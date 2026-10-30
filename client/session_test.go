package client

import (
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

	mu        sync.Mutex
	estimates int
	sent      []*ethtypes.Transaction
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
			node.mu.Lock()
			node.estimates++
			node.mu.Unlock()
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

func (node *fakeEvmNode) counts() (estimates int, sent []*ethtypes.Transaction) {
	node.mu.Lock()
	defer node.mu.Unlock()
	return node.estimates, append([]*ethtypes.Transaction(nil), node.sent...)
}

// deleteBucket sends a storage precompile call the way the client's EVM paths do.
func deleteBucket(t *testing.T, client *ethclient.Client, gasLimit uint64) error {
	t.Helper()
	txOpts, err := CreateTxOpts(context.Background(), client, testEvmPrivateKey, big.NewInt(5151), gasLimit, 0)
	require.NoError(t, err)
	session, err := CreateStorageSession(client, *txOpts, mocadTypes.StorageAddress)
	require.NoError(t, err)
	_, err = session.DeleteBucket("bucket")
	return err
}

func TestStorageSession_SendsBufferedEstimate(t *testing.T) {
	node := &fakeEvmNode{estimate: 100_000}
	require.NoError(t, deleteBucket(t, newFakeEvmNode(t, node), GasLimitEstimate))

	estimates, sent := node.counts()
	require.Equal(t, 1, estimates)
	require.Len(t, sent, 1)
	require.Equal(t, uint64(135_000), sent[0].Gas())
}

func TestStorageSession_UnestimableCallIsNotSent(t *testing.T) {
	node := &fakeEvmNode{estimateErr: "execution reverted: No such bucket"}
	err := deleteBucket(t, newFakeEvmNode(t, node), GasLimitEstimate)
	require.ErrorContains(t, err, "failed to estimate gas")
	require.ErrorContains(t, err, "No such bucket")

	_, sent := node.counts()
	require.Empty(t, sent, "a call the node cannot estimate must not be sent with a fallback gas limit")
}

func TestStorageSession_ExplicitGasLimitIsNotEstimated(t *testing.T) {
	node := &fakeEvmNode{estimateErr: "must not be called"}
	require.NoError(t, deleteBucket(t, newFakeEvmNode(t, node), 300_000))

	estimates, sent := node.counts()
	require.Zero(t, estimates)
	require.Len(t, sent, 1)
	require.Equal(t, uint64(300_000), sent[0].Gas())
}

func TestTxGasLimit(t *testing.T) {
	require.Equal(t, uint64(GasLimitEstimate), txGasLimit(nil))
	require.Equal(t, uint64(GasLimitEstimate), txGasLimit(&gnfdSdkTypes.TxOption{GasLimit: 70_000}),
		"without NoSimulate a gas limit is not explicit, as for a Cosmos transaction")
	require.Equal(t, uint64(70_000), txGasLimit(&gnfdSdkTypes.TxOption{NoSimulate: true, GasLimit: 70_000}))
}

func TestTransfer_GasLimit(t *testing.T) {
	newClient := func(node *fakeEvmNode) *Client {
		return &Client{privateKey: testEvmPrivateKey, evmClient: newFakeEvmNode(t, node), evmChainID: big.NewInt(5151)}
	}
	to := "0x00000000000000000000000000000000000000ff"

	t.Run("estimated", func(t *testing.T) {
		node := &fakeEvmNode{estimate: 21_000}
		_, err := newClient(node).Transfer(context.Background(), to, math.NewInt(1), gnfdSdkTypes.TxOption{})
		require.NoError(t, err)
		_, sent := node.counts()
		require.Len(t, sent, 1)
		require.Equal(t, uint64(36_250), sent[0].Gas())
	})
	t.Run("not estimable is not sent", func(t *testing.T) {
		node := &fakeEvmNode{estimateErr: "insufficient funds for transfer"}
		_, err := newClient(node).Transfer(context.Background(), to, math.NewInt(1), gnfdSdkTypes.TxOption{})
		require.ErrorContains(t, err, "insufficient funds for transfer")
		_, sent := node.counts()
		require.Empty(t, sent)
	})
	t.Run("explicit", func(t *testing.T) {
		node := &fakeEvmNode{estimateErr: "must not be called"}
		_, err := newClient(node).Transfer(context.Background(), to, math.NewInt(1),
			gnfdSdkTypes.TxOption{NoSimulate: true, GasLimit: 30_000})
		require.NoError(t, err)
		estimates, sent := node.counts()
		require.Zero(t, estimates)
		require.Len(t, sent, 1)
		require.Equal(t, uint64(30_000), sent[0].Gas())
	})
}
