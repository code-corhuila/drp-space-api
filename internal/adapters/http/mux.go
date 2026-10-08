package httpadapter

import (
	"net/http"

	"github.com/code-corhuila/drp-space-api/internal/app"
)

type Deps struct {
	Service string
	Catalog app.Catalog
	Tokens  app.TokenVerifier
}

func NewMux(d Deps) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /health", HealthHandler(d.Service))
	if d.Tokens != nil {
		auth := RequireBearer(d.Tokens)
		mux.Handle("GET /api/v1/spaces", auth(ListSpaces(d.Catalog)))
		mux.Handle("GET /api/v1/spaces/{spaceId}", auth(GetSpace(d.Catalog)))
	}
	mux.Handle("/", NotFoundHandler())
	return WithCorrelation(stripSpoofedIdentity(mux))
}
