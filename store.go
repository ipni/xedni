package xedni

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ipni/go-indexer-core"
	"github.com/libp2p/go-libp2p/core/peer"
	_ "github.com/marcboeker/go-duckdb"
	"github.com/mr-tron/base58"
	"github.com/multiformats/go-multihash"
	"github.com/parquet-go/parquet-go"
	"github.com/parquet-go/parquet-go/compress/zstd"
)

var (
	_ indexer.Interface = (*Store)(nil)
	_ Sampler           = (*Store)(nil)
)

type (
	Store struct {
		home     string
		delegate indexer.Interface
		db       *sql.DB
	}
	ParquetRecord struct {
		Multihash multihash.Multihash
	}
)

func NewStore(home string, delegate indexer.Interface) (*Store, error) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		return nil, err
	}

	return &Store{
		home:     filepath.Clean(home),
		delegate: delegate,
		db:       db,
	}, nil
}

func (s *Store) Sample(ctx context.Context, population Population) ([]multihash.Multihash, error) {
	seed, err := seedFromBeacon(population.Beacon)
	if err != nil {
		return nil, err
	}

	pid := population.ProviderID.String()
	ctxid := base58.Encode(population.ContextID)
	dataset := filepath.Join(s.home, pid, ctxid, "*.parquet")

	// Prepared statement for sample size and seed doesn't seem to work. Hence, the
	// manual query crafting
	query := fmt.Sprintf(
		`SELECT Multihash FROM read_parquet('%s') TABLESAMPLE reservoir(%d) REPEATABLE (%d);`,
		dataset, population.MaxSamples, seed)
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var samples []multihash.Multihash
	for rows.Next() {
		var sample multihash.Multihash
		if err := rows.Scan(&sample); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		samples = append(samples, sample)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate rows: %w", err)
	}
	return samples, nil
}

func seedFromBeacon(beacon []byte) (int32, error) {
	beaconLength := len(beacon)
	if beaconLength < 1 || beaconLength > 32 {
		return 0, fmt.Errorf("beacon must be at least 1 and at most 32 bytes, got length: %d", len(beacon))
	}

	split := beaconLength / 2
	highBytes := beacon[:split]
	lowBytes := beacon[split:]

	// Extract a positive in32 value from the beacon to use as seed for reservoir
	// sampling. DuckDB does not support int64 seeds by the looks of it.
	highPart := binary.LittleEndian.Uint16(highBytes[:min(2, split)])
	lowPart := binary.LittleEndian.Uint16(lowBytes[:min(2, split)])
	unsignedValue := (uint32(highPart) << 16) | uint32(lowPart)
	return int32(unsignedValue & 0x7FFFFFFF), nil
}

func (s *Store) Get(multihash multihash.Multihash) ([]indexer.Value, bool, error) {
	return s.delegate.Get(multihash)
}

func (s *Store) Put(value indexer.Value, mhs ...multihash.Multihash) error {
	if err := s.delegate.Put(value, mhs...); err != nil {
		return err
	}

	pid := value.ProviderID.String()
	ctxid := base58.Encode(value.ContextID)
	basename := fmt.Sprintf("%d.parquet", time.Now().UnixNano())
	out := filepath.Join(s.home, pid, ctxid, basename)
	if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", filepath.Dir(out), err)
	}

	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()

	records := make([]ParquetRecord, 0, len(mhs))
	for _, mh := range mhs {
		record := ParquetRecord{
			Multihash: mh,
		}
		records = append(records, record)
	}
	return parquet.WriteFile(out, records,
		parquet.Compression(&zstd.Codec{
			Level:       zstd.DefaultLevel,
			Concurrency: zstd.DefaultConcurrency,
		}),
		parquet.KeyValueMetadata("ProviderID", value.ProviderID.String()),
		parquet.KeyValueMetadata("ContextID", ctxid),
		parquet.KeyValueMetadata("Metadata", string(value.MetadataBytes)),
	)
}

func (s *Store) Remove(value indexer.Value, multihash ...multihash.Multihash) error {
	return s.delegate.Remove(value, multihash...)

	// Nothing else to do here. Individual multihash removal isn't supported.
}

func (s *Store) RemoveProvider(ctx context.Context, id peer.ID) error {
	if err := s.delegate.RemoveProvider(ctx, id); err != nil {
		return err
	}
	pid := id.String()
	dir := filepath.Join(s.home, pid)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("failed to remove xedni provider directory %s: %w", dir, err)
	}
	return nil
}

func (s *Store) RemoveProviderContext(id peer.ID, ctx []byte) error {
	if err := s.delegate.RemoveProviderContext(id, ctx); err != nil {
		return err
	}
	pid := id.String()
	ctxid := base64.StdEncoding.EncodeToString(ctx)
	dir := filepath.Join(s.home, pid, ctxid)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("failed to remove xedni contextID directory %s: %w", dir, err)
	}
	return nil
}

func (s *Store) Size() (int64, error) {
	return s.delegate.Size()
}

func (s *Store) Flush() error {
	return s.delegate.Flush()
}

func (s *Store) Close() error {
	defer func() {
		_ = s.db.Close()
	}()
	return s.delegate.Close()
}

func (s *Store) Iter() (indexer.Iterator, error) {
	return s.delegate.Iter()
}

func (s *Store) Stats() (*indexer.Stats, error) {
	return s.delegate.Stats()
}
