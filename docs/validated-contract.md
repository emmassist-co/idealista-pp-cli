# Validated Contract

This CLI intentionally exposes a narrow, read-only subset of the Idealista.pt website contract.

Validated today:

- location suggestions
- saved-search/session summary
- canonical result URL construction for observed house-filter shapes
- `georeach` validation for numeric and room filters
- internal listing results endpoint (`/ajax/listingcontroller/listingajax.ajax`) for the same observed filter subset
- internal listing totals endpoint (`/ajax/listingcontroller/totals/listingajax.ajax`) for the same observed filter subset
- listing photo and gallery metadata

Out of scope:

- cookie acquisition or challenge bypass
- write flows against the public website
- broad scraping or full public-site cloning beyond the HAR-validated endpoints above
