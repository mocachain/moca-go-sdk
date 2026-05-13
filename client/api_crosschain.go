package client

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	oracletypes "github.com/cosmos/cosmos-sdk/x/oracle/types"
	gnfdSdkTypes "github.com/mocachain/moca/v2/sdk/types"
)

type ICrossChainClient interface {
	Claims(ctx context.Context, srcShainId, destChainId uint32, sequence uint64, timestamp uint64, payload []byte, voteAddrSet []uint64, aggSignature []byte, txOption gnfdSdkTypes.TxOption) (*sdk.TxResponse, error)
}

// Claims is the only cross-chain helper that remains exposed in moca/v2.
// Legacy mirror/query APIs were removed together with the bridge-module-based flows
// so callers do not compile against methods that are guaranteed to fail at runtime.
// Claims - Claim cross-chain packages from BSC to Moca, used by relayers which run by validators
//
// - ctx: Context variables for the current API call.
//
// - srcChainId: The source chain id.
//
// - destChainId: The destination chain id.
//
// - sequence: The sequence of the claim.
//
// - timestamp: The timestamp of the cross-chain packages.
//
// - payload: The payload of the claim.
//
// - voteAddrSet: The bitset of the voted validators.
//
// - aggSignature: The aggregated bls signature of the claim.
//
// - txOption: The txOption for sending transactions.
//
// - ret1: Transaction response from Moca.
//
// - ret2: Return error if transaction failed, otherwise return nil.
func (c *Client) Claims(ctx context.Context, srcChainId, destChainId uint32, sequence uint64,
	timestamp uint64, payload []byte, voteAddrSet []uint64, aggSignature []byte, txOption gnfdSdkTypes.TxOption,
) (*sdk.TxResponse, error) {
	msg := oracletypes.NewMsgClaim(
		c.MustGetDefaultAccount().GetAddress().String(),
		srcChainId,
		destChainId,
		sequence,
		timestamp,
		payload,
		voteAddrSet,
		aggSignature)

	txResp, err := c.BroadcastTx(ctx, []sdk.Msg{msg}, &txOption)
	if err != nil {
		return nil, err
	}
	return txResp.TxResponse, nil
}
