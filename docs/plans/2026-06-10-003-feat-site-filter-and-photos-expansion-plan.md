# Plan: Site Filter And Photos Expansion

## Goal

Expand the CLI around two validated surfaces:

- richer house-search filters
- listing photo/gallery retrieval

## Search Direction

Prefer the observed website/georeach contract for:

- combined price ranges
- combined size ranges
- room bands
- amenities
- recency
- energy class
- sort

## Photos Direction

Add a shaped listing-photos flow over the existing detail/gallery endpoints.

## Boundary

Return metadata and image URLs first. Do not assume automatic asset download or mirroring.
