# Contributing to LocalFinance

Thank you for your interest in contributing to **LocalFinance**! LocalFinance is an offline-first, privacy-respecting financial dashboard tailored for the Indian banking and credit card ecosystem.

---

## 🛠️ Development Setup

### Prerequisites

* **Go 1.27.2+** (must satisfy the version in `go.mod`)
* **Node.js 20+**
* **pnpm** (v10+ recommended)

Frontend dependencies and scripts are declared in `frontend/package.json`. The patched transitive dependency override is declared in `frontend/pnpm-workspace.yaml`, which is the project configuration supported by pnpm 10 and 11. Keep its override and `pnpm-lock.yaml` synchronized when updating dependencies.

### Clone & Install

```bash
git clone https://github.com/usmslm102/local-finance.git
cd local-finance

# Install frontend dependencies
cd frontend
pnpm install
cd ..
```

---

## 💻 Development Workflows

### 1. Unified Run (`make dev` or `make serve`)

Compiles the frontend and boots up the Go server with auto-browser opening:

```bash
make dev
# Opens http://localhost:8080
```

### 2. Standalone Frontend Development

Run the Vite dev server with Hot Module Reload (HMR):

```bash
cd frontend
pnpm dev
# Opens Vite dev server on http://localhost:5173
```

---

## 🧪 Testing & Verification

Always make sure all tests and builds pass before submitting a Pull Request:

### Backend Tests

```bash
# Run all Go tests
go test -v ./...

# Run parser tests with sample statements
go test -v ./internal/parser/...
```

### Frontend Typechecking & Build

```bash
cd frontend
pnpm run build
```

---

## 🛡️ Privacy & Sensitive Data Guidelines

* **Never commit real statements or credentials**: LocalFinance is a privacy-first tool. Tests must use entirely fictional data. Ready-to-import fixtures belong in `samples/`; existing bank samples can be regenerated using `cmd/generate_samples`. Investment samples are checked-in fictional workbooks and do not require generator tooling. Tests may adapt fictional fixtures in memory to cover layout variations. Never derive fixtures from a user's personal statement.
* **No Telemetry / No Cloud APIs**: Do not add third-party tracking, analytics, or external cloud sync APIs without explicit discussion and architecture approval.
* **Keep Dependencies Minimal**: The Go backend utilizes pure-Go SQLite (`modernc.org/sqlite`) for zero-CGO cross-compilation. Avoid dependencies that require CGO.

---

## 🔀 Submitting Pull Requests

1. Fork the repository and create a feature branch (`git checkout -b feature/my-new-feature`).
2. Commit your changes with descriptive messages.
3. Ensure `go test ./...` and `pnpm --prefix frontend build` pass cleanly.
4. Push your branch to GitHub and open a Pull Request against `main`.
