# Run with: powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\windows\tests\Update.Tests.ps1
# These checks use fictional bytes only, never a real installation or database.
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$scriptsDirectory = Split-Path -Parent $PSScriptRoot
$module = Import-Module (Join-Path $scriptsDirectory 'LocalFinance.Update.psm1') -Force -PassThru
# No GUI is started by these tests. Isolate process detection from the user's
# real app so safety helpers can be exercised exclusively against fixtures.
& $module { function script:Assert-LocalFinanceStopped {} }

function Assert-True([bool]$Condition, [string]$Message) {
    if (-not $Condition) { throw $Message }
}
function Assert-Throws([scriptblock]$Action, [string]$Message) {
    $thrown = $false
    try { & $Action } catch { $thrown = $true }
    Assert-True $thrown $Message
}

$project = Split-Path -Parent (Split-Path -Parent $scriptsDirectory)
$testRoot = Join-Path $project ('.cache\update-script-tests-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $testRoot | Out-Null
try {
    # Parameter defaults must not depend on script automatic variables being
    # populated. This reproduced the original empty-PSScriptRoot exception.
    $tokens = $null
    $parseErrors = $null
    $entrypoint = Join-Path $scriptsDirectory 'Update-LocalFinance.ps1'
    $ast = [Management.Automation.Language.Parser]::ParseFile($entrypoint, [ref]$tokens, [ref]$parseErrors)
    & ([scriptblock]::Create($ast.ParamBlock.Extent.Text))
    Push-Location $testRoot
    try {
        $startup = @(& $entrypoint -CheckOnly)
        Assert-True ($startup -contains "Project: $project") 'Default project path did not resolve from the saved script.'
    }
    finally { Pop-Location }
    # Syntax-check both entrypoints and the module without executing them.
    foreach ($script in Get-ChildItem -LiteralPath $scriptsDirectory -File | Where-Object { $_.Extension -in @('.ps1', '.psm1') }) {
        $tokens = $null
        $parseErrors = $null
        [void][Management.Automation.Language.Parser]::ParseFile($script.FullName, [ref]$tokens, [ref]$parseErrors)
        Assert-True ($parseErrors.Count -eq 0) "Syntax error in $($script.Name): $parseErrors"
    }
    $source = Join-Path $testRoot 'Data'
    New-Item -ItemType Directory -Path $source | Out-Null
    [IO.File]::WriteAllText((Join-Path $source 'local_finance.db'), 'fictional database')
    [IO.File]::WriteAllText((Join-Path $source 'local_finance.db-wal'), 'fictional WAL')
    [IO.File]::WriteAllText((Join-Path $source 'local_finance.db.desktop-port'), '54321')
    $snapshot = Join-Path $testRoot 'Backup'
    $hashes = Backup-LocalFinanceData -DataDirectory $source -Destination $snapshot
    Assert-True ($hashes.Count -eq 2) 'Database and WAL were not both fingerprinted.'
    Assert-True ((Get-Content -LiteralPath (Join-Path $snapshot 'local_finance.db.desktop-port')) -eq '54321') 'Desktop port was not preserved.'
    Assert-LedgerUnchanged -DataDirectory $source -Expected $hashes
    Assert-Throws { Backup-LocalFinanceData -DataDirectory $source -Destination $snapshot } 'Existing backup was overwritten.'

    $writer = [IO.File]::Open((Join-Path $source 'local_finance.db'), [IO.FileMode]::Open, [IO.FileAccess]::ReadWrite, [IO.FileShare]::ReadWrite)
    try {
        Assert-Throws { Backup-LocalFinanceData -DataDirectory $source -Destination (Join-Path $testRoot 'BusyBackup') } 'Backup accepted a database with an active writer.'
    }
    finally { $writer.Dispose() }
    [IO.File]::WriteAllText((Join-Path $source 'local_finance.db'), 'changed fictional database')
    Assert-Throws { Assert-LedgerUnchanged -DataDirectory $source -Expected $hashes } 'Changed live database went undetected.'
    [IO.File]::WriteAllText((Join-Path $source 'local_finance.db-shm'), 'new sidecar')
    Assert-Throws { Assert-LedgerUnchanged -DataDirectory $source -Expected $hashes } 'New sidecar went undetected.'
    Assert-LedgerUnchanged -DataDirectory $snapshot -Expected $hashes

    $old = Join-Path $testRoot 'Installed.exe'
    $new = Join-Path $testRoot 'Built.exe'
    [IO.File]::WriteAllText($old, 'old executable fixture')
    [IO.File]::WriteAllText($new, 'new executable fixture')
    $newHash = (Get-FileHash -LiteralPath $new).Hash
    Assert-Throws { Install-VerifiedExecutable -Source $new -Destination $old -ExpectedHash 'incorrect' } 'Unverified build was installed.'
    Assert-True ((Get-Content -LiteralPath $old) -eq 'old executable fixture') 'Rejected update altered the installed app.'
    Install-VerifiedExecutable -Source $new -Destination $old -ExpectedHash $newHash
    Assert-True ((Get-FileHash -LiteralPath $old).Hash -eq $newHash) 'Atomic replacement did not install the verified file.'

    $manifest = @{ InstalledExecutable = $old; OriginalExecutableHash = $newHash }
    $manifest | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $testRoot 'backup.json') -Encoding UTF8
    Copy-Item -LiteralPath $new -Destination (Join-Path $testRoot 'LocalFinance-before-update.exe')
    [IO.File]::WriteAllText($old, 'later executable fixture')
    & (Join-Path $scriptsDirectory 'Restore-LocalFinanceApp.ps1') -BackupDirectory $testRoot
    Assert-True ((Get-FileHash -LiteralPath $old).Hash -eq $newHash) 'Rollback failed.'
    Assert-LedgerUnchanged -DataDirectory $snapshot -Expected $hashes
    Write-Host 'PASS: syntax, verified backups, active-writer rejection, database-change detection, verified replacement, and executable rollback.' -ForegroundColor Green
    Write-Host "Fictional test fixtures: $testRoot"
}
finally { Remove-Module $module.Name -ErrorAction SilentlyContinue }
