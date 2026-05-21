package main

import (
	"log"

	"github.com/mocachain/moca-go-sdk/client"
	"github.com/mocachain/moca-go-sdk/types"
)

// it is the example of cross-chain SDKs usage
func TestCrossChain() {
	account, err := types.NewAccountFromPrivateKey("test", privateKey)
	if err != nil {
		log.Fatalf("New account from private key error, %v", err)
	}
	cli, err := client.New(chainId, rpcAddr, evmRpcAddr, privateKey, client.Option{DefaultAccount: account})
	if err != nil {
		log.Fatalf("unable to new moca client, %v", err)
	}
	log.Println("legacy cross-chain SDK helpers are not supported on current moca main")
	_ = cli
}
