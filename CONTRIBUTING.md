# Contributing to GoScouter

Thanks for taking the time to contribute. Bug reports, documentation fixes, new
modules, and core changes are all welcome.

> ⚠️ GoScouter is intended for authorized security testing, research, and
> network administration. Contributions that exist only to make unauthorized
> scanning easier — evasion, mass targeting, or denial of service — will be
> declined.

## Ways to contribute

- **Report a bug** — [open an issue](https://github.com/GoScouter/goscouter/issues/new/choose)
  with the bug report template. Include the `gs` version, your OS, and the
  steps that reproduce it.
- **Request a feature** — open an issue with the feature request template and
  describe the problem before the solution.
- **Write a module** — modules are standalone binaries, not patches to this
  repository. See [Writing a module](#writing-a-module).
- **Improve the docs** — the README, this guide, and doc comments all count.

## Before you start

For anything larger than a bug fix, open an issue first and describe what you
have in mind. It's cheaper to agree on an approach in an issue than to redo a
finished branch.

## Development setup

You need the [Go toolchain](https://go.dev/dl/) (**1.26.4** or newer — the
version CI builds with), `make`, and `git`.

```bash
git clone https://github.com/GoScouter/goscouter.git
cd goscouter
make build      # produces ./bin/gs
```

## Making a change

1. **Fork** the repository and create a branch off `main`.

   Branch names follow `<type>/<short-description>`, matching the change types
   below:

   ```
   feat/module-timeout-flag
   fix/installer-stdin-prompt
   docs/contributing-guide
   refactor/standard-go-project-layout
   ```

2. **Write the change.** Match the surrounding code — its naming, its comment
   density, its idiom. The
   [`.editorconfig`](.editorconfig) covers whitespace; your editor should pick
   it up automatically.

3. **Add tests.** Packages here are tested alongside the code
   (`foo.go` / `foo_test.go`), using the standard library's `testing` package
   with table-driven cases. New behavior needs a test; a bug fix needs a test
   that fails without it.

4. **Commit** using [Conventional Commits](https://www.conventionalcommits.org/):

   ```
   feat: add --timeout flag to module runner
   fix: handle stdin prompt in interactive vs piped installer
   docs: document the module protocol handshake
   refactor: adopt standard Go project layout
   chore: bump sdk to v1.0.9
   test: cover dns record parsing edge cases
   ```

   Keep the subject in the imperative mood and under ~72 characters. Explain
   *why* in the body when the reason isn't obvious from the diff.

5. **Open a pull request** against `main` and fill in the template. Link the
   issue it closes (`Closes #123`), and apply the labels that fit — `type/*`
   for what kind of change it is, `area/*` for what it touches (see
   [`.github/labels.json`](.github/labels.json)).

## Review

Every pull request is reviewed by [@IdanKoblik](https://github.com/IdanKoblik),
who is requested automatically when you open it. Expect questions — they're
about the code, not about you. Push follow-up commits to the same branch rather
than force-pushing over the review, so the diff stays readable; squashing
happens at merge time.

## Writing a module

Modules are **standalone executables**, not code in this repository. A module
implements the [`sdk.Module`](https://pkg.go.dev/github.com/GoScouter/sdk)
interface and hands itself to `sdk.Serve`:

```go
func main() {
	if err := sdk.Serve(whois{}); err != nil {
		log.Fatal(err)
	}
}
```

The host and module speak JSON over the module's stdin and stdout, which means
a few rules are load-bearing:

- **Never print to stdout.** The protocol owns it. Use stderr for logging — it
  is passed through to the host.
- **`Scout` must be safe for concurrent use.** The host multiplexes concurrent
  calls over one subprocess and handles each request in its own goroutine.
- **`Scout` must never end the process.** Report failure by returning an error.
  A `log.Fatal` on one bad target takes down every call in flight and every
  call after it.

[`GoScouter/nginx`](https://github.com/GoScouter/nginx) is a short, complete
module to read before writing your own. Publish yours in its own repository —
users install it with `install <repo-url>` from the `gs` shell, and nothing
needs to be merged here.

## Reporting a security issue

Please don't open a public issue for a vulnerability in GoScouter itself.
Report it privately through
[GitHub Security Advisories](https://github.com/GoScouter/goscouter/security/advisories/new).

## License

By contributing, you agree that your contributions are licensed under the same
terms as this project — see [LICENSE.md](LICENSE.md).
