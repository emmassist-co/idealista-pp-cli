# Idealista Site Cookie Management

## Scope

The CLI can use an already-valid website session cookie. It does not mint or refresh anti-bot cookies itself.

## Safe Model

Supported:

- operator supplies a valid cookie
- CLI stores it in config or reads it from `IDEALISTA_COOKIE`
- CLI checks whether it is still usable
- future secret-sync tooling can distribute the cookie

Not in scope:

- bypassing anti-bot protections
- automatic challenge solving
- unattended cookie harvesting or renewal

## Practical Workflow

1. Obtain a valid cookie from an authorized browser session.
2. Load it with:
   - `idealista-pp-cli cookie set 'datadome=...; ...'`
   - or `export IDEALISTA_COOKIE='datadome=...; ...'`
3. Validate it with:
   - `idealista-pp-cli cookie source`
   - `idealista-pp-cli cookie check`

## Refresh Model

Treat refresh as `refresh on failure`, not `refresh every N hours`.

The CLI should:

- detect stale cookies
- fail clearly on `403`/challenge responses
- ask for a replacement cookie

## Deferred Follow-Up

`agentcookie` can be useful later as a storage/sync layer, but not as a cookie minting mechanism.
