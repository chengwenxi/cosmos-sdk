package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	channel "github.com/cosmos/cosmos-sdk/x/ibc/04-channel"
	commitment "github.com/cosmos/cosmos-sdk/x/ibc/23-commitment"
)

func RegisterCodec(cdc *codec.Codec) {
	cdc.RegisterConcrete(MsgRecvPacket{}, "ibcmockbank/MsgRecvPacket", nil)
}

var ModuleCdc = codec.New()

func init() {
	RegisterCodec(ModuleCdc)
	channel.RegisterCodec(ModuleCdc)
	commitment.RegisterCodec(ModuleCdc)
}
