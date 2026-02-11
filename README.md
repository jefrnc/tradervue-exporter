# tvue - Tradervue Trade Exporter & Analyzer

A fast CLI tool to export all your trades from [Tradervue](https://www.tradervue.com) and generate daily summaries. Built in Go for speed and simplicity.

## Features

- **Incremental export** - First run downloads everything, subsequent runs only fetch new trades
- **Daily summaries** - See your P&L, win rate, and symbols traded per day at a glance
- **Long/Short tracking** - Know exactly what direction you traded each ticker
- **CSV export** - Pipe your data into spreadsheets or other tools
- **Credential flexibility** - Use CLI flags or `.env` file
- **Zero config** - Just provide your Tradervue credentials and go

## Installation

```bash
# Clone the repository
git clone https://github.com/jefrnc/tradervue-utils.git
cd tradervue-utils

# Build
make build

# Binary will be at ./bin/tvue
```

**Requirements:** Go 1.21+

## Quick Start

```bash
# Set up credentials (option A: .env file)
cp .env.example .env
# Edit .env with your Tradervue username and password

# Export all trades
./bin/tvue export

# Or pass credentials directly (option B: flags)
./bin/tvue export -u your_username -p your_password

# View daily summaries
./bin/tvue summary
```

## Usage

### Export Trades

```bash
# Full export (first time)
tvue export

# Incremental update (runs automatically after first export)
tvue export

# Export specific date range
tvue export --from 2025-01-01 --to 2025-01-31

# Force re-export (overwrite existing data)
tvue export --from 2025-01-01 --force

# Include individual executions (slower, more detailed)
tvue export --with-executions
```

### View Summaries

```bash
# All-time summary
tvue summary

# Filter by date range
tvue summary --from 2025-01-01 --to 2025-01-31

# Export to CSV
tvue summary --csv -o report.csv
```

**Example output:**

```
DATE        TRADES  GROSS P&L   NET P&L    WIN%  VOLUME  SYMBOLS
────        ──────  ─────────   ───────    ────  ──────  ───────
2025-01-15  8       +$234.50    +$219.30   75%   45000   SNGX(L)+$120.00 MULN(L)+$114.50
2025-01-16  5       -$45.20     -$58.70    40%   22000   ATER(L)-$45.20
2025-01-17  12      +$189.00    +$171.40   67%   68000   FFIE(S)+$89.00 NBEV(L)+$100.00
────        ──────  ─────────   ───────    ────  ──────  ───────
TOTAL       25      +$378.30    +$332.00   64%   135000  3 days
```

## Configuration

### Environment Variables (.env)

```bash
TRADERVUE_USERNAME=your_username
TRADERVUE_PASSWORD=your_password
TVUE_DATA_DIR=./data              # optional, default: ./data
```

### CLI Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--username` | `-u` | Tradervue username |
| `--password` | `-p` | Tradervue password |
| `--data-dir` | `-d` | Data directory (default: `./data`) |
| `--from` | | Start date (yyyy-mm-dd) |
| `--to` | | End date (yyyy-mm-dd) |
| `--with-executions` | | Fetch individual fills per trade |
| `--force` | | Re-export existing dates |
| `--csv` | | Output summary as CSV |
| `--output` | `-o` | Write to file instead of stdout |

CLI flags take priority over `.env` values.

## How It Works

1. **Export** connects to the [Tradervue API](https://github.com/tradervue/api-docs) using your credentials
2. Discovers your first trade date and paginates through all trades
3. Saves one JSON file per trading day in `data/trades/`
4. Tracks progress in `data/state.json` for incremental updates
5. **Summary** reads the local JSON files (no API calls) and computes daily stats

### Data Storage

```
data/
├── state.json              # Export progress tracker
└── trades/
    ├── 2025-01-15.json     # All trades for that day
    ├── 2025-01-16.json
    └── ...
```

Each day file contains the full trade data from Tradervue including symbol, side, P&L, volume, tags, and optionally individual executions.

## API Usage

This tool uses the official [Tradervue REST API](https://github.com/tradervue/api-docs):

- Only accesses **your own data** with **your own credentials**
- Uses HTTP Basic Auth over SSL as documented
- Includes rate limiting (200ms between requests) to be a good API citizen
- Identifies itself via the `User-Agent` header as recommended

## Contributing

Contributions are welcome! Feel free to open issues or submit pull requests.

## License

MIT
