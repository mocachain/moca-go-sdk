package client

import (
	hashlib "github.com/mocachain/moca-common/go/hash"
	storageTypes "github.com/mocachain/moca/v2/x/storage/types"
)

func toStorageRedundancyType(redundancyType hashlib.RedundancyType) storageTypes.RedundancyType {
	switch redundancyType {
	case hashlib.RedundancyReplicaType:
		return storageTypes.REDUNDANCY_REPLICA_TYPE
	default:
		return storageTypes.REDUNDANCY_EC_TYPE
	}
}
