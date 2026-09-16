package client

import (
	"net/http"
	"strings"
	"testing"

	httplib "github.com/mocachain/moca-common/go/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetRequestNonceIsUniquePerRequest(t *testing.T) {
	req1, _ := http.NewRequest(http.MethodGet, "https://sp.example.com/x", nil)
	req2, _ := http.NewRequest(http.MethodGet, "https://sp.example.com/x", nil)
	require.NoError(t, setRequestNonce(req1))
	require.NoError(t, setRequestNonce(req2))

	n1 := req1.Header.Get(httplib.HTTPHeaderNonce)
	n2 := req2.Header.Get(httplib.HTTPHeaderNonce)
	assert.Len(t, n1, 32)
	assert.Len(t, n2, 32)
	assert.NotEqual(t, n1, n2)
}

func TestNonceJoinsTheSignedCanonicalRequest(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "https://sp.example.com/x", nil)
	baseline := httplib.GetMsgToSignInGNFD1Auth(req)
	require.NoError(t, setRequestNonce(req))
	signed := httplib.GetMsgToSignInGNFD1Auth(req)
	assert.NotEqual(t, baseline, signed)
}

func TestSetContentHashBindsReplayableBody(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPost, "https://sp.example.com/x", strings.NewReader("payload"))
	require.NoError(t, setContentHash(req))
	sum := req.Header.Get(httplib.HTTPHeaderContentSHA256)
	// sha256("payload")
	assert.Equal(t, "239f59ed55e737c77147cf55ad0c1b030b6d7ee748a7426952f9b852d5a935e5", sum)

	get, _ := http.NewRequest(http.MethodGet, "https://sp.example.com/x", nil)
	require.NoError(t, setContentHash(get))
	assert.Empty(t, get.Header.Get(httplib.HTTPHeaderContentSHA256))
}
