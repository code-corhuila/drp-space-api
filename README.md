# drp-space-api

Space catalog API (Go). **Hexagonal:** `internal/domain` has no web/DB imports. DDL in [`drp-space-db`](https://github.com/code-corhuila/drp-space-db). Engine: [`drp-infra-postgres`](https://github.com/code-corhuila/drp-infra-postgres). This repo must not define a database container.

Contract: `drp-docs` `07-api/contracts/openapi/space-service.yaml` and `07-api/api-contract.md` (E-05, E-07). RS256 is validated **here** against identity JWKS. The gateway only checks credential presence.

## This increment

`GET /health`, `GET /api/v1/spaces` (E-05), `GET /api/v1/spaces/{spaceId}` (E-07). Catalog is an **in-memory seed** (like identity Corte 2). Flyway/`space_app` persistence is a later `feat/`. No SQL in this repo.

Lists return `{data, meta}`. Path `{spaceId}` that is not a canonical UUID is **400** (D-C12), missing is **404**. `available` is the catalog flag, not period occupancy (that is `drp-availability-api`).

```bash
go test ./...
go run ./cmd/api
curl http://localhost:8082/health
```

Compose (infra network must exist; identity JWKS must be reachable to authenticate):

```bash
docker compose --env-file .env.example -f deploy/compose.yml up --build
```

Child of `develop` named `feat/…`. Promote with `cherry-pick -x`.
