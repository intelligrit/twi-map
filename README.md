# TWI Map

<p align="center">
  <img src="internal/web/static/favicon.png" alt="TWI Map compass rose" width="128" />
</p>

Interactive, spoiler-free map of [The Wandering Inn](https://wanderinginn.com) web serial. Explore Innworld as you read — the map reveals locations only as they appear in the story.

## Features

- **Spoiler-free chapter slider** — set your reading progress and the map only shows what you've encountered so far
- **Multi-format navigation** — supports web serial chapters, audiobook books, and ebook editions
- **Clickable relationship lines** — see spatial connections between locations with source quotes from the text
- **Ghost provenance lines** — faded lines show connections to hidden locations so you keep spatial context
- **Searchable sidebar** — filter, hide, and locate any of 600+ extracted locations
- **Accessible** — WCAG AA contrast, full keyboard navigation, tabbable map markers, screen reader popup announcements, zero axe-core violations
- **Single binary** — compiles to one executable with all assets embedded via `go:embed`

## How It Works

TWI Map uses a multi-stage pipeline to transform 807 chapters (~12 million words) into a browsable map:

1. **Scrape** — Downloads the table of contents and chapter text from wanderinginn.com
2. **Extract** — Sends each chapter to Claude (Sonnet 4) to identify locations, relationships, and containment hierarchies
3. **Aggregate** — Deduplicates locations, merges canonical names, assigns coordinates via containment-based inheritance from hand-seeded reference points
4. **Serve** — Launches an interactive Leaflet map with a chapter slider for spoiler control

## Prerequisites

- Go 1.25+
- An [Anthropic API key](https://console.anthropic.com/) (for the `extract` step only)

## Install

```bash
go install github.com/robertmeta/twi-map@latest
```

Or build from source:

```bash
git clone https://github.com/robertmeta/twi-map.git
cd twi-map
make build
```

## Usage

Run the pipeline steps in order:

```bash
# 1. Fetch table of contents
twi-map scrape-toc

# 2. Download chapter text (by volume)
twi-map scrape-chapters --volume vol-1

# 3. Extract locations via Claude API (needs ANTHROPIC_API_KEY)
export ANTHROPIC_API_KEY=sk-ant-...
twi-map extract --volume vol-1

# 4. Merge extractions into unified dataset
twi-map aggregate

# 5. Launch the map
twi-map serve --addr localhost:8090
```

Check pipeline progress at any time:

```bash
twi-map status
```

### Volumes

The serial is split into 10 volumes (`vol-1` through `vol-10`). Scrape and extract each volume separately, then aggregate once at the end.

## Architecture

```
cmd/                  CLI commands (Cobra)
internal/
  scraper/            TOC + chapter HTML parsing
  extractor/          Anthropic API client, prompt templates
  aggregator/         Deduplication, coordinate assignment
  store/              DuckDB persistence layer
  web/                HTTP server, API handlers, embedded static files
  model/              Shared data types
```

All static assets (HTML, CSS, JS, Leaflet) are embedded into the binary via `go:embed`.

## Development

```bash
make test     # Run all tests
make fmt      # Format Go code
make vet      # Run go vet
make check    # fmt + vet + test
make serve    # Build and serve on localhost:8090
```

## Dependencies

All dependencies use permissive open source licenses:

| Dependency | License |
|---|---|
| [Cobra](https://github.com/spf13/cobra) (CLI framework) | Apache 2.0 |
| [goquery](https://github.com/PuerkitoBio/goquery) (HTML parsing) | BSD 3-Clause |
| [DuckDB Go driver](https://github.com/duckdb/duckdb-go) (embedded database) | MIT |
| [golang.org/x/time](https://pkg.go.dev/golang.org/x/time) (rate limiting) | BSD 3-Clause |
| [Leaflet.js](https://leafletjs.com) (map rendering) | BSD 2-Clause |

See [THIRD_PARTY_LICENSES](THIRD_PARTY_LICENSES) for full license texts of bundled libraries.

## Disclaimer

This is an unofficial fan project. It is not affiliated with, endorsed by, or connected to pirateaba or The Wandering Inn in any way.

*[The Wandering Inn](https://wanderinginn.com)* is written by pirateaba. All story content, including location names and world-building elements referenced by this tool, belong to their respective creator. No copyrighted chapter text is included in this repository — the extraction pipeline processes text at runtime via the Anthropic API.

This tool scrapes content from wanderinginn.com. Users are responsible for their own compliance with the site's terms of service. Please be respectful of the author's work and the site's resources.

## License

MIT — see [LICENSE](LICENSE).
