---
name: pp-idealista
description: "Printing Press CLI for Idealista. Curated website-facing CLI spec derived from an authorized Idealista.pt HAR capture."
author: "Alexandre Santos"
license: "Apache-2.0"
argument-hint: "<command> [args] | install cli|mcp"
allowed-tools: "Read Bash"
metadata:
  openclaw:
    requires:
      bins:
        - idealista-pp-cli
---

# Idealista — Printing Press CLI

## Prerequisites: Install the CLI

This skill drives the `idealista-pp-cli` binary. **You must verify the CLI is installed before invoking any command from this skill.** If it is missing, install it first:

1. Install via the Printing Press installer:
   ```bash
   npx -y @mvanhorn/printing-press-library install idealista --cli-only
   ```
2. Verify: `idealista-pp-cli --version`
3. Ensure `$GOPATH/bin` (or `$HOME/go/bin`) is on `$PATH`.

If the `npx` install fails before this CLI has a public-library category, install Node or use the category-specific Go fallback after publish.

If `--version` reports "command not found" after install, the install step did not put the binary on `$PATH`. Do not proceed with skill commands until verification succeeds.

Curated website-facing CLI spec derived from an authorized Idealista.pt HAR capture.

## When Not to Use This CLI

Do not activate this CLI for requests that require creating, updating, deleting, publishing, commenting, upvoting, inviting, ordering, sending messages, booking, purchasing, or changing remote state. This printed CLI exposes read-only commands for inspection, export, sync, and analysis.

## HTTP Transport

This CLI uses Chrome-compatible HTTP transport over HTTP/3 for browser-facing endpoints. It does not require a resident browser process for normal API calls.

## Command Reference

**addetail-recommendation** — Detail recommendation modules

- `idealista-pp-cli addetail-recommendation <addetail_recommendation_id>` — GET addetail-recommendation {addetail_recommendation_id}

**detail** — Listing detail metadata

- `idealista-pp-cli detail <detail_id>` — GET detail {detail_id} datalayer

**home** — Home and search-box metadata

- `idealista-pp-cli home` — GET home searchbox operation-typology

**login-and-register-api** — Session and login-state helpers

- `idealista-pp-cli login-and-register-api` — GET login-and-register-api lara sl od

**pt** — Portuguese portal endpoints

- `idealista-pp-cli pt get-configuration` — GET pt detail {detail_id} configuration
- `idealista-pp-cli pt get-open-detail-gallery` — GET pt openDetailGallery {opendetailgallery_id}
- `idealista-pp-cli pt list-ad-contact-info-for-detail.ajax` — GET pt ajax listingController adContactInfoForDetail.ajax
- `idealista-pp-cli pt list-home` — GET pt locationsSuggest sale home
- `idealista-pp-cli pt list-pager` — GET pt detail pager
- `idealista-pp-cli pt list-user-searches` — GET pt home user-searches

**search** — Website-native search helpers plus local FTS

- `idealista-pp-cli search local <query>` — Local full-text search over synced data
- `idealista-pp-cli search locations <query>` — Website location suggestions
- `idealista-pp-cli search saved` — Compact saved-search/session summary
- `idealista-pp-cli search results-url ...` — Build validated result URLs and georeach query state from the supported website filter subset
- `idealista-pp-cli search results-live ...` — Call the internal listing results endpoint observed in the HAR with the supported filter subset
- `idealista-pp-cli search results-enriched ...` — Parse result cards first, then enrich a bounded shortlist through the listing detail endpoints
- `idealista-pp-cli search totals-live ...` — Call the internal listing totals endpoint observed in the HAR with the supported filter subset
- `idealista-pp-cli listing inspect <listing_id>` — Get a shaped listing summary across detail, configuration, gallery, and contact endpoints
- `idealista-pp-cli listing photos <listing_id>` — Get listing photos, primary image, and gallery metadata


### Finding the right command

When you know what you want to do but not which command does it, ask the CLI directly:

```bash
idealista-pp-cli which "<capability in your own words>"
```

`which` resolves a natural-language capability query to the best matching command from this CLI's curated feature index. Exit code `0` means at least one match; exit code `2` means no confident match — fall back to `--help` or use a narrower query.

For website search work, prefer these explicit flows over a generic `search <query>`:

- `search locations <query>` for place autocomplete
- `search saved` for current-session saved-search context
- `search results-url` for supported result-page filters
- `search results-live` for the internal results HTML endpoint
- `search results-enriched` for shortlist-first structured listing search
- `search totals-live` for the internal totals endpoint
- `listing inspect <listing_id>` for the aggregated listing detail workflow
- `listing photos <listing_id>` for listing image and gallery inspection

Validated `search results-url` filter subset:

- `--min-price` and `--max-price`
- `--min-size` and `--max-size`
- `--bedrooms t0,t1,t2,t3,t4-t5`
- `--bathrooms 1,2,3`
- `--amenities elevador,garagem,arrecadacao,arcondicionado,roupeiros-embutidos,vista-mar`
- `--energy-class alta,media,baixa`
- `--published-within 48h,week,month`
- `--sort preco_medio-asc,precos-asc,atualizado-desc`

Current boundary:

- result URLs and georeach validation are related but not identical: the browse URL now follows the observed Arroios token grammar, while georeach covers the numeric and room-based subset the website exposes through AJAX
- the listing results and totals endpoints are now exposed directly, but they still depend on a usable website cookie and may return DataDome challenge HTML when the session is blocked
- listing photos returns metadata and image URLs only; it does not download assets locally

## Auth Setup

No authentication required.

Run `idealista-pp-cli doctor` to verify setup.

## Agent Mode

Add `--agent` to any command. Expands to: `--json --compact --no-input --no-color --yes`.

- **Pipeable** — JSON on stdout, errors on stderr
- **Filterable** — `--select` keeps a subset of fields. Dotted paths descend into nested structures; arrays traverse element-wise. Critical for keeping context small on verbose APIs:

  ```bash
  idealista-pp-cli addetail-recommendation mock-value --agent --select id,name,status
  ```
- **Previewable** — `--dry-run` shows the request without sending
- **Offline-friendly** — sync/search commands can use the local SQLite store when available
- **Non-interactive** — never prompts, every input is a flag
- **Read-only** — do not use this CLI for create, update, delete, publish, comment, upvote, invite, order, send, or other mutating requests

### Response envelope

Commands that read from the local store or the API wrap output in a provenance envelope:

```json
{
  "meta": {"source": "live" | "local", "synced_at": "...", "reason": "..."},
  "results": <data>
}
```

Parse `.results` for data and `.meta.source` to know whether it's live or local. A human-readable `N results (live)` summary is printed to stderr only when stdout is a terminal AND no machine-format flag (`--json`, `--csv`, `--compact`, `--quiet`, `--plain`, `--select`) is set — piped/agent consumers and explicit-format runs get pure JSON on stdout.

## Agent Feedback

When you (or the agent) notice something off about this CLI, record it:

```
idealista-pp-cli feedback "the --since flag is inclusive but docs say exclusive"
idealista-pp-cli feedback --stdin < notes.txt
idealista-pp-cli feedback list --json --limit 10
```

Entries are stored locally at `~/.local/share/idealista-pp-cli/feedback.jsonl`. They are never POSTed unless `IDEALISTA_FEEDBACK_ENDPOINT` is set AND either `--send` is passed or `IDEALISTA_FEEDBACK_AUTO_SEND=true`. Default behavior is local-only.

Write what *surprised* you, not a bug report. Short, specific, one line: that is the part that compounds.

## Output Delivery

Every command accepts `--deliver <sink>`. The output goes to the named sink in addition to (or instead of) stdout, so agents can route command results without hand-piping. Three sinks are supported:

| Sink | Effect |
|------|--------|
| `stdout` | Default; write to stdout only |
| `file:<path>` | Atomically write output to `<path>` (tmp + rename) |
| `webhook:<url>` | POST the output body to the URL (`application/json` or `application/x-ndjson` when `--compact`) |

Unknown schemes are refused with a structured error naming the supported set. Webhook failures return non-zero and log the URL + HTTP status on stderr.

## Named Profiles

A profile is a saved set of flag values, reused across invocations. Use it when a scheduled agent calls the same command every run with the same configuration - HeyGen's "Beacon" pattern.

```
idealista-pp-cli profile save briefing --json
idealista-pp-cli --profile briefing addetail-recommendation mock-value
idealista-pp-cli profile list --json
idealista-pp-cli profile show briefing
idealista-pp-cli profile delete briefing --yes
```

Explicit flags always win over profile values; profile values win over defaults. `agent-context` lists all available profiles under `available_profiles` so introspecting agents discover them at runtime.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 2 | Usage error (wrong arguments) |
| 3 | Resource not found |
| 5 | API error (upstream issue) |
| 7 | Rate limited (wait and retry) |
| 10 | Config error |

## Argument Parsing

Parse `$ARGUMENTS`:

1. **Empty, `help`, or `--help`** → show `idealista-pp-cli --help` output
2. **Starts with `install`** → ends with `mcp` → MCP installation; otherwise → see Prerequisites above
3. **Anything else** → Direct Use (execute as CLI command with `--agent`)

## MCP Server Installation

Install the MCP binary from this CLI's published public-library entry or pre-built release, then register it:

```bash
claude mcp add idealista-pp-mcp -- idealista-pp-mcp
```

Verify: `claude mcp list`

## Direct Use

1. Check if installed: `which idealista-pp-cli`
   If not found, offer to install (see Prerequisites at the top of this skill).
2. Match the user query to the best command from the Unique Capabilities and Command Reference above.
3. Execute with the `--agent` flag:
   ```bash
   idealista-pp-cli <command> [subcommand] [args] --agent
   ```
4. If ambiguous, drill into subcommand help: `idealista-pp-cli <command> --help`.
