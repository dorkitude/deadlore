# deadlore

`deadlore` uses the community-maintained [Deadlock Wiki](https://deadlock.wiki) as its authoritative source, with [Deadlock.io](https://deadlock.io) as a separately cached fallback only when the Wiki does not provide the requested data. It fetches canonical Wiki pages only and does not call its crawler-disallowed API endpoints; Deadlock.io is read through its public JSON source.

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
deadlore hero weapon Haze
deadlore hero weapon rank fire-rate
deadlore hero weapon compare Haze Wraith --boons 10
deadlore hero rank health
deadlore hero compare Haze Wraith
deadlore hero find --tag Assassin
deadlore build Haze
deadlore build Haze --sort all-time --limit 10
deadlore build Haze --language English
deadlore ability "Sleep Dagger"
deadlore item "Heroic Aura"
deadlore hero list
deadlore item list
deadlore ability list
deadlore mechanic "Soul sharing"
deadlore source Infuser
deadlore timers cheatsheet
deadlore cheat
deadlore timers camps
deadlore timers pickups
deadlore timers objectives
deadlore --json item "Heroic Aura"
deadlore --no-color Haze
deadlore --refresh Haze
deadlore cache status
deadlore cache clear Haze
deadlore cache clear --all
```

Each response reports the canonical URL, wiki revision when available, wiki last-modified text, and local retrieval time. Cached data is refreshed after six hours by default; use `--refresh` to fetch it now.

## Sources

The Wiki is used whenever it has the requested page, catalog, ability, or weapon data. Only a Wiki miss triggers the Deadlock.io fallback, which is clearly labeled in human output and reports `"fallback": true` in `--json`; ordinary Wiki responses do not query it. Wiki, Deadlock.io, and public-build data retain separate caches: `cache status` reports all three, and `cache clear` clears a matching Wiki/Deadlock.io entry while `cache clear --all` clears every cache. Use `--wiki-url`, `--deadlockio-url`, or `--deadlock-api-url` only when testing an alternate source endpoint.

## Timers

`deadlore timers cheatsheet` (or simply `deadlore cheat`) is a built-in, offline reference for the map's spawn and respawn timers. It covers jungle camps, Sinner's Sacrifice, breakables, power-ups, Mid-Boss, the Soul Urn, and the Unstable Rift. Use `camps`, `pickups`, or `objectives` to narrow the output. The reference identifies approximate or variable timers with `~` and reports the patch date through which it was checked; it never fetches the wiki.

## Hero analysis

Hero lookups also support comparisons and rankings from canonical hero pages. `deadlore hero weapon` shows one hero's weapon stats; `hero weapon rank` accepts `dps`, `fire-rate`, `bullet-damage`, `ammo`, `reload-time`, and `bullet-velocity`. `hero rank` accepts `health`, `health-regen`, `move-speed`, and `sprint-speed`. Add `--boons N` to a weapon view, rank, or comparison to evaluate stats at a particular boon count. `hero find --tag Assassin` filters the roster by a wiki hero tag. These aggregate commands use the local cache after their first lookup; `--json` includes provenance for every hero entry.

Interactive terminals get a compact, in-game-inspired stat HUD with ANSI color on modern Windows, macOS, and Linux. Color turns off automatically for redirected output, `NO_COLOR`, `TERM=dumb`, `--no-color`, and `--json`.

## Public builds

`deadlore build <hero>` lists the most-popular public in-game builds from [Deadlock API](https://api.deadlock-api.com/docs), defaulting to weekly favorites. Use `--sort all-time` or `--sort recent`, `--limit N`, and optionally `--language English`. The CLI deliberately displays only compact metadata—title, favorites, update date, and build ID—and links every result to its full build viewer on Deadlock Labs. It does not copy item paths, descriptions, or annotations. Build data has its own local cache, which appears in `cache status` and is cleared by `cache clear --all`.

## Notes on sourcing

The wiki's text is licensed CC BY-NC-SA 4.0, with exceptions for media and game content. This CLI preserves source metadata and is deliberately designed for individual, low-volume lookups. Obtain permission from the wiki maintainers before adding API sync, indexing, bulk fetches, or hosted redistribution.

## License

Deadlore is licensed under [CC BY-NC-SA 4.0](LICENSE), matching the license used for the Deadlock Wiki text it retrieves. No wiki text or media is bundled with this repository. CC BY-NC-SA is source-available but is not an OSI-approved open-source license because it restricts commercial use.
