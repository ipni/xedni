# `xedni`

> IPNI Reverse Index

[![Go](https://github.com/ipni/xedni/actions/workflows/build.yaml/badge.svg)](https://github.com/ipni/xedni/actions/workflows/build.yaml)

`xedni` is an IPNI index store wrapper designed to deterministically sample multihashes advertised by a provider using
context ID.

## Install

```shell
go get github.com/ipni/xedni@latest
```

## Usage

Xedni wraps with any existing implementation of IPNI index backing store, [
`Indexer.Interface`](https://github.com/ipni/go-indexer-core/tree/main/store), and exposes one
additional [interface](sampler.go) that can sample multihashes from a provider using a context ID.

A random beacon may be optionally specified to deterministically sample the multihashes from the provider along with a
maximum sample size.

In addition to the core sampling logic implementation, the package also provides a simple HTTP API to expose the
sampling
capability documented [here](openapi.yaml).

The example below illustrates how Xedni can be used to wrap an existing indexer store implementation and expose the
sampling API:

```go
package main

import (
	"github.com/ipni/go-indexer-core"
	"github.com/ipni/go-indexer-core/engine"
	"github.com/ipni/xedni"
)

func main() {
	// The delegate indexer store backend may be any one of the stores currently supported by go-indexer-core.
	// See: https://github.com/ipni/go-indexer-core/tree/main/store
	var delegate indexer.Interface
	rx, err := xedni.New(
		xedni.WithStorePath(".store"), xedni.WithDelegateIndexer(delegate))
	if err != nil {
		//...
	}
	// Start the indexer engine as usual, _but_ with the warapped xedni store.
	eng := engine.New(rx.Store())
	// pass engine to storetheindex ingester or any other ingestion implementation...
	
	// Start Xedni to expose samling HTTP API.
	if err:= rx.Start(context.Background); err != nil {
	//...
	}
}
```

You can then use the HTTP API to sample multihashes from the provider using a context ID:

```shell
$ curl http://localhost:40080/ipni/v0/sample/<provider-id>/<context-id>?max=3&beacon=5865646e690a
```

Example response:

```json
{
    "samples": [
        "QmNrkWz2DFy2MKKpK84B4MJQhLM8GptkttQFonBXrHBr68",
        "QmQt3QgrXrnEsJDDFWfwoXgp1CYusTdACGCd7mxnU1H9Zc",
        "QmbuvjR4kZroSvADmEMh3tqdw88eovBeACZUBR4djbdX39"
    ]
}
```

See [options](option.go) for further configurable fields.

## License

[SPDX-License-Identifier: Apache-2.0 OR MIT](LICENSE.md)
