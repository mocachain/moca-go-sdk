package basesuite

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	storageTypes "github.com/mocachain/moca/v2/x/storage/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/mocachain/moca-go-sdk/client"
	"github.com/mocachain/moca-go-sdk/types"
	"github.com/stretchr/testify/suite"
)

var (
	ChainID      = envOrDefault("MOCA_E2E_CHAIN_ID", "moca_5151-1")
	LocalupDir   = resolveLocalupDir()
	MocadPath    = resolveMocadPath()
	Endpoint     = resolveEndpoint()
	GRPCEndpoint = resolveGRPCEndpoint()
	EVMEndpoint  = resolveEVMEndpoint()
)

func envOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func resolveEndpoint() string {
	if value := os.Getenv("MOCA_E2E_ENDPOINT"); value != "" {
		return value
	}
	return "http://localhost:26657"
}

func resolveEVMEndpoint() string {
	if value := os.Getenv("MOCA_E2E_EVM_ENDPOINT"); value != "" {
		return value
	}
	return "http://localhost:8545"
}

func resolveGRPCEndpoint() string {
	if value := os.Getenv("MOCA_E2E_GRPC_ENDPOINT"); value != "" {
		return value
	}
	return "localhost:19090"
}

func resolveLocalupDir() string {
	if value := os.Getenv("MOCA_E2E_LOCALUP_DIR"); value != "" {
		return value
	}

	candidates := []string{
		"../moca/deployment/localup/.local",
		"../../moca/deployment/localup/.local",
		"../../../moca/deployment/localup/.local",
		"../../../../moca/deployment/localup/.local",
	}

	for _, candidate := range candidates {
		cleaned := filepath.Clean(candidate)
		if _, err := os.Stat(cleaned); err == nil {
			return cleaned
		}
	}

	// Fall back to the CI sibling checkout layout.
	return filepath.Clean("../../../moca/deployment/localup/.local")
}

func resolveMocadPath() string {
	if value := os.Getenv("MOCA_E2E_MOCAD"); value != "" {
		return value
	}

	candidates := []string{
		filepath.Join(LocalupDir, "..", "..", "..", "build", "mocad"),
		"../moca/build/mocad",
		"../../moca/build/mocad",
		"../../../moca/build/mocad",
		"../../../../moca/build/mocad",
	}

	for _, candidate := range candidates {
		cleaned := filepath.Clean(candidate)
		if info, err := os.Stat(cleaned); err == nil && !info.IsDir() {
			return cleaned
		}
	}

	return filepath.Clean(filepath.Join(LocalupDir, "..", "..", "..", "build", "mocad"))
}

func exportLocalPrivateKey(name, homeDir string) (string, error) {
	cmd := exec.Command(
		MocadPath,
		"keys",
		"export",
		name,
		"--unarmored-hex",
		"--unsafe",
		"--keyring-backend",
		"test",
		"--home",
		filepath.Clean(homeDir),
	)
	cmd.Stdin = strings.NewReader("y\n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("export private key for %s failed: %w: %s", name, err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

func loadLocalAccount(name, homeDir string) (*types.Account, string, error) {
	privateKey, err := exportLocalPrivateKey(name, homeDir)
	if err != nil {
		return nil, "", err
	}

	account, err := types.NewAccountFromPrivateKey(name, privateKey)
	if err != nil {
		return nil, "", err
	}
	return account, privateKey, nil
}

type BaseSuite struct {
	suite.Suite
	DefaultAccount    *types.Account
	DefaultPrivateKey string
	Client            client.IClient
	ClientContext     context.Context
	ChallengeClient   client.IClient
}

func (s *BaseSuite) NewChallengeClient() {
	challengerHome := filepath.Join(LocalupDir, "challenger0")
	if _, err := os.Stat(challengerHome); err != nil {
		s.T().Logf("challenge client skipped: %s not found", challengerHome)
		return
	}

	challengeAcc, priKey, err := loadLocalAccount("challenger0", filepath.Join(LocalupDir, "challenger0"))
	s.Require().NoError(err)
	s.ChallengeClient, err = client.New(ChainID, Endpoint, EVMEndpoint, priKey, client.Option{
		DefaultAccount: challengeAcc,
		GrpcAddress:    GRPCEndpoint,
		GrpcDialOption: grpc.WithTransportCredentials(insecure.NewCredentials()),
	})
	s.Require().NoError(err)
}

func (s *BaseSuite) SetupSuite() {
	account, priKey, err := loadLocalAccount("validator0", filepath.Join(LocalupDir, "validator0"))
	s.Require().NoError(err)
	s.DefaultPrivateKey = priKey
	s.Client, err = client.New(ChainID, Endpoint, EVMEndpoint, priKey, client.Option{
		DefaultAccount: account,
		GrpcAddress:    GRPCEndpoint,
		GrpcDialOption: grpc.WithTransportCredentials(insecure.NewCredentials()),
	})
	s.Require().NoError(err)
	s.ClientContext = context.Background()
	s.DefaultAccount = account
	s.NewChallengeClient()
}

func (s *BaseSuite) WaitSealObject(bucketName string, objectName string) {
	startCheckTime := time.Now()
	var (
		objectDetail *types.ObjectDetail
		err          error
	)

	// wait 300s
	for i := 0; i < 100; i++ {
		objectDetail, err = s.Client.HeadObject(s.ClientContext, bucketName, objectName)
		s.Require().NoError(err)
		if objectDetail.ObjectInfo.GetObjectStatus() == storageTypes.OBJECT_STATUS_SEALED && !objectDetail.ObjectInfo.GetIsUpdating() {
			break
		}
		time.Sleep(3 * time.Second)
	}

	s.Require().Equal(objectDetail.ObjectInfo.GetObjectStatus().String(), "OBJECT_STATUS_SEALED")
	s.T().Logf("---> Wait Seal Object cost %d ms, <---", time.Since(startCheckTime).Milliseconds())
}
