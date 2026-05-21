package client

import (
	"context"
	"errors"
	"strings"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	oracletypes "github.com/cosmos/cosmos-sdk/x/oracle/types"
	bsccommon "github.com/mocachain/moca-go-sdk/common"
	gnfdSdkTypes "github.com/mocachain/moca/v2/sdk/types"
	mocadTypes "github.com/mocachain/moca/v2/types"
)

var errCrossChainFeatureRemoved = errors.New("cross-chain chain module APIs are not supported by github.com/mocachain/moca/v2 main")

type ICrossChainClient interface {
	TransferOut(ctx context.Context, toAddress string, amount math.Int, txOption gnfdSdkTypes.TxOption) (*sdk.TxResponse, error)
	Claims(ctx context.Context, srcShainId, destChainId uint32, sequence uint64, timestamp uint64, payload []byte, voteAddrSet []uint64, aggSignature []byte, txOption gnfdSdkTypes.TxOption) (*sdk.TxResponse, error)
	GetChannelSendSequence(ctx context.Context, destChainId sdk.ChainID, channelId uint32) (uint64, error)
	GetChannelReceiveSequence(ctx context.Context, destChainId sdk.ChainID, channelId uint32) (uint64, error)
	GetInturnRelayer(ctx context.Context, req *oracletypes.QueryInturnRelayerRequest) (*oracletypes.QueryInturnRelayerResponse, error)
	GetCrossChainPackage(ctx context.Context, destChainId sdk.ChainID, channelId uint32, sequence uint64) ([]byte, error)
	MirrorGroup(ctx context.Context, destChainId sdk.ChainID, groupId math.Uint, groupName string, txOption gnfdSdkTypes.TxOption) (*sdk.TxResponse, error)
	MirrorBucket(ctx context.Context, destChainId sdk.ChainID, bucketId math.Uint, bucketName string, txOption gnfdSdkTypes.TxOption) (*sdk.TxResponse, error)
	MirrorObject(ctx context.Context, destChainId sdk.ChainID, objectId math.Uint, bucketName, objectName string, txOption gnfdSdkTypes.TxOption) (*sdk.TxResponse, error)
}

// TransferOut uses the ERC20 precompile because the old cross-chain module path was removed from moca main.
func (c *Client) TransferOut(ctx context.Context, toAddress string, amount math.Int, txOption gnfdSdkTypes.TxOption) (*sdk.TxResponse, error) {
	txHash, err := c.sendTransferOutEvmTx(ctx, toAddress, amount)
	if err != nil {
		return nil, err
	}
	return &sdk.TxResponse{TxHash: txHash}, nil
}

func (c *Client) sendTransferOutEvmTx(ctx context.Context, to string, amount math.Int) (string, error) {
	nonce, err := c.chainClient.GetNonce(context.Background())
	if err != nil {
		return "", err
	}
	chainID, err := c.evmClient.ChainID(ctx)
	if err != nil {
		return "", err
	}
	txOpts, err := CreateTxOpts(context.Background(), c.evmClient, c.privateKey, chainID, DefaultGasLimit, nonce)
	if err != nil {
		return "", err
	}

	parsedABI, err := abi.JSON(strings.NewReader(bsccommon.TokenABI))
	if err != nil {
		return "", err
	}
	contract := bindContract(common.HexToAddress(mocadTypes.Erc20Address), parsedABI, c.evmClient)
	tx, err := contract.Transact(txOpts, "transferOut", common.HexToAddress(to), amount.BigInt())
	if err != nil {
		return "", err
	}
	return tx.Hash().String(), nil
}

func (c *Client) Claims(ctx context.Context, srcChainId, destChainId uint32, sequence uint64,
	timestamp uint64, payload []byte, voteAddrSet []uint64, aggSignature []byte, txOption gnfdSdkTypes.TxOption,
) (*sdk.TxResponse, error) {
	return nil, errCrossChainFeatureRemoved
}

func (c *Client) GetChannelSendSequence(ctx context.Context, destChainId sdk.ChainID, channelId uint32) (uint64, error) {
	return 0, errCrossChainFeatureRemoved
}

func (c *Client) GetChannelReceiveSequence(ctx context.Context, destChainId sdk.ChainID, channelId uint32) (uint64, error) {
	return 0, errCrossChainFeatureRemoved
}

func (c *Client) GetInturnRelayer(ctx context.Context, req *oracletypes.QueryInturnRelayerRequest) (*oracletypes.QueryInturnRelayerResponse, error) {
	return nil, errCrossChainFeatureRemoved
}

func (c *Client) GetCrossChainPackage(ctx context.Context, destChainId sdk.ChainID, channelId uint32, sequence uint64) ([]byte, error) {
	return nil, errCrossChainFeatureRemoved
}

func (c *Client) MirrorGroup(ctx context.Context, destChainId sdk.ChainID, groupId math.Uint, groupName string, txOption gnfdSdkTypes.TxOption) (*sdk.TxResponse, error) {
	return nil, errCrossChainFeatureRemoved
}

func (c *Client) MirrorBucket(ctx context.Context, destChainId sdk.ChainID, bucketId math.Uint, bucketName string, txOption gnfdSdkTypes.TxOption) (*sdk.TxResponse, error) {
	return nil, errCrossChainFeatureRemoved
}

func (c *Client) MirrorObject(ctx context.Context, destChainId sdk.ChainID, objectId math.Uint, bucketName, objectName string, txOption gnfdSdkTypes.TxOption) (*sdk.TxResponse, error) {
	return nil, errCrossChainFeatureRemoved
}
