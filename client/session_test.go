package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"cosmossdk.io/math"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"

	gnfdSdkTypes "github.com/mocachain/moca/v2/sdk/types"
	mocadTypes "github.com/mocachain/moca/v2/types"
)

const testEvmPrivateKey = "2a3f0f19fbcb057e053696879207324c24f601ab47db92676cc4958ea9089761"

// fakeEvmNode answers the JSON-RPC calls the EVM send paths make. eth_estimateGas
// returns estimate, or estimateErr as an RPC error; eth_sendRawTransaction records
// the transaction.
type fakeEvmNode struct {
	estimate    uint64
	estimateErr string

	mu   sync.Mutex
	sent []*ethtypes.Transaction
}

func newFakeEvmNode(t *testing.T, node *fakeEvmNode) *ethclient.Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var req struct {
			ID     json.RawMessage   `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		require.NoError(t, json.Unmarshal(body, &req))
		reply := map[string]any{"jsonrpc": "2.0", "id": req.ID}
		switch req.Method {
		case "eth_gasPrice":
			reply["result"] = "0x3b9aca00"
		case "eth_getTransactionCount":
			reply["result"] = "0x0"
		case "eth_chainId":
			reply["result"] = "0x141f"
		case "eth_estimateGas":
			if node.estimateErr != "" {
				reply["error"] = map[string]any{"code": 3, "message": node.estimateErr}
			} else {
				reply["result"] = hexutil.EncodeUint64(node.estimate)
			}
		case "eth_sendRawTransaction":
			var raw hexutil.Bytes
			require.NoError(t, json.Unmarshal(req.Params[0], &raw))
			tx := new(ethtypes.Transaction)
			require.NoError(t, tx.UnmarshalBinary(raw))
			node.mu.Lock()
			node.sent = append(node.sent, tx)
			node.mu.Unlock()
			reply["result"] = tx.Hash()
		default:
			reply["error"] = map[string]any{"code": -32601, "message": "method not found: " + req.Method}
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(reply))
	}))
	t.Cleanup(server.Close)
	client, err := ethclient.Dial(server.URL)
	require.NoError(t, err)
	t.Cleanup(client.Close)
	return client
}

// sentGas returns the gas limit of the one transaction the node received.
func (node *fakeEvmNode) sentGas(t *testing.T) uint64 {
	t.Helper()
	node.mu.Lock()
	defer node.mu.Unlock()
	require.Len(t, node.sent, 1)
	return node.sent[0].Gas()
}

type logLine struct {
	Level    string `json:"level"`
	Message  string `json:"message"`
	Error    string `json:"error"`
	Estimate uint64 `json:"estimate"`
	GasLimit uint64 `json:"gas_limit"`
}

// captureLogs redirects the client's logs for the rest of the test and returns a
// function that reads the lines logged so far.
func captureLogs(t *testing.T) func() []logLine {
	t.Helper()
	var buf bytes.Buffer
	prev := log.Logger
	log.Logger = zerolog.New(&buf).Level(zerolog.DebugLevel)
	t.Cleanup(func() { log.Logger = prev })
	return func() []logLine {
		var lines []logLine
		dec := json.NewDecoder(bytes.NewReader(buf.Bytes()))
		for dec.More() {
			var l logLine
			require.NoError(t, dec.Decode(&l))
			lines = append(lines, l)
		}
		return lines
	}
}

// deleteBucket sends a storage precompile call the way the client's EVM paths do.
func deleteBucket(t *testing.T, client *ethclient.Client) {
	t.Helper()
	txOpts, err := CreateTxOpts(context.Background(), client, testEvmPrivateKey, big.NewInt(5151), GasLimitEstimate, 0)
	require.NoError(t, err)
	session, err := CreateStorageSession(client, *txOpts, mocadTypes.StorageAddress)
	require.NoError(t, err)
	_, err = session.DeleteBucket("bucket")
	require.NoError(t, err)
}

func TestStorageSession_SendsBufferedEstimate(t *testing.T) {
	logs := captureLogs(t)
	node := &fakeEvmNode{estimate: 100_000}
	deleteBucket(t, newFakeEvmNode(t, node))

	require.Equal(t, uint64(135_000), node.sentGas(t))
	require.Equal(t, []logLine{{Level: "debug", Message: "gas estimated", Estimate: 100_000, GasLimit: 135_000}}, logs())
}

func TestStorageSession_FallsBackWhenEstimateFails(t *testing.T) {
	logs := captureLogs(t)
	node := &fakeEvmNode{estimateErr: "execution reverted: No such bucket"}
	deleteBucket(t, newFakeEvmNode(t, node))

	require.Equal(t, uint64(DefaultGasLimit), node.sentGas(t))
	require.Equal(t, []logLine{{
		Level:    "warn",
		Message:  "gas estimation failed, sending with the fallback gas limit",
		Error:    "execution reverted: No such bucket",
		GasLimit: DefaultGasLimit,
	}}, logs())
}

func TestTransfer_GasLimit(t *testing.T) {
	transfer := func(node *fakeEvmNode) {
		c := &Client{privateKey: testEvmPrivateKey, evmClient: newFakeEvmNode(t, node), evmChainID: big.NewInt(5151)}
		_, err := c.Transfer(context.Background(), "0x00000000000000000000000000000000000000ff", math.NewInt(1), gnfdSdkTypes.TxOption{})
		require.NoError(t, err)
	}

	t.Run("estimated", func(t *testing.T) {
		logs := captureLogs(t)
		node := &fakeEvmNode{estimate: 21_000}
		transfer(node)
		require.Equal(t, uint64(36_250), node.sentGas(t))
		require.Equal(t, []logLine{{Level: "debug", Message: "gas estimated", Estimate: 21_000, GasLimit: 36_250}}, logs())
	})
	t.Run("fallback", func(t *testing.T) {
		logs := captureLogs(t)
		node := &fakeEvmNode{estimateErr: "insufficient funds for transfer"}
		transfer(node)
		require.Equal(t, uint64(21_000), node.sentGas(t))
		require.Equal(t, []logLine{{
			Level:    "warn",
			Message:  "gas estimation failed, sending with the fallback gas limit",
			Error:    "insufficient funds for transfer",
			GasLimit: 21_000,
		}}, logs())
	})
}
