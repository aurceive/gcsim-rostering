[CmdletBinding()]
param(
  [string]$OutputDirectory,
  [int]$RetentionMonths = 1
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

if ([string]::IsNullOrWhiteSpace($OutputDirectory)) {
  $repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
  $OutputDirectory = Join-Path $repoRoot 'output\roster_lab'
}
if (-not (Test-Path -LiteralPath $OutputDirectory -PathType Container)) {
  throw "Output directory not found: $OutputDirectory"
}
$OutputDirectory = (Resolve-Path -LiteralPath $OutputDirectory).Path
$archiveRoot = Join-Path $OutputDirectory 'archive'
New-Item -ItemType Directory -Force -Path $archiveRoot | Out-Null
$cutoff = [datetime]::Today.AddMonths(-$RetentionMonths)
$archived = 0
$removed = 0

foreach ($file in Get-ChildItem -LiteralPath $OutputDirectory -File -Filter '*.ods') {
  if ($file.Name -notmatch '^(?<date>\d{8})_(?:rl|roster_lab)_(?<project>.+)\.ods$') {
    continue
  }
  $date = [datetime]::ParseExact($Matches['date'], 'yyyyMMdd', [Globalization.CultureInfo]::InvariantCulture)
  $directory = Join-Path $archiveRoot $Matches['project']
  New-Item -ItemType Directory -Force -Path $directory | Out-Null
  Copy-Item -LiteralPath $file.FullName -Destination (Join-Path $directory $file.Name) -Force
  $archived++
  if ($date -lt $cutoff) {
    Remove-Item -LiteralPath $file.FullName
    $removed++
  }
}

Write-Host "Archived roster_lab copies: $archived"
Write-Host "Removed old roster_lab results: $removed"
