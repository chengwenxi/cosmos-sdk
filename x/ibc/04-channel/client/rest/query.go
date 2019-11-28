package rest

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/types/rest"
	"github.com/cosmos/cosmos-sdk/x/ibc/04-channel/types"
)

func registerQueryRoutes(cliCtx context.CLIContext, r *mux.Router, queryRoute string) {
	r.HandleFunc(fmt.Sprintf("/ibc/ports/{%s}/channels/{%s}", RestPortID, RestChannelID), queryChannelHandlerFn(cliCtx, queryRoute)).Methods("GET")
}

// queryChannelHandlerFn implements a channel querying route
//
// @Summary Query channel
// @Tags IBC
// @Produce  json
// @Param port-id path string true "Port ID"
// @Param channel-id path string true "Channel ID"
// @Param prove query boolean false "Proof of result"
// @Success 200 {object} QueryChannel "OK"
// @Failure 400 {object} rest.ErrorResponse "Invalid port id or channel id"
// @Failure 500 {object} rest.ErrorResponse "Internal Server Error"
// @Router /ibc/ports/{port-id}/channels/{channel-id} [get]
func queryChannelHandlerFn(cliCtx context.CLIContext, queryRoute string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		portID := vars[RestPortID]
		channelID := vars[RestChannelID]

		cliCtx, ok := rest.ParseQueryHeightOrReturnBadRequest(w, cliCtx, r)
		if !ok {
			return
		}

		req := abci.RequestQuery{
			Path:  "store/ibc/key",
			Data:  types.KeyChannel(portID, channelID),
			Prove: rest.ParseQueryProve(r),
		}

		res, err := cliCtx.QueryABCI(req)
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		var channel types.Channel
		if err := cliCtx.Codec.UnmarshalBinaryLengthPrefixed(res.Value, &channel); err != nil {
			rest.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		cliCtx = cliCtx.WithHeight(res.Height)
		rest.PostProcessResponse(w, cliCtx, types.NewChannelResponse(portID, channelID, channel, res.Proof, res.Height))
	}
}
