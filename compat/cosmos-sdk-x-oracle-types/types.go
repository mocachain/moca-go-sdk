package types

// ClaimSrcChain matches the legacy enum exposed by the removed oracle module.
// The SDK keeps these public types only to avoid a compile-time API break for
// callers that still reference GetInturnRelayer while the method returns an
// unsupported-feature error.
type ClaimSrcChain int32

const (
	CLAIM_SRC_CHAIN_UNSPECIFIED ClaimSrcChain = 0
	CLAIM_SRC_CHAIN_BSC         ClaimSrcChain = 1
	CLAIM_SRC_CHAIN_OP_BNB      ClaimSrcChain = 2
)

// RelayInterval holds the active window of the in-turn relayer.
type RelayInterval struct {
	Start uint64 `json:"start,omitempty"`
	End   uint64 `json:"end,omitempty"`
}

// QueryInturnRelayerRequest is the legacy request type kept for API
// compatibility only.
type QueryInturnRelayerRequest struct {
	ClaimSrcChain ClaimSrcChain `json:"claim_src_chain,omitempty"`
}

// QueryInturnRelayerResponse is the legacy response type kept for API
// compatibility only.
type QueryInturnRelayerResponse struct {
	BlsPubKey     string         `json:"bls_pub_key,omitempty"`
	RelayInterval *RelayInterval `json:"relay_interval,omitempty"`
}
