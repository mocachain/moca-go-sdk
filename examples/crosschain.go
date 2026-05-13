package main

import (
	"context"
	"log"

	"github.com/mocachain/moca-go-sdk/client"
	"github.com/mocachain/moca-go-sdk/types"
	gnfdSdkTypes "github.com/mocachain/moca/v2/sdk/types"
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
	ctx := context.Background()

	claimCrossChainPackage(cli, ctx)
}

func claimCrossChainPackage(cli client.IClient, ctx context.Context) {
	// In moca/v2 the SDK keeps claim support for relayer workflows.
	// Mirror/query helpers that depended on the legacy bridge module are no longer exposed.
	log.Println("submitting a sample cross-chain claim transaction")
	_, err := cli.Claims(
		ctx,
		1,
		uint32(crossChainDestBsChainId),
		1,
		0,
		nil,
		nil,
		nil,
		gnfdSdkTypes.TxOption{},
	)
	handleErr(err, "Claims")
}
