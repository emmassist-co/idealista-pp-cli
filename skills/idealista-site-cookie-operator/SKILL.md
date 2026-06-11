---
name: idealista-site-cookie-operator
description: Extract a valid Idealista website session cookie from an authorized browser session for use with the local CLI.
allowed-tools: Read Bash
---

# Idealista Site Cookie Operator

Use this when the CLI needs a valid website session cookie.

## Goal

Produce a cookie string in this shape:

```text
datadome=...; cookie2=...; cookie3=...
```

## Browser Method

1. Open `https://www.idealista.pt` in a normal browser session.
2. Open DevTools.
3. Go to `Network`.
4. Refresh the page.
5. Click a first-party `www.idealista.pt` request.
6. Copy the full `Cookie` request header value.
7. Remove the `Cookie: ` prefix if present.

## JSON Export Method

If you exported browser cookies as JSON:

1. Find the `datadome` cookie for `www.idealista.pt`.
2. Reconstruct the cookie header string from the exported cookies.
3. Use the final semicolon-separated string with the CLI.

## CLI Use

Either:

```bash
idealista-pp-cli cookie set 'datadome=...; other_cookie=...'
```

Or:

```bash
export IDEALISTA_COOKIE='datadome=...; other_cookie=...'
```

Then verify:

```bash
idealista-pp-cli cookie source
idealista-pp-cli cookie check
```
