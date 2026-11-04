package client

import (
	"encoding/hex"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	hashlib "github.com/mocachain/moca-common/go/hash"
	httplib "github.com/mocachain/moca-common/go/http"
	"github.com/mocachain/moca-go-sdk/types"
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

func TestSignRequestSendsNonceOnlyWhenEnabled(t *testing.T) {
	account, _, err := types.NewAccount("test")
	require.NoError(t, err)
	for _, enabled := range []bool{false, true} {
		c := &Client{defaultAccount: account, sendAuthNonce: enabled}
		req, err := http.NewRequest(http.MethodGet, "https://sp.example.com/x", nil)
		require.NoError(t, err)
		require.NoError(t, c.signRequest(req))

		nonce := req.Header.Get(httplib.HTTPHeaderNonce)
		if !enabled {
			assert.Empty(t, nonce)
			continue
		}
		assert.Len(t, nonce, 32)
		// recover the signer from the canonical request the way the SP does
		sig, err := hex.DecodeString(strings.TrimPrefix(req.Header.Get(types.HTTPHeaderAuthorization), httplib.Gnfd1Ecdsa+", Signature="))
		require.NoError(t, err)
		signer, _, err := hashlib.RecoverAddr(httplib.GetMsgToSignInGNFD1Auth(req), sig)
		require.NoError(t, err)
		assert.Equal(t, account.GetAddress(), signer)
	}
}
