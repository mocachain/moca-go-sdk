# moca Go SDK

## Disclaimer

**The software and related documentation are under active development, all subject to potential future change without
notification and not ready for production use. The code and security audit have not been fully completed and not ready
for any bug bounty. We advise you to be careful and experiment on the network at your own risk. Stay safe out there.**

## Instruction

The moca-GO-SDK provides a thin wrapper for interacting with moca storage network.

Rich SDKs is provided to operate moca resources or query status of resources.

### Requirement

Go version above 1.20

## Getting started

To get started working with the SDK setup your project for Go modules, and retrieve the SDK dependencies with `go get`.
This example shows how you can use the moca go SDK to interact with the moca storage network,

### Initialize Project

```sh
mkdir ~/hellomoca
cd ~/hellomoca
go mod init hellomoca
```

### Add SDK Dependencies

```sh
go get github.com/MocaFoundation/moca-go-sdk
```

replace dependencies

```go.mod
cosmossdk.io/api => github.com/MocaFoundation/moca-cosmos-sdk/api v0.0.0-20250908083143-e28da82c630a
cosmossdk.io/math => github.com/MocaFoundation/moca-cosmos-sdk/math v0.0.0-20250908083143-e28da82c630a
cosmossdk.io/simapp => github.com/MocaFoundation/moca-cosmos-sdk/simapp v0.0.0-20250908083143-e28da82c630a
github.com/cometbft/cometbft => github.com/MocaFoundation/moca-cometbft v1.0.0-alpha.1
github.com/cometbft/cometbft-db => github.com/MocaFoundation/moca-cometbft-db v1.0.0-alpha.1
github.com/consensys/gnark-crypto => github.com/consensys/gnark-crypto v0.7.0
github.com/cosmos/cosmos-sdk => github.com/MocaFoundation/moca-cosmos-sdk v1.0.0-alpha.1
github.com/cosmos/iavl => github.com/MocaFoundation/moca-iavl v1.0.0-alpha.1
```

### Initialize Client

The moca client requires the following parameters to connect to moca chain and storage providers.

| Parameter     | Description                                       |
| :------------ | :------------------------------------------------ |
| rpcAddr       | the tendermit address of moca chain               |
| chainId       | the chain id of moca                              |
| client.Option | All the options such as DefaultAccount and secure |

The DefaultAccount is need to set in options if you need send request to SP or send txn to moca

```go
package main

import (
 "context"
 "log"

 "github.com/MocaFoundation/moca-go-sdk/client"
 "github.com/MocaFoundation/moca-go-sdk/types"
)

func main() {
 privateKey := "<Your own private key>"
 account, err := types.NewAccountFromPrivateKey("test", privateKey)
 if err != nil {
  log.Fatalf("New account from private key error, %v", err)
 }

 rpcAddr := "https://devnet-rpc.mocachain.org:443"
 chainId := "moca_5151-1"
 
 gnfdCLient, err := client.New(chainId, rpcAddr, client.Option{DefaultAccount: account})
 if err != nil {
  log.Fatalf("unable to new moca client, %v", err)
 }
}

```

### Quick Start Examples

The examples directory provides a wealth of examples to guide users in using the SDK's various features, including basic storage upload and download functions,
group functions, permission functions, as well as payment and cross-chain related functions.

The **basic.go** includes the basic functions to fetch the blockchain info.

The **storage.go** includes the most storage functions such as creating a bucket, uploading files, downloading files, heading and deleting resource.

The **group.go** includes the group related functions such as creating a group and updating group member.

The **payment.go** includes the payment related functions to manage payment accounts.

The **permission.go** includes the permission related functions to manage resources(bucket, object, group) policy.

The **crosschain.go** includes the cross chain related functions to transfer or mirror resource to BSC.

#### Config Examples

You need to modify the variables in "common.go" under the "examples" directory to set the initialization information for the client, including "rpcAddr", "chainId", and "privateKey", etc. In addition,
you also need to set basic parameters such as "bucket name" and "object name" to run the basic functionality of storage.

#### Run Examples

The steps to run example are as follows

```shell
make examples
cd examples
./storage 
```

You can also directly execute "go run" to run a specific example.
For example, execute "go run storage.go common.go" to run the relevant example for storage.
Please note that the "permission.go" example must be run after "storage.go" because resources such as objects need to be created first before setting permissions.

## Reference

- [Moca](https://github.com/MocaFoundation/moca): The Golang implementation of the Moca Blockchain.
- [Moca-Contracts](https://github.com/MocaFoundation/moca-contracts): the cross chain contract for Moca that deployed on ethereum-compatible network.
- [Moca-Storage-Provider](https://github.com/MocaFoundation/moca-storage-provider): the storage service infrastructures provided by either organizations or individuals.
- [moca-relayer](https://github.com/MocaFoundation/moca-relayer): the service that relay cross chain package to both chains.
- [moca-cmd](https://github.com/MocaFoundation/moca-cmd): the most powerful command line to interact with Moca system.
- [Awesome Cosmos](https://github.com/cosmos/awesome-cosmos): Collection of Cosmos related resources which also fits moca.

## Fork Information

This project is forked from [greenfield-go-sdk](https://github.com/bnb-chain/greenfield-go-sdk). Significant changes have been made to adapt the project for specific use cases, but much of the core functionality comes from the original project.
