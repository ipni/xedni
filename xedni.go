package xedni

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/ipfs/go-log/v2"
)

var logger = log.Logger("xedni")

type Xedni struct {
	*options
	server http.Server
	store  *Store
}

func New(o ...Option) (*Xedni, error) {
	var x Xedni
	var err error
	if x.options, err = newOptions(o...); err != nil {
		return nil, err
	}
	x.store, err = NewStore(x.options.storePath, x.delegateIndexer)
	if err != nil {
		return nil, fmt.Errorf("creating store: %s", err)
	}
	x.server = http.Server{
		Addr:    x.options.httpServerListenAddr,
		Handler: x.ServeMux(),
	}
	return &x, nil
}

func (x *Xedni) Start(context.Context) error {
	listen, err := net.Listen("tcp", x.httpServerListenAddr)
	if err != nil {
		return err
	}
	go func() {
		if err := x.server.Serve(listen); !errors.Is(err, http.ErrServerClosed) {
			logger.Errorw("Sever stopped unexpectedly.", "err", err)
		} else {
			logger.Info("Sever stopped.")
		}
	}()
	return nil
}

func (x *Xedni) Shutdown(ctx context.Context) (_err error) {
	defer func() {
		err := x.store.Close()
		if _err == nil {
			_err = err
		}
	}()
	return x.server.Shutdown(ctx)
}
