package client

import (
	"context"
	"fmt"
	"math/big"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/mocachain/moca/v2/precompiles/bank"
	"github.com/mocachain/moca/v2/precompiles/payment"
	"github.com/mocachain/moca/v2/precompiles/storage"
	"github.com/mocachain/moca/v2/sdk/types"
)

const (
	// DefaultGasLimit is no longer used. EVM transactions used to fall back to it
	// when gas estimation failed and were sent anyway, which a call that would
	// revert paid for; the fee market charges at least half the gas limit, so every
	// such transaction was charged 2,500,000 gas. A call the node cannot estimate is
	// now not sent.
	//
	// Deprecated: EVM transactions carry the estimated gas, or the caller's explicit
	// limit (see GasLimitEstimate).
	DefaultGasLimit = 5000000
	// GasLimitEstimate is the sentinel that leaves TransactOpts.GasLimit==0 so
	// geth's bind estimates gas per call (via estimatingBackend) instead of
	// pinning a fixed value — precompile gas is dynamic (moca #332 store gas +
	// #356 flat+per-byte RequiredGas), so a fixed pin over- or under-shoots.
	// A caller sets an explicit limit the same way as for a Cosmos transaction:
	// TxOption.NoSimulate with TxOption.GasLimit (see txGasLimit).
	GasLimitEstimate = 0
)

// estimatingBackend wraps the EVM client so bind's auto gas-estimation works for
// moca's stateful precompiles: (1) precompile addresses have no bytecode, so a
// dummy PendingCodeAt avoids bind's ErrNoCode guard; (2) cosmos/evm's
// eth_estimateGas returns the exact binary-searched minimum with no headroom, so
// EstimateGas adds the same 1.25x + 10000 safety buffer the cosmos tx path uses
// (state can shift between the estimate block and execution). When the node cannot
// estimate the call (it would revert, or the node failed), EstimateGas returns that
// error and bind neither signs nor sends the transaction.
type estimatingBackend struct {
	*ethclient.Client
}

func (estimatingBackend) PendingCodeAt(_ context.Context, _ common.Address) ([]byte, error) {
	return []byte{0x01}, nil
}

func (b estimatingBackend) EstimateGas(ctx context.Context, call ethereum.CallMsg) (uint64, error) {
	est, err := b.Client.EstimateGas(ctx, call)
	if err != nil {
		return 0, fmt.Errorf("failed to estimate gas: %w", err)
	}
	return withGasBuffer(est), nil
}

// withGasBuffer adds the 1.25x + 10000 buffer to a gas estimate.
func withGasBuffer(estimate uint64) uint64 {
	return estimate*125/100 + 10000
}

// txGasLimit returns the gas limit a caller set explicitly for an EVM transaction:
// TxOption.GasLimit together with TxOption.NoSimulate, as for a Cosmos transaction.
// Without both it returns GasLimitEstimate.
func txGasLimit(txOpt *types.TxOption) uint64 {
	if txOpt == nil || !txOpt.NoSimulate {
		return GasLimitEstimate
	}
	return txOpt.GasLimit
}

func CreateTxOpts(ctx context.Context, client *ethclient.Client, hexPrivateKey string, chain *big.Int, gasLimit uint64, nonce uint64) (*bind.TransactOpts, error) {
	// create private key
	privateKey, err := crypto.HexToECDSA(hexPrivateKey)
	if err != nil {
		return nil, err
	}

	// Build transact tx opts with private key
	txOpts, err := bind.NewKeyedTransactorWithChainID(privateKey, chain)
	if err != nil {
		return nil, err
	}

	// set gas limit and gas price
	txOpts.GasLimit = gasLimit
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}
	txOpts.GasPrice = gasPrice

	txOpts.Nonce = big.NewInt(int64(nonce))

	return txOpts, nil
}

func CreateStorageSession(client *ethclient.Client, txOpts bind.TransactOpts, contractAddress string) (*storage.IStorageSession, error) {
	contract, err := storage.NewIStorage(common.HexToAddress(contractAddress), estimatingBackend{client})
	if err != nil {
		return nil, err
	}
	session := &storage.IStorageSession{
		Contract: contract,
		CallOpts: bind.CallOpts{
			Pending: false,
		},
		TransactOpts: txOpts,
	}
	return session, nil
}

func CreateBankSession(client *ethclient.Client, txOpts bind.TransactOpts, contractAddress string) (*bank.IBankSession, error) {
	contract, err := bank.NewIBank(common.HexToAddress(contractAddress), estimatingBackend{client})
	if err != nil {
		return nil, err
	}
	session := &bank.IBankSession{
		Contract: contract,
		CallOpts: bind.CallOpts{
			Pending: false,
		},
		TransactOpts: txOpts,
	}
	return session, nil
}

func CreatePaymentSession(client *ethclient.Client, txOpts bind.TransactOpts, contractAddress string) (*payment.IPaymentSession, error) {
	contract, err := payment.NewIPayment(common.HexToAddress(contractAddress), estimatingBackend{client})
	if err != nil {
		return nil, err
	}
	session := &payment.IPaymentSession{
		Contract: contract,
		CallOpts: bind.CallOpts{
			Pending: false,
		},
		TransactOpts: txOpts,
	}
	return session, nil
}
