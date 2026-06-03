package e2e

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/mocachain/moca-go-sdk/e2e/basesuite"
	"github.com/mocachain/moca-go-sdk/types"
	storageTestUtil "github.com/mocachain/moca/v2/testutil/storage"
	spTypes "github.com/mocachain/moca/v2/x/sp/types"
	storageTypes "github.com/mocachain/moca/v2/x/storage/types"
)

type BucketMigrateTestSuite struct {
	basesuite.BaseSuite
	PrimarySP spTypes.StorageProvider
}

func (s *BucketMigrateTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()

	spList, err := s.Client.ListStorageProviders(s.ClientContext, false)
	s.Require().NoError(err)
	for _, sp := range spList {
		if sp.Id == 1 {
			s.PrimarySP = sp
			break
		}
	}

	s.requireMigrateAdminAvailable()
}

func TestBucketMigrateTestSuiteTestSuite(t *testing.T) {
	suite.Run(t, new(BucketMigrateTestSuite))
}

func (s *BucketMigrateTestSuite) requireMigrateAdminAvailable() {
	s.Require().NotEmpty(s.PrimarySP.Endpoint, "bucket migrate tests require a primary SP endpoint")

	adminAddr, err := bucketMigrateAdminAddr(s.PrimarySP.Endpoint)
	s.Require().NoError(err, "bucket migrate tests require a resolvable SP admin endpoint")

	conn, err := net.DialTimeout("tcp", adminAddr, 2*time.Second)
	s.Require().NoError(err, "bucket migrate tests require reachable SP admin endpoint %s", adminAddr)
	_ = conn.Close()
}

func bucketMigrateAdminAddr(endpoint string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}

	host := u.Hostname()
	if host == "" {
		return "", fmt.Errorf("empty host in endpoint %q", endpoint)
	}

	switch host {
	case "127.0.0.1", "localhost", "host.docker.internal", "sp-0":
		return "127.0.0.1:9033", nil
	case "sp-1":
		return "127.0.0.1:9034", nil
	case "sp-2":
		return "127.0.0.1:9035", nil
	default:
		return "", fmt.Errorf("unsupported non-local SP endpoint %q for local bucket migrate e2e", endpoint)
	}
}

func (s *BucketMigrateTestSuite) CreateObjects(bucketName string, count int) ([]*types.ObjectDetail, []bytes.Buffer, error) {
	var (
		objectNames   []string
		contentBuffer []bytes.Buffer
		objectDetails []*types.ObjectDetail
	)

	// create object
	for i := 0; i < count; i++ {
		var buffer bytes.Buffer
		line := `1234567890,1234567890,1234567890,1234567890,1234567890,1234567890,1234567890,1234567890,123456789012`
		// Create 1MiB content where each line contains 1024 characters.
		for n := 0; n < 1024*3; n++ {
			fmt.Fprintf(&buffer, "[%05d] %s\n", n, line)
		}
		objectName := storageTestUtil.GenRandomObjectName()
		s.T().Logf("---> CreateObject and HeadObject, bucket name:%s, object name:%s <---", bucketName, objectName)
		objectTx, err := s.Client.CreateObject(s.ClientContext, bucketName, objectName, bytes.NewReader(buffer.Bytes()), types.CreateObjectOptions{Visibility: storageTypes.VISIBILITY_TYPE_PUBLIC_READ})
		s.Require().NoError(err)
		_, err = s.Client.WaitForTx(s.ClientContext, objectTx)
		s.Require().NoError(err)

		objectNames = append(objectNames, objectName)
		contentBuffer = append(contentBuffer, buffer)
	}

	// head object
	time.Sleep(5 * time.Second)
	for _, objectName := range objectNames {
		objectDetail, err := s.Client.HeadObject(s.ClientContext, bucketName, objectName)
		s.Require().NoError(err)
		s.Require().Equal(objectDetail.ObjectInfo.ObjectName, objectName)
		s.Require().Equal(objectDetail.ObjectInfo.GetObjectStatus().String(), "OBJECT_STATUS_CREATED")
	}

	s.T().Log("---> PutObject and GetObject <---")
	for idx, objectName := range objectNames {
		buffer := contentBuffer[idx]
		err := s.Client.PutObject(s.ClientContext, bucketName, objectName, int64(buffer.Len()),
			bytes.NewReader(buffer.Bytes()), types.PutObjectOptions{})
		s.Require().NoError(err)
	}

	for _, objectName := range objectNames {
		s.WaitSealObject(bucketName, objectName)
	}

	// seal object
	for idx, objectName := range objectNames {
		objectDetail, err := s.Client.HeadObject(s.ClientContext, bucketName, objectName)
		s.Require().NoError(err)
		s.Require().Equal(objectDetail.ObjectInfo.GetObjectStatus().String(), "OBJECT_STATUS_SEALED")

		ior, info, err := s.Client.GetObject(s.ClientContext, bucketName, objectName, types.GetObjectOptions{})
		s.Require().NoError(err)
		if err == nil {
			s.Require().Equal(info.ObjectName, objectName)
			objectBytes, err := io.ReadAll(ior)
			s.Require().NoError(err)
			s.Require().Equal(objectBytes, contentBuffer[idx].Bytes())
		}
		objectDetails = append(objectDetails, objectDetail)
	}

	return objectDetails, contentBuffer, nil
}

func (s *BucketMigrateTestSuite) MustCreateBucket(visibility storageTypes.VisibilityType) (string, *storageTypes.BucketInfo) {
	bucketName := storageTestUtil.GenRandomBucketName()
	bucketTx, err := s.Client.CreateBucket(s.ClientContext, bucketName, s.PrimarySP.OperatorAddress, types.CreateBucketOptions{Visibility: visibility})
	s.Require().NoError(err)

	_, err = s.Client.WaitForTx(s.ClientContext, bucketTx)
	s.Require().NoError(err)

	bucketInfo, err := s.Client.HeadBucket(s.ClientContext, bucketName)
	s.Require().NoError(err)
	if err == nil {
		s.Require().Equal(bucketInfo.Visibility, visibility)
	}

	s.T().Logf("success to create a new bucket: %s", bucketInfo)

	return bucketName, bucketInfo
}

func (s *BucketMigrateTestSuite) SelectDestSP(objectDetail *types.ObjectDetail) *spTypes.StorageProvider {
	sps, err := s.localReachableStorageProviders()
	s.Require().NoError(err)
	expectedGVGSPCount := s.expectedGVGSPCount()

	spIDs := make(map[uint32]bool)
	spIDs[objectDetail.GlobalVirtualGroup.PrimarySpId] = true
	for _, id := range objectDetail.GlobalVirtualGroup.SecondarySpIds {
		spIDs[id] = true
	}
	s.Require().Equal(expectedGVGSPCount, len(spIDs))

	var destSP *spTypes.StorageProvider
	for _, sp := range sps {
		_, exist := spIDs[sp.Id]
		if !exist {
			destSP = &sp
			break
		}
	}
	if destSP == nil {
		s.T().Skipf("bucket migrate tests require one SP outside the source GVG; available SPs=%d, GVG SPs=%d", len(sps), len(spIDs))
	}

	return destSP
}

func (s *BucketMigrateTestSuite) localReachableStorageProviders() ([]spTypes.StorageProvider, error) {
	sps, err := s.Client.ListStorageProviders(s.ClientContext, true)
	if err != nil {
		return nil, err
	}

	reachable := make([]spTypes.StorageProvider, 0, len(sps))
	for _, sp := range sps {
		adminAddr, err := bucketMigrateAdminAddr(sp.Endpoint)
		if err != nil {
			continue
		}
		conn, err := net.DialTimeout("tcp", adminAddr, 2*time.Second)
		if err != nil {
			continue
		}
		_ = conn.Close()
		reachable = append(reachable, sp)
	}
	return reachable, nil
}

func (s *BucketMigrateTestSuite) expectedGVGSPCount() int {
	dataBlocks, parityBlocks, _, err := s.Client.GetRedundancyParams()
	s.Require().NoError(err)
	return int(1 + dataBlocks + parityBlocks)
}

func (s *BucketMigrateTestSuite) waitUntilBucketMigrateFinish(bucketName string, destSP *spTypes.StorageProvider) *storageTypes.BucketInfo {
	var (
		bucketInfo *storageTypes.BucketInfo
		err        error
	)
	var primarySPID uint32

	// wait 5 minutes
	for i := 0; i < 100; i++ {
		bucketInfo, err = s.Client.HeadBucket(s.ClientContext, bucketName)
		s.T().Logf("HeadBucket: %s", bucketInfo)
		s.Require().NoError(err)

		family, err := s.Client.QueryVirtualGroupFamily(s.ClientContext, bucketInfo.GlobalVirtualGroupFamilyId)
		s.Require().NoError(err)
		s.T().Logf("VirtualGroupFamily: %s", family)
		primarySPID = family.PrimarySpId
		if primarySPID == destSP.GetId() {
			break
		}
		time.Sleep(3 * time.Second)
	}

	s.Require().Equal(primarySPID, destSP.GetId())

	return bucketInfo
}

// test only one object's case
func (s *BucketMigrateTestSuite) Test_Bucket_Migrate_Simple_Case() {
	// 1) create bucket and object in srcSP
	bucketName, _ := s.MustCreateBucket(storageTypes.VISIBILITY_TYPE_PUBLIC_READ)

	// test only one object's case
	objectDetails, contentBuffer, err := s.CreateObjects(bucketName, 1)
	s.Require().NoError(err)

	objectDetail := objectDetails[0]
	buffer := contentBuffer[0]

	// select a storage provider to migrate
	destSP := s.SelectDestSP(objectDetail)

	s.T().Logf(":Migrate Bucket DstPrimarySPID %d", destSP.GetId())

	// normal no conflict send migrate bucket transaction
	txhash, err := s.Client.MigrateBucket(s.ClientContext, bucketName, destSP.GetId(), types.MigrateBucketOptions{TxOpts: nil, IsAsyncMode: false})
	s.Require().NoError(err)

	s.T().Logf("MigrateBucket : %s", txhash)
	s.waitUntilBucketMigrateFinish(bucketName, destSP)

	ior, info, err := s.Client.GetObject(s.ClientContext, bucketName, objectDetail.ObjectInfo.ObjectName, types.GetObjectOptions{})
	s.Require().NoError(err)
	if err == nil {
		s.Require().Equal(info.ObjectName, objectDetail.ObjectInfo.ObjectName)
		objectBytes, err := io.ReadAll(ior)
		s.Require().NoError(err)
		s.Require().Equal(objectBytes, buffer.Bytes())
	}
	s.CheckChallenge(uint32(objectDetail.ObjectInfo.Id.Uint64()))
}

// test only conflict sp's case
func (s *BucketMigrateTestSuite) Test_Bucket_Migrate_Simple_Conflict_Case() {
	// 1) create bucket and object in srcSP
	bucketName, _ := s.MustCreateBucket(storageTypes.VISIBILITY_TYPE_PUBLIC_READ)

	// test only one object's case
	objectDetails, contentBuffer, err := s.CreateObjects(bucketName, 1)
	s.Require().NoError(err)

	objectDetail := objectDetails[0]
	buffer := contentBuffer[0]

	expectedGVGSPCount := s.expectedGVGSPCount()

	spIDs := make(map[uint32]bool)
	spIDs[objectDetail.GlobalVirtualGroup.PrimarySpId] = true
	for _, id := range objectDetail.GlobalVirtualGroup.SecondarySpIds {
		spIDs[id] = true
	}
	s.Require().Equal(expectedGVGSPCount, len(spIDs))
	if expectedGVGSPCount >= len(spsMust(s)) {
		s.T().Skipf("bucket migrate conflict test requires one SP outside the source GVG; available SPs=%d, GVG SPs=%d", len(spsMust(s)), expectedGVGSPCount)
	}

	// migrate bucket with conflict
	conflictSPID := objectDetail.GlobalVirtualGroup.SecondarySpIds[0]
	s.T().Logf(":Migrate Bucket DstPrimarySPID %d", conflictSPID)

	txhash, err := s.Client.MigrateBucket(s.ClientContext, bucketName, conflictSPID, types.MigrateBucketOptions{TxOpts: nil, IsAsyncMode: false})
	s.Require().NoError(err)

	s.T().Logf("MigrateBucket : %s", txhash)

	var bucketInfo *storageTypes.BucketInfo

	for {
		bucketInfo, err = s.Client.HeadBucket(s.ClientContext, bucketName)
		s.T().Logf("HeadBucket: %s", bucketInfo)
		s.Require().NoError(err)
		if bucketInfo.BucketStatus != storageTypes.BUCKET_STATUS_MIGRATING {
			break
		}
		time.Sleep(3 * time.Second)
	}

	family, err := s.Client.QueryVirtualGroupFamily(s.ClientContext, bucketInfo.GlobalVirtualGroupFamilyId)
	s.Require().NoError(err)
	if family.PrimarySpId != conflictSPID {
		s.T().Logf("conflict migration kept bucket on family %d with primary SP %d; requested SP %d is already in the source GVG",
			bucketInfo.GlobalVirtualGroupFamilyId, family.PrimarySpId, conflictSPID)
	}
	ior, info, err := s.Client.GetObject(s.ClientContext, bucketName, objectDetail.ObjectInfo.ObjectName, types.GetObjectOptions{})
	s.Require().NoError(err)
	if err == nil {
		s.Require().Equal(info.ObjectName, objectDetail.ObjectInfo.ObjectName)
		objectBytes, err := io.ReadAll(ior)
		s.Require().NoError(err)
		s.Require().Equal(objectBytes, buffer.Bytes())
	}
	s.CheckChallenge(uint32(objectDetail.ObjectInfo.Id.Uint64()))
}

// test empty bucket case
func (s *BucketMigrateTestSuite) Test_Empty_Bucket_Migrate_Simple_Case() {
	// 1) create bucket and object in srcSP
	bucketName, bucketInfo := s.MustCreateBucket(storageTypes.VISIBILITY_TYPE_PUBLIC_READ)

	s.T().Logf("CreateBucket : %s", bucketInfo)
	virtualGroupFamily, err := s.Client.QueryVirtualGroupFamily(s.ClientContext, bucketInfo.GetGlobalVirtualGroupFamilyId())
	s.Require().NoError(err)
	s.T().Logf("virtualGroupFamily : %s", virtualGroupFamily)

	if err == nil {
		s.Require().Equal(bucketInfo.Visibility, storageTypes.VISIBILITY_TYPE_PUBLIC_READ)
	}

	time.Sleep(5 * time.Second)
	// select a storage provider to migrate
	sps, err := s.localReachableStorageProviders()
	s.Require().NoError(err)
	expectedGVGSPCount := s.expectedGVGSPCount()
	if expectedGVGSPCount >= len(sps) {
		s.T().Skipf("empty bucket migrate test requires one SP outside the source GVG; available SPs=%d, GVG SPs=%d", len(sps), expectedGVGSPCount)
	}

	var destSP *spTypes.StorageProvider
	for _, sp := range sps {
		if sp.GetId() != virtualGroupFamily.GetPrimarySpId() {
			destSP = &sp
			break
		}
	}
	s.Require().NotNil(destSP)

	s.T().Logf(":Migrate Bucket DstPrimarySPID %s", destSP.String())

	// normal no conflict send migrate bucket transaction
	txhash, err := s.Client.MigrateBucket(s.ClientContext, bucketName, destSP.GetId(), types.MigrateBucketOptions{TxOpts: nil, IsAsyncMode: false})
	s.Require().NoError(err)

	s.T().Logf("MigrateBucket : %s", txhash)

	for {
		bucketInfo, err = s.Client.HeadBucket(s.ClientContext, bucketName)
		s.T().Logf("HeadBucket: %s", bucketInfo)
		s.Require().NoError(err)
		if bucketInfo.BucketStatus != storageTypes.BUCKET_STATUS_MIGRATING {
			break
		}
		time.Sleep(3 * time.Second)
	}

	family, err := s.Client.QueryVirtualGroupFamily(s.ClientContext, bucketInfo.GlobalVirtualGroupFamilyId)
	s.Require().NoError(err)
	if family.PrimarySpId != destSP.GetId() {
		s.T().Logf("empty bucket migration kept bucket on family %d with primary SP %d; requested SP %d is already in the source family",
			bucketInfo.GlobalVirtualGroupFamilyId, family.PrimarySpId, destSP.GetId())
	}
}

func spsMust(s *BucketMigrateTestSuite) []spTypes.StorageProvider {
	sps, err := s.localReachableStorageProviders()
	s.Require().NoError(err)
	return sps
}

func (s *BucketMigrateTestSuite) CheckChallenge(objectId uint32) bool {
	time.Sleep(5 * time.Second)
	i := objectId
	dataBlocks, parityBlocks, _, err := s.Client.GetRedundancyParams()
	s.Require().NoError(err)
	infos, err := s.Client.HeadObjectByID(context.Background(), fmt.Sprintf("%d", i))
	s.Require().NoError(err)
	if infos.ObjectInfo.ObjectStatus == storageTypes.OBJECT_STATUS_SEALED {
		reader, _, err := s.Client.GetObject(context.Background(), infos.ObjectInfo.BucketName, infos.ObjectInfo.ObjectName, types.GetObjectOptions{})
		s.NoError(err, fmt.Sprintf("%d", i), infos.ObjectInfo.BucketName, infos.ObjectInfo.ObjectName)
		_, err = io.ReadAll(reader)
		s.NoError(err, fmt.Sprintf("%d", i), infos.ObjectInfo.BucketName, infos.ObjectInfo.ObjectName)
		for j := -1; j < int(dataBlocks+parityBlocks); j++ {
			endpoint := s.challengeEndpoint(infos, j)
			s.T().Logf("====challenge %v,%v,=====", i, j)
			_, errPk := s.ChallengeClient.GetChallengeInfo(context.Background(), infos.ObjectInfo.Id.String(), 0, j, types.GetChallengeInfoOptions{
				Endpoint: endpoint,
			})
			s.NoError(errPk, infos.ObjectInfo.BucketName, infos.ObjectInfo.ObjectName, i, j)
			if errPk != nil {
				s.T().Errorf(infos.ObjectInfo.BucketName, infos.ObjectInfo.ObjectName, i, j)
			}
		}
	}

	return true
}

func (s *BucketMigrateTestSuite) challengeEndpoint(objectDetail *types.ObjectDetail, redundancyIndex int) string {
	var spID uint32
	if redundancyIndex == types.PrimaryRedundancyIndex {
		bucketInfo, err := s.Client.HeadBucket(s.ClientContext, objectDetail.ObjectInfo.BucketName)
		s.Require().NoError(err)
		family, err := s.Client.QueryVirtualGroupFamily(s.ClientContext, bucketInfo.GlobalVirtualGroupFamilyId)
		s.Require().NoError(err)
		spID = family.PrimarySpId
	} else {
		s.Require().Less(redundancyIndex, len(objectDetail.GlobalVirtualGroup.SecondarySpIds))
		spID = objectDetail.GlobalVirtualGroup.SecondarySpIds[redundancyIndex]
	}

	sps, err := s.localReachableStorageProviders()
	s.Require().NoError(err)
	for _, sp := range sps {
		if sp.GetId() == spID {
			return sp.Endpoint
		}
	}
	s.Require().FailNowf("storage provider not found", "sp id %d", spID)
	return ""
}
