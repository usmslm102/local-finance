#Requires -Version 5.1
[CmdletBinding()]
param(
    [string]$ProjectDirectory,
    [string]$DataDirectory = (Join-Path $env:USERPROFILE '.localfinance'),
    [string]$InstalledExecutable = (Join-Path $env:LOCALAPPDATA 'Programs\LocalFinance\LocalFinance.exe'),
    [string]$BackupRoot = (Join-Path $env:USERPROFILE 'Documents\LocalFinance-backups'),
    [string]$GoExecutable,
    [string]$PnpmExecutable,
    [switch]$CheckOnly
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$scriptDirectory = $PSScriptRoot
if (-not $scriptDirectory -and $PSCommandPath) { $scriptDirectory = Split-Path -Parent $PSCommandPath }
if (-not $scriptDirectory -and $MyInvocation.MyCommand.Path) { $scriptDirectory = Split-Path -Parent $MyInvocation.MyCommand.Path }
if (-not $scriptDirectory) { throw 'Run the saved script with powershell -File, rather than pasting its contents into the console.' }
if (-not $ProjectDirectory) { $ProjectDirectory = [IO.Path]::GetFullPath((Join-Path $scriptDirectory '..\..')) }
Import-Module (Join-Path $scriptDirectory 'LocalFinance.Update.psm1') -Force
if ([Environment]::OSVersion.Platform -ne [PlatformID]::Win32NT) { throw 'This script builds the Windows desktop app.' }
$ProjectDirectory = (Resolve-Path -LiteralPath $ProjectDirectory).Path
if (-not (Test-Path -LiteralPath (Join-Path $ProjectDirectory 'cmd\desktop\go.mod'))) { throw 'ProjectDirectory is not the LocalFinance repository.' }
if ($CheckOnly) {
    Write-Output "Project: $ProjectDirectory"
    Write-Output "Data directory: $DataDirectory"
    Write-Output "Installed executable: $InstalledExecutable"
    Write-Output 'Startup check passed. No files were changed and no app was launched.'
    return
}
$DataDirectory = (Resolve-Path -LiteralPath $DataDirectory).Path
$InstalledExecutable = (Resolve-Path -LiteralPath $InstalledExecutable).Path
$BackupRoot = [IO.Path]::GetFullPath($BackupRoot)
$dataPrefix = $DataDirectory.TrimEnd('\', '/') + [IO.Path]::DirectorySeparatorChar
if ($BackupRoot.Equals($DataDirectory, [StringComparison]::OrdinalIgnoreCase) -or $BackupRoot.StartsWith($dataPrefix, [StringComparison]::OrdinalIgnoreCase)) {
    throw 'BackupRoot must be outside the live data directory.'
}
$oldPath = $env:PATH
$oldCGO = $env:CGO_ENABLED
$oldGOOS = $env:GOOS
$oldGOARCH = $env:GOARCH
$resourcePath = $null
$resourceOriginal = $null
$resourceGenerated = $false

try {
    Assert-LocalFinanceStopped
    # Prefer installed tools, then the tools already used by this Codex project.
    $runtime = Join-Path $env:USERPROFILE '.cache\codex-runtimes\codex-primary-runtime\dependencies'
    if (-not (Get-Command node -ErrorAction SilentlyContinue)) { $env:PATH = "$(Join-Path $runtime 'node\bin');$env:PATH" }
    if (-not $GoExecutable) {
        $command = Get-Command go -ErrorAction SilentlyContinue
        if ($command) { $GoExecutable = $command.Source }
        else { $GoExecutable = Join-Path $env:LOCALAPPDATA 'Temp\local-finance-toolchain\go\bin\go.exe' }
    }
    if (-not $PnpmExecutable) {
        $command = Get-Command pnpm.cmd -ErrorAction SilentlyContinue
        if (-not $command) { $command = Get-Command pnpm -ErrorAction SilentlyContinue }
        if ($command) { $PnpmExecutable = $command.Source }
        else { $PnpmExecutable = Join-Path $runtime 'bin\fallback\pnpm.cmd' }
    }
    if (-not (Test-Path -LiteralPath $GoExecutable -PathType Leaf)) { throw 'Go is missing. Install the Go version required by go.mod or supply -GoExecutable.' }
    if (-not (Test-Path -LiteralPath $PnpmExecutable -PathType Leaf)) { throw 'pnpm is missing. Install pnpm or supply -PnpmExecutable.' }
    if (-not (Get-Command node -ErrorAction SilentlyContinue)) { throw 'Node.js is missing. Install Node.js before building.' }
    $env:PATH = "$(Split-Path -Parent $GoExecutable);$env:PATH"
    $env:CGO_ENABLED = '0'
    $env:GOOS = 'windows'
    $env:GOARCH = ''
    $arch = (& $GoExecutable env GOHOSTARCH | Out-String).Trim()
    if ($LASTEXITCODE -ne 0 -or $arch -notin @('amd64', 'arm64')) { throw 'Cannot determine a supported Windows build architecture.' }
    $env:GOARCH = $arch

    $session = Join-Path $BackupRoot ((Get-Date -Format 'yyyyMMdd-HHmmss') + '-' + [guid]::NewGuid().ToString('N').Substring(0,8))
    New-Item -ItemType Directory -Path $session -ErrorAction Stop | Out-Null
    $snapshot = Join-Path $session 'DataBackup'
    Write-Host 'Backing up the closed app and its complete data folder...'
    $fingerprint = Backup-LocalFinanceData -DataDirectory $DataDirectory -Destination $snapshot
    $oldApp = Join-Path $session 'LocalFinance-before-update.exe'
    Copy-Item -LiteralPath $InstalledExecutable -Destination $oldApp
    $oldAppHash = (Get-FileHash -LiteralPath $oldApp -Algorithm SHA256).Hash
    $manifest = [ordered]@{
        InstalledExecutable = $InstalledExecutable
        DataDirectory = $DataDirectory
        OriginalExecutableHash = $oldAppHash
        DatabaseHashes = $fingerprint
        CreatedAt = (Get-Date).ToString('o')
    }
    $manifest | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath (Join-Path $session 'backup.json') -Encoding UTF8
    Write-Host "Backup: $session"

    Push-Location $ProjectDirectory
    try {
        Push-Location 'frontend'
        try {
            Invoke-CheckedCommand $PnpmExecutable @('install', '--frozen-lockfile')
            Invoke-CheckedCommand $PnpmExecutable @('build')
        }
        finally { Pop-Location }
        # Focus on the affected bank/PDF behavior; the pre-existing Splitwise
        # CRLF fixture failure is separate from this update.
        Invoke-CheckedCommand $GoExecutable @('test', './internal/parser/...', './internal/service', '-run', 'Test(ICICI|IdentityH|BankHistory)')
        Invoke-CheckedCommand $GoExecutable @('vet', './...')
        New-Item -ItemType Directory -Force -Path 'dist\desktop' | Out-Null
        $buildDirectory = Join-Path $session 'Build'
        New-Item -ItemType Directory -Path $buildDirectory | Out-Null
        $builtApp = Join-Path $buildDirectory 'LocalFinance.exe'
        Push-Location 'cmd\desktop'
        try {
            $resourcePath = Join-Path (Get-Location).Path "rsrc_windows_$arch.syso"
            if (Test-Path -LiteralPath $resourcePath) { $resourceOriginal = [IO.File]::ReadAllBytes($resourcePath) }
            $resourceGenerated = $true
            Invoke-CheckedCommand $GoExecutable @('run', './tools/resources', '-arch', $arch, '-version', 'dev-icicistatements')
            Invoke-CheckedCommand $GoExecutable @('build', '-trimpath', '-tags', 'production', '-ldflags', '-s -w -H windowsgui -X local-finance/internal/updater.CurrentVersion=dev-icicistatements', '-o', $builtApp, '.')
        }
        finally { Pop-Location }
    }
    finally { Pop-Location }
    $builtHash = (Get-FileHash -LiteralPath $builtApp -Algorithm SHA256).Hash
    $testDirectory = Join-Path $session 'TestData'
    Copy-Item -LiteralPath $snapshot -Destination $testDirectory -Recurse
    $testDatabase = Join-Path $testDirectory 'local_finance.db'
    Assert-LocalFinanceStopped

    Write-Host ''
    Write-Host 'MANUAL VERIFICATION - the next window uses ONLY the database copy.' -ForegroundColor Cyan
    Write-Host '1. Check existing accounts, balances, categories, notes and tags.'
    Write-Host '2. Check ICICI savings preview (110 transactions) and credit card preview (16 transactions), then import.'
    Write-Host '   Select the correct card account; the portal PDF does not print a card number.'
    Write-Host '3. Reimport and confirm no new duplicate transactions are created.'
    Write-Host '4. Close the test app, then type UPDATE below. Any other answer cancels installation.'
    Write-Host "Test database: $testDatabase"
    $testProcess = Start-Process -FilePath $builtApp -ArgumentList @('-db', ('"' + $testDatabase + '"')) -WindowStyle Normal -PassThru
    $answer = Read-Host 'After verification and closing the test app, enter UPDATE'
    if ($answer -cne 'UPDATE') { Write-Host 'Installation cancelled. Backup and test copy were retained.'; return }
    $testProcess.Refresh()
    if (-not $testProcess.HasExited) { throw 'The test app is still running. Close it and rerun this script.' }
    $testProcess.WaitForExit()
    if ($testProcess.ExitCode -ne 0) { throw "Test app exited with code $($testProcess.ExitCode). Installation cancelled." }
    Assert-LocalFinanceStopped
    Assert-LedgerUnchanged -DataDirectory $DataDirectory -Expected $fingerprint
    if ((Get-FileHash -LiteralPath $InstalledExecutable -Algorithm SHA256).Hash -ne $oldAppHash) { throw 'The installed app changed during verification. Installation cancelled.' }
    Install-VerifiedExecutable -Source $builtApp -Destination $InstalledExecutable -ExpectedHash $builtHash
    Assert-LedgerUnchanged -DataDirectory $DataDirectory -Expected $fingerprint
    Write-Host ''
    Write-Host 'Update complete. Your live database and its sidecars are byte-for-byte unchanged.' -ForegroundColor Green
    Write-Host 'Launch LocalFinance from your usual shortcut, then import your statements there.'
    Write-Host 'The test imports were made only in the copy. The app may apply existing schema migrations when you launch it.'
    Write-Host "Keep this backup for rollback: $session"
}
finally {
    if ($resourceGenerated -and $resourcePath) {
        if ($null -ne $resourceOriginal) { [IO.File]::WriteAllBytes($resourcePath, $resourceOriginal) }
        elseif (Test-Path -LiteralPath $resourcePath) { Remove-Item -LiteralPath $resourcePath }
    }
    $env:PATH = $oldPath
    $env:CGO_ENABLED = $oldCGO
    $env:GOOS = $oldGOOS
    $env:GOARCH = $oldGOARCH
}
