Set-StrictMode -Version Latest

function Invoke-CheckedCommand {
    param([string]$FilePath, [string[]]$Arguments)
    & $FilePath @Arguments
    if ($LASTEXITCODE -ne 0) { throw "Command failed (exit $LASTEXITCODE): $FilePath $($Arguments -join ' ')" }
}

function Assert-LocalFinanceStopped {
    $running = @(Get-Process | Where-Object {
        $_.ProcessName -in @('LocalFinance', 'local-finance-desktop', 'local-finance', 'local-finance-desktop-prototype')
    })
    if ($running.Count -gt 0) {
        throw "Close LocalFinance and its test/portable instances first. Running process IDs: $($running.Id -join ', '). Nothing will be force-stopped."
    }
}

function Get-LedgerFingerprint {
    param([string]$DataDirectory)
    $result = @{}
    foreach ($name in @('local_finance.db', 'local_finance.db-wal', 'local_finance.db-shm')) {
        $path = Join-Path $DataDirectory $name
        if (Test-Path -LiteralPath $path -PathType Leaf) {
            $result[$name] = (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash
        }
    }
    return $result
}

function Assert-LedgerUnchanged {
    param([string]$DataDirectory, [hashtable]$Expected)
    $actual = Get-LedgerFingerprint $DataDirectory
    if ($actual.Count -ne $Expected.Count) { throw 'Live database/sidecar files changed since backup. Run the update again after closing the app.' }
    foreach ($name in $Expected.Keys) {
        if (-not $actual.ContainsKey($name) -or $actual[$name] -ne $Expected[$name]) {
            throw "Live database file changed since backup: $name. Run the update again after closing the app."
        }
    }
}

function Backup-LocalFinanceData {
    param([string]$DataDirectory, [string]$Destination)
    if (-not (Test-Path -LiteralPath (Join-Path $DataDirectory 'local_finance.db') -PathType Leaf)) {
        throw "Existing database not found in $DataDirectory. Supply -DataDirectory for your installation."
    }
    if (Test-Path -LiteralPath $Destination) { throw "Backup destination already exists: $Destination" }
    Assert-LocalFinanceStopped
    $handles = @()
    try {
        # Reject an active SQLite writer, including an older or renamed server.
        # Retain read-sharing handles throughout the copy to exclude new writers.
        foreach ($name in @('local_finance.db', 'local_finance.db-wal', 'local_finance.db-shm')) {
            $path = Join-Path $DataDirectory $name
            if (Test-Path -LiteralPath $path -PathType Leaf) {
                $handles += [IO.File]::Open($path, [IO.FileMode]::Open, [IO.FileAccess]::Read, [IO.FileShare]::Read)
            }
        }
        $fingerprint = Get-LedgerFingerprint $DataDirectory
        Copy-Item -LiteralPath $DataDirectory -Destination $Destination -Recurse -ErrorAction Stop
        Assert-LedgerUnchanged -DataDirectory $Destination -Expected $fingerprint
        Assert-LedgerUnchanged -DataDirectory $DataDirectory -Expected $fingerprint
        return $fingerprint
    }
    finally { foreach ($handle in $handles) { $handle.Dispose() } }
}

function Install-VerifiedExecutable {
    param([string]$Source, [string]$Destination, [string]$ExpectedHash)
    if ((Get-FileHash -LiteralPath $Source -Algorithm SHA256).Hash -ne $ExpectedHash) {
        throw 'Executable changed after verification. Installation cancelled.'
    }
    if (-not (Test-Path -LiteralPath $Destination -PathType Leaf)) { throw "Installed executable missing: $Destination" }
    $pending = Join-Path (Split-Path -Parent $Destination) ('.LocalFinance-update-' + [guid]::NewGuid().ToString('N') + '.exe')
    $previous = Join-Path (Split-Path -Parent $Destination) ('.LocalFinance-previous-' + [guid]::NewGuid().ToString('N') + '.exe')
    try {
        Copy-Item -LiteralPath $Source -Destination $pending -ErrorAction Stop
        if ((Get-FileHash -LiteralPath $pending -Algorithm SHA256).Hash -ne $ExpectedHash) { throw 'Staged executable checksum mismatch.' }
        # Same-volume atomic replacement; a failed replacement retains the old app.
        [IO.File]::Replace($pending, $Destination, $previous)
        if ((Get-FileHash -LiteralPath $Destination -Algorithm SHA256).Hash -ne $ExpectedHash) { throw "Installed executable checksum mismatch. Previous app retained at $previous" }
        Remove-Item -LiteralPath $previous -ErrorAction Stop
    }
    finally {
        if (Test-Path -LiteralPath $pending) { Remove-Item -LiteralPath $pending -ErrorAction SilentlyContinue }
    }
}

Export-ModuleMember -Function Invoke-CheckedCommand, Assert-LocalFinanceStopped, Get-LedgerFingerprint, Assert-LedgerUnchanged, Backup-LocalFinanceData, Install-VerifiedExecutable
