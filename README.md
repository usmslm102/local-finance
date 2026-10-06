# 🇮🇳 LocalFinance

[![CI](https://github.com/usmslm102/local-finance/actions/workflows/ci.yml/badge.svg)](https://github.com/usmslm102/local-finance/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Website](https://img.shields.io/badge/Website-usamaansari.com-emerald)](https://usamaansari.com/local-finance/)
[![Platform](https://img.shields.io/badge/Platform-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey)](https://github.com/usmslm102/local-finance/releases)
[![Architecture](https://img.shields.io/badge/Architecture-100%25%20Offline-success)](https://github.com/usmslm102/local-finance)

> **A privacy-first, 100% offline personal finance intelligence app tailored for the Indian banking & credit card ecosystem.**  
> Distributed as a **single standalone executable binary** with an embedded React frontend and an embedded pure-Go SQLite database.

> [!IMPORTANT]
> ### ⚖️ Disclaimer & Notice of Liability (Use at Your Own Risk)
> **LocalFinance is an independent open-source software project distributed under the terms of the [MIT License](LICENSE).**
>
> - **"AS IS" & Use at Your Own Risk**: This software is provided **"AS IS"**, without warranty of any kind, express or implied, including but not limited to the warranties of merchantability, fitness for a particular purpose, and non-infringement. You use this application entirely at your own discretion and risk.
> - **No Financial, Tax, or Legal Advice**: LocalFinance is an offline analytical and record-keeping tool. It does not provide financial planning, accounting, investment, tax, or legal advice.
> - **No Banking Affiliation**: LocalFinance is completely independent and is not affiliated with, endorsed by, sponsored by, or connected to HDFC Bank, ICICI Bank, Axis Bank, State Bank of India (SBI), the Reserve Bank of India (RBI), or any other financial institution. All bank names, brand trademarks, and logos are property of their respective owners and are referenced solely for identification and parser compatibility.
> - **Zero Liability**: To the maximum extent permitted by applicable law, in no event shall the authors, maintainers, copyright holders, or contributors be held liable for any claim, damages, losses, or legal liabilities—whether in contract, tort (including negligence), or otherwise—arising from, out of, or in connection with the software, its calculations, parser extractions, categorization rules, data loss, miscalculated figures, financial decisions, tax assessments, or any consequences resulting from its use.
> - **User Responsibility**: Users are solely responsible for reviewing and verifying the accuracy of all imported transactions, balances, and calculations against their original official bank and credit card statements before taking any financial action.

---

## 🌟 Key Highlights

- **100% Offline & Private**: Zero cloud sync, zero telemetry, no SMS scraping, and no third-party account aggregators. All your financial data stays strictly on your local machine in an open SQLite database file (`local_finance.db` or `~/.localfinance/local_finance.db`).
- **Single-Binary Portability**: Built in **Go (Golang)** with the React 19 single-page app bundled directly into the executable via `go:embed`. Cross-compiles across Windows (`.exe`), macOS, and Linux with zero CGO dependencies (`CGO_ENABLED=0`).
- **Zero-Config Database Migrations**: Schema migrations managed automatically via embedded Goose (`github.com/pressly/goose/v3`) on startup—no external migration CLI needed.
- **Generic & Extensible Parser Engine**: Pluggable architecture that **auto-detects** bank formats, account types (Savings, Current, Credit Cards), and file structures (PDF, CSV, Excel/XLS/XLSX).
- **Native Password-Protected PDF Support**: Upload encrypted bank statements directly through the UI; LocalFinance decrypts and parses them in-memory without altering original file formatting.
- **Multi-Account & Multi-Year Bulk Statements**: Handles 10+ year bulk historic statements without missing records, correctly distinguishing multiple accounts (e.g. separate HDFC Savings and Current accounts).
- **Indian Narration Intelligence**: Native regex cleaning engine for:
  - **UPI Transfers**: Extracts Payee names, VPAs (e.g., `merchant@icici`, `user@okaxis`), Reference/UTR numbers, and apps used (GPay, PhonePe, Paytm).
  - **Card Swipes / POS**: Cleans raw terminal dumps (e.g., `POS 40124300 SWIGGY BANGALORE IN` &rarr; `Swiggy`).
  - **IMPS / NEFT / RTGS**: Resolves beneficiary names and 12-to-16 digit UTR numbers.
  - **ATM Withdrawals & Bank Charges**: Distinguishes cash withdrawals, SMS alert fees, AMC charges, and GST.
- **Credit Card Billing Intelligence**:
  - Automatically captures billing cycles, statement dates, payment due dates, minimum due, total due, finance charges, reward points, and cashback.
- **Duplicate Detection & Smart Upsert**:
  - Deterministic `SHA-256` transaction hashing prevents duplicates when uploading overlapping statement periods.
  - Idempotent SQLite upserts (`ON CONFLICT`) safely update balances while **strictly preserving** your custom category overrides, notes, and tags.
- **Optional Local Security / Password Protection**:
  - Protect local database access with an optional PIN/password stored with PBKDF2/argon2 hashing, complete with automatic lock timeout.
- **Built-in Release Checks & Updates**:
  - Checks public GitHub Releases automatically on launch (cached for four hours). Disable **Automatic Update Checks** in Settings to opt out; **Check for Updates** remains available on demand.
  - Standalone executables support **Update & Restart** when writable. macOS `.app` installations offer the complete DMG instead: quit LocalFinance and replace the app in Applications to preserve its signature. Financial data remains in the separate local database.
  - macOS releases use local ad-hoc code signatures; they are not Developer ID signed or notarized.
- **Modern Interactive Dashboard**:
  - Built with **React 19**, **TypeScript**, **Tailwind CSS v4**, **TanStack Router**, **TanStack Table**, and **Recharts**.

---

## 🏦 Supported Banks & Statements Matrix

LocalFinance features dedicated parsers for major Indian banks, with native extraction of statements in PDF (including password-encrypted files decrypted losslessly in memory), CSV, and Excel formats.

### Compatibility Matrix

| Bank | Savings Account | Current Account | Core / Premium CC | Swiggy HDFC | Amazon Pay ICICI | Flipkart Axis | RuPay UPI CC | Supported Formats |
| :--- | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :--- |
| **HDFC Bank** | ✅ Supported | ✅ Supported | ✅ Supported <br><sub>*(Regalia, Millennia, Infinia)*</sub> | ✅ Supported | ➖ *(N/A)* | ➖ *(N/A)* | ✅ Supported <br><sub>*(Tata Neu, RuPay)*</sub> | PDF, CSV, Excel (`.xls`, `.xlsx`) |
| **ICICI Bank** | ✅ Supported | ⏳ Planned | ✅ Supported <br><sub>*(Coral, Rubyx, Sapphiro)*</sub> | ➖ *(N/A)* | ✅ Supported | ➖ *(N/A)* | ⏳ Planned | PDF (Savings, Credit Card) |
| **Union Bank of India** | ✅ Supported | ⏳ Planned | ⏳ Planned | ➖ *(N/A)* | ➖ *(N/A)* | ➖ *(N/A)* | ⏳ Planned | PDF (Savings) |
| **Axis Bank** | ⏳ Planned <sup>*</sup> | ⏳ Planned | ✅ Supported <br><sub>*(ACE, Magnus, Atlas, Neo)*</sub> | ➖ *(N/A)* | ➖ *(N/A)* | ✅ Supported | ⏳ Planned | PDF (Credit Card) |
| **State Bank of India (SBI)** | ⏳ Planned <sup>*</sup> | ⏳ Planned | ⏳ Planned | ➖ *(N/A)* | ➖ *(N/A)* | ➖ *(N/A)* | ⏳ Planned | Generic CSV |
| **Kotak Mahindra Bank** | ⏳ Planned <sup>*</sup> | ⏳ Planned | ⏳ Planned | ➖ *(N/A)* | ➖ *(N/A)* | ➖ *(N/A)* | ⏳ Planned | Generic CSV |

> <sup>*</sup> **Universal CSV Support**: Any bank statement exported as CSV (including SBI, Kotak, ICICI Savings, etc.) can be parsed and ingested using LocalFinance's built-in delimiter-sniffing generic CSV engine.

Synthetic ICICI and Union Bank savings PDFs in `samples/savings/` exercise parser detection and full PDF extraction in the test suite. They contain only fabricated names, descriptions, dates, and amounts.

### Credit Card Variants Breakdown

| Bank | Card Variant / Series | Network | Supported Formats | Extracted Intelligence | Status |
| :--- | :--- | :---: | :--- | :--- | :---: |
| **HDFC Bank** | **Regalia / Regalia Gold** | VISA | PDF, CSV | Billing period, due date, reward points, credit limit | ✅ Supported |
| **HDFC Bank** | **Millennia** | VISA / MC | PDF, CSV | Billing period, due date, cashback, reward points | ✅ Supported |
| **HDFC Bank** | **Infinia** | VISA | PDF, CSV | Billing period, due date, reward points, credit limit | ✅ Supported |
| **HDFC Bank** | **Swiggy HDFC** | Mastercard | PDF, CSV | Cashback earned & credited, billing period, due date | ✅ Supported |
| **HDFC Bank** | **Tata Neu / RuPay UPI** | RuPay | PDF, CSV | UPI merchant transactions, NeuCoins/rewards, due date | ✅ Supported |
| **ICICI Bank** | **Amazon Pay ICICI** | VISA | PDF | 5%/2%/1% cashback calculation, due dates, reward tracking | ✅ Supported |
| **ICICI Bank** | **Coral / Rubyx / Sapphiro** | VISA / MC | PDF | Purchases/charges, reward points, limits, due dates | ✅ Supported |
| **Axis Bank** | **Flipkart Axis** | Mastercard / VISA | PDF | Cashback earned & credited, merchant categories, due dates | ✅ Supported |
| **Axis Bank** | **ACE / Magnus / Atlas / Neo** | VISA / MC | PDF | Itemized spends, reward points, credit limits, due dates | ✅ Supported |

---

## 🏗️ Architecture

```mermaid
graph TD
    A[Bank & Credit Card Statements<br/>PDF, CSV, Excel, Bulk Historic] --> B[Extensible Parser Engine]
    
    subgraph "Extensible Parser Engine"
        B --> B1[Format Sniffer & Confidence Scorer]
        B1 --> B2[In-Memory Decryption & Positional Extractor]
        B2 --> B3[Bank Adapters<br/>HDFC, ICICI, Axis, SBI...]
        B3 --> B4[Indian Narration & UPI Regex Engine]
        B4 --> B5[Deterministic SHA-256 Fingerprinter]
    end

    B5 --> C[(Embedded Pure-Go SQLite DB<br/>modernc.org/sqlite + Goose Migrations)]
    
    subgraph "Go Backend (Gin Framework)"
        C --> D[Transaction & Ingestion Service]
        C --> E[Analytics, Budget & Cash Flow Engine]
        C --> S[App Security & Auth Store]
        D --> F[Local REST API Server<br/>127.0.0.1:8080]
        E --> F
        S --> F
    end

    subgraph "Embedded Frontend (React 19 + Vite)"
        F --> G[Embedded Static File Server<br/>go:embed all:frontend/dist]
        G --> H[TanStack Router SPA<br/>Overview, Ledger, Importer, Rules, Security]
    end

    H --> I[Default Browser Auto-Open<br/>or http://127.0.0.1:8080]
```

---

## 🛠️ Technology Stack

| Layer | Component | Description |
| :--- | :--- | :--- |
| **Backend Core** | **Go 1.22+** (Go 1.26 toolchain) | High-performance, low-memory footprint, single-binary compilation with `CGO_ENABLED=0`. |
| **API Framework** | **Gin (`gin-gonic/gin`)** | High-speed HTTP router, multipart file upload handling, CORS, and embedded static asset serving. |
| **Database Engine** | **SQLite (`modernc.org/sqlite`)** | Pure Go SQLite engine (zero CGO required), WAL mode enabled with busy timeout pragmas. |
| **Schema Migrations** | **Goose (`pressly/goose/v3`)** | Embedded SQL migrations executed automatically on startup via `embed.FS`. |
| **Asset Embedding** | **Go `embed`** | Packages the compiled React production bundle directly into the Go binary. |
| **Frontend Framework**| **React 19 + TypeScript + Vite** | High-speed modern UI with type-safety and hot module replacement. |
| **Package Manager** | **`pnpm`** | Strict dependency resolution (use `pnpm` exclusively for all frontend tasks). |
| **UI Components** | **shadcn/ui + Radix UI** | Accessible, headless UI components styled with Tailwind CSS. |
| **Routing** | **TanStack Router** | Client-side routing for `/`, `/transactions`, `/import`, `/categories`, `/settings`. |
| **Table & Ledger** | **TanStack Table v8** | Virtualized transaction table with multi-column sorting, filtering, and pagination. |
| **State & Fetching** | **TanStack Query v5** | Server-state caching and automatic cache invalidation on imports. |
| **Styling** | **Tailwind CSS v4** | Modern design tokens and dark financial aesthetic. |
| **Visualizations** | **Recharts** | Interactive donut spending breakdown and cash flow bar charts. |

---

## 🚀 Running & Developing LocalFinance

### ⚡ Quick Start: Pre-built Standalone Releases
You don't need Go or Node.js installed to use LocalFinance. Download the pre-compiled package for your operating system from **[GitHub Releases](https://github.com/usmslm102/local-finance/releases)**.

#### 🍏 macOS (Universal: Apple Silicon M1/M2/M3/M4 & Intel)
1. Download **`LocalFinance.dmg`** directly from Releases.
2. Open the `.dmg` and drag **LocalFinance** into your **Applications** folder.
3. Launch **LocalFinance** from Applications or Spotlight.
   - *Note on Gatekeeper*: Because LocalFinance is open-source and distributed without an Apple Developer ID certificate, macOS may show a prompt:
     > *"LocalFinance" cannot be opened because Apple cannot check it for malicious software.*
   - To open: **Right-click** (or Control-click) `LocalFinance.app` in Finder, choose **Open**, and click **Open** on the prompt. Alternatively, go to **System Settings** → **Privacy & Security**, scroll down to **Security**, and click **Open Anyway**.
   - *(CLI Alternative)*: A universal command-line binary archive is also available as `local-finance-darwin-universal.tar.gz`.

#### 🪟 Windows
1. Download **`LocalFinance.exe`** directly from Releases (no unzipping required).
2. Double-click `LocalFinance.exe` to launch.
   - If Windows SmartScreen appears (*"Windows protected your PC"*), click **More info** → **Run anyway**.

#### 🐧 Linux
1. Download `local-finance-linux-amd64.tar.gz` (or `local-finance-linux-arm64.tar.gz`).
2. Extract and run:
   ```bash
   tar -xzf local-finance-linux-amd64.tar.gz
   ./local-finance
   ```

---

### Prerequisites (For Building from Source)
- **Go 1.22+** (configured with Go 1.26 toolchain)
- **Node.js 20+**
- **pnpm** (install via `npm install -g pnpm` or `brew install pnpm`)

---

### Development Workflows

#### 1. Quick Unified Run (`make dev` or `make serve`)
Compiles the React frontend and boots up the Go server with auto-browser opening:
```bash
make dev
```

#### 2. Live Development Mode (Dual Process with Hot Reload)
When actively building React UI components or making backend changes:

* **Terminal 1: Go Backend Server**
  ```bash
  make dev-backend
  # Or manually:
  go run ./cmd/server/main.go -port 8080 -db ./local_finance.db -open=false
  ```
  *(Optional: Use [`air`](https://github.com/air-verse/air) for live Go auto-recompilation: `air -c .air.toml`)*

* **Terminal 2: Frontend Vite Dev Server**
  ```bash
  make dev-frontend
  # Or manually:
  cd frontend && pnpm dev
  ```
  Open **`http://localhost:5173`** in your browser. All UI edits reflect instantly via Hot Module Replacement (HMR), and API requests (`/api/*`) are automatically proxied to the Go backend on port `8080`.

#### 3. Production Build (Single Standalone Binary)
Builds the production React bundle, embeds it into Go, and outputs the standalone executable:
```bash
make build
# Or manually:
cd frontend && pnpm build && cd .. && go build -o local-finance ./cmd/server/main.go
```

#### 4. Run the Production Binary
```bash
./local-finance -port 8080 -open
```

#### Available CLI Flags:
| Flag | Default | Description |
| :--- | :--- | :--- |
| `-port` | `8080` | Port for the local HTTP server. |
| `-db` | `~/.localfinance/local_finance.db` | Path to the SQLite database file. |
| `-open` | `true` | Automatically opens the application in your default web browser on startup (`-open=false` to disable). |

#### 5. Clean Build Artifacts
```bash
make clean
# Deletes frontend/dist, compiled binary, and test databases
```

#### 6. Run Automated Tests
```bash
make test
# Or:
go test -v ./...
```

---

### Cross-Compiling for Other Operating Systems

Because LocalFinance uses pure Go SQLite (`modernc.org/sqlite`), cross-compilation requires **zero CGO** (`CGO_ENABLED=0`):

```bash
# 1. Build frontend assets
cd frontend && pnpm build && cd ..

# 2. Compile for macOS (Apple Silicon M1/M2/M3/M4)
GOOS=darwin GOARCH=arm64 go build -o local-finance-darwin-arm64 ./cmd/server/main.go

# 3. Compile for macOS (Intel x86_64)
GOOS=darwin GOARCH=amd64 go build -o local-finance-darwin-amd64 ./cmd/server/main.go

# 4. Compile for Windows 64-bit (.exe)
GOOS=windows GOARCH=amd64 go build -o local-finance-windows-amd64.exe ./cmd/server/main.go

# 5. Compile for Linux 64-bit
GOOS=linux GOARCH=amd64 go build -o local-finance-linux-amd64 ./cmd/server/main.go
```

---

## 🔒 Password-Protected PDF Statements & Optional Decryption

### Native Decryption (Recommended)
You do **not** need to decrypt or remove passwords from your statements before uploading!
- LocalFinance natively decrypts protected PDF statements in memory using the password input in the upload modal.
- It parses the full multi-page document in its native layout without touching your filesystem or saving decrypted files to disk.

---

### ⚠️ Avoid macOS Preview "Print to PDF"
When removing passwords from bank statements, **do not** use **File &rarr; Print &rarr; Save as PDF** in macOS Preview:
1. **Canvas Rescaling**: Preview's virtual printer often downsizes wide/landscape bank statements (e.g. from 730 pt landscape down to 245 pt portrait), which can break column alignments in standard parsers.
2. **Missing Pages**: The print dialog frequently defaults to printing **only Page 1**, accidentally truncating multi-page statements (e.g. losing 13 out of 14 pages).

---

### 💡 How to Safely Remove Passwords via `qpdf` (Lossless)

If you wish to remove passwords from bank statements for local archival or inspection, the safest and cleanest utility is **`qpdf`**.

`qpdf` performs a **pure cryptographic decryption** on the PDF binary stream without re-rendering, rescaling, or rasterizing vector coordinates:

#### 1. Install `qpdf`
* **macOS** (Homebrew):
  ```bash
  brew install qpdf
  ```
* **Ubuntu / Debian**:
  ```bash
  sudo apt install qpdf
  ```
* **Windows** (Chocolatey or Scoop):
  ```bash
  choco install qpdf
  # Or: scoop install qpdf
  ```

#### 2. Decrypt Statement
```bash
qpdf --decrypt --password="YOUR_PASSWORD" "protected_statement.pdf" "unlocked_statement.pdf"
```

#### 3. Batch Decrypt All Statements in a Folder (macOS/Linux)
```bash
for f in *.pdf; do
    qpdf --decrypt --password="YOUR_PASSWORD" "$f" "unlocked_${f}"
done
```

> **Why `qpdf`?** It preserves 100% of the original PDF layout, font streams, page counts, and coordinate dimensions with zero data loss.

---

#### Alternative GUI Method: macOS Preview "Export" (Not Print)
If you prefer not to use the terminal:
1. Open the protected PDF in **Preview** and enter your password.
2. Click **File &rarr; Export...** (do **NOT** use *Export as PDF* or *Print*).
3. In the export dialog, ensure the **Encrypt** checkbox is **unchecked**.
4. Save the file. This preserves all pages and canvas dimensions.

---

## 📁 Repository Structure

```
local-finance/
├── cmd/
│   └── server/
│       └── main.go                 # Application entry point: CLI flags, DB bootstrap, auto-browser
├── embed.go                        # go:embed directives bundling frontend/dist into Go binary
├── frontend/                       # React 19 + TypeScript + Vite frontend
│   ├── dist/                       # Production frontend build output (embedded into Go binary)
│   ├── src/
│   │   ├── components/
│   │   │   ├── dashboard/          # KPI Cards, Spend Breakdown Donut, Cash Flow Bar Chart
│   │   │   ├── import/             # Statement upload dropzone & upsert audit summary
│   │   │   ├── layout/             # Top Navbar with active navigation links
│   │   │   ├── rules/              # Categories & auto-categorization rule inspector
│   │   │   ├── settings/           # Local Auth & database lock settings
│   │   │   └── transactions/       # TanStack Table ledger with filtering & pagination
│   │   ├── lib/
│   │   │   ├── api.ts              # Typed API client for Go backend endpoints
│   │   │   └── utils.ts            # Formatting helpers (INR currency, dates, class merging)
│   │   ├── types/                  # TypeScript interfaces (Transaction, Account, Analytics)
│   │   ├── index.css               # Tailwind CSS v4 design tokens & theme
│   │   ├── main.tsx                # QueryClientProvider & root mount
│   │   └── router.tsx              # TanStack Router configuration
│   ├── package.json                # Frontend dependencies (ALWAYS managed with pnpm)
│   └── vite.config.ts              # Vite configuration (port 5173 with proxy to backend :8080)
├── go.mod                          # Go module dependencies
├── go.sum                          # Go checksums
├── internal/
│   ├── api/
│   │   ├── handlers.go             # Gin HTTP route handlers
│   │   └── routes.go               # Router setup, CORS, static SPA fallback handler
│   ├── db/
│   │   ├── db.go                   # SQLite connection, Goose migration runner, queries, seeders
│   │   ├── db_test.go              # Database & migration unit tests
│   │   └── migrations/             # Embedded Goose SQL migration files
│   │       ├── 00001_initial_schema.sql
│   │       ├── ...
│   │       └── 00010_add_account_number.sql
│   ├── models/
│   │   └── models.go               # Shared domain structs & enums
│   ├── parser/
│   │   ├── cleaner.go              # Indian UPI, POS terminal, IMPS/NEFT narration regex engine
│   │   ├── extractor/              # Generic format extractors (Excel, PDF, CSV, normalizers)
│   │   │   ├── csv.go              # Auto-delimiter & BOM stripping CSV extractor
│   │   │   ├── excel.go            # OpenXML (.xlsx) & legacy BIFF8 (.xls) / HTML extractor
│   │   │   ├── helper.go           # Indian currency & date normalizers, ColumnSpec finder
│   │   │   └── pdf.go              # Decryption & positional text token extractor
│   │   ├── hdfc_cc_csv.go          # HDFC Credit Card CSV parser plugin
│   │   ├── hdfc_cc_pdf.go          # HDFC Credit Card PDF parser plugin
│   │   ├── hdfc_savings_csv.go     # HDFC Savings/Current Account CSV parser plugin
│   │   ├── hdfc_savings_pdf.go     # HDFC Savings/Current Account PDF parser plugin (bulk & standard)
│   │   ├── hdfc_savings_xls.go     # HDFC Savings/Current Account Excel parser plugin
│   │   ├── icici_cc_pdf.go         # ICICI Bank Credit Card PDF parser plugin
│   │   ├── axis_cc_pdf.go          # Axis Bank Credit Card PDF parser plugin
│   │   └── parser.go               # Generic StatementParser interface & auto-detection Registry
│   └── service/
│       └── transaction_service.go  # Ingestion pipeline, SHA-256 fingerprinting & smart upserts
├── Makefile                        # Build and development automation targets
├── ROADMAP.md                      # Product specifications, completed ledger & future roadmap
└── README.md                       # Comprehensive user and developer manual
```

---

## 🧩 Adding a New Bank Parser (Extensibility)

Adding support for a new bank or format (e.g. SBI, Kotak, Axis, Amex) is completely modular:

1. Create a new file in `internal/parser/<bank>_<account_type>_<format>.go` (e.g. `internal/parser/sbi_savings_csv.go`).
2. Implement the `StatementParser` interface:

```go
package parser

import (
	"io"
	"strings"
	"local-finance/internal/models"
)

type SBISavingsCSVParser struct{}

func init() {
	// Automatically registers parser with the global registry
	DefaultRegistry.Register(&SBISavingsCSVParser{})
}

func (p *SBISavingsCSVParser) ID() string {
	return "sbi_savings_csv_v1"
}

func (p *SBISavingsCSVParser) Name() string {
	return "State Bank of India Savings CSV"
}

func (p *SBISavingsCSVParser) SupportedTypes() []StatementType {
	return []StatementType{TypeSavingsCSV}
}

func (p *SBISavingsCSVParser) CanParse(filename string, sample []byte) (float64, string, models.AccountType) {
	content := strings.ToUpper(string(sample))
	confidence := 0.0
	if strings.Contains(content, "STATE BANK OF INDIA") || strings.Contains(content, "SBI") {
		confidence += 0.5
	}
	if strings.Contains(content, "TXN DATE") && strings.Contains(content, "DESCRIPTION") {
		confidence += 0.4
	}
	return confidence, "State Bank of India", models.AccountTypeSavings
}

func (p *SBISavingsCSVParser) Parse(r io.Reader, opts ParseOptions) ([]ParsedTransaction, StatementMeta, error) {
	// 1. Read rows using extractor.ExtractCSV
	// 2. Normalize dates using extractor.NormalizeIndianDate
	// 3. Clean narrations using CleanNarration(raw)
	// 4. Return []ParsedTransaction and StatementMeta
}
```

---

## Monthly Review

Overview now includes a compact review of the latest month with imported data. Choose **Understand what changed** to open the detailed review in Cash Flow, or use **Review month** there to inspect another month.

- Expand statement coverage to see how many days each tracked account's reported statement ranges cover in both periods. Overlaps count once; gaps remain visible. This is a date-coverage check, not a balance audit.
- Inspect the largest category changes and the merchants contributing to them. **View transactions** shows the exact recorded debits for either period, with pagination.
- Use **Plan next month** to open the existing budget editor for the following month and category. Nothing is saved automatically.

Historical months compare full calendar months. The current month compares elapsed days, capped at the previous month's last day when it is shorter. Transfers and excluded transactions do not count as spending; credits and refunds are not deducted. Missing statement coverage can distort comparisons, so the review flags it explicitly.

The review is read-only, runs offline, and uses the app's existing authentication and discreet mode.

## 🔌 REST API Reference

| Endpoint | Method | Description |
| :--- | :--- | :--- |
| `/api/health` | `GET` | Server health check and version info |
| `/api/accounts` | `GET` | List all discovered bank accounts and credit cards with current balances |
| `/api/transactions` | `GET` | List transactions with filters (`account_id`, `category_id`, `tx_type`, `search`, `start_date`, `end_date`, `page`, `page_size`) |
| `/api/statements/upload` | `POST` | Ingest statement file (multipart `file`, optional `password`, `account_id`, `parser_id`) |
| `/api/statements/preview` | `POST` | Dry-run statement parse without saving to DB (shows detected metadata and transactions) |
| `/api/statements` | `GET` | List history of uploaded statements with checksums and date ranges |
| `/api/categories` | `GET` | List spending categories with icons and color tokens |
| `/api/rules` | `GET` | List active auto-categorization keyword & regex rules |
| `/api/analytics/overview`| `GET` | Get total income, total expense, net savings, category breakdown, and monthly cash flow |
| `/api/analytics/monthly-review` | `GET` | Spending comparison and statement coverage; optional `month=YYYY-MM`, defaulting to the latest month with imported data |
| `/api/analytics/monthly-review/transactions` | `GET` | Supporting debits for `month`, `category` (empty means uncategorized), `period=current\|previous`, and `page`; 50 rows per page |
| `/api/parsers` | `GET` | List all registered bank parser plugins |

---

## 🛡️ Privacy & Local Security

- **100% Offline**: LocalFinance does not make external network requests, send telemetry, or connect to third-party servers.
- **Local SQLite Storage**: Your data lives entirely in `local_finance.db` (or `~/.localfinance/local_finance.db`). Backups can be made simply by copying this single file.
- **Local App Lock**: An optional PIN/Password can be enabled in settings to restrict local access to the dashboard.

---

## 📄 License
MIT License. Free and open source for local personal finance intelligence.
