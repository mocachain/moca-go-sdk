package client

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"cosmossdk.io/errors"
	protov2 "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"github.com/cometbft/cometbft/proto/tendermint/p2p"
	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	bfttypes "github.com/cometbft/cometbft/types"
	"github.com/cometbft/cometbft/votepool"
	sdkclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	clitx "github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	xauthsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"google.golang.org/grpc"
	txsigning "cosmossdk.io/x/tx/signing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	gosdktypes "github.com/mocachain/moca-go-sdk/types"
	mocacmdconfig "github.com/mocachain/moca/v2/cmd/config"
	"github.com/mocachain/moca/v2/sdk/types"
	"github.com/mocachain/moca/v2/x/evm/precompiles/storage"
	storageTypes "github.com/mocachain/moca/v2/x/storage/types"
)

// IBasicClient interface defines basic functions of moca Client.
type IBasicClient interface {
	EnableTrace(outputStream io.Writer, onlyTraceErr bool)

	GetNodeInfo(ctx context.Context) (*p2p.DefaultNodeInfo, *cmtservice.VersionInfo, error)
	GetStatus(ctx context.Context) (*ctypes.ResultStatus, error)
	GetCommit(ctx context.Context, height int64) (*ctypes.ResultCommit, error)
	GetLatestBlockHeight(ctx context.Context) (int64, error)
	GetLatestBlock(ctx context.Context) (*bfttypes.Block, error)
	GetSyncing(ctx context.Context) (bool, error)
	GetBlockByHeight(ctx context.Context, height int64) (*bfttypes.Block, error)
	GetBlockResultByHeight(ctx context.Context, height int64) (*ctypes.ResultBlockResults, error)

	GetValidatorSet(ctx context.Context) (int64, []*bfttypes.Validator, error)
	GetValidatorsByHeight(ctx context.Context, height int64) ([]*bfttypes.Validator, error)

	WaitForBlockHeight(ctx context.Context, height int64) error
	WaitForTx(ctx context.Context, hash string) (*ctypes.ResultTx, error)
	WaitForNBlocks(ctx context.Context, n int64) error
	WaitForNextBlock(ctx context.Context) error

	SimulateTx(ctx context.Context, msgs []sdk.Msg, txOpt types.TxOption, opts ...grpc.CallOption) (*txtypes.SimulateResponse, error)
	SimulateRawTx(ctx context.Context, txBytes []byte, opts ...grpc.CallOption) (*txtypes.SimulateResponse, error)
	BroadcastTx(ctx context.Context, msgs []sdk.Msg, txOpt *types.TxOption, opts ...grpc.CallOption) (*txtypes.BroadcastTxResponse, error)
	BroadcastRawTx(ctx context.Context, txBytes []byte, sync bool) (*sdk.TxResponse, error)

	BroadcastVote(ctx context.Context, vote votepool.Vote) error
	QueryVote(ctx context.Context, eventType int, eventHash []byte) (*ctypes.ResultQueryVote, error)
	SetTag(ctx context.Context, resourceGRN string, tags storageTypes.ResourceTags, opts gosdktypes.SetTagsOptions) (string, error)
}

// EnableTrace support trace error info the request and the response
func (c *Client) EnableTrace(output io.Writer, onlyTraceErr bool) {
	if output == nil {
		output = os.Stdout
	}

	c.onlyTraceError = onlyTraceErr

	c.traceOutput = output
	c.isTraceEnabled = true
}

// GetNodeInfo - Get the current node info of the moca that the Client is connected to.
//
// - ctx: Context variables for the current API call.
//
// - ret1: The Node info.
//
// - ret2: The Version info.
//
// - ret3: Return error when the request failed, otherwise return nil.
func (c *Client) GetNodeInfo(ctx context.Context) (*p2p.DefaultNodeInfo, *cmtservice.VersionInfo, error) {
	nodeInfoResponse, err := c.chainClient.TmClient.GetNodeInfo(ctx, &cmtservice.GetNodeInfoRequest{})
	if err != nil {
		return nil, nil, err
	}
	return nodeInfoResponse.DefaultNodeInfo, nodeInfoResponse.ApplicationVersion, nil
}

// GetStatus - Get the status of connected Node.
//
// - ctx: Context variables for the current API call.
//
// - ret1: The detail of Node status.
//
// - ret2: Return error when the request failed, otherwise return nil.
func (c *Client) GetStatus(ctx context.Context) (*ctypes.ResultStatus, error) {
	return c.chainClient.GetStatus(ctx)
}

// GetCommit - Get the block commit detail.
//
// - ctx: Context variables for the current API call.
//
// - height: The block height.
//
// - ret1: The commit result.
//
// - ret2: Return error when the request failed, otherwise return nil.
func (c *Client) GetCommit(ctx context.Context, height int64) (*ctypes.ResultCommit, error) {
	return c.chainClient.GetCommit(ctx, height)
}

// BroadcastRawTx - Broadcast raw transaction bytes to a Tendermint node.
//
// - ctx: Context variables for the current API call.
//
// - txBytes: The transaction bytes.
//
// - sync: A flag to specify the transaction mode. If it is true, the transaction is broadcast synchronously. If it is false, the transaction is broadcast asynchronously.
//
// - ret1: Transaction response, it can indicate both success and failed transaction.
//
// - ret2: Return error when the request failed, otherwise return nil.
func (c *Client) BroadcastRawTx(ctx context.Context, txBytes []byte, sync bool) (*sdk.TxResponse, error) {
	var mode txtypes.BroadcastMode
	if sync {
		mode = txtypes.BroadcastMode_BROADCAST_MODE_SYNC
	} else {
		mode = txtypes.BroadcastMode_BROADCAST_MODE_ASYNC
	}
	broadcastTxResponse, err := c.chainClient.TxClient.BroadcastTx(ctx, &txtypes.BroadcastTxRequest{TxBytes: txBytes, Mode: mode})
	if err != nil {
		return nil, err
	}
	return broadcastTxResponse.TxResponse, nil
}

// SimulateRawTx - Simulate the execution of a raw transaction on the blockchain without broadcasting it to the network.
//
// - ctx: Context variables for the current API call.
//
// - txBytes: The transaction bytes.
//
// - opts: The grpc option(s) if Client is using grpc connection.
//
// - ret1: The simulation result.
//
// - ret2: Return error when the request failed, otherwise return nil.
func (c *Client) SimulateRawTx(ctx context.Context, txBytes []byte, opts ...grpc.CallOption) (*txtypes.SimulateResponse, error) {
	simulateResponse, err := c.chainClient.TxClient.Simulate(
		ctx,
		&txtypes.SimulateRequest{
			TxBytes: txBytes,
		},
		opts...,
	)
	if err != nil {
		return nil, err
	}
	return simulateResponse, nil
}

// GetLatestBlock - Get the latest block from the chain.
//
// - ctx: Context variables for the current API call.
//
// - ret1: The block result.
//
// - ret2: Return error when the request failed, otherwise return nil.
func (c *Client) GetLatestBlock(ctx context.Context) (*bfttypes.Block, error) {
	res, err := c.chainClient.GetBlock(ctx, nil)
	if err != nil {
		return nil, err
	}
	return res.Block, nil
}

// GetLatestBlockHeight - Get the height of the latest block from the chain.
//
// - ctx: Context variables for the current API call.
//
// - ret1: The block height.
//
// - ret2: Return error when the request failed, otherwise return nil.
func (c *Client) GetLatestBlockHeight(ctx context.Context) (int64, error) {
	resp, err := c.chainClient.GetStatus(ctx)
	if err != nil {
		return 0, errors.Wrap(err, "get latest block height")
	}
	return resp.SyncInfo.LatestBlockHeight, nil
}

// WaitForBlockHeight - Wait until a specified block height is committed.
//
// - ctx: Context variables for the current API call.
//
// - ret: Return error when the request failed, otherwise return nil.
func (c *Client) WaitForBlockHeight(ctx context.Context, h int64) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		latestBlockHeight, err := c.GetLatestBlockHeight(ctx)
		if err != nil {
			return err
		}
		if latestBlockHeight >= h {
			return nil
		}
		select {
		case <-ctx.Done():
			return errors.Wrap(ctx.Err(), "timeout exceeded waiting for block")
		case <-ticker.C:
		}
	}
}

// WaitForNextBlock - Wait until the next block is committed since current block.
//
// - ctx: Context variables for the current API call.
//
// - ret: Return error when the request failed, otherwise return nil.
func (c *Client) WaitForNextBlock(ctx context.Context) error {
	return c.WaitForNBlocks(ctx, 1)
}

// WaitForNBlocks - Wait for another n blocks to be committed since current block.
//
// - ctx: Context variables for the current API call.
//
// - n: number of blocks to be waited.
//
// - ret: Return error when the request failed, otherwise return nil.
func (c *Client) WaitForNBlocks(ctx context.Context, n int64) error {
	start, err := c.GetLatestBlock(ctx)
	if err != nil {
		return err
	}
	return c.WaitForBlockHeight(ctx, start.Header.Height+n)
}

// WaitForTx - Wait for a transaction to be confirmed onchian, if transaction not found in current block, wait for the next block. API ends when a transaction is found or context is canceled.
//
// - ctx: Context variables for the current API call.
//
// - hash: The hex representation of transaction hash.
//
// - ret1: The transaction result details.
//
// - ret2: Return error when the request failed, otherwise return nil.
func (c *Client) WaitForTx(ctx context.Context, hash string) (*ctypes.ResultTx, error) {
	for {
		txResponse, txErr := c.tryWaitForTx(ctx, hash)
		if txErr == nil {
			return txResponse, nil
		}
		if txErr != errTxNotFound {
			return nil, txErr
		}

		evmResponse, evmErr := c.tryWaitForEvmTx(ctx, hash)
		if evmErr == nil {
			return evmResponse, nil
		}
		if evmErr != errTxNotFound {
			return nil, evmErr
		}

		if err := c.WaitForNextBlock(ctx); err != nil {
			return nil, errors.Wrap(err, "waiting for next block")
		}
	}
}

var errTxNotFound = fmt.Errorf("tx not found")

func (c *Client) tryWaitForTx(ctx context.Context, hash string) (*ctypes.ResultTx, error) {
	var (
		txResponse *ctypes.ResultTx
		err        error
		waitTxCtx  context.Context
		cancelFunc context.CancelFunc
	)
	queryHash := strings.TrimPrefix(hash, "0x")

	// when websocket conn is used, use a short timeout context to achieve the retry mechanism
	if c.useWebsocketConn {
		waitTxCtx, cancelFunc = context.WithTimeout(context.Background(), gosdktypes.WaitTxContextTimeOut)
		txResponse, err = c.chainClient.Tx(waitTxCtx, queryHash)
		cancelFunc()
	} else {
		txResponse, err = c.chainClient.Tx(ctx, queryHash)
	}
	if err != nil {
		if strings.Contains(err.Error(), "not found") || (c.useWebsocketConn && (waitTxCtx.Err() == context.DeadlineExceeded)) {
			return nil, errTxNotFound
		}
		return nil, errors.Wrapf(err, "fetching tx '%s'", hash)
	}
	if txResponse == nil {
		return nil, errTxNotFound
	}
	return txResponse, nil
}

func (c *Client) tryWaitForEvmTx(ctx context.Context, hash string) (*ctypes.ResultTx, error) {
	receipt, err := c.evmClient.TransactionReceipt(ctx, common.HexToHash(hash))
	if err != nil {
		if err == ethereum.NotFound {
			return nil, errTxNotFound
		}
		return nil, err
	}

	if receipt == nil {
		return nil, errTxNotFound
	}

	if receipt.Status != ethtypes.ReceiptStatusSuccessful {
		return nil, fmt.Errorf("transaction %s failed with status: %d", hash, receipt.Status)
	}

	h, _ := hex.DecodeString(hash)
	return &ctypes.ResultTx{
		Hash:   h,
		Height: receipt.BlockNumber.Int64(),
		// todo: fill in the rest of the fields
	}, nil
}

// BroadcastTx - Broadcast a transaction containing the provided message(s) to the chain.
//
// - ctx: Context variables for the current API call.
//
// - msgs: Message(s) to be broadcast to blockchain.
//
// - txOpt: txOpt contains options for customizing the transaction.
//
// - opts: The grpc option(s) if Client is using grpc connection.
//
// - ret1: transaction response, it can indicate both success and failed transaction.
//
// - ret2: Return error when the request failed, otherwise return nil.
func (c *Client) BroadcastTx(ctx context.Context, msgs []sdk.Msg, txOpt *types.TxOption, opts ...grpc.CallOption) (*txtypes.BroadcastTxResponse, error) {
	if len(msgs) == 0 {
		return nil, fmt.Errorf("msg is not provided in the transaction")
	}
	for _, msg := range msgs {
		if validateBasic, ok := msg.(sdk.HasValidateBasic); ok {
			if err := validateBasic.ValidateBasic(); err != nil {
				return nil, err
			}
		}
	}
	resp, err := c.broadcastTxWithMocaAddressCodec(ctx, msgs, txOpt, opts...)
	if err != nil {
		return nil, err
	}
	if resp.TxResponse.Code != 0 {
		return resp, fmt.Errorf(
			"the tx has failed with response code: %d, codespace:%s, raw_log:%s",
			resp.TxResponse.Code,
			resp.TxResponse.Codespace,
			resp.TxResponse.RawLog,
		)
	}
	return resp, nil
}

// SimulateTx - Simulate a transaction containing the provided message(s) on the chain.
//
// - ctx: Context variables for the current API call.
//
// - msgs: Message(s) to be broadcast to blockchain.
//
// - txOpt: TxOpt contains options for customizing the transaction.
//
// - opts: The grpc option(s) if Client is using grpc connection.
//
// - ret1: The simulation result.
//
// - ret2: Return error when the request failed, otherwise return nil.
func (c *Client) SimulateTx(ctx context.Context, msgs []sdk.Msg, txOpt types.TxOption, opts ...grpc.CallOption) (*txtypes.SimulateResponse, error) {
	return c.simulateTxWithMocaAddressCodec(ctx, msgs, &txOpt, opts...)
}

func (c *Client) newTxConfigWithMocaAddressCodec() (sdkclient.TxConfig, error) {
	signingOptions := &txsigning.Options{
		FileResolver: c.chainClient.GetCodec().InterfaceRegistry(),
		AddressCodec: mocacmdconfig.NewMultiPrefixBech32AccCodec(),
		CustomGetSigners: map[protoreflect.FullName]txsigning.GetSignersFunc{
			protoreflect.FullName("moca.payment.MsgCreatePaymentAccount"): func(msg protov2.Message) ([][]byte, error) {
				creatorField := msg.ProtoReflect().Descriptor().Fields().ByName("creator")
				if creatorField == nil {
					return nil, fmt.Errorf("creator field not found in %s", msg.ProtoReflect().Descriptor().FullName())
				}
				signer, err := sdk.AccAddressFromHexUnsafe(msg.ProtoReflect().Get(creatorField).String())
				if err != nil {
					return nil, err
				}
				return [][]byte{signer}, nil
			},
		},
	}
	return authtx.NewTxConfigWithOptions(c.chainClient.GetCodec(), authtx.ConfigOptions{
		EnabledSignModes: []signing.SignMode{signing.SignMode_SIGN_MODE_EIP_712},
		SigningOptions:   signingOptions,
	})
}

func (c *Client) simulateTxWithMocaAddressCodec(ctx context.Context, msgs []sdk.Msg, txOpt *types.TxOption, opts ...grpc.CallOption) (*txtypes.SimulateResponse, error) {
	txConfig, err := c.newTxConfigWithMocaAddressCodec()
	if err != nil {
		return nil, err
	}
	txBuilder := txConfig.NewTxBuilder()
	if err := c.chainClientConstructTx(ctx, msgs, txOpt, txBuilder); err != nil {
		return nil, err
	}
	txBytes, err := txConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return nil, err
	}
	return c.chainClient.TxClient.Simulate(ctx, &txtypes.SimulateRequest{TxBytes: txBytes}, opts...)
}

func (c *Client) broadcastTxWithMocaAddressCodec(ctx context.Context, msgs []sdk.Msg, txOpt *types.TxOption, opts ...grpc.CallOption) (*txtypes.BroadcastTxResponse, error) {
	txConfig, err := c.newTxConfigWithMocaAddressCodec()
	if err != nil {
		return nil, err
	}
	txBuilder := txConfig.NewTxBuilder()
	if err := c.chainClientConstructTxWithGasInfo(ctx, msgs, txOpt, txConfig, txBuilder, opts...); err != nil {
		return nil, err
	}
	txSignedBytes, err := c.chainClientSignTx(ctx, txConfig, txBuilder, txOpt)
	if err != nil {
		return nil, err
	}

	mode := txtypes.BroadcastMode_BROADCAST_MODE_SYNC
	if txOpt != nil && txOpt.Mode != nil {
		mode = *txOpt.Mode
	}

	return c.chainClient.TxClient.BroadcastTx(ctx, &txtypes.BroadcastTxRequest{
		Mode:    mode,
		TxBytes: txSignedBytes,
	}, opts...)
}

func (c *Client) chainClientSignTx(ctx context.Context, txConfig sdkclient.TxConfig, txBuilder sdkclient.TxBuilder, txOpt *types.TxOption) ([]byte, error) {
	km, err := c.chainClient.GetKeyManager()
	if err != nil {
		return nil, err
	}
	if txOpt != nil && txOpt.OverrideKeyManager != nil {
		km = *txOpt.OverrideKeyManager
	}

	account, err := c.chainClient.GetAccountByAddr(ctx, km.GetAddr())
	if err != nil {
		return nil, err
	}
	nonce := account.GetSequence()
	if txOpt != nil && txOpt.Nonce != 0 {
		nonce = txOpt.Nonce
	}

	chainID, err := c.chainClient.GetChainID()
	if err != nil {
		return nil, err
	}
	signerData := xauthsigning.SignerData{
		ChainID:       chainID,
		AccountNumber: account.GetAccountNumber(),
		Sequence:      nonce,
	}
	sig, err := clitx.SignWithPrivKey(ctx, signing.SignMode_SIGN_MODE_EIP_712, signerData, txBuilder, km, txConfig, nonce)
	if err != nil {
		return nil, err
	}
	if err := txBuilder.SetSignatures(sig); err != nil {
		return nil, err
	}
	return txConfig.TxEncoder()(txBuilder.GetTx())
}

func (c *Client) chainClientSetSignerInfo(ctx context.Context, txBuilder sdkclient.TxBuilder, txOpt *types.TxOption) error {
	km, err := c.chainClient.GetKeyManager()
	if err != nil {
		return err
	}
	if txOpt != nil && txOpt.OverrideKeyManager != nil {
		km = *txOpt.OverrideKeyManager
	}

	account, err := c.chainClient.GetAccountByAddr(ctx, km.GetAddr())
	if err != nil {
		return err
	}
	nonce := account.GetSequence()
	if txOpt != nil && txOpt.Nonce != 0 {
		nonce = txOpt.Nonce
	}

	return txBuilder.SetSignatures(signing.SignatureV2{
		PubKey: km.PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode: signing.SignMode_SIGN_MODE_EIP_712,
		},
		Sequence: nonce,
	})
}

func (c *Client) chainClientConstructTx(ctx context.Context, msgs []sdk.Msg, txOpt *types.TxOption, txBuilder sdkclient.TxBuilder) error {
	for _, msg := range msgs {
		if validateBasic, ok := msg.(sdk.HasValidateBasic); ok {
			if err := validateBasic.ValidateBasic(); err != nil {
				return err
			}
		}
	}
	if err := txBuilder.SetMsgs(msgs...); err != nil {
		return err
	}
	if txOpt != nil {
		if txOpt.Memo != "" {
			txBuilder.SetMemo(txOpt.Memo)
		}
		if !txOpt.FeePayer.Empty() {
			txBuilder.SetFeePayer(txOpt.FeePayer)
		}
		if !txOpt.FeeGranter.Empty() {
			txBuilder.SetFeeGranter(txOpt.FeeGranter)
		}
	}
	return c.chainClientSetSignerInfo(ctx, txBuilder, txOpt)
}

func (c *Client) chainClientConstructTxWithGasInfo(ctx context.Context, msgs []sdk.Msg, txOpt *types.TxOption, txConfig sdkclient.TxConfig, txBuilder sdkclient.TxBuilder, opts ...grpc.CallOption) error {
	if err := c.chainClientConstructTx(ctx, msgs, txOpt, txBuilder); err != nil {
		return err
	}
	txBytes, err := txConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return err
	}

	if txOpt != nil && txOpt.NoSimulate {
		isFeeAmtZero, err := isZeroFeeAmount(txOpt.FeeAmount)
		if err != nil {
			return err
		}
		if txOpt.GasLimit == 0 || isFeeAmtZero {
			return types.ErrGasInfoNotProvided
		}
		txBuilder.SetGasLimit(txOpt.GasLimit)
		txBuilder.SetFeeAmount(txOpt.FeeAmount)
		return nil
	}

	simulateResponse, err := c.chainClient.TxClient.Simulate(ctx, &txtypes.SimulateRequest{TxBytes: txBytes}, opts...)
	if err != nil {
		return err
	}
	gasLimit := simulateResponse.GasInfo.GetGasUsed()
	gasPrice, err := sdk.ParseCoinNormalized(simulateResponse.GasInfo.GetMinGasPrice())
	if err != nil {
		return err
	}
	if gasPrice.IsNil() || gasPrice.IsZero() {
		return types.ErrSimulatedGasPrice
	}
	txBuilder.SetGasLimit(gasLimit)
	txBuilder.SetFeeAmount(sdk.NewCoins(
		sdk.NewCoin(gasPrice.Denom, gasPrice.Amount.MulRaw(int64(gasLimit))),
	))
	return nil
}

func isZeroFeeAmount(feeAmount sdk.Coins) (bool, error) {
	if len(feeAmount) == 0 {
		return true, nil
	}
	if len(feeAmount) != 1 {
		return false, types.ErrFeeAmountNotValid
	}
	if feeAmount[0].Amount.IsNil() {
		return false, types.ErrFeeAmountNotValid
	}
	return feeAmount[0].IsZero(), nil
}

// GetSyncing - Retrieve the syncing status of the node.
//
// - ctx: Context variables for the current API call.
//
// - ret1: The boolean value which indicates whether the node has caught up the latest block.
//
// - ret2: Return error when the request failed, otherwise return nil.
func (c *Client) GetSyncing(ctx context.Context) (bool, error) {
	syncing, err := c.chainClient.GetSyncing(ctx, &cmtservice.GetSyncingRequest{})
	if err != nil {
		return false, err
	}
	return syncing.Syncing, nil
}

// GetBlockByHeight - Retrieve the block at the given height from the chain.
//
// - ctx: Context variables for the current API call.
//
// - height: The block height.
//
// - ret1: The boolean value which indicates whether the node has caught up the latest block.
//
// - ret2: Return error when the request failed, otherwise return nil.
func (c *Client) GetBlockByHeight(ctx context.Context, height int64) (*bfttypes.Block, error) {
	blockByHeight, err := c.chainClient.GetBlock(ctx, &height)
	if err != nil {
		return nil, err
	}
	return blockByHeight.Block, nil
}

// GetBlockResultByHeight - Retrieve the block result at the given height from the chain.
//
// - ctx: Context variables for the current API call.
//
// - height: The block height.
//
// - ret1: The boolean value which indicates whether the node has caught up the latest block.
//
// - ret2: Return error when the request failed, otherwise return nil.
func (c *Client) GetBlockResultByHeight(ctx context.Context, height int64) (*ctypes.ResultBlockResults, error) {
	return c.chainClient.GetBlockResults(ctx, &height)
}

// GetValidatorSet - Retrieve the latest validator set from the chain.
//
// - ctx: Context variables for the current API call.
//
// - ret1: The latest height of block that validators set info retrieved from.
//
// - ret2: The list of validators.
//
// - ret3: Return error when the request failed, otherwise return nil.
func (c *Client) GetValidatorSet(ctx context.Context) (int64, []*bfttypes.Validator, error) {
	validatorSetResponse, err := c.chainClient.GetValidators(ctx, nil)
	if err != nil {
		return 0, nil, err
	}
	return validatorSetResponse.BlockHeight, validatorSetResponse.Validators, nil
}

// GetValidatorsByHeight - Retrieve the validator set at a given block height from the chain.
//
// - ctx: Context variables for the current API call.
//
// - height: The block height.
//
// - ret1: The list of validators.
//
// - ret2: Return error when the request failed, otherwise return nil.
func (c *Client) GetValidatorsByHeight(ctx context.Context, height int64) ([]*bfttypes.Validator, error) {
	validatorSetResponse, err := c.chainClient.GetValidators(ctx, &height)
	if err != nil {
		return nil, err
	}
	return validatorSetResponse.Validators, nil
}

// BroadcastVote - Broadcast a vote to the Node's VotePool, it is used by Moca relayer and challengers by now.
//
// - ctx: Context variables for the current API call.
//
// - vote: Contains vote details.
//
// - ret: Return error when the request failed, otherwise return nil.
func (c *Client) BroadcastVote(ctx context.Context, vote votepool.Vote) error {
	return c.chainClient.BroadcastVote(ctx, vote)
}

// QueryVote - Query a vote from the Node's VotePool, it is used by Moca relayer and challengers by now.
//
// - ctx: Context variables for the current API call.
//
// - eventType: The type of vote to be queried.
//
// - eventHash: The hash bytes of vote
//
// - ret1: The vote result
//
// - ret2: Return error when the request failed, otherwise return nil.
func (c *Client) QueryVote(ctx context.Context, eventType int, eventHash []byte) (*ctypes.ResultQueryVote, error) {
	return c.chainClient.QueryVote(ctx, eventType, eventHash)
}

// SetTag - Set tag for a given existing resource GRN (a bucket, a object or a group)
//
// This API sends a request to the moca chain to set tags for the given resource.
//
// - ctx: Context variables for the current API call.
//
// - resourceGRN: The GRN of resource that needs to set tags
//
// - tags: the tags to be set for the given resource
//
// - opts: The Options indicates the meta to construct setTag msg and the way to send transaction
//
// - ret1: Transaction hash return from blockchain.
//
// - ret2: Return error if SetTag failed, otherwise return nil.
func (c *Client) SetTag(ctx context.Context, resourceGRN string, tags storageTypes.ResourceTags, opts gosdktypes.SetTagsOptions) (string, error) {
	msgSetTag := storageTypes.NewMsgSetTag(c.MustGetDefaultAccount().GetAddress(), resourceGRN, &tags)
	return c.sendSetTagEvmTxn(ctx, msgSetTag)
}

func (c *Client) sendSetTagEvmTxn(ctx context.Context, msg *storageTypes.MsgSetTag) (string, error) {
	backoffDelay := gosdktypes.NonceBackOffDelay
	var lastErr error

	for retry := 0; retry < gosdktypes.MaxNonceRetryTime; retry++ {
		session, err := c.createStorageEvmSession(ctx, c.privateKey)
		if err != nil {
			return "", err
		}

		txRsp, err := session.SetTag(
			msg.Resource,
			toStorageTags(msg.Tags),
		)
		if err != nil {
			lastErr = err
			errorMsg := strings.ToLower(err.Error())

			// Check if it's a nonce-related error that we should retry
			if strings.Contains(errorMsg, gosdktypes.InvalidNonceErr) ||
				strings.Contains(errorMsg, gosdktypes.InvalidSequenceErr) {

				if retry == gosdktypes.MaxNonceRetryTime-1 {
					// This is the last retry, return the error
					return "", fmt.Errorf("failed to set tags after %d attempts due to nonce errors: %w", gosdktypes.MaxNonceRetryTime, err)
				}

				// Wait before retrying
				time.Sleep(backoffDelay)
				backoffDelay *= 2
				continue
			}

			// For non-nonce errors, don't retry
			return "", err
		}

		// Success
		return txRsp.Hash().String(), nil
	}

	// This should not be reached, but just in case
	return "", fmt.Errorf("failed to set tags after %d attempts: %w", gosdktypes.MaxNonceRetryTime, lastErr)
}

func toStorageTags(t *storageTypes.ResourceTags) []storage.Tag {
	var tags []storage.Tag
	for _, tag := range t.Tags {
		tags = append(tags, *toStorageTag(&tag))
	}
	return tags
}

func toStorageTag(t *storageTypes.ResourceTags_Tag) *storage.Tag {
	return &storage.Tag{
		Key:   t.Key,
		Value: t.Value,
	}
}
