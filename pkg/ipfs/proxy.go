package ipfs

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"gitlab.com/distributed_lab/ape"
	"gitlab.com/distributed_lab/ape/problems"
	"gitlab.com/distributed_lab/kit/comfig"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/distributed_lab/logan/v3/errors"
)

// ProxyConfig - config for proxy server
type ProxyConfig interface {
	comfig.Logger
	comfig.Listenerer
	IPFSer
}

func RunProxy(ctx context.Context, cfg ProxyConfig) {
	log := cfg.Log().WithField("runner", "ipfs_proxy")

	r := chi.NewRouter()

	const slowRequestDurationThreshold = 3 * time.Second
	ape.DefaultMiddlewares(r, cfg.Log(), slowRequestDurationThreshold)
	handler := proxyHandler{
		ipfs: cfg.IPFS(),
	}

	r.Get("/readiness", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r.Get("/liveness", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r.Get("/ipfs/*", handler.Handle)

	log.WithFields(logan.F{
		"service": "api",
		"addr":    cfg.Listener().Addr(),
	}).Info("listening for http requests")
	ape.Serve(ctx, r, cfg, ape.ServeOpts{})
}

type proxyHandler struct {
	ipfs Gateway
}

func (p *proxyHandler) Handle(w http.ResponseWriter, r *http.Request) {
	resp, err := p.ipfs.GetReader(r.Context(), r.URL.Path)
	if err != nil {
		if errors.Cause(err) == ErrNotFound {
			ape.RenderErr(w, problems.NotFound())
			return
		}

		panic(errors.Wrap(err, "failed to get resource from IPFS"))
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	defer resp.Close()
	_, err = io.Copy(w, resp)
	if err != nil {
		panic(errors.Wrap(err, "failed to copy result"))
	}

}
