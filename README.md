# drp-space-api

Space catalog API (Go). **Hexagonal:** `internal/domain` has no web/DB imports. DDL in [`drp-space-db`](https://github.com/code-corhuila/drp-space-db).

This increment is the **Space aggregate + BlockedPeriod** (kinds, capacity > 0, `endAt > startAt`). HTTP adapters come later. CONFIRMED overlap is not modeled here.

```bash
go test ./...
```

Child of `develop` named `feat/…`. Promote with `cherry-pick -x`.
