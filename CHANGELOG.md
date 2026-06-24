# Changelog

All notable changes to **openbank-funds** are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/).

## [1.0.0] - 2026-06-24

Initial release of the Funds microservice (OBIE CBPII confirmation of funds).

### Added

- CBPII `/funds-confirmations`, validating consent and checking availability via the Accounts service.
- In-memory and Postgres repository adapters; SQL migrations applied on startup.
- OBIE-shaped HTTP API with FAPI interaction-id, structured logging and panic recovery.
- Unit/handler test suite plus Postgres integration tests (testcontainers).
- GitHub Actions CI and MIT license.
