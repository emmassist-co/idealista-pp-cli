# Validated Contract

This CLI intentionally exposes a narrow, read-only subset of the Idealista.pt website contract.

Validated today:

- location suggestions
- saved-search/session summary
- canonical result URL construction for observed house-filter shapes
- `georeach` validation for numeric and room filters
- canonical website results page HTML for live listing cards and result counts
- parser-side exclusion of listings tagged or described as rented / tenant-occupied
- listing detail aggregation across datalayer, configuration, gallery, and contact endpoints
- listing photo and gallery metadata

Out of scope:

- cookie acquisition or challenge bypass
- guaranteed request-side occupancy filtering on the website search form
- write flows against the public website
- broad scraping or full public-site cloning beyond the validated page/detail flows above
