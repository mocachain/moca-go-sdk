package basesuite

import (
	"context"
	"fmt"
	"net"
	"net/http"
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
	return "http://127.0.0.1:26657"
}

func resolveEVMEndpoint() string {
	if value := os.Getenv("MOCA_E2E_EVM_ENDPOINT"); value != "" {
		return value
	}
	return "http://127.0.0.1:8545"
}

func resolveGRPCEndpoint() string {
	if value := os.Getenv("MOCA_E2E_GRPC_ENDPOINT"); value != "" {
		return value
	}
	return "127.0.0.1:9090"
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
	if privateKey, err := exportLocalPrivateKeyFromDocker(name, homeDir); err == nil {
		return privateKey, nil
	}

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

func exportLocalPrivateKeyFromDocker(name, homeDir string) (string, error) {
	containerName := imageStackContainerName(homeDir)
	if containerName == "" {
		return "", fmt.Errorf("no docker container mapped for %s", homeDir)
	}

	relHome, err := filepath.Rel(LocalupDir, filepath.Clean(homeDir))
	if err != nil {
		return "", fmt.Errorf("resolve docker key path failed: %w", err)
	}

	sharedHome := filepath.ToSlash(filepath.Join("/shared", relHome))
	keyringDir := filepath.ToSlash(filepath.Join(sharedHome, "keyring-test"))
	tmpHome := filepath.ToSlash(filepath.Join("/tmp/moca-e2e-export", strings.ReplaceAll(relHome, "/", "-")))

	script := fmt.Sprintf(
		"set -euo pipefail\n"+
			"rm -rf %[1]s %[2]s\n"+
			"mkdir -p %[1]s %[2]s\n"+
			"cp -R %[3]s/. %[2]s/\n"+
			"printf 'y\\n' | mocad keys export %[4]s --unarmored-hex --unsafe --keyring-backend test --home %[1]s --keyring-dir %[2]s",
		shellQuote(tmpHome),
		shellQuote(filepath.ToSlash(filepath.Join(tmpHome, "keyring-test"))),
		shellQuote(keyringDir),
		shellQuote(name),
	)

	cmd := exec.Command("docker", "exec", containerName, "/bin/bash", "-lc", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker export private key for %s failed: %w: %s", name, err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

func imageStackContainerName(homeDir string) string {
	base := filepath.Base(filepath.Clean(homeDir))
	switch {
	case strings.HasPrefix(base, "validator-"):
		return "e2e-" + base + "-1"
	case strings.HasPrefix(base, "challenger-"):
		return "e2e-validator-0-1"
	default:
		return ""
	}
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
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

func loadFirstLocalAccount(candidates ...struct {
	name string
	home string
}) (*types.Account, string, error) {
	var lastErr error
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate.home); err != nil {
			lastErr = err
			continue
		}
		account, privateKey, err := loadLocalAccount(candidate.name, candidate.home)
		if err == nil {
			return account, privateKey, nil
		}
		lastErr = err
	}
	return nil, "", fmt.Errorf("load local account failed: %w", lastErr)
}

type BaseSuite struct {
	suite.Suite
	DefaultAccount    *types.Account
	DefaultPrivateKey string
	Client            client.IClient
	ClientContext     context.Context
	ChallengeClient   client.IClient
}

type LocalE2ETransport struct {
	base http.RoundTripper
}

func (t LocalE2ETransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL == nil {
		return t.transport().RoundTrip(req)
	}

	host := req.URL.Hostname()
	port := req.URL.Port()
	targetHost, targetPort, ok := localE2EHostPort(host, port)
	if !ok {
		return t.transport().RoundTrip(req)
	}

	cloned := req.Clone(req.Context())
	u := *req.URL
	u.Host = net.JoinHostPort(targetHost, targetPort)
	cloned.URL = &u
	if cloned.Host == "" {
		cloned.Host = req.URL.Host
	}
	return t.transport().RoundTrip(cloned)
}

func localE2EHostPort(host, port string) (string, string, bool) {
	if port == "" {
		port = "80"
	}
	if strings.EqualFold(host, "host.docker.internal") {
		return "127.0.0.1", port, true
	}
	if port != "9033" {
		return "", "", false
	}
	switch strings.ToLower(host) {
	case "sp-0":
		return "127.0.0.1", "9033", true
	case "sp-1":
		return "127.0.0.1", "9034", true
	case "sp-2":
		return "127.0.0.1", "9035", true
	case "sp-3":
		return "127.0.0.1", "9036", true
	default:
		return "", "", false
	}
}

func (t LocalE2ETransport) transport() http.RoundTripper {
	if t.base != nil {
		return t.base
	}
	return http.DefaultTransport
}

func LocalE2EClientOption(account *types.Account, transport http.RoundTripper) client.Option {
	return client.Option{
		DefaultAccount: account,
		GrpcAddress:    GRPCEndpoint,
		GrpcDialOption: grpc.WithTransportCredentials(insecure.NewCredentials()),
		Transport:      transport,
	}
}

func (s *BaseSuite) NewChallengeClient() {
	challengeAcc, priKey, err := loadFirstLocalAccount(
		struct {
			name string
			home string
		}{"challenger0", filepath.Join(LocalupDir, "challenger0")},
		struct {
			name string
			home string
		}{"challenger-0", filepath.Join(LocalupDir, "challenger-0")},
	)
	s.Require().NoError(err)
	s.ChallengeClient, err = client.New(ChainID, Endpoint, EVMEndpoint, priKey, LocalE2EClientOption(challengeAcc, LocalE2ETransport{}))
	s.Require().NoError(err)
}

func (s *BaseSuite) SetupSuite() {
	account, priKey, err := loadFirstLocalAccount(
		struct {
			name string
			home string
		}{"validator0", filepath.Join(LocalupDir, "validator0")},
		struct {
			name string
			home string
		}{"validator-0", filepath.Join(LocalupDir, "validator-0")},
	)
	s.Require().NoError(err)
	s.DefaultPrivateKey = priKey
	s.Client, err = client.New(ChainID, Endpoint, EVMEndpoint, priKey, LocalE2EClientOption(account, LocalE2ETransport{}))
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
	sealedCount := 0
	for i := 0; i < 100; i++ {
		objectDetail, err = s.Client.HeadObject(s.ClientContext, bucketName, objectName)
		s.Require().NoError(err)
		if objectDetail.ObjectInfo.GetObjectStatus() == storageTypes.OBJECT_STATUS_SEALED && !objectDetail.ObjectInfo.GetIsUpdating() {
			sealedCount++
			if sealedCount >= 2 {
				break
			}
		} else {
			sealedCount = 0
		}
		time.Sleep(3 * time.Second)
	}

	s.Require().Equal("OBJECT_STATUS_SEALED", objectDetail.ObjectInfo.GetObjectStatus().String())
	s.T().Logf("---> Wait Seal Object cost %d ms, <---", time.Since(startCheckTime).Milliseconds())
}
