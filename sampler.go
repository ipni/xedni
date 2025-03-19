package xedni

import (
	"context"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multihash"
)

type Population struct {
	ProviderID      peer.ID
	ContextID       []byte
	Beacon          []byte
	MaxSamples      int
	FederationEpoch uint64
}

type Sampler interface {
	Sample(context.Context, Population) ([]multihash.Multihash, error)
}
