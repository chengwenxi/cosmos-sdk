package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypestm "github.com/cosmos/cosmos-sdk/x/ibc/02-client/types/tendermint"
	connection "github.com/cosmos/cosmos-sdk/x/ibc/03-connection"
	channel "github.com/cosmos/cosmos-sdk/x/ibc/04-channel"
	"github.com/cosmos/cosmos-sdk/x/ibc/20-transfer/types"
	commitment "github.com/cosmos/cosmos-sdk/x/ibc/23-commitment"
	abci "github.com/tendermint/tendermint/abci/types"
)

func (suite *KeeperTestSuite) createClient() {
	suite.app.Commit()
	commitID := suite.app.LastCommitID()

	suite.app.BeginBlock(abci.RequestBeginBlock{Header: abci.Header{Height: suite.app.LastBlockHeight() + 1}})
	suite.ctx = suite.app.BaseApp.NewContext(false, abci.Header{})

	consensusState := clienttypestm.ConsensusState{
		ChainID: testChainID,
		Height:  uint64(commitID.Version),
		Root:    commitment.NewRoot(commitID.Hash),
	}

	_, err := suite.app.IBCKeeper.ClientKeeper.CreateClient(suite.ctx, testClient, testClientType, consensusState)
	suite.NoError(err)
}

func (suite *KeeperTestSuite) createConnection(state connection.State) {
	connection := connection.ConnectionEnd{
		State:    state,
		ClientID: testClient,
		Counterparty: connection.Counterparty{
			ClientID:     testClient,
			ConnectionID: testConnection,
			Prefix:       suite.app.IBCKeeper.ConnectionKeeper.GetCommitmentPrefix(),
		},
		Versions: connection.GetCompatibleVersions(),
	}

	suite.app.IBCKeeper.ConnectionKeeper.SetConnection(suite.ctx, testConnection, connection)
}

func (suite *KeeperTestSuite) createChannel(portID string, chanID string, connID string, counterpartyPort string, counterpartyChan string, state channel.State) {
	ch := channel.Channel{
		State:    state,
		Ordering: testChannelOrder,
		Counterparty: channel.Counterparty{
			PortID:    counterpartyPort,
			ChannelID: counterpartyChan,
		},
		ConnectionHops: []string{connID},
		Version:        testChannelVersion,
	}

	suite.app.IBCKeeper.ChannelKeeper.SetChannel(suite.ctx, portID, chanID, ch)
}

func (suite *KeeperTestSuite) TestSendTransfer() {
	// test the situation where the source is true
	isSourceChain := true

	err := suite.app.IBCKeeper.TransferKeeper.SendTransfer(suite.ctx, testPort1, testChannel1, testCoins, testAddr1, testAddr2, isSourceChain)
	suite.NotNil(err) // channel does not exist

	suite.createChannel(testPort1, testChannel1, testConnection, testPort2, testChannel2, channel.OPEN)
	err = suite.app.IBCKeeper.TransferKeeper.SendTransfer(suite.ctx, testPort1, testChannel1, testCoins, testAddr1, testAddr2, isSourceChain)
	suite.NotNil(err) // next send sequence not found

	nextSeqSend := uint64(1)
	suite.app.IBCKeeper.ChannelKeeper.SetNextSequenceSend(suite.ctx, testPort1, testChannel1, nextSeqSend)
	err = suite.app.IBCKeeper.TransferKeeper.SendTransfer(suite.ctx, testPort1, testChannel1, testCoins, testAddr1, testAddr2, isSourceChain)
	suite.NotNil(err) // sender has insufficient coins

	_ = suite.app.BankKeeper.SetCoins(suite.ctx, testAddr1, testCoins)
	err = suite.app.IBCKeeper.TransferKeeper.SendTransfer(suite.ctx, testPort1, testChannel1, testCoins, testAddr1, testAddr2, isSourceChain)
	suite.Nil(err) // successfully executed

	senderCoins := suite.app.BankKeeper.GetCoins(suite.ctx, testAddr1)
	suite.Equal(sdk.Coins(nil), senderCoins)

	escrowCoins := suite.app.BankKeeper.GetCoins(suite.ctx, types.GetEscrowAddress(testPort1, testChannel1))
	suite.Equal(testCoins, escrowCoins)

	newNextSeqSend, found := suite.app.IBCKeeper.ChannelKeeper.GetNextSequenceSend(suite.ctx, testPort1, testChannel1)
	suite.True(found)
	suite.Equal(nextSeqSend+1, newNextSeqSend)

	packetCommitment := suite.app.IBCKeeper.ChannelKeeper.GetPacketCommitment(suite.ctx, testPort1, testChannel1, nextSeqSend)
	suite.NotNil(packetCommitment)

	// test the situation where the source is false
	isSourceChain = false

	_ = suite.app.BankKeeper.SetCoins(suite.ctx, testAddr1, testPrefixedCoins2)
	err = suite.app.IBCKeeper.TransferKeeper.SendTransfer(suite.ctx, testPort1, testChannel1, testPrefixedCoins2, testAddr1, testAddr2, isSourceChain)
	suite.NotNil(err) // incorrect denom prefix

	_ = suite.app.BankKeeper.SetCoins(suite.ctx, testAddr1, testPrefixedCoins1)
	err = suite.app.IBCKeeper.TransferKeeper.SendTransfer(suite.ctx, testPort1, testChannel1, testPrefixedCoins1, testAddr1, testAddr2, isSourceChain)
	suite.Nil(err) // successfully executed

	senderCoins = suite.app.BankKeeper.GetCoins(suite.ctx, testAddr1)
	suite.Equal(sdk.Coins(nil), senderCoins)
}

func (suite *KeeperTestSuite) TestReceiveTransfer() {
	packetData := types.PacketData{
		Amount:   testPrefixedCoins1,
		Sender:   testAddr1,
		Receiver: testAddr2,
	}

	// test the situation where the source is true
	packetData.Source = true

	err := suite.app.IBCKeeper.TransferKeeper.ReceiveTransfer(suite.ctx, testPort1, testChannel1, testPort2, testChannel2, packetData)
	suite.NotNil(err) // incorrect denom prefix

	packetData.Amount = testPrefixedCoins2
	err = suite.app.IBCKeeper.TransferKeeper.ReceiveTransfer(suite.ctx, testPort1, testChannel1, testPort2, testChannel2, packetData)
	suite.Nil(err) // successfully executed

	receiverCoins := suite.app.BankKeeper.GetCoins(suite.ctx, packetData.Receiver)
	suite.Equal(testPrefixedCoins2, receiverCoins)

	// test the situation where the source is false
	packetData.Source = false

	packetData.Amount = testPrefixedCoins2
	err = suite.app.IBCKeeper.TransferKeeper.ReceiveTransfer(suite.ctx, testPort1, testChannel1, testPort2, testChannel2, packetData)
	suite.NotNil(err) // incorrect denom prefix

	packetData.Amount = testPrefixedCoins1
	err = suite.app.IBCKeeper.TransferKeeper.ReceiveTransfer(suite.ctx, testPort1, testChannel1, testPort2, testChannel2, packetData)
	suite.NotNil(err) // insufficient coins in the corresponding escrow account

	escrowAddress := types.GetEscrowAddress(testPort2, testChannel2)
	_ = suite.app.BankKeeper.SetCoins(suite.ctx, escrowAddress, testPrefixedCoins1)
	_ = suite.app.BankKeeper.SetCoins(suite.ctx, packetData.Receiver, sdk.Coins{})
	err = suite.app.IBCKeeper.TransferKeeper.ReceiveTransfer(suite.ctx, testPort1, testChannel1, testPort2, testChannel2, packetData)
	suite.Nil(err) // successfully executed

	escrowCoins := suite.app.BankKeeper.GetCoins(suite.ctx, escrowAddress)
	suite.Equal(sdk.Coins{}, escrowCoins)

	receiverCoins = suite.app.BankKeeper.GetCoins(suite.ctx, packetData.Receiver)
	suite.Equal(testPrefixedCoins1, receiverCoins)
}
