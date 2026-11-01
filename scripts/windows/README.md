# Windows local update

Close all LocalFinance desktop and portable instances first. From the repository
root, run this in PowerShell (no administrator privileges are needed):

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\windows\Update-LocalFinance.ps1
```

The execution policy override applies only to this command, not your machine's
saved settings. Review the scripts before running them.

To check script startup and the default paths without changing files or launching
the app, append `-CheckOnly` to the update command. The project path is resolved
from the saved script's location, so the command also works from another directory
when you supply the full script path.

The script uses the current working tree, including uncommitted parser fixes.
It finds Go/Node.js/pnpm on PATH, falling back to the bundled tools used in this
chat. Go must meet the version in `go.mod`. Dependency downloads need internet
access; the running finance application remains offline. WebView2 must already
be installed, as it is for the existing desktop app.

It creates a unique folder under `Documents\LocalFinance-backups` containing an
untouched `DataBackup`, the previous executable, the new build, and a separate
`TestData` database copy. Keep these private: they contain your financial data.
It builds the frontend and Windows desktop host, runs focused ICICI/bank-history/
PDF tests and `go vet`, then launches the new app against `TestData`.

**There is one manual verification checkpoint:** inspect your existing data,
test the two ICICI imports, confirm reimport creates no duplicate transactions,
close the test app, and type `UPDATE` in the terminal. Any other answer cancels
installation. A test that crashes, a failed build/check, a changed live database,
or an app still running blocks installation. Cancellation retains the backup and
test app; close that app before rerunning.

The script replaces only the installed executable after verifying its checksum.
It does not launch the installed app or open the live database. Database and WAL/
SHM checksums must remain unchanged. Launch through your usual shortcut after
success; startup may run existing migrations. The test imports are only in the
copy, so import the statements into the real app afterward.

Default data: `%USERPROFILE%\.localfinance`. Default executable:
`%LOCALAPPDATA%\Programs\LocalFinance\LocalFinance.exe`. Override nonstandard
locations using `-DataDirectory`, `-InstalledExecutable`, or `-BackupRoot`.
Explicit tool paths use `-GoExecutable` and `-PnpmExecutable`.

The backup/replacement safeguards can be checked without building or launching
the app, using fictional files only:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\windows\tests\Update.Tests.ps1
```

To restore only the previous app after closing LocalFinance:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\windows\Restore-LocalFinanceApp.ps1 -BackupDirectory 'C:\path\printed\by\the\update'
```

Rollback saves the current executable too and never restores/deletes financial
data. If an older executable cannot read a migrated database, keep the app closed
and recover the entire `DataBackup` separately, preserving the newer data folder
first. Do not combine an old database with newer WAL/SHM files. Do not run
`make clean` to update this app: that target deletes project-local databases.
