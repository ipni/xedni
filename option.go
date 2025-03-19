package xedni

import (
	"errors"

	"github.com/ipni/go-indexer-core"
)

type (
	options struct {
		httpServerListenAddr string
		storePath            string
		delegateIndexer      indexer.Interface
	}
	Option func(*options) error
)

func newOptions(option ...Option) (*options, error) {
	opts := &options{
		httpServerListenAddr: "0.0.0.0:40080",
		storePath:            ".",
	}
	for _, configure := range option {
		if err := configure(opts); err != nil {
			return nil, err
		}
	}
	if opts.delegateIndexer == nil {
		return nil, errors.New("delegate indexer must be set")
	}
	return opts, nil
}

func WithHTTPServerListenAddr(addr string) Option {
	return func(o *options) error {
		o.httpServerListenAddr = addr
		return nil
	}
}

func WithStorePath(path string) Option {
	return func(o *options) error {
		o.storePath = path
		return nil
	}
}

func WithDelegateIndexer(indexer indexer.Interface) Option {
	return func(o *options) error {
		if indexer == nil {
			return errors.New("delegate cannot be nil")
		}
		o.delegateIndexer = indexer
		return nil
	}
}
