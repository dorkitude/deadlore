# Deadlore development guide

## Local build and test

- Build the root-level executable with `go build -o deadlore ./cmd/deadlore`.
- Run `go test ./...` and `go vet ./...` before committing.
- After every shipped CLI change, rebuild the root executable and reinstall the published Homebrew formula for this machine:
  ```bash
  go build -o deadlore ./cmd/deadlore
  brew tap dorkitude/tap https://github.com/dorkitude/homebrew-tap
  brew list --formula deadlore >/dev/null 2>&1 && brew reinstall deadlore || brew install dorkitude/tap/deadlore
  ```
  Leave the formula installed; this makes `deadlore` available on the user's `PATH`.
- Use a temporary cache for live smoke tests so development lookups do not affect the user cache:
  ```bash
  ./deadlore --cache-dir /tmp/deadlore-smoke Haze
  ./deadlore --cache-dir /tmp/deadlore-smoke Leech
  ./deadlore --cache-dir /tmp/deadlore-smoke hero list
  ./deadlore --cache-dir /tmp/deadlore-smoke item list
  ./deadlore --cache-dir /tmp/deadlore-smoke ability "Sleep Dagger"
  ./deadlore --cache-dir /tmp/deadlore-smoke --json ability list
  ```

The root `deadlore` binary is ignored by Git. `--json` is a public machine-facing interface; keep it structured and free of terminal formatting.

## Wiki access

Deadlore is intentionally a low-volume client of `deadlock.wiki`:

- Fetch canonical article pages only; do not use the wiki API, REST endpoints, or bulk-crawl the site.
- Preserve provenance in every human-facing lookup: canonical URL, revision, wiki last-modified text, and local fetch time.
- `ability list` derives ability names from canonical hero pages and relies on the local cache after its first lookup.
- Wiki text is CC BY-NC-SA 4.0. Do not bundle wiki text or media without confirming the licensing implications.

## Releasing a new version

1. Build/test, commit, and push the source change.
2. Create and push an annotated `vX.Y.Z` tag.
3. Build Windows archives and publish a GitHub release with `gh`:
   ```bash
   release_dir=$(mktemp -d /tmp/deadlore-vX.Y.Z.XXXXXX)
   mkdir -p "$release_dir/windows-amd64" "$release_dir/windows-arm64"
   GOOS=windows GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o "$release_dir/windows-amd64/deadlore.exe" ./cmd/deadlore
   GOOS=windows GOARCH=arm64 go build -trimpath -ldflags='-s -w' -o "$release_dir/windows-arm64/deadlore.exe" ./cmd/deadlore
   (cd "$release_dir/windows-amd64" && zip -q "$release_dir/deadlore_X.Y.Z_windows_amd64.zip" deadlore.exe)
   (cd "$release_dir/windows-arm64" && zip -q "$release_dir/deadlore_X.Y.Z_windows_arm64.zip" deadlore.exe)
   gh release create vX.Y.Z "$release_dir"/*.zip --repo dorkitude/deadlore --title "Deadlore vX.Y.Z" --generate-notes
   ```
4. Update `/Users/dorkitude/a/dev/homebrew-tap/Formula/deadlore.rb` with the new source URL and its SHA-256:
   ```bash
   curl --fail --silent --show-error --location "https://github.com/dorkitude/deadlore/archive/refs/tags/vX.Y.Z.tar.gz" | shasum -a 256
   ```
   Commit/push the tap, then test the actual user flow: `brew tap dorkitude/tap https://github.com/dorkitude/homebrew-tap`, `brew install dorkitude/tap/deadlore`, `brew test deadlore`, `brew uninstall deadlore`, `brew untap dorkitude/tap`.
5. Update `/Users/dorkitude/a/dev/scoop-bucket/deadlore.json` with the new version, asset URLs, and SHA-256 hashes. Commit/push it, validate JSON with `jq empty deadlore.json`, and download each release asset to verify its hash.
6. `scripts/install.ps1` reads the Scoop manifest at install time, so it needs no version bump unless the manifest format or installation behavior changes.
