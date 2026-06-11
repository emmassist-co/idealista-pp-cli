# Idealista Website Capture Playbook

Use this when capturing new website behavior to extend the CLI safely from real browser traffic.

## Goal

Produce a clean HAR that shows:

- search/result URL shapes
- filter combinations
- listing detail requests
- gallery/photo requests

## Setup

1. Open `https://www.idealista.pt` in a normal, already-working browser session.
2. Open DevTools.
3. Go to `Network`.
4. Enable `Preserve log`.
5. Clear old requests.

## Capture Script

Start from a stable result page, for example:

`https://www.idealista.pt/comprar-casas/lisboa/arroios/`

Then apply filters in this order, waiting for each reload to finish:

1. `min price`
2. `max price`
3. `min + max price`
4. `min size`
5. `max size`
6. `min + max size`
7. bedroom bands such as `t1`, `t2`, `t3`, `t4+`
8. bathroom counts such as `1`, `2`, `3+`
9. one amenity at a time:
   - elevator
   - parking
   - terrace
   - balcony
   - garden
   - pool
10. condition/state filters
11. recency filters
12. sort options

After that:

1. Open one listing detail page.
2. Open the photo gallery.
3. Click through a few photos.
4. Open any floorplan if present.

## Export

Export:

- the HAR
- a cookie export if the session changed
- optional example listing URLs you care about

## Naming

Helpful file prefixes:

- `list-...`
- `photos-...`

## Why This Matters

This lets the CLI adopt only observed, validated behavior:

- canonical result URL grammar
- combined filter support
- listing/gallery/media contracts
- useful operator commands instead of guessed abstractions
