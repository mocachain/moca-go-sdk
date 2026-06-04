package client

import (
	"testing"

	"cosmossdk.io/math"
	"cosmossdk.io/x/feegrant"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	cmdcfg "github.com/mocachain/moca/v2/cmd/config"
	sdkkeys "github.com/mocachain/moca/v2/sdk/keys"
	sdktypes "github.com/mocachain/moca/v2/sdk/types"
)

func TestFeegrantNormalizeAccountAddress_HexInputReturnsBech32(t *testing.T) {
	km, err := sdkkeys.NewPrivateKeyManager("2a3f0f19fbcb057e053696879207324c24f601ab47db92676cc4958ea9089761")
	require.NoError(t, err)

	acc, normalizedAddr, err := normalizeFeeGrantAccountAddress(km.GetAddr().String())
	require.NoError(t, err)
	require.Equal(t, km.GetAddr(), acc)
	expectedAddr, err := cmdcfg.NewMultiPrefixBech32AccCodec().BytesToString(km.GetAddr())
	require.NoError(t, err)
	require.Equal(t, expectedAddr, normalizedAddr)
}

func TestFeegrantNormalizeAccountAddress_CosmosBech32InputReturnsBech32(t *testing.T) {
	km, err := sdkkeys.NewPrivateKeyManager("2a3f0f19fbcb057e053696879207324c24f601ab47db92676cc4958ea9089761")
	require.NoError(t, err)

	cosmosCodec := cmdcfg.NewMultiPrefixBech32Codec("cosmos", "moca")
	cosmosBech32, err := cosmosCodec.BytesToString(km.GetAddr())
	require.NoError(t, err)

	acc, normalizedAddr, err := normalizeFeeGrantAccountAddress(cosmosBech32)
	require.NoError(t, err)
	require.Equal(t, km.GetAddr(), acc)
	expectedAddr, err := cmdcfg.NewMultiPrefixBech32AccCodec().BytesToString(km.GetAddr())
	require.NoError(t, err)
	require.Equal(t, expectedAddr, normalizedAddr)
}

func TestFeegrantNewGrantAllowanceMsg_UsesBech32Addresses(t *testing.T) {
	granterKM, err := sdkkeys.NewPrivateKeyManager("2a3f0f19fbcb057e053696879207324c24f601ab47db92676cc4958ea9089761")
	require.NoError(t, err)
	granteeKM, err := sdkkeys.NewPrivateKeyManager("e04eb74dc6bf9ceb89e584ee57d0c2f4f86d88c1604ec54f76a8c76cffb34417")
	require.NoError(t, err)

	allowance := feegrant.BasicAllowance{
		SpendLimit: sdk.NewCoins(sdk.NewCoin(sdktypes.Denom, math.NewIntWithDecimal(1, 18))),
	}

	msg, err := newFeeGrantAllowanceMsg(granterKM.GetAddr(), granteeKM.GetAddr().String(), &allowance)
	require.NoError(t, err)

	_, granterAddrBech32, err := normalizeFeeGrantAccountAddress(granterKM.GetAddr().String())
	require.NoError(t, err)
	_, granteeAddrBech32, err := normalizeFeeGrantAccountAddress(granteeKM.GetAddr().String())
	require.NoError(t, err)

	require.Equal(t, granterAddrBech32, msg.Granter)
	require.Equal(t, granteeAddrBech32, msg.Grantee)
}

func TestFeegrantNewRevokeMsg_UsesBech32Addresses(t *testing.T) {
	granterKM, err := sdkkeys.NewPrivateKeyManager("2a3f0f19fbcb057e053696879207324c24f601ab47db92676cc4958ea9089761")
	require.NoError(t, err)
	granteeKM, err := sdkkeys.NewPrivateKeyManager("e04eb74dc6bf9ceb89e584ee57d0c2f4f86d88c1604ec54f76a8c76cffb34417")
	require.NoError(t, err)

	msg, err := newFeeGrantRevokeMsg(granterKM.GetAddr(), granteeKM.GetAddr().String())
	require.NoError(t, err)

	_, granterAddrBech32, err := normalizeFeeGrantAccountAddress(granterKM.GetAddr().String())
	require.NoError(t, err)
	_, granteeAddrBech32, err := normalizeFeeGrantAccountAddress(granteeKM.GetAddr().String())
	require.NoError(t, err)

	require.Equal(t, granterAddrBech32, msg.Granter)
	require.Equal(t, granteeAddrBech32, msg.Grantee)
}
