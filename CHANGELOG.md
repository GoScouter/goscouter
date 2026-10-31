# Changelog

All notable changes to GoScouter are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- A module manifest can list several binaries per platform, so one repository
  can ship several modules (e.g. `recon:ports`, `recon:vhosts` and
  `recon:dirs`) and one `gs --install` installs them all. Every binary is
  downloaded and verified before any is installed, and if one fails, nothing
  is. Binaries that are already installed are skipped. The single-binary
  format keeps working unchanged.

## [2.1.0] - 2026-10-31

### Added

- Run system shell commands from the prompt by prefixing them with `!`, e.g.
  `!ls -la`. Commands run through `sh -c` (`cmd /C` on Windows), their output
  goes into the shell's scrollback, Ctrl-C stops the command rather than the
  shell, and a failing command is reported in gray as `process failed`.
  Interactive programs such as `vim` are not supported.
- The shell takes over the terminal like vim does: it opens on a clean screen
  and gives back what the terminal showed before when it exits.
- Scrollback: everything printed is kept (up to 5,000 lines) and scrolls with
  the mouse wheel. While scrolled up, new output doesn't move the view, and
  typing jumps back to the bottom.
- Drag with the mouse to select text; releasing copies it to the clipboard
  (via OSC 52) and shows a "Copied" toast. Pastes show a "Pasted" toast, and
  newlines in a paste become spaces so a paste never runs a command.
- Status bar showing the target, the running command with a spinner and
  elapsed time, the usage of the command being typed, and shortcut tips. Long
  targets are shortened in the middle, keeping the domain ending (`.co.il`).
- Ctrl-C cancels the running command. `scan` and `subdomains` stop mid-run,
  external modules are killed, and other modules are abandoned.
- Command history is saved between sessions (the last 1,000 commands, in the
  user config directory under `gs/history`). Ctrl-R searches it.
- Ctrl-F searches all output, highlighting matches; Enter/↑ and ↓ step
  through them and Esc closes the search.
- Clicking a command's prompt line folds its output away, and clicking again
  brings it back.
- `save <file>` writes the last module result as JSON (`.json`), a Markdown
  report (`.md`), or plain text without colors (any other extension).
- `help <command>` shows a command's usage, short name and flags, and `help`
  now lists the keyboard and mouse shortcuts.
- URLs in output are links: terminals that support OSC 8 make them clickable,
  and Ctrl+click opens them in the browser.
- Line editing: ←/→, Home/End, Delete, Ctrl-A/E (start/end), Ctrl-U/K (delete
  to start/end), Ctrl-W (delete word) and Ctrl-L (clear screen).
- Completion while typing: the rest of the best match shows in gray, and Tab
  or → accepts it. It completes command names, also from the part after the
  namespace (`subdo` → `subdomains`), and arguments: flags such as
  `http --ssl`, previous targets for `target`, command names for `help` and
  file names for `save`. Tab lists the options when several match.
- Wildcard DNS detection for subdomain enumeration: random names under the
  target are resolved, and if they all point to the same addresses, subdomains
  resolving to those addresses are filtered out of the results.

### Changed

- Every outbound HTTP request — module installs, the update check, site
  checks, and subdomain sources — now goes through a shared client that sends
  a `goscouter/<version>` User-Agent.
- Typing an unknown command now points to `help` instead of only reporting
  that the command does not exist.
- Subdomain enumeration timeout raised from 5 to 15 seconds, since crt.sh
  alone regularly takes 4-5 seconds to answer.
- Commands run by their full name (`internal:dns`) or by their short name
  (`dns`). A short name shared by two modules is rejected with the full names
  to choose from, instead of silently running one of them.
- Running the same command twice in a row adds it to the history once.
- `Command.Exec` takes a `context.Context`, and internal modules can implement
  `ScoutContext` to stop when it is cancelled.

### Fixed

- Subdomain enumeration no longer fails with `context deadline exceeded` on
  every source: wildcard detection now runs alongside the sources instead of
  using up their time budget first, and wildcard filtering gets its own.
- Subdomain enumeration no longer prints debug lines that broke the shell's
  layout.
- Subdomains whose DNS lookup fails are kept instead of being dropped by the
  wildcard filter.
- The `scan` command enumerates subdomains of the target's host rather than
  its full URL, and subdomain names given as URLs are normalized to hostnames.
- Typing part of a name, such as `s` or `p`, no longer runs whichever command
  happens to end with it (`internal:dns`, `help`).
- On Linux and macOS, output no longer staircases across the screen: raw mode
  now applies to input only, so a plain `\n` starts a new line again.
- ↑ now starts from the last command instead of the second-oldest, and ↓
  returns to what was being typed.
- Delete, Home, End and ←/→ no longer leave stray characters such as `~` in
  the line.
- SIGTERM (e.g. `kill <pid>`) now closes the shell and restores the terminal;
  before, it only took effect after the next command was entered.

## [2.0.0] - 2026-08-02

### Added

- Module management from the CLI: `gs --install <module-url@version>` and
  `gs --uninstall <author:name>`. When the version is omitted, the `latest`
  version declared in the module's `manifest.json` is used.
- Module dependency graph resolution, so modules can depend on other modules
  and are loaded in the right order.
- PDF summary output for network scans.
- `scan` command: enumerates the target's subdomains and runs every installed
  module against each one, returning the whole run as JSON. The shell prints
  it as an indented tree instead of raw JSON.
- Live scan progress with the usual `[+]`, `[-]`, and `[!]` markers, so a run
  reports as it goes rather than only at the end. Added `internal/logging` for
  the concurrency-safe output behind it, and wired it into the DNS and HTTP
  modules.
- Scans skip subdomains that no longer resolve, so dead names left behind in
  certificate transparency logs no longer cost a timeout per module.
- Community and contribution scaffolding: `CONTRIBUTING.md`, issue templates,
  and a PR review request workflow.

### Changed

- Rewrote the module subsystem around a central manager and runner, replacing
  the previous per-command install/uninstall/scan implementations.
- Replaced the standalone `internal/logger` package with output helpers in
  `../sdk/style`, unifying how commands and modules print to the terminal.
- Reworked DNS, HTTP, and subdomain record handling in `pkg/records` and
  `pkg/subdomains`.
- The HTTP module fetches its target once instead of twice: it checked the
  site status and then refetched the same URL for the records.
- Subdomain enumeration during a scan is bounded by a timeout, so an
  unresponsive certificate transparency source can no longer hang the run.
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
