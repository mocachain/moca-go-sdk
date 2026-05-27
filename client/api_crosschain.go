package client

import (
	"context"
	"errors"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gnfdSdkTypes "github.com/mocachain/moca/v2/sdk/types"
)

var errCrossChainFeatureRemoved = errors.New("cross-chain chain module APIs are not supported by github.com/mocachain/moca/v2 main")

type ICrossChainClient interface {
	TransferOut(ctx context.Context, toAddress string, amount math.Int, txOption gnfdSdkTypes.TxOption) (*sdk.TxResponse, error)
	Claims(ctx context.Context, srcShainId, destChainId uint32, sequence uint64, timestamp uint64, payload []byte, voteAddrSet []uint64, aggSignature []byte, txOption gnfdSdkTypes.TxOption) (*sdk.TxResponse, error)
	GetChannelSendSequence(ctx context.Context, destChainId sdk.ChainID, channelId uint32) (uint64, error)
	GetChannelReceiveSequence(ctx context.Context, destChainId sdk.ChainID, channelId uint32) (uint64, error)
	GetCrossChainPackage(ctx context.Context, destChainId sdk.ChainID, channelId uint32, sequence uint64) ([]byte, error)
	MirrorGroup(ctx context.Context, destChainId sdk.ChainID, groupId math.Uint, groupName string, txOption gnfdSdkTypes.TxOption) (*sdk.TxResponse, error)
	MirrorBucket(ctx context.Context, destChainId sdk.ChainID, bucketId math.Uint, bucketName string, txOption gnfdSdkTypes.TxOption) (*sdk.TxResponse, error)
	MirrorObject(ctx context.Context, destChainId sdk.ChainID, objectId math.Uint, bucketName, objectName string, txOption gnfdSdkTypes.TxOption) (*sdk.TxResponse, error)
}

func (c *Client) TransferOut(ctx context.Context, toAddress string, amount math.Int, txOption gnfdSdkTypes.TxOption) (*sdk.TxResponse, error) {
	return nil, errCrossChainFeatureRemoved
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
