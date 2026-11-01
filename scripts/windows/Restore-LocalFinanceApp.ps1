#Requires -Version 5.1
[CmdletBinding()]
param([Parameter(Mandatory = $true)][string]$BackupDirectory)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
Import-Module (Join-Path $PSScriptRoot 'LocalFinance.Update.psm1') -Force
Assert-LocalFinanceStopped
$manifest = Get-Content -LiteralPath (Join-Path $BackupDirectory 'backup.json') -Raw | ConvertFrom-Json
$source = Join-Path $BackupDirectory 'LocalFinance-before-update.exe'
# Save the current app too, so rolling back never discards either executable.
$currentCopy = Join-Path $BackupDirectory ('LocalFinance-before-rollback-' + [guid]::NewGuid().ToString('N') + '.exe')
Copy-Item -LiteralPath $manifest.InstalledExecutable -Destination $currentCopy
Install-VerifiedExecutable -Source $source -Destination $manifest.InstalledExecutable -ExpectedHash $manifest.OriginalExecutableHash
Write-Host 'Previous executable restored. No database files were changed.'
Write-Host 'If the new app already migrated your database, verify compatibility before opening the older app.'
Write-Host 'The untouched DataBackup folder is available for a separate database recovery if needed.'
