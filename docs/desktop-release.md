# Desktop builds and releases

LocalFinance ships two hosts around the same Go application: the Wails desktop
window and the portable CGO-free browser/server executable. Finance logic and
MCP remain shared; neither distribution needs a separate backend installation.
Wails v3 is pinned to beta.28. Build automation is implemented, but Windows/Linux
runtime sign-off and the native interaction checklist below remain release gates.

## Artifacts

| Host | Platform | Release artifact | Runtime requirement |
| --- | --- | --- | --- |
| Desktop | macOS Intel + Apple Silicon | `LocalFinance-Desktop.dmg` containing Universal `LocalFinance.app` | macOS 13+ |
| Desktop | Windows x64 | `LocalFinance-Desktop-windows-amd64-setup.exe` | WebView2 |
| Desktop | Windows ARM64 | `LocalFinance-Desktop-windows-arm64-setup.exe` | WebView2 ARM64 |
| Desktop | Linux x64/ARM64 | `local-finance-desktop-linux-{arch}.deb` and `.tar.gz` | Ubuntu 24.04+, GTK4, WebKitGTK 6 |
| Portable server | macOS Universal | `local-finance-darwin-universal.tar.gz`, legacy `LocalFinance.dmg` | system browser |
| Portable server | Windows x64 | `LocalFinance.exe` | system browser |
| Portable server | Linux x64/ARM64 | `local-finance-linux-{arch}.tar.gz` | system browser |

`checksums.txt` covers all published files. Desktop asset names deliberately differ
from portable assets so the updater cannot install a server binary into the native
host. The legacy macOS DMG still opens the browser; choose **Desktop** for Wails.
Linux tarballs contain the executable and require the same system libraries as
the Debian package. The Linux native build baseline is Ubuntu 24.04 / glibc
2.39; older Ubuntu/Debian systems are not supported by these desktop artifacts.
The Debian package declares `libc6 (>= 2.39)` so installation fails clearly on
older systems rather than succeeding with an unusable executable. Other
distributions have not been validated; use the CGO-free portable server there.
Windows installers install per user and do not remove the
ledger on uninstall. They do not silently download WebView2; install that system
runtime in advance (use Microsoft's offline installer for disconnected machines).

## Pipelines

- `ci.yml`: portable Go tests/vet/vulnerability check, frontend lint/tests/build,
  dependency audit, and the reusable desktop build matrix.
- `desktop.yml`: macOS Universal, Windows x64/ARM64, and native Linux x64/ARM64
  runners; nested-module tests/vet, reachable-vulnerability scans, production builds, optional signing, and
  installer packaging. Production builds omit Wails developer-control MCP.
- `pages.yml`: deploys native download links after the latest public release has
  the matching desktop artifacts; publishing the release triggers deployment.
  Before that, the previous site remains live.
- `release.yml`: tag builds also run backend and frontend checks, build both
  distributions, generate checksums and create a **draft** GitHub Release. A manual
  workflow dispatch builds downloadable workflow artifacts without publishing.

To release:

1. Merge a validated change and choose the next version under the existing `v*`
   tag convention. Do not reuse a version that has already shipped.
2. Push the tag. Wait for every backend, frontend, desktop and portable build.
3. Download the actual artifacts from the draft release and verify checksums.
   Run the platform checklist below, especially both macOS architectures and
   both Windows architectures. Inspect installer metadata and release notes.
4. Document the current unsigned/notarization status in release notes. Publish
   the draft after verification. Drafts are not returned by the app's latest
   public-release check.

Draft release assets can be rebuilt; published releases are immutable in the
workflow. Do not retag a published version as a substitute for a new release. Builds stay local
until pushed; this implementation does not itself publish a version.

## Signing without an Apple account

No Apple account is required for the default path. macOS bundles receive ad-hoc
signatures and signature verification, but are **not notarized** and do not identify
a trusted developer. Downloaded apps may be blocked on first launch. After
trying to launch, approve a trusted download through System Settings → Privacy &
Security → Open Anyway, as described by [Apple](https://support.apple.com/en-us/102445).
On macOS 15+, use this Settings flow rather than relying on Control-click Open.
If that flow is unavailable for a trusted ad-hoc build, first verify the downloaded
DMG against `checksums.txt` and the installed app signature with
`codesign --verify --deep --strict /Applications/LocalFinance.app`. Then remove
only this app's quarantine attribute:

```sh
xattr -dr com.apple.quarantine /Applications/LocalFinance.app
```

This is a manual exception for that app; it does not notarize the build. A
"damaged" alert can also indicate corruption or an invalid signature, so do not
use the command to bypass failed verification. Do not disable Gatekeeper globally.
Managed Macs may prevent the override.
Windows installers are unsigned without a configured certificate and may show
publisher/SmartScreen warnings.

If Developer ID becomes available later, configure repository Actions secrets:

| Secret | Purpose |
| --- | --- |
| `APPLE_CERTIFICATE_P12` | Base64 Developer ID Application certificate plus private key |
| `APPLE_CERTIFICATE_PASSWORD` | Export password |
| `APPLE_SIGNING_IDENTITY` | Developer ID Application identity |
| `APPLE_ID`, `APPLE_TEAM_ID`, `APPLE_APP_PASSWORD` | Notary submission credentials |
| `WINDOWS_CERTIFICATE_PFX` | Optional Base64 Authenticode certificate plus private key |
| `WINDOWS_CERTIFICATE_PASSWORD` | Export password |

The workflow imports credentials into an ephemeral runner keychain/file, signs
the app before packaging and the DMG container before submission, notarizes and staples the DMG when Developer ID is
configured, and cleans up signing material even after failures. An explicitly
configured Developer ID path fails if notarization credentials are absent. The
Windows path signs both executable and installer and verifies the installer.
Never put these credentials in source control. Public timestamp/notarization
services are used only by release tooling, never by finance operations.

## Runtime checks before publication

Use disposable ledgers for automated and manual import/migration experiments.
Check both a fresh ledger and a copy of an existing ledger, never a live database.

- First launch, subsequent launch, native app quit/window close, and second launch
  focusing the existing instance. Starting the updated portable server against
  the owned ledger must fail; it must succeed once desktop exits.
- Password setup/login/lock, re-import idempotence, retained manual categories,
  notes and tags, statement upload, PDF preview, native exports and cancellation.
- Settings backup and restore; verify the restored data after restarting. UI
  preferences should survive a restart on the persisted loopback port.
- Enable opt-in finance MCP; authenticate a read-only client, refresh the UI after
  a change, and verify both listeners stop when the app quits.
- With background release checks opted out, restart and confirm no automatic
  check occurs. Explicit checks may contact GitHub. Desktop updates must offer
  desktop installers and reject executable replacement.
- Disconnect networking and exercise finance features. Test Windows with its
  runtime installed and check a clear missing-runtime failure. Install the Linux
  package on a clean supported distribution to verify shared library dependencies.
- Verify the downloaded macOS DMG and Gatekeeper first-launch behavior, rather
  than relying solely on a locally built app without download quarantine.

Wails v3 beta updates must be deliberate: update the pinned module, rerun the
matrix and native checks, and review framework release notes. Build success alone
does not establish native UI or platform compatibility.

## Current local verification

The portable Go suite, new ownership/update/auth checks, frontend lint/typecheck/
build and 64 frontend tests passed. Native macOS ARM64/Intel compilation, Universal packaging, bundle/plist, DMG integrity
and ad-hoc signature checks were exercised locally. The packaged app passed
a disposable-ledger startup/import/idempotence/shutdown/restart smoke test and
reopened the same data in the portable server. Windows x64/ARM64 cross-compilation
also passed. Workflow syntax passed
`actionlint`; the desktop vulnerability scan found no reachable vulnerabilities
(one advisory, [GO-2026-5932](https://pkg.go.dev/vuln/GO-2026-5932), concerns
`golang.org/x/crypto/openpgp`, which this application does not import).
Linux and Windows packaging execute in CI; they have not yet been
run on this branch's GitHub runners. The user subsequently verified the macOS native window, single-instance behavior,
native save/export dialogs and loopback listener interactively. Windows/Linux
installation and interaction checks remain required before publishing a release.

## Performance

A WebView host avoids opening a separate full browser, which may change memory
and startup overhead. Finance services, import parsers, SQLite and React rendering
are unchanged, and HTTP requests stay on loopback. Wails alone does not establish
an overall speedup. Compare cold/warm startup, resident memory, representative
imports and large-ledger navigation on the same hardware before claiming one.

References: [Wails platform builds](https://v3.wails.io/guides/build/cross-platform/)
and [v3 status](https://v3.wails.io/status/).

## Review follow-up

The updater now force-closes HTTP connections if graceful shutdown expires,
closes the database and releases ownership before spawning a replacement. Any
remaining shutdown error aborts the child launch and wakes the waiting main
goroutine. Export protection conservatively treats macOS paths as case-insensitive,
including absent sidecar files. Web links can open on the locked screen while
save actions remain authenticated; local host/origin checks still guard both.

Port persistence is intentional: changing the loopback origin loses browser-local
preferences, including a user's background release-check opt-out. We therefore
do not silently reassign an occupied saved port. The startup error now gives an
explicit recovery path: close the conflicting process, or remove only
`<database>.desktop-port` to allocate a new port. After choosing a new origin,
reconfigure Settings preferences before relying on their previous values.

The claim that `ubuntu-24.04-arm` requires public beta access or larger-runner
credits is outdated: [GitHub announced standard ARM runners for private
repositories in January 2026](https://github.blog/changelog/2026-01-29-arm64-standard-runners-are-now-available-in-private-repositories/).
Normal Actions quotas and repository policies still apply.

Apple recommends signing the DMG as well as its contents in its
[distribution packaging guide](https://developer.apple.com/documentation/xcode/packaging-mac-software-for-distribution).
The lack of Developer ID/notarization remains an explicitly disclosed distribution
constraint, not a guarantee that every downloaded app is corrupt or cannot be
approved through [Privacy & Security](https://support.apple.com/en-us/102445).
Wails beta status and actual Windows/Linux interaction checks remain release gates;
these code fixes do not replace that verification.
