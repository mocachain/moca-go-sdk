package client

import (
	"context"
	"math/big"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/mocachain/moca/v2/precompiles/bank"
	"github.com/mocachain/moca/v2/precompiles/payment"
	"github.com/mocachain/moca/v2/precompiles/storage"
)

const (
	// DefaultGasLimit is the fallback ceiling used only when gas estimation
	// fails (e.g. a tx that would revert); the tx is still sent and reverts
	// on-chain as before. Sized to cover the heaviest precompile store gas.
	DefaultGasLimit = 5000000
	// GasLimitEstimate is the sentinel that leaves TransactOpts.GasLimit==0 so
	// geth's bind estimates gas per call (via estimatingBackend) instead of
	// pinning a fixed value — precompile gas is dynamic (moca #332 store gas +
	// #356 flat+per-byte RequiredGas), so a fixed pin over- or under-shoots.
	GasLimitEstimate = 0
)

// estimatingBackend wraps the EVM client so bind's auto gas-estimation works for
// moca's stateful precompiles: (1) precompile addresses have no bytecode, so a
// dummy PendingCodeAt avoids bind's ErrNoCode guard; (2) cosmos/evm's
// eth_estimateGas returns the exact binary-searched minimum with no headroom, so
// EstimateGas adds the same 1.25x + 10000 safety buffer the cosmos tx path uses
// (state can shift between the estimate block and execution). On estimate failure
// it falls back to DefaultGasLimit so the tx still sends (and reverts on-chain).
type estimatingBackend struct {
	*ethclient.Client
}

func (estimatingBackend) PendingCodeAt(_ context.Context, _ common.Address) ([]byte, error) {
	return []byte{0x01}, nil
}

func (b estimatingBackend) EstimateGas(ctx context.Context, call ethereum.CallMsg) (uint64, error) {
	est, err := b.Client.EstimateGas(ctx, call)
	if err != nil {
		return DefaultGasLimit, nil
	}
	return est*125/100 + 10000, nil
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
