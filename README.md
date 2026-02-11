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
git clone https://github.com/jefrnc/tradervue-exporter.git
cd tradervue-exporter

# Build
make build

# Binary will be at ./bin/tvue
```

**Requirements:** Go 1.21+

Or download a prebuilt binary from [Releases](https://github.com/jefrnc/tradervue-exporter/releases).

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
# Full export (first time) - discovers your first trade and downloads everything
./bin/tvue export

# Incremental update (subsequent runs only fetch new trades)
./bin/tvue export

# Export specific date range
./bin/tvue export --from 2025-06-01 --to 2025-06-30

# Force re-export (overwrite existing data)
./bin/tvue export --from 2025-07-01 --force

# Include individual executions/fills (slower, one API call per trade)
./bin/tvue export --with-executions
```

**Example - first run:**

```
$ ./bin/tvue export
First run: discovering first trade date...
  Scanning page 2...
  Scanning page 3...
  ...
First trade found on: 2025-05-07
Exporting trades from 2025-05-07 to 2026-02-09...
  2025-05-07: 4 trades [ASST(L)]
  2025-05-08: 12 trades [VEEE(L) ASST(S) JZ(L) QBTS(L)]
  2025-05-09: 4 trades [NVVE(L)]
  2025-05-12: 6 trades [VFF(L) KDLY(L) ASST(S)]
  ...
Export complete: 173 days, 1818 trades
```

**Example - incremental update:**

```
$ ./bin/tvue export
Exporting trades from 2026-02-10 to 2026-02-10...
  2026-02-10: 5 trades [SNGX(L) MULN(S)]
Export complete: 1 days, 5 trades
```

### View Summaries

```bash
# All-time summary
./bin/tvue summary

# Filter by date range
./bin/tvue summary --from 2026-02-01 --to 2026-02-09

# Export to CSV for spreadsheets
./bin/tvue summary --csv -o report.csv
```

**Example - weekly summary:**

```
$ ./bin/tvue summary --from 2026-02-02 --to 2026-02-09
DATE        TRADES  GROSS P&L  NET P&L   WIN%  VOLUME  SYMBOLS
────        ──────  ─────────  ───────   ────  ──────  ───────
2026-02-02  3       +$24.86    +$24.34   100%  440     INLF(L)+$6.86 FUSE(L)+$9.60 NAMM(L)+$8.40
2026-02-03  3       +$54.48    +$53.32   100%  1168    WTO(L)+$1.46 EGHT(L)+$5.10 CYN(L)+$47.92
2026-02-04  9       -$3.67     -$6.65    89%   2588    DHX(L)+$1.30 CISS(L)+$11.22 GDTC(L)+$2.91 GOOGL(L)+$9.67 ...
2026-02-05  6       +$50.05    +$49.36   83%   1452    MSTR(L)-$5.41 AMZN(L)+$5.30 GWAV(L)+$11.70 ...
2026-02-06  5       +$36.63    +$36.50   80%   538     JZXN(L)+$13.30 SMX(L)-$10.09 FLYE(L)+$11.15 ...
2026-02-09  6       +$51.48    +$49.99   100%  1594    ABP(L)+$1.70 ENSC(L)+$11.26 IFBD(L)+$12.50 ...
────        ──────  ─────────  ───────   ────  ──────  ───────
TOTAL       32      +$213.84   +$206.87  91%   7780    6 days
```

**Example - CSV output:**

```
$ ./bin/tvue summary --from 2026-02-03 --to 2026-02-05 --csv
date,trades,gross_pl,net_pl,commission,fees,win_rate,winners,losers,volume,symbols
2026-02-03,3,54.48,53.32,1.75,-0.59,100.0,3,0,1168,WTO(L) EGHT(L) CYN(L)
2026-02-04,9,-3.67,-6.65,3.88,-0.90,88.9,8,1,2588,DHX(L) CISS(L) GDTC(L) GOOGL(L) ...
2026-02-05,6,50.05,49.36,2.18,-1.49,83.3,5,1,1452,MSTR(L) AMZN(L) GWAV(L) WTO(L) ...
```

## Configuration

### Environment Variables (.env)

```bash
TRADERVUE_USERNAME=your_username
TRADERVUE_PASSWORD=your_password
TVUE_DATA_DIR=./data              # optional, default: ./data
```

### CLI Flags

**Export command:**

| Flag | Short | Description |
|------|-------|-------------|
| `--username` | `-u` | Tradervue username |
| `--password` | `-p` | Tradervue password |
| `--data-dir` | `-d` | Data directory (default: `./data`) |
| `--from` | | Start date (yyyy-mm-dd) |
| `--to` | | End date (yyyy-mm-dd) |
| `--with-executions` | | Fetch individual fills per trade |
| `--force` | | Re-export existing dates |

**Summary command:**

| Flag | Short | Description |
|------|-------|-------------|
| `--data-dir` | `-d` | Data directory (default: `./data`) |
| `--from` | | Start date filter (yyyy-mm-dd) |
| `--to` | | End date filter (yyyy-mm-dd) |
| `--csv` | | Output as CSV instead of table |
| `--output` | `-o` | Write to file instead of stdout |

CLI flags take priority over `.env` values.

## How It Works

1. **Export** connects to the [Tradervue API](https://github.com/tradervue/api-docs) using your credentials
2. Discovers your first trade date and paginates through all trades
3. Saves one JSON file per trading day in `data/trades/`
4. Tracks progress in `data/state.json` for incremental updates
5. **Summary** reads the local JSON files (no API calls needed) and computes daily stats

### Data Storage

```
data/
├── state.json              # Export progress tracker
└── trades/
    ├── 2025-05-07.json     # All trades for that day
    ├── 2025-05-08.json
    ├── 2025-05-09.json
    └── ...
```

Each day file contains the full trade data from Tradervue including symbol, side (Long/Short), P&L, volume, commissions, fees, tags, notes, and optionally individual executions.

## API Usage

This tool uses the official [Tradervue REST API](https://github.com/tradervue/api-docs):

- Only accesses **your own data** with **your own credentials**
- Uses HTTP Basic Auth over SSL as documented
- Includes rate limiting (200ms between requests) to be a good API citizen
- Identifies itself via the `User-Agent` header as recommended by Tradervue

## Contributing

Contributions are welcome! Feel free to open issues or submit pull requests.

## License

MIT
