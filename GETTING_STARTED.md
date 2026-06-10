# Getting Started

## Build

```bash
make build
```

This produces `bin/idealista-pp-cli`.

## Test

```bash
make test
```

For the live website smoke checks, provide a valid browser session cookie:

```bash
export IDEALISTA_COOKIE='datadome=...; other_cookie=...'
go test ./internal/cli -run TestSiteDogfood -v
```

## First Commands

```bash
idealista-pp-cli doctor
idealista-pp-cli cookie check
idealista-pp-cli search locations lisboa
idealista-pp-cli search results-url --location-path comprar-casas/lisboa/arroios --bedrooms t2,t3
idealista-pp-cli listing photos 34998327
```
