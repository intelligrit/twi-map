# TWI Map

Interactive, spoiler-free map of [The Wandering Inn](https://wanderinginn.com) web serial. Explore Innworld as you read — the map reveals locations only as they appear in the story.

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

## License

MIT — see [LICENSE](LICENSE).
