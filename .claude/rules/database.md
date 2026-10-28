---
paths:
  - "backend/migrations/**"
  - "backend/internal/*/infra/**"
  - "backend/internal/platform/database/**"
  - "backend/internal/platform/cache/**"
---

# Database rules

## Postgres and GORM

- Use GORM with the pgx driver (`gorm.io/driver/postgres`). Use no sqlc (see ADR 0001).
- GORM models are only in `infra/`. Each repository changes its models into domain types, and domain types into models. Domain types have no `gorm` tags.
- Give the request context to each query with `db.WithContext(ctx)`.
- Set `TranslateError: true` in the GORM config. In the repository, change `gorm.ErrRecordNotFound` and `gorm.ErrDuplicatedKey` into domain errors.
- When the domain enforces a rule that Postgres can also enforce, add the database constraint too. For example, add a unique index on the email address.
- IDs are UUIDv7 values that the domain makes. Times are `timestamptz` values in UTC.

## Migrations

- Change the schema only with goose SQL files in `backend/migrations/`. Do not use GORM `AutoMigrate`.
- Each migration has an `Up` part and a `Down` part.
- Do not change a migration after it is merged. Add a new migration.
- Run migrations with `make migrate-up` and `make migrate-down`.

## Redis cache

- The cache uses Redis DB 0.
- Cache keys have the form `<context>:<entity>:<id>`, for example `user:user:<id>`.
- Give each key a TTL. A cache of data that changes seldom can use 5 minutes.
- Read through the cache. Read the key first. If the key is not there, read Postgres and write the key.
- Postgres is the source of truth. If Redis fails, log the error and read from Postgres. Do not write to a cache that just failed to read.
- A call to Redis must fail fast: the cache client has timeouts of under 1 second and no retries. A cache value that the adapter cannot read, or that is for another ID, counts as a miss.
- When a change affects a cached entity, delete its key after the transaction commits.
