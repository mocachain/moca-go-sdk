package client

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gnfdSdkTypes "github.com/evmos/evmos/v12/sdk/types"
	mocadTypes "github.com/evmos/evmos/v12/types"
	"github.com/evmos/evmos/v12/x/evm/precompiles/payment"
	paymentTypes "github.com/evmos/evmos/v12/x/payment/types"
	"github.com/rs/zerolog/log"
	"github.com/mocachain/moca-go-sdk/pkg/utils"
	"github.com/mocachain/moca-go-sdk/types"
)

// IPaymentClient - Client APIs for operating and querying Moca payment accounts and stream records.
type IPaymentClient interface {
	GetStreamRecord(ctx context.Context, streamAddress string) (*paymentTypes.StreamRecord, error)
	Deposit(ctx context.Context, toAddress string, amount math.Int, txOption gnfdSdkTypes.TxOption) (string, error)
	Withdraw(ctx context.Context, fromAddress string, amount math.Int, txOption gnfdSdkTypes.TxOption) (string, error)
	DisableRefund(ctx context.Context, paymentAddress string, txOption gnfdSdkTypes.TxOption) (string, error)
	ListUserPaymentAccounts(ctx context.Context, opts types.ListUserPaymentAccountsOptions) (types.ListUserPaymentAccountsResult, error)
}

// GetStreamRecord - Retrieve stream record information for a given stream address.
//
// - ctx: Context variables for the current API call.
//
// - streamAddress: The address of the stream record to be queried.
//
// - ret1: The stream record information, including balances and net flow rate.
//
// - ret2: Return error when getting challenge info failed, otherwise return nil.
func (c *Client) GetStreamRecord(ctx context.Context, streamAddress string) (*paymentTypes.StreamRecord, error) {
	accAddress, err := sdk.AccAddressFromHexUnsafe(streamAddress)
	if err != nil {
		return nil, err
	}
	pa, err := c.chainClient.StreamRecord(ctx, &paymentTypes.QueryGetStreamRecordRequest{Account: accAddress.String()})
	if err != nil {
		return nil, err
	}
	return &pa.StreamRecord, nil
}

// Deposit - Deposit amoca to a payment account.
//
// - ctx: Context variables for the current API call.
//
// - toAddress: The address of the stream record to receive the deposit.
//
// - amount: The amount to deposit.
//
// - txOption: The options for sending the tx.
//
// - ret1: The response of Moca transaction.
//
// - ret2: Return error when deposit tx failed, otherwise return nil.
func (c *Client) Deposit(ctx context.Context, toAddress string, amount math.Int, txOption gnfdSdkTypes.TxOption) (string, error) {
	accAddress, err := sdk.AccAddressFromHexUnsafe(toAddress)
	if err != nil {
		return "", err
	}
	msgDeposit := &paymentTypes.MsgDeposit{
		Creator: c.MustGetDefaultAccount().GetAddress().String(),
		To:      accAddress.String(),
		Amount:  amount,
	}
	// return c.sendTxn(ctx, msgDeposit, &txOption)
	return c.sendDepositEvmTxn(ctx, msgDeposit)
}

func (c *Client) sendDepositEvmTxn(ctx context.Context, msg *paymentTypes.MsgDeposit) (string, error) {
	session, err := c.createPaymentEvmSession(ctx, c.privateKey)
	if err != nil {
		return "", err
	}
	txRsp, err := session.Deposit(msg.To, msg.Amount.BigInt())
	if err != nil {
		return "", err
	}
	return txRsp.Hash().String(), nil
}

// Withdraw - Withdraws amoca from a payment account.
//
// Withdrawal will trigger settlement, i.e., updating static balance and buffer balance.
// If the withdrawal amount is greater than the static balance after settlement it will fail.
// If the withdrawal amount is equal to or greater than 100amoca, it will be timelock-ed for 1 day duration.
// And after the duration, a message without `from` field should be sent to get the funds.
//
// - ctx: Context variables for the current API call.
//
// - fromAddress: The address of the stream record to withdraw from.
//
// - amount: The amount to withdraw.
//
// - txOption: The options for sending the tx.
//
// - ret1: The response of Moca transaction.
//
// - ret2: Return error when withdrawal tx failed, otherwise return nil.
func (c *Client) Withdraw(ctx context.Context, fromAddress string, amount math.Int, txOption gnfdSdkTypes.TxOption) (string, error) {
	accAddress, err := sdk.AccAddressFromHexUnsafe(fromAddress)
	if err != nil {
		return "", err
	}
	msgWithdraw := &paymentTypes.MsgWithdraw{
		Creator: c.MustGetDefaultAccount().GetAddress().String(),
		From:    accAddress.String(),
		Amount:  amount,
	}
	// return c.sendTxn(ctx, msgWithdraw, &txOption)
	return c.sendWithdrawEvmTxn(ctx, msgWithdraw)
}

func (c *Client) sendWithdrawEvmTxn(ctx context.Context, msg *paymentTypes.MsgWithdraw) (string, error) {
	session, err := c.createPaymentEvmSession(ctx, c.privateKey)
	if err != nil {
		return "", err
	}
	txRsp, err := session.Withdraw(msg.From, msg.Amount.BigInt())
	if err != nil {
		return "", err
	}
	return txRsp.Hash().String(), nil
}

// DisableRefund - Disable refund/withdrawal for a payment account.
//
// After disabling withdrawal of a payment account, no more withdrawal can be executed. The action cannot be reverted.
//
// - ctx: Context variables for the current API call.
//
// - paymentAddress: The address of the payment account to disable refund/withdrawal.
//
// - txOption: The options for sending the tx.
//
// - ret1: The response of Moca transaction.
//
// - ret2: Return error when disable refund tx failed, otherwise return nil.
func (c *Client) DisableRefund(ctx context.Context, paymentAddress string, txOption gnfdSdkTypes.TxOption) (string, error) {
	accAddress, err := sdk.AccAddressFromHexUnsafe(paymentAddress)
	if err != nil {
		return "", err
	}
	msgDisableRefund := &paymentTypes.MsgDisableRefund{
		Owner: c.MustGetDefaultAccount().GetAddress().String(),
		Addr:  accAddress.String(),
	}
	tx, err := c.BroadcastTx(ctx, []sdk.Msg{msgDisableRefund}, &txOption)
	if err != nil {
		return "", err
	}
	return tx.TxResponse.TxHash, nil
}

// ListUserPaymentAccounts - List payment info by a user address.
//
// - ctx: Context variables for the current API call.
//
// - opts: The options to define the user address for querying.
//
// - ret1: The response of streams records for the user address.
//
// - ret2: Return error when querying payment accounts failed, otherwise return nil.
func (c *Client) ListUserPaymentAccounts(ctx context.Context, opts types.ListUserPaymentAccountsOptions) (types.ListUserPaymentAccountsResult, error) {
	params := url.Values{}
	params.Set("user-payments", "")

	account := opts.Account
	if account == "" {
		acc, err := c.GetDefaultAccount()
		if err != nil {
			log.Error().Msg(fmt.Sprintf("failed to get default account: %s", err.Error()))
			return types.ListUserPaymentAccountsResult{}, err
		}
		account = acc.GetAddress().String()
	}

	reqMeta := requestMeta{
		urlValues:     params,
		contentSHA256: types.EmptyStringSHA256,
		userAddress:   account,
	}

	sendOpt := sendOptions{
		method:           http.MethodGet,
		disableCloseBody: true,
	}

	endpoint, err := c.getEndpointByOpt(&types.EndPointOptions{
		Endpoint:  opts.Endpoint,
		SPAddress: opts.SPAddress,
	})
	if err != nil {
		log.Error().Msg(fmt.Sprintf("get endpoint by option failed %s", err.Error()))
		return types.ListUserPaymentAccountsResult{}, err
	}

	resp, err := c.sendReq(ctx, reqMeta, &sendOpt, endpoint)
	if err != nil {
		return types.ListUserPaymentAccountsResult{}, errors.New("send request error" + err.Error())
	}
	defer utils.CloseResponse(resp)

	paymentAccounts := types.ListUserPaymentAccountsResult{}
	// unmarshal the json content from response body
	buf := new(strings.Builder)
	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		return types.ListUserPaymentAccountsResult{}, errors.New("unmarshal response error" + err.Error())
	}

	bufStr := buf.String()
	err = xml.Unmarshal([]byte(bufStr), &paymentAccounts)
	if err != nil {
		return types.ListUserPaymentAccountsResult{}, err
	}

	return paymentAccounts, nil
}

func (c *Client) createPaymentEvmSession(ctx context.Context, privKey string) (*payment.IPaymentSession, error) {
	nonce, err := c.chainClient.GetNonce(context.Background())
	if err != nil {
		return nil, err
	}
	chainId, err := c.evmClient.ChainID(ctx)
	if err != nil {
		return nil, err
	}

	txOpts, err := CreateTxOpts(context.Background(), c.evmClient, privKey, chainId, DefaultGasLimit, nonce)
	if err != nil {
		return nil, err
	}

	session, err := CreatePaymentSession(c.evmClient, *txOpts, mocadTypes.PaymentAddress)
	if err != nil {
		return nil, err
	}
	return session, nil
}
