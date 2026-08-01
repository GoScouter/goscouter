# Changelog

All notable changes to GoScouter are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Module management from the CLI: `gs --install <module-url@version>` and
  `gs --uninstall <author:name>`. When the version is omitted, the `latest`
  version declared in the module's `manifest.json` is used.
- Module dependency graph resolution, so modules can depend on other modules
  and are loaded in the right order.
- PDF summary output for network scans.
- Community and contribution scaffolding: `CONTRIBUTING.md`, issue templates,
  and a PR review request workflow.

### Changed

- Rewrote the module subsystem around a central manager and runner, replacing
  the previous per-command install/uninstall/scan implementations.
- Replaced the standalone `internal/logger` package with output helpers in
  `internal/style`, unifying how commands and modules print to the terminal.
- Reworked DNS, HTTP, and subdomain record handling in `pkg/records` and
  `pkg/subdomains`.
- Documentation updates to `README.md` covering installation, module
  management, and usage.

### Removed

- `internal/logger` — superseded by the styled output helpers.

## [1.0.0] - 2026-07-18

First stable release.

### Added

- Interactive shell interface: run `gs --target <url>` to drop into the `gs>`
  prompt, with built-in `help`, `info`, `install`, and `exit` commands.
- Modular architecture — DNS, HTTP, and subdomain probing each live in their
  own module, loaded on demand.
- Cross-platform binaries for Linux and macOS.
- Install script (`scripts/install.sh`) that selects the right prebuilt binary
  for the platform, verifies its SHA-256 checksum, and installs it to
  `/usr/local/bin` or `~/.local/bin`.
- `make build` and `make release-build` targets for building from source and
  cross-compiling release artifacts.

### Changed

- Complete rewrite of the project on top of the standard Go project layout
  (`cmd/`, `internal/`, `pkg/`).

[Unreleased]: https://github.com/GoScouter/GoScouter/compare/1.0.0...HEAD
[1.0.0]: https://github.com/GoScouter/GoScouter/compare/2026-b1-rc1...1.0.0
