package client

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/evmos/evmos/v12/x/evm/precompiles/bank"
	"github.com/evmos/evmos/v12/x/evm/precompiles/payment"
	"github.com/evmos/evmos/v12/x/evm/precompiles/storage"
)

const (
	DefaultGasLimit = 180000
)

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
	contract, err := storage.NewIStorage(common.HexToAddress(contractAddress), client)
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
	contract, err := bank.NewIBank(common.HexToAddress(contractAddress), client)
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
	contract, err := payment.NewIPayment(common.HexToAddress(contractAddress), client)
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
