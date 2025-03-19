package xedni_test

import (
	"context"
	"encoding/hex"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ipni/go-indexer-core"
	"github.com/ipni/xedni"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multihash"
	"github.com/stretchr/testify/require"
)

var _ indexer.Interface = (*noopStore)(nil)

func TestT(t *testing.T) {
	out := t.TempDir()
	subject, err := xedni.NewStore(out, noopStore{})
	require.NoError(t, err)

	rng := rand.New(rand.NewSource(1413))
	var size int
	var mhs []multihash.Multihash

	generateBytes := func() []byte {
		bytes := make([]byte, 32)
		read, err := rng.Read(bytes[:])
		require.NoError(t, err)
		return bytes[:read]
	}

	for range 10_000 {
		bytes := generateBytes()
		mhs = append(mhs, bytes)
		size += len(bytes)
	}

	var value indexer.Value
	value.ContextID = generateBytes()
	value.MetadataBytes = generateBytes()
	value.ProviderID = peer.ID(generateBytes())

	now := time.Now()
	require.NoError(t, subject.Put(value, mhs...))
	t.Log("Put Took", time.Since(now))

	decodeString, err := hex.DecodeString("3439d92d58e47d342131d446a3abe264396dd264717897af30525c98408c834f")
	require.NoError(t, err)

	now = time.Now()
	samples, err := subject.Sample(context.Background(), xedni.Population{
		ProviderID:      value.ProviderID,
		ContextID:       value.ContextID,
		Beacon:          decodeString,
		MaxSamples:      5,
		FederationEpoch: 0,
	})
	require.NoError(t, err)
	t.Log("Sample Took", time.Since(now))
	t.Log("Size:", getDirectorySize(t, out))
	t.Log("Raw: ", size)

	require.Len(t, samples, 5)
	for i, sample := range samples {
		require.Contains(t, mhs, sample)
		t.Log(i, sample.String())
	}
}

func getDirectorySize(t *testing.T, dir string) int64 {
	var size int64
	err := filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		require.NoError(t, err)
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	require.NoError(t, err)
	return size
}

type noopStore struct{}

func (noopStore) Get(multihash.Multihash) ([]indexer.Value, bool, error) { return nil, false, nil }
func (noopStore) Put(indexer.Value, ...multihash.Multihash) error        { return nil }
func (noopStore) Remove(indexer.Value, ...multihash.Multihash) error     { return nil }
func (noopStore) RemoveProvider(context.Context, peer.ID) error          { return nil }
func (noopStore) RemoveProviderContext(peer.ID, []byte) error            { return nil }
func (noopStore) Size() (int64, error)                                   { return 0, nil }
func (noopStore) Flush() error                                           { return nil }
func (noopStore) Close() error                                           { return nil }
func (noopStore) Iter() (indexer.Iterator, error)                        { return nil, nil }
func (noopStore) Stats() (*indexer.Stats, error)                         { return nil, nil }
