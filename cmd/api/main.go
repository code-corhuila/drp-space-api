package main

import (
	"log"
	"net/http"
	"os"
	"time"

	httpadapter "github.com/code-corhuila/drp-space-api/internal/adapters/http"
	"github.com/code-corhuila/drp-space-api/internal/adapters/memory"
	"github.com/code-corhuila/drp-space-api/internal/adapters/security"
	"github.com/code-corhuila/drp-space-api/internal/app"
)

func main() {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8082"
	}
	service := os.Getenv("SERVICE_NAME")
	if service == "" {
		service = "space-service"
	}
	jwksURL := os.Getenv("IDENTITY_JWKS_URL")
	if jwksURL == "" {
		jwksURL = "http://identity-api:8081/api/v1/auth/jwks"
	}

	spaces, err := memory.Corte2()
	if err != nil {
		log.Fatal(err)
	}
	tokens := security.NewVerifier(security.HTTPJWKS{
		URL: jwksURL,
		Client: &http.Client{
			Timeout: 3 * time.Second,
		},
	})

	mux := httpadapter.NewMux(httpadapter.Deps{
		Service: service,
		Catalog: app.Catalog{Spaces: spaces},
		Tokens:  tokens,
	})

	log.Printf("drp-space-api listening on %s (memory catalog; JWKS %s; no DDL in this repo)", addr, jwksURL)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
