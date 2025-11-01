package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/mocachain/moca-go-sdk/client"
)

// The config information is consistent with the testnet of moca
// You need to set the privateKey, bucketName, objectName and groupName to make the basic examples work well
const (
	rpcAddr                 = "http://localhost:26657"
	evmRpcAddr              = "http://localhost:8545"
	chainId                 = "moca_5151-1"
	crossChainDestBsChainId = 97
	privateKey              = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	objectSize              = 1000
	groupMember             = "0x.." // used for group examples
	principal               = "0x.." // used for permission examples
	bucketName              = "test-bucket"
	objectName              = "test-object"
	groupName               = "test-group"
	toAddress               = "0x.." // used for cross chain transfer
	httpsAddr               = ""
	paymentAddr             = ""
	bscRpcAddr              = "https://data-seed-prebsc-1-s1.binance.org:8545/"
	bscPrivateKey           = "a6f2041aeca9a09159c937b77316c9c7e2c0f1c5b7241832f84bf1d37eb49661"
)

func handleErr(err error, funcName string) {
	if err != nil {
		log.Fatalln("fail to " + funcName + ": " + err.Error())
	}
}

func waitObjectSeal(cli client.IClient, bucketName, objectName string) {
	ctx := context.Background()
	// wait for the object to be sealed
	timeout := time.After(15 * time.Second)
	ticker := time.NewTicker(2 * time.Second)

	for {
		select {
		case <-timeout:
			err := errors.New("object not sealed after 15 seconds")
			handleErr(err, "HeadObject")
		case <-ticker.C:
			objectDetail, err := cli.HeadObject(ctx, bucketName, objectName)
			handleErr(err, "HeadObject")
			if objectDetail.ObjectInfo.GetObjectStatus().String() == "OBJECT_STATUS_SEALED" {
				ticker.Stop()
				fmt.Printf("put object %s successfully \n", objectName)
				return
			}
		}
	}
}
