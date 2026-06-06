# audiobookshelf-status

A GitHub-contributions-style heatmap of your [Audiobookshelf](https://www.audiobookshelf.org/) listening history. Zoom from a year-at-a-glance overview all the way down to per-day cards showing covers, progress, and time listened.

**🔎 Live demo:** https://michondr.github.io/audiobookshelf-status/
*(demo data is synthetic — real books & covers from [Open Library](https://openlibrary.org/), ~3 years of generated listening)*

## What it does

- Pulls your listening sessions from your Audiobookshelf server and stores them in SQLite.
- Buckets them by calendar day and renders a zoomable timeline:
  **year overview → heatmap → finished books → covers → compact → full detail.**
- Downloads and downscales book covers locally.
- Syncs in the background on each page load — no cron needed.

## Run it

A single Go container serves both the frontend and API. Configure via a `.env` file:

```env
ABS_URL=http://your-audiobookshelf:13378
ABS_TOKEN=your_api_token
ABS_PUBLIC_URL=https://abs.example.com   # optional: enables "Open in Audiobookshelf" links
TZ=Europe/Prague
```

```sh
docker compose up -d
```

## Demo

`go run . -gendemo dist` builds the static demo into `dist/` (book list + covers fetched live from Open Library, ~3 years of listening synthesized through the same aggregation as production). On every push, `.github/workflows/demo.yml` builds it and publishes to GitHub Pages.
