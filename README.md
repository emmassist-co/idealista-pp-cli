# Idealista Site CLI

Read-only Go CLI for a curated set of Idealista.pt website workflows, grounded in authorized browser HAR captures and live verification.

This repo focuses on operator-friendly access to a narrow, validated subset of the public website contract:

- session and health checks
- location suggestions
- saved-search/session summaries
- canonical result URL construction for validated house filters
- listing photo and gallery metadata

## Install

### Build from source

```bash
go build -o bin/idealista-pp-cli ./cmd/idealista-pp-cli
go build -o bin/idealista-pp-mcp ./cmd/idealista-pp-mcp
```

### With Make

```bash
make build-all
```

## Quick Start

### 1. Verify Setup

```bash
idealista-pp-cli doctor
```

This checks your configuration.

### 2. Try Your First Command

```bash
idealista-pp-cli search locations lisboa
```

## Usage

Run `idealista-pp-cli --help` for the full command reference and flag list.

## Commands

### Search helpers

- **`idealista-pp-cli search local <query>`** - local SQLite FTS over synced data
- **`idealista-pp-cli search locations <query>`** - website location suggestions for queries like `lisboa` or `porto`
- **`idealista-pp-cli search saved`** - compact session-backed saved-search summary
- **`idealista-pp-cli search results-url ...`** - build validated website result URLs and georeach query state from the current supported filter subset
- **`idealista-pp-cli search results-live ...`** - fetch the canonical website results page and parse structured listing cards from the current HTML
- **`idealista-pp-cli search results-enriched ...`** - fetch result cards first, then enrich a bounded shortlist through listing detail endpoints
- **`idealista-pp-cli search totals-live ...`** - fetch the canonical website results page and extract the live result count
- **`idealista-pp-cli listing inspect <listing_id>`** - shaped listing summary across detail, configuration, gallery, and contact endpoints
- **`idealista-pp-cli listing photos <listing_id>`** - shaped listing photo and gallery summary for a listing ID

Important: `idealista-pp-cli search <query>` is local-search only. Live website search is exposed through the explicit path-shaped helpers above, not a free-text API.

### Low-level website endpoints

The repo also exposes the lower-level generated endpoint commands for listing detail, pager, configuration, and gallery access. Use `idealista-pp-cli --help` or `idealista-pp-cli which "<capability>"` to discover them.


## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
idealista-pp-cli search locations lisboa

# JSON for scripting and agents
idealista-pp-cli search locations lisboa --json

# Filter to specific fields
idealista-pp-cli listing photos 34998327 --json --select listing_id,primary_image_url,image_count

# Dry run — show the request without sending
idealista-pp-cli search results-url --location-path comprar-casas/lisboa/arroios --bedrooms t2,t3 --dry-run
idealista-pp-cli search results-live --location-path comprar-casas/lisboa/arroios --max-price 300000 --dry-run
idealista-pp-cli search results-enriched --location-path comprar-casas/lisboa/arroios --max-price 300000 --shortlist-limit 3 --dry-run
idealista-pp-cli search results-live --location-path comprar-casas/lisboa/arroios --max-price 300000 --exclude-tenanted --agent
idealista-pp-cli search results-enriched --location-path comprar-casas/lisboa/arroios --max-price 300000 --bedrooms t2,t3 --exclude-tenanted --shortlist-limit 3 --agent

# Agent mode — JSON + compact + no prompts in one flag
idealista-pp-cli search saved --agent
```

## Agent Usage

This CLI is designed for AI agent consumption:

- **Non-interactive** - never prompts, every input is a flag
- **Pipeable** - `--json` output to stdout, errors to stderr
- **Filterable** - `--select id,name` returns only fields you need
- **Previewable** - `--dry-run` shows the request without sending
- **Read-only by default** - this CLI does not create, update, delete, publish, send, or mutate remote resources
- **Offline-friendly** - sync/search commands can use the local SQLite store when available
- **Agent-safe by default** - no colors or formatting unless `--human-friendly` is set

Exit codes: `0` success, `2` usage error, `3` not found, `5` API error, `7` rate limited, `10` config error.

## Health Check

```bash
idealista-pp-cli doctor
```

Verifies configuration and connectivity to the API.

## Configuration

Config file: `~/.config/idealista-pp-cli/config.toml`

Static request headers can be configured under `headers`; per-command header overrides take precedence.

For a session-backed website workflow, the preferred operator flow is:

```bash
idealista-pp-cli cookie setup --launch
idealista-pp-cli cookie set 'datadome=...; other_cookie=...'
idealista-pp-cli cookie source
idealista-pp-cli cookie check
```

If you copied the raw request header from DevTools, stdin mode avoids shell quoting and strips a leading `Cookie: ` prefix automatically:

```bash
pbpaste | idealista-pp-cli cookie set --stdin
idealista-pp-cli cookie check
```

Or with an env-backed session:

```bash
export IDEALISTA_COOKIE='datadome=...; other_cookie=...'
idealista-pp-cli cookie source
idealista-pp-cli cookie check
```

Equivalent config file shape:

```toml
base_url = "https://www.idealista.pt"

[headers]
Cookie = "datadome=...; other_cookie=..."
```

Refresh model: this CLI does not mint or renew the website cookie. It checks freshness on demand and tells you when a replacement cookie is required.

## Validated Website Search Contract

The website-backed search support in this repo is intentionally narrow and evidence-based.

Supported now:

- `search locations <query>` for live location suggestions
- `search saved` for saved-search/session state
- `search results-url` for the validated filter subset:
  - `--location-path comprar-casas/...`
  - `--min-price`
  - `--max-price`
  - `--min-size`
  - `--max-size`
  - `--bedrooms t0,t1,t2,t3,t4-t5`
  - `--bathrooms 1,2,3`
  - `--amenities elevador,garagem,arrecadacao,arcondicionado,roupeiros-embutidos,vista-mar`
  - `--energy-class alta,media,baixa`
  - `--published-within 48h,week,month`
  - `--sort preco_medio-asc,precos-asc,atualizado-desc`
- `search results-live`, `search totals-live`, and `search results-enriched` for the canonical website results page HTML
  - `--exclude-tenanted` drops listings visibly tagged or described as rented / tenant-occupied
- `listing photos <listing_id>` for image URLs, primary photo, and gallery metadata

Current boundary:

- browse URLs are now canonicalized from the observed Arroios house-search grammar, including combined price and area ranges plus room-token and amenity combinations that were revalidated live
- `georeach` validation still covers the numeric and room-based subset the website exposes through that AJAX contract
- this round returns photo metadata and URLs, not local image downloads or mirroring

Example:

```bash
idealista-pp-cli search results-url \
  --location-path comprar-casas/lisboa/arroios \
  --min-price 220000 \
  --max-price 750000 \
  --min-size 60 \
  --max-size 120 \
  --bedrooms t2,t3,t4-t5 \
  --bathrooms 2,3 \
  --amenities elevador,garagem \
  --published-within month \
  --sort atualizado-desc

idealista-pp-cli search results-url \
  --location-path comprar-casas/lisboa/arroios \
  --bedrooms t1,t4-t5 \
  --amenities elevador,garagem \
  --sort atualizado-desc

idealista-pp-cli listing photos 34998327 --json --select listing_id,primary_image_url,image_count,images
idealista-pp-cli search results-live --location-path comprar-casas/lisboa/arroios --max-price 300000 --bedrooms t2,t3 --exclude-tenanted --agent
```

Live dogfood path:

```bash
IDEALISTA_COOKIE='datadome=...; other_cookie=...' \
go test ./internal/cli -run TestSiteDogfood
```

## Development

```bash
make test
make lint
```

See [GETTING_STARTED.md](GETTING_STARTED.md) for a tighter first-run path.
```

## Troubleshooting
**Not found errors (exit code 3)**
- Check the resource ID is correct
- Run the `list` command to see available items

## HTTP Transport

This CLI uses Chrome-compatible HTTP transport over HTTP/3 for browser-facing endpoints. It does not require a resident browser process for normal API calls.

---

Generated by [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press)
