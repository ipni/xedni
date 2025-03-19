package xedni

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/mr-tron/base58"
	"github.com/multiformats/go-multihash"
)

type (
	SampleResponse struct {
		Samples []multihash.Multihash `json:"samples"`
	}
	Error struct {
		Error string `json:"error"`
	}
)

func (x *Xedni) ServeMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/ipni/v0/sample/{provider_id}/{context_id}", x.sampleHandler)
	return mux
}

func (x *Xedni) sampleHandler(w http.ResponseWriter, r *http.Request) {
	pidPath := r.PathValue("provider_id")
	crxidPath := r.PathValue("context_id")

	var pop Population
	var err error
	pop.ProviderID, err = peer.Decode(pidPath)
	if err != nil || pop.ProviderID.Validate() != nil {
		x.writeJson(w, http.StatusBadRequest, Error{
			Error: "invalid provider ID",
		})
		return
	}
	pop.ContextID, err = base58.Decode(crxidPath)
	if err != nil || len(pop.ContextID) == 0 {
		x.writeJson(w, http.StatusBadRequest, Error{
			Error: "invalid Context ID",
		})
		return
	}

	query := r.URL.Query()
	if query.Has("beacon") {
		pop.Beacon, err = hex.DecodeString(query.Get("beacon"))
		if err != nil || len(pop.Beacon) > 32 || len(pop.Beacon) < 1 {
			x.writeJson(w, http.StatusBadRequest, Error{
				Error: "invalid Beacon: must be at least 1 and ad most 32 hex encoded bytes",
			})
			return
		}
	} else {
		pop.Beacon = make([]byte, 0, 32)
		if _, err := rand.Read(pop.Beacon); err != nil {
			logger.Errorw("Failed to generate beacon", "error", err)
			x.writeJson(w, http.StatusInternalServerError, Error{
				Error: "failed to generate Beacon",
			})
			return
		}
	}

	if query.Has("max") {
		maxSamples, err := strconv.ParseInt(query.Get("max"), 10, 32)
		if err != nil || maxSamples > 10 || maxSamples < 1 {
			x.writeJson(w, http.StatusBadRequest, Error{
				Error: "invalid max sample count: must be at least 1 and no more than 10",
			})
			return
		}
		pop.MaxSamples = int(maxSamples)
	} else {
		pop.MaxSamples = 1
	}

	if query.Has("federation_epoch") {
		fedEpoch, err := strconv.ParseUint(query.Get("max"), 10, 64)
		if err != nil {
			x.writeJson(w, http.StatusBadRequest, Error{
				Error: "invalid federation epoch",
			})
			return
		}
		pop.FederationEpoch = fedEpoch
	}

	var response SampleResponse
	response.Samples, err = x.store.Sample(r.Context(), pop)
	if err != nil {
		logger.Errorw("Failed to sample store", "error", err)
		x.writeJson(w, http.StatusInternalServerError, Error{
			Error: "failed to sample store",
		})
		return
	}
	x.writeJson(w, http.StatusOK, response)
}

func (x *Xedni) writeJson(w http.ResponseWriter, statusCode int, v any) {
	w.WriteHeader(statusCode)
	h := w.Header()
	h.Set("Content-Type", "application/json; charset=utf-8")
	h.Set("X-Content-Type-Options", "nosniff")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		logger.Errorw("Failed to write JSON", "status", statusCode, "error", err)
	}
}
