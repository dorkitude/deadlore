# deadlore

`deadlore` is a source-aware CLI for the community-maintained [Deadlock Wiki](https://deadlock.wiki).
It performs one-page, canonical article lookups and keeps a small local cache; it does not bulk crawl the wiki or call its crawler-disallowed API endpoints.

## Install

### Homebrew (macOS and Linux)

```bash
brew tap dorkitude/tap
brew install deadlore
```

### Scoop (Windows)

```powershell
scoop bucket add dorkitude https://github.com/dorkitude/scoop-bucket
scoop install dorkitude/deadlore
```

### Direct Windows install

Run this from PowerShell or `cmd.exe`—no package manager required:

```powershell
powershell -ExecutionPolicy Bypass -Command "irm https://raw.githubusercontent.com/dorkitude/deadlore/main/scripts/install.ps1 | iex"
```

It installs the matching x64/ARM64 release to `%LocalAppData%\Programs\deadlore`, verifies its SHA-256 hash, and adds that directory to your user `PATH`.

### Go

```bash
go install ./cmd/deadlore
```

After the repository is published, install it from GitHub with:

```bash
go install github.com/dorkitude/deadlore/cmd/deadlore@latest
```

For one-off local development:

```bash
go run ./cmd/deadlore Haze
```

## Usage

```bash
deadlore Haze
deadlore hero Haze
deadlore ability "Sleep Dagger"
deadlore item "Heroic Aura"
deadlore hero list
deadlore item list
deadlore ability list
deadlore mechanic "Soul sharing"
deadlore source Infuser
deadlore --json item "Heroic Aura"
deadlore --no-color Haze
deadlore --refresh Haze
deadlore cache status
deadlore cache clear Haze
deadlore cache clear --all
```

Each response reports the canonical URL, wiki revision when available, wiki last-modified text, and local retrieval time. Cached data is refreshed after six hours by default; use `--refresh` to fetch it now.

Interactive terminals get a compact, in-game-inspired stat HUD with ANSI color on modern Windows, macOS, and Linux. Color turns off automatically for redirected output, `NO_COLOR`, `TERM=dumb`, `--no-color`, and `--json`.

## Notes on sourcing

The wiki's text is licensed CC BY-NC-SA 4.0, with exceptions for media and game content. This CLI preserves source metadata and is deliberately designed for individual, low-volume lookups. Obtain permission from the wiki maintainers before adding API sync, indexing, bulk fetches, or hosted redistribution.

## License

Deadlore is licensed under [CC BY-NC-SA 4.0](LICENSE), matching the license used for the Deadlock Wiki text it retrieves. No wiki text or media is bundled with this repository. CC BY-NC-SA is source-available but is not an OSI-approved open-source license because it restricts commercial use.
