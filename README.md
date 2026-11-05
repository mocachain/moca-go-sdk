# moca Go SDK

## Disclaimer

**The software and related documentation are under active development, all subject to potential future change without
notification and not ready for production use. The code and security audit have not been fully completed and not ready
for any bug bounty. We advise you to be careful and experiment on the network at your own risk. Stay safe out there.**

## Instruction

The moca-GO-SDK provides a thin wrapper for interacting with moca storage network.

Rich SDKs is provided to operate moca resources or query status of resources.

### Requirement

Go 1.26.10 or later

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
go get github.com/mocachain/moca-go-sdk
```

Replace dependencies

Go does not apply the `replace` directives of a dependency, so copy the ones in this SDK's
[go.mod](go.mod) into your own `go.mod`. They match moca `v2.0.0-rc1`:

```go.mod
replace (
	cosmossdk.io/api => github.com/mocachain/moca-cosmos-sdk/api v1.4.0-rc0.0.20261029124406-176ec71f8cf4
	cosmossdk.io/client/v2 => github.com/mocachain/moca-cosmos-sdk/client/v2 v2.4.0-rc0.0.20261029124406-176ec71f8cf4
	cosmossdk.io/math => github.com/mocachain/moca-cosmos-sdk/math v1.4.0-rc0.0.20261029124406-176ec71f8cf4
	cosmossdk.io/simapp => github.com/mocachain/moca-cosmos-sdk/simapp v1.4.0-rc0.0.20261029124406-176ec71f8cf4
	cosmossdk.io/store => github.com/mocachain/moca-cosmos-sdk/store v1.4.0-rc0.0.20261029124406-176ec71f8cf4
	cosmossdk.io/x/circuit => github.com/mocachain/moca-cosmos-sdk/x/circuit v1.4.0-rc0.0.20261029124406-176ec71f8cf4
	cosmossdk.io/x/feegrant => github.com/mocachain/moca-cosmos-sdk/x/feegrant v1.4.0-rc0.0.20261029124406-176ec71f8cf4
	cosmossdk.io/x/nft => github.com/mocachain/moca-cosmos-sdk/x/nft v1.4.0-rc0.0.20261029124406-176ec71f8cf4
	cosmossdk.io/x/tx => github.com/mocachain/moca-cosmos-sdk/x/tx v1.4.0-rc0.0.20261029124406-176ec71f8cf4
	github.com/0xPolygon/polygon-edge v1.3.3 => github.com/mocachain/polygon-edge v1.3.3-moca.1
	github.com/99designs/keyring => github.com/cosmos/keyring v1.2.0
	github.com/btcsuite/btcd => github.com/btcsuite/btcd v0.22.1
	github.com/btcsuite/btcd/btcec/v2 => github.com/btcsuite/btcd/btcec/v2 v2.3.4
	github.com/cometbft/cometbft => github.com/mocachain/moca-cometbft v1.3.0-rc1
	github.com/confio/ics23/go => github.com/cosmos/ics23/go v0.10.0
	github.com/cosmos/cosmos-sdk => github.com/mocachain/moca-cosmos-sdk v1.4.0-rc1
	github.com/cosmos/evm => github.com/mocachain/cosmos-evm v0.6.3-moca.0
	github.com/ethereum/go-ethereum => github.com/cosmos/go-ethereum v1.16.2-cosmos-1
	github.com/gin-gonic/gin => github.com/gin-gonic/gin v1.9.1
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
	github.com/lyft/protoc-gen-validate => github.com/bufbuild/protoc-gen-validate v0.0.13
	github.com/syndtr/goleveldb => github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7
	nhooyr.io/websocket => nhooyr.io/websocket v1.8.17
)
```

The SDK also replaces `github.com/cosmos/cosmos-sdk/x/oracle/types` with the local module in
[compat/cosmos-sdk-x-oracle-types](compat/cosmos-sdk-x-oracle-types). That module is not published,
so copy the directory into your project and point the same `replace` at your copy.

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

 "github.com/mocachain/moca-go-sdk/client"
 "github.com/mocachain/moca-go-sdk/types"
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

The cross-chain governance and query helpers were removed together with the crosschain module in `moca/v2` main. Legacy cross-chain examples are retained only as placeholders that now return an unsupported error.

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

- [Moca](https://github.com/mocachain/moca): The Golang implementation of the Moca Blockchain.
- [Moca-Contracts](https://github.com/mocachain/moca-contracts): the cross chain contract for Moca that deployed on ethereum-compatible network.
- [Moca-Storage-Provider](https://github.com/mocachain/moca-storage-provider): the storage service infrastructures provided by either organizations or individuals.
- [moca-relayer](https://github.com/mocachain/moca-relayer): the service that relay cross chain package to both chains.
- [moca-cmd](https://github.com/mocachain/moca-cmd): the most powerful command line to interact with Moca system.
- [Awesome Cosmos](https://github.com/cosmos/awesome-cosmos): Collection of Cosmos related resources which also fits moca.

## Fork Information

This project is forked from [greenfield-go-sdk](https://github.com/bnb-chain/greenfield-go-sdk). Significant changes have been made to adapt the project for specific use cases, but much of the core functionality comes from the original project.
