package client

import (
	"context"
	"time"

	"cosmossdk.io/math"
	"cosmossdk.io/x/feegrant"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cmdcfg "github.com/mocachain/moca/v2/cmd/config"
	gnfdsdktypes "github.com/mocachain/moca/v2/sdk/types"
	mocautils "github.com/mocachain/moca/v2/utils"
)

type IFeeGrantClient interface {
	GrantBasicAllowance(ctx context.Context, granteeAddr string, feeAllowanceAmount math.Int, expiration *time.Time, txOption gnfdsdktypes.TxOption) (string, error)
	QueryBasicAllowance(ctx context.Context, granterAddr, granteeAddr string) (*feegrant.BasicAllowance, error)

	// for generic allowance(BasicAllowance, PeriodicAllowance, AllowedMsgAllowance)
	GrantAllowance(ctx context.Context, granteeAddr string, allowance feegrant.FeeAllowanceI, txOption gnfdsdktypes.TxOption) (string, error)
	QueryAllowance(ctx context.Context, granterAddr, granteeAddr string) (*feegrant.Grant, error)
	QueryAllowances(ctx context.Context, granteeAddr string) ([]*feegrant.Grant, error)

	RevokeAllowance(ctx context.Context, granteeAddr string, txOption gnfdsdktypes.TxOption) (string, error)
}

// GrantBasicAllowance grants the grantee the BasicAllowance with specified amount and expiration.
func (c *Client) GrantBasicAllowance(ctx context.Context, granteeAddr string, feeAllowanceAmount math.Int, expiration *time.Time, txOption gnfdsdktypes.TxOption) (string, error) {
	grantee, granteeAddrBech32, err := normalizeFeeGrantAccountAddress(granteeAddr)
	if err != nil {
		return "", err
	}
	amoca := sdk.NewCoins(sdk.NewCoin(gnfdsdktypes.Denom, feeAllowanceAmount))
	allowance := feegrant.BasicAllowance{
		SpendLimit: amoca,
		Expiration: expiration,
	}
	msg, err := feegrant.NewMsgGrantAllowance(&allowance, c.defaultAccount.GetAddress(), grantee)
	if err != nil {
		return "", err
	}
	msg.Grantee = granteeAddrBech32
	msg.Granter, err = feeGrantAddressCodec.BytesToString(c.defaultAccount.GetAddress())
	if err != nil {
		return "", err
	}
	return c.sendTxn(ctx, msg, &txOption)
}

// GrantAllowance provides a generic way to grant different types of allowance(BasicAllowance, PeriodicAllowance, AllowedMsgAllowance), the user needs to construct the desired type of allowance
func (c *Client) GrantAllowance(ctx context.Context, granteeAddr string, allowance feegrant.FeeAllowanceI, txOption gnfdsdktypes.TxOption) (string, error) {
	grantee, granteeAddrBech32, err := normalizeFeeGrantAccountAddress(granteeAddr)
	if err != nil {
		return "", err
	}
	msg, err := feegrant.NewMsgGrantAllowance(allowance, c.defaultAccount.GetAddress(), grantee)
	if err != nil {
		return "", err
	}
	msg.Grantee = granteeAddrBech32
	msg.Granter, err = feeGrantAddressCodec.BytesToString(c.defaultAccount.GetAddress())
	if err != nil {
		return "", err
	}
	resp, err := c.BroadcastTx(ctx, []sdk.Msg{msg}, &txOption)
	if err != nil {
		return "", err
	}
	return resp.TxResponse.TxHash, nil
}

// RevokeAllowance revokes allowance on a grantee by the granter
func (c *Client) RevokeAllowance(ctx context.Context, granteeAddr string, txOption gnfdsdktypes.TxOption) (string, error) {
	grantee, granteeAddrBech32, err := normalizeFeeGrantAccountAddress(granteeAddr)
	if err != nil {
		return "", err
	}
	msg := feegrant.NewMsgRevokeAllowance(c.defaultAccount.GetAddress(), grantee)
	msg.Grantee = granteeAddrBech32
	msg.Granter, err = feeGrantAddressCodec.BytesToString(c.defaultAccount.GetAddress())
	if err != nil {
		return "", err
	}
	resp, err := c.BroadcastTx(ctx, []sdk.Msg{&msg}, &txOption)
	if err != nil {
		return "", err
	}
	return resp.TxResponse.TxHash, nil
}

// QueryBasicAllowance queries the BasicAllowance
func (c *Client) QueryBasicAllowance(ctx context.Context, granterAddr, granteeAddr string) (*feegrant.BasicAllowance, error) {
	allowance, err := c.QueryAllowance(ctx, granterAddr, granteeAddr)
	if err != nil {
		return nil, err
	}
	basicAllowance := &feegrant.BasicAllowance{}
	if err = c.chainClient.GetCodec().Unmarshal(allowance.Allowance.GetValue(), basicAllowance); err != nil {
		return nil, err
	}
	return basicAllowance, nil
}

func (c *Client) QueryAllowance(ctx context.Context, granterAddr, granteeAddr string) (*feegrant.Grant, error) {
	_, granterAddrBech32, err := normalizeFeeGrantAccountAddress(granterAddr)
	if err != nil {
		return nil, err
	}
	_, granteeAddrBech32, err := normalizeFeeGrantAccountAddress(granteeAddr)
	if err != nil {
		return nil, err
	}
	req := &feegrant.QueryAllowanceRequest{
		Granter: granterAddrBech32,
		Grantee: granteeAddrBech32,
	}
	response, err := c.chainClient.FeegrantQueryClient.Allowance(ctx, req)
	if err != nil {
		return nil, err
	}
	return response.Allowance, nil
}

func (c *Client) QueryAllowances(ctx context.Context, granteeAddr string) ([]*feegrant.Grant, error) {
	_, granteeAddrBech32, err := normalizeFeeGrantAccountAddress(granteeAddr)
	if err != nil {
		return nil, err
	}
	req := &feegrant.QueryAllowancesRequest{
		Grantee: granteeAddrBech32,
	}
	response, err := c.chainClient.FeegrantQueryClient.Allowances(ctx, req)
	if err != nil {
		return nil, err
	}
	return response.Allowances, nil
}

func (c *Client) QueryGranterAllowances(ctx context.Context, granterAddr string) ([]*feegrant.Grant, error) {
	_, granterAddrBech32, err := normalizeFeeGrantAccountAddress(granterAddr)
	if err != nil {
		return nil, err
	}
	req := &feegrant.QueryAllowancesByGranterRequest{
		Granter: granterAddrBech32,
	}
	response, err := c.chainClient.FeegrantQueryClient.AllowancesByGranter(ctx, req)
	if err != nil {
		return nil, err
	}
	return response.Allowances, nil
}

var feeGrantAddressCodec = cmdcfg.NewMultiPrefixBech32AccCodec()

func normalizeFeeGrantAccountAddress(addr string) (sdk.AccAddress, string, error) {
	acc, err := sdk.AccAddressFromHexUnsafe(addr)
	if err != nil {
		acc, err = mocautils.GetMocaAddressFromBech32(addr)
		if err != nil {
			return nil, "", err
		}
	}

	bech32Addr, err := feeGrantAddressCodec.BytesToString(acc)
	if err != nil {
		return nil, "", err
	}
	return acc, bech32Addr, nil
}
