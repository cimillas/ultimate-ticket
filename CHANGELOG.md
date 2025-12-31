# Changelog

All notable changes to this project will be documented in this file.

This project follows Semantic Versioning (SemVer).

## [Unreleased]

## [0.2.0]
- Added admin endpoints for managing events/zones in local tooling.
- Added CORS allow-list support via `CORS_ORIGINS`.
- Added shared `.env.example`, `.env` auto-loading, and warnings for defaults.
- Added a minimal frontend and improved layout for clarity on desktop.
- Upgraded Vite to v7 and added `.nvmrc` for frontend tooling consistency.
- Standardized API errors as JSON with `error` and `code`, including not-found responses.
- Documented API error codes and domain concepts.
- Removed QA naming from docs and UI.

## [0.1.0]
- Initial project structure and API skeleton (holds, confirms, migrations, tests).
