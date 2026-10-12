# LocalFinance desktop

The Wails v3 native host packages the existing React application, Gin API, Go
services, SQLite and optional finance MCP into one process. Wails is pinned to
`v3.0.0-beta.28` in this separate module. macOS/Linux desktop builds permit CGO;
the portable server module continues to build with `CGO_ENABLED=0`.

From the repository root, with Go, pnpm and platform prerequisites installed:

```sh
make desktop        # build and launch
make build-desktop  # build without launching
```

On macOS, the app is `dist/desktop/LocalFinance.app`. Build a local DMG with:

```sh
VERSION=v1.2.0 PACKAGE_DMG=1 sh cmd/desktop/package-macos.sh
```

`make desktop-prototype` remains an alias, but it now runs the real desktop host:
there is no automatic demo seeding and no temporary default ledger.

## Data and lifecycle

The default ledger is **the same** `~/.localfinance/local_finance.db` used by the
portable application. Existing transactions, password settings, MCP configuration
and imports remain available. To test without opening real data, invoke the built
executable with `-db /absolute/path/disposable.db`.

Both updated hosts acquire an OS lock before migrations or MCP startup. A second
host cannot open the same canonical path concurrently. Repeating a desktop launch
focuses the current window. Stop an older portable version before launching the
desktop app: older versions do not participate in this lock. File/directory
symlinks are resolved; hard links and separate copies are not a supported way to
share a database. Keep the database on a local filesystem.

Gin listens only on `127.0.0.1`. On the first desktop launch, the OS chooses a free
port; the port is saved in `<database>.desktop-port` and reused across launches.
This preserves the WebView origin and browser-stored preferences, including update
check opt-out. If another process occupies that port, startup reports an error
instead of silently changing origin. Deleting the port file allocates a new
origin and therefore resets browser-stored preferences. The finance MCP listener
uses its separately configured port and remains disabled by default.

Closing the application stops MCP, drains Gin requests, and closes SQLite.
Focus and View → Refresh finance data invalidate UI query caches after changes
from an MCP client. There is no background/tray mode. Exports use a native save
dialog, including browser-generated CSVs and PDF/backup links. External HTTP(S)
links open in the system browser. OS actions use the existing authenticated Gin
interface; finance operations retain their existing REST transport.

Desktop release checks retain the four-hour cache, explicit Check for Updates,
and background-check opt-out. The desktop updater selects full desktop installers
and never replaces the running executable. Quit and install the newer package;
the ledger remains outside the application bundle. The portable server retains
its existing executable update flow.

## Distribution and verification

See [desktop release operations](../../docs/desktop-release.md) for CI targets,
artifacts, optional signing credentials, release procedure and platform checks.

macOS requires 13+. Without an Apple Developer account, the bundle is ad-hoc
signed and not notarized. Windows needs WebView2 installed beforehand; the
application does not download a runtime. Linux packages target Ubuntu 24.04+
with GTK4 and WebKitGTK 6. Native builds use platform runners rather than attempting
CGO cross-compilation from the portable server workflow.

A native window may change startup time or memory use relative to a full browser.
Parsing, SQLite queries and React rendering use the same implementations; no
performance improvement has been measured or claimed. Profile a representative
ledger before changing transport or splitting logic into more packages.
