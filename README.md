# centropy-affilate

Backend for سنتروپی's customer management / loyalty-club system. It owns a
small database of its own (admin logins, a synced customer directory) and
reads the AlefGym production database read-only for everything about
purchase behavior — orders, course expiry — so it can classify every
customer into one of the six segments defined in `loyalty-club-roadmap.html`
without duplicating AlefGym's data model.

Architecture mirrors `findra/backend`: DDD-ish domain packages, a
generics-based in-process CQRS bus (`pkg/cqrs`), ent for this service's own
Postgres, chi for HTTP, JWT for admin auth.

## What's implemented (v1)

This is phases 0–1 of the roadmap, end to end:

- **Segmentation** (`internal/domain/segment`, `internal/infrastructure/alefgym`):
  the six-bucket rule (newcomer / cold / hero / at-risk / churned / one-time),
  computed live from AlefGym's `Users`, `Orders` and `Courses.ExpiredAt` — no
  new fields needed on the AlefGym side, matching the roadmap's own finding.
  Cached in Redis for 10 minutes so an admin dashboard doesn't hammer a
  production database it doesn't own.
- **Customer sync** (`internal/domain/customer`): a manually-triggered pull
  of the AlefGym user directory into a local table, so future loyalty
  features (points, referrals — not built yet) have a local row to attach
  state to.
- **Admin auth**: email/password + JWT, since this is an internal ops tool,
  not a customer-facing surface (that stays in AlefGym).
- **Non-purchasers & delayed-program complainers** (`internal/domain/complaint`):
  who registered and never bought, by month; and who explicitly complained
  about a late program (keyword search over their own chat/tickets) and
  bought nothing since — a heuristic signal, documented as such.
- **Overdue-without-repurchase + AI analysis** (`internal/domain/renewal`,
  `internal/domain/analysis`): customers whose last delivered program is
  >=50 days old with zero orders since (rule-based, free), each optionally
  enriched with a GapGPT-read verdict on *why* — one LLM call per customer,
  only when they have chat/ticket messages newer than their last analysis
  (a customer who's gone silent costs nothing to re-check). Runs on an
  in-process daily ticker (`ANALYSIS_RUN_INTERVAL`, default 24h) plus a
  manual, capped-at-20 trigger (`POST /admin/analysis/run`) for testing.
- **Complaint verification** (`internal/domain/complaint`'s `Verification`):
  the delayed-program keyword search catches false positives (word
  co-occurrence isn't meaning), so a second GapGPT pass reads each matched
  excerpt and judges whether it's a genuine delay complaint. Same
  cache-by-cursor idea as analysis (keyed to `ComplaintAt` this time), same
  daily ticker, same manual capped trigger (`POST
  /admin/complaints/delayed-program/verify`).

## What's explicitly NOT here yet

Points ledger, referral codes, promotion engine, and the customer-facing
loyalty UI are phases 2–4 of the roadmap — deliberately left out rather than
stubbed, so nothing in this repo is a half-finished shell.

## Running locally

```bash
cp .env.example .env
# fill in ALEFGYM_DATABASE_DSN with the real read-only credential

make up          # starts this service's own Postgres + Redis (docker)
make generate    # generates ent code from ent/schema (first run only)
make run
```

On first boot, if `ADMIN_SEED_EMAIL`/`ADMIN_SEED_PASSWORD` are set and no
admin exists yet, one is created so there's a way to get the first token.

```bash
curl -X POST localhost:8090/api/v1/auth/login \
  -H 'content-type: application/json' \
  -d '{"email":"admin@centropy.ir","password":"changeme123"}'

curl localhost:8090/api/v1/admin/segments \
  -H "authorization: Bearer <token>"
```

## API

| Method | Path                                    | Auth  |
|--------|------------------------------------------|-------|
| GET    | `/healthz`                                | none  |
| POST   | `/api/v1/auth/login`                      | none  |
| GET    | `/api/v1/admin/segments`                  | admin |
| GET    | `/api/v1/admin/segments/{segment}/customers` | admin |
| GET    | `/api/v1/admin/customers`                 | admin |
| POST   | `/api/v1/admin/customers/sync`            | admin |
| GET    | `/api/v1/admin/segments/non-purchasers`   | admin |
| GET    | `/api/v1/admin/segments/non-purchasers/monthly` | admin |
| GET    | `/api/v1/admin/complaints/delayed-program` | admin |
| GET    | `/api/v1/admin/complaints/delayed-program/verified` | admin |
| POST   | `/api/v1/admin/complaints/delayed-program/verify?limit=5` | admin |
| GET    | `/api/v1/admin/renewals/overdue?days=50`  | admin |
| GET    | `/api/v1/admin/analysis/overdue?days=50`  | admin |
| POST   | `/api/v1/admin/analysis/run?limit=5`      | admin |

`{segment}` is one of `NEWCOMER`, `COLD`, `HERO`, `AT_RISK`, `CHURNED`, `ONE_TIME`.
