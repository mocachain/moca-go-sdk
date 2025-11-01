package utils

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/evmos/evmos/v12/types/common"
	permTypes "github.com/evmos/evmos/v12/x/permission/types"
	"github.com/mocachain/moca-go-sdk/types"
)

// NewStatement return the statement of permission module
func NewStatement(actions []permTypes.ActionType, effect permTypes.Effect,
	resource []string, opts types.NewStatementOptions,
) permTypes.Statement {
	statement := permTypes.Statement{
		Actions:        actions,
		Effect:         effect,
		Resources:      resource,
		ExpirationTime: opts.StatementExpireTime,
	}

	if opts.LimitSize != 0 {
		statement.LimitSize = &common.UInt64Value{Value: opts.LimitSize}
	}

	return statement
}

// NewPrincipalWithAccount return the marshaled Principal string which indicates the account
func NewPrincipalWithAccount(principalAddr sdk.AccAddress) (types.Principal, error) {
	p := permTypes.NewPrincipalWithAccount(principalAddr)
	principalBytes, err := p.Marshal()
	if err != nil {
		return "", err
	}
	return types.Principal(principalBytes), nil
}

// NewPrincipalWithGroupId return the marshaled Principal string which indicates the group
func NewPrincipalWithGroupId(groupId uint64) (types.Principal, error) {
	p := permTypes.NewPrincipalWithGroupID(sdkmath.NewUint(groupId))
	principalBytes, err := p.Marshal()
	if err != nil {
		return "", err
	}
	return types.Principal(principalBytes), nil
}
