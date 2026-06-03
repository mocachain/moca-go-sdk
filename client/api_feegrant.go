package client

import (
	"context"
	"time"

	"cosmossdk.io/math"
	"cosmossdk.io/x/feegrant"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cmdcfg "github.com/mocachain/moca/v2/cmd/config"
	gnfdsdktypes "github.com/mocachain/moca/v2/sdk/types"
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
	amoca := sdk.NewCoins(sdk.NewCoin(gnfdsdktypes.Denom, feeAllowanceAmount))
	allowance := feegrant.BasicAllowance{
		SpendLimit: amoca,
		Expiration: expiration,
	}
	msg, err := newFeeGrantAllowanceMsg(c.defaultAccount.GetAddress(), granteeAddr, &allowance)
	if err != nil {
		return "", err
	}
	return c.sendTxn(ctx, msg, &txOption)
}

// GrantAllowance provides a generic way to grant different types of allowance(BasicAllowance, PeriodicAllowance, AllowedMsgAllowance), the user needs to construct the desired type of allowance
func (c *Client) GrantAllowance(ctx context.Context, granteeAddr string, allowance feegrant.FeeAllowanceI, txOption gnfdsdktypes.TxOption) (string, error) {
	msg, err := newFeeGrantAllowanceMsg(c.defaultAccount.GetAddress(), granteeAddr, allowance)
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
	msg, err := newFeeGrantRevokeMsg(c.defaultAccount.GetAddress(), granteeAddr)
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
	response, err := c.chainClient.Allowance(ctx, req)
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
	response, err := c.chainClient.Allowances(ctx, req)
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
	response, err := c.chainClient.AllowancesByGranter(ctx, req)
	if err != nil {
		return nil, err
	}
	return response.Allowances, nil
}

var feeGrantAddressCodec = cmdcfg.NewMultiPrefixBech32AccCodec()

func newFeeGrantAllowanceMsg(granter sdk.AccAddress, granteeAddr string, allowance feegrant.FeeAllowanceI) (*feegrant.MsgGrantAllowance, error) {
	grantee, granteeAddrBech32, err := normalizeFeeGrantAccountAddress(granteeAddr)
	if err != nil {
		return nil, err
	}
	_, granterAddrBech32, err := normalizeFeeGrantAccountAddress(granter.String())
	if err != nil {
		return nil, err
	}

	msg, err := feegrant.NewMsgGrantAllowance(allowance, granter, grantee)
	if err != nil {
		return nil, err
	}
	msg.Grantee = granteeAddrBech32
	msg.Granter = granterAddrBech32
	return msg, nil
}

func newFeeGrantRevokeMsg(granter sdk.AccAddress, granteeAddr string) (feegrant.MsgRevokeAllowance, error) {
	grantee, granteeAddrBech32, err := normalizeFeeGrantAccountAddress(granteeAddr)
	if err != nil {
		return feegrant.MsgRevokeAllowance{}, err
	}
	_, granterAddrBech32, err := normalizeFeeGrantAccountAddress(granter.String())
	if err != nil {
		return feegrant.MsgRevokeAllowance{}, err
	}

	msg := feegrant.NewMsgRevokeAllowance(granter, grantee)
	msg.Grantee = granteeAddrBech32
	msg.Granter = granterAddrBech32
	return msg, nil
}

func normalizeFeeGrantAccountAddress(addr string) (sdk.AccAddress, string, error) {
	acc, err := sdk.AccAddressFromHexUnsafe(addr)
	if err != nil {
		bz, err := feeGrantAddressCodec.StringToBytes(addr)
		if err != nil {
			return nil, "", err
		}
		if err := sdk.VerifyAddressFormat(bz); err != nil {
			return nil, "", err
		}
		acc = sdk.AccAddress(bz)
	}
	addrBech32, err := feeGrantAddressCodec.BytesToString(acc)
	if err != nil {
		return nil, "", err
	}
	return acc, addrBech32, nil
}
