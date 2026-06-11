# Plan: Site Cookie Lifecycle

## Goal

Add first-class operator commands for website session cookie handling.

## Planned Commands

- `cookie set`
- `cookie check`
- `cookie clear`
- `cookie source`

## Rules

- store cookie in existing config headers
- let `IDEALISTA_COOKIE` override config
- mask cookie values in output
- classify `403` as refresh-required

## Non-Goals

- automatic cookie acquisition
- automatic renewal
- anti-bot bypass logic
