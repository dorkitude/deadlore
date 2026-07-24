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
deadlore item "Heroic Aura"
deadlore mechanic "Soul sharing"
deadlore source Infuser
deadlore --json item "Heroic Aura"
deadlore --refresh Haze
deadlore cache status
deadlore cache clear Haze
deadlore cache clear --all
```

Each response reports the canonical URL, wiki revision when available, wiki last-modified text, and local retrieval time. Cached data is refreshed after six hours by default; use `--refresh` to fetch it now.

## Notes on sourcing

The wiki's text is licensed CC BY-NC-SA 4.0, with exceptions for media and game content. This CLI preserves source metadata and is deliberately designed for individual, low-volume lookups. Obtain permission from the wiki maintainers before adding API sync, indexing, bulk fetches, or hosted redistribution.

## License

Deadlore is licensed under [CC BY-NC-SA 4.0](LICENSE), matching the license used for the Deadlock Wiki text it retrieves. No wiki text or media is bundled with this repository. CC BY-NC-SA is source-available but is not an OSI-approved open-source license because it restricts commercial use.
