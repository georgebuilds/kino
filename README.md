# Kino

A local-first personal finance desktop app built with Go, Vue 3, and Wails.

## Features

- **Multi-account tracking** — checking, savings, credit cards, investments, loans, crypto, cash
- **Transaction management** — create, edit, categorize, reconcile, and filter transactions
- **Import** — CSV and OFX/QFX file import with exact and fuzzy duplicate detection
- **Budgets** — monthly, weekly, and annual budgets with optional rollover
- **Cash flow** — Sankey diagram showing income sources flowing to expense categories
- **Net worth** — 12 or 24-month history snapshots
- **Goals** — savings goal tracking linked to accounts
- **Local data** — all data lives in a single `.kino` SQLite file you own and control

## Privacy

No accounts, no sync servers, no telemetry. Your `.kino` file can be stored anywhere — iCloud Drive, Dropbox, Google Drive, OneDrive, or just your local disk.

## Development

**Prerequisites:** Go 1.24+, Node 18+, [Wails v2](https://wails.io/docs/gettingstarted/installation)

```bash
# Install Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Live development (hot reload)
wails dev

# Production build
wails build
```

The dev server also exposes a browser UI at `http://localhost:34115` with access to all Go methods via devtools.

## Architecture

| Layer | Technology |
|-------|-----------|
| Desktop shell | Wails v2 |
| Backend | Go |
| Frontend | Vue 3 + TypeScript + Vite + Tailwind CSS |
| Database | SQLite (WAL mode, via `modernc.org/sqlite`) |
| Charts | Chart.js |

**Backend modules:**

| File | Responsibility |
|------|---------------|
| `app.go` | DB lifecycle, file open/create/move, MRU tracking |
| `app_accounts.go` | Account CRUD |
| `app_transactions.go` | Transaction CRUD, filters, transfer pairs |
| `app_categories.go` | Category CRUD (hierarchical) |
| `app_budgets.go` | Budget CRUD with rollover logic |
| `app_cashflow.go` | Sankey diagram data |
| `app_networth.go` | Net worth history snapshots |
| `app_summary.go` | Monthly aggregation (income, expenses, savings) |
| `app_import.go` | CSV/OFX import, duplicate detection |
| `internal/db/` | SQLite connection, schema migrations, all queries |
| `internal/models/` | Shared data types |
| `internal/importer/` | CSV and OFX parsers |

**Key conventions:**
- All monetary values are stored as `int64` cents (no floating-point)
- Transfer transactions are linked in pairs via `TransferPairID`
- Schema migrations are idempotent and version-gated via `kino_meta`
- The frontend calls Go methods directly via Wails RPC bindings
