# Deadlore development guide

## Local build and test

- Build the root-level executable with `go build -o deadlore ./cmd/deadlore`.
- Run `go test ./...` and `go vet ./...` before committing.
- After every shipped CLI change, rebuild the root executable and reinstall the published Homebrew cask for this machine:
  ```bash
  go build -o deadlore ./cmd/deadlore
  brew tap dorkitude/tap https://github.com/dorkitude/homebrew-tap
  brew list --cask deadlore >/dev/null 2>&1 && brew reinstall --cask deadlore || brew install dorkitude/tap/deadlore
  ```
  Leave the cask installed; this makes `deadlore` available on the user's `PATH`. (If an old source-built formula is still installed, `brew uninstall deadlore` first.)
The root `deadlore` binary is ignored by Git. `--json` is a public machine-facing interface; keep it structured and free of terminal formatting. Human output may use ANSI color in interactive terminals, but must remain readable without it: honor `NO_COLOR`, `TERM=dumb`, `--no-color`, redirected output, and `--json`.

When changing parsed `wiki.Page` output, bump `wiki.CurrentCacheVersion` so older cached pages cannot hide newly supported fields from users.

## Wiki access

Deadlore is intentionally a low-volume client of `deadlock.wiki`:

- Fetch canonical article pages only; do not use the wiki API, REST endpoints, or bulk-crawl the site.
- Preserve provenance in every human-facing lookup: canonical URL, revision, wiki last-modified text, and local fetch time.
- `ability list` derives ability names from canonical hero pages and relies on the local cache after its first lookup.
- Wiki text is CC BY-NC-SA 4.0. Do not bundle wiki text or media without confirming the licensing implications.

## Releasing a new version

A tag push runs `.github/workflows/release.yml`: vet and tests, then GoReleaser (`.goreleaser.yaml`), which publishes the GitHub release with macOS (arm64, amd64, and a universal `darwin_all`), Linux, and Windows archives plus `checksums.txt`. The version comes from the tag: GoReleaser sets `-X github.com/dorkitude/deadlore/internal/cli.version=X.Y.Z`, and `go install ...@vX.Y.Z` builds read it from the module info. There is no version constant to bump.

1. Build/test, commit, and push the source change. Wait for CI to pass.
2. Tag and push:
   ```bash
   git tag -a vX.Y.Z -m "vX.Y.Z: short summary"
   git push origin vX.Y.Z
   ```
3. Watch it: `gh run watch -R dorkitude/deadlore`.
4. Publish Homebrew and Scoop: `scripts/publish-packages.sh vX.Y.Z` (`DRY_RUN=1` to preview). It writes the cask `Casks/deadlore.rb` in `dorkitude/homebrew-tap` (removing the old `Formula/deadlore.rb`) and `deadlore.json` in `dorkitude/scoop-bucket`. **Don't skip this**: the tag push updates only GitHub, and `brew`/`scoop` users stay on the previous version otherwise.
5. Verify: `brew update && brew upgrade --cask deadlore && deadlore --version` (or `brew install dorkitude/tap/deadlore`), and on Windows `scoop update deadlore`.
6. `scripts/install.ps1` reads the Scoop manifest at install time, so it needs no version bump unless the manifest format or installation behavior changes.

If the workflow fails after GoReleaser has published, fix the problem, delete the release and tag (`gh release delete vX.Y.Z --cleanup-tag -R dorkitude/deadlore`), and tag again. Don't reuse a version once a package channel has published it; bump instead.

macOS signing and notarization are wired but off: `.goreleaser.yaml` enables them only when `MACOS_SIGN_P12` is set. To turn them on, add the `MACOS_SIGN_P12`, `MACOS_SIGN_PASSWORD`, `MACOS_NOTARY_KEY`, `MACOS_NOTARY_KEY_ID`, and `MACOS_NOTARY_ISSUER_ID` secrets (base64 `.p12` and `.p8` contents) to `dorkitude/deadlore`.
