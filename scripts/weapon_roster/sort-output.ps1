[CmdletBinding()]
param(
  [string]$OutputDirectory
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

if ([string]::IsNullOrWhiteSpace($OutputDirectory)) {
  $repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
  $OutputDirectory = Join-Path $repoRoot (Join-Path 'output' 'weapon_roster')
}
elseif (-not (Test-Path -LiteralPath $OutputDirectory -PathType Container)) {
  throw "Output directory not found: $OutputDirectory"
}
else {
  $OutputDirectory = (Resolve-Path -LiteralPath $OutputDirectory).Path
}

if (-not (Test-Path -LiteralPath $OutputDirectory -PathType Container)) {
  throw "Output directory not found: $OutputDirectory"
}

$archiveDirectory = Join-Path $OutputDirectory 'archive'
New-Item -ItemType Directory -Force -Path $archiveDirectory | Out-Null
$cutoffDate = [datetime]::Today.AddMonths(-1)
$rosterNamePattern = '^(?<date>\d{8})_weapon_roster_(?<character>[^_]+)_.+\.xlsx$'

function Get-RosterFileInfo {
  param(
    [Parameter(Mandatory = $true)][System.IO.FileInfo]$File
  )

  if ($File.Name -notmatch $rosterNamePattern) {
    return $null
  }

  $date = [datetime]::ParseExact(
    $Matches['date'],
    'yyyyMMdd',
    [Globalization.CultureInfo]::InvariantCulture
  )

  [pscustomobject]@{
    Date = $date
    Character = $Matches['character']
  }
}

$archivedCount = 0
$prunedCount = 0
$archiveFiles = @(Get-ChildItem -LiteralPath $archiveDirectory -File -Recurse)

foreach ($file in $archiveFiles) {
  $rosterInfo = Get-RosterFileInfo -File $file
  if ($null -eq $rosterInfo) {
    continue
  }

  $characterDirectory = Join-Path $archiveDirectory $rosterInfo.Character
  New-Item -ItemType Directory -Force -Path $characterDirectory | Out-Null
  $destination = Join-Path $characterDirectory $file.Name

  if ([string]::Equals(
      [System.IO.Path]::GetFullPath($file.FullName),
      [System.IO.Path]::GetFullPath($destination),
      [System.StringComparison]::OrdinalIgnoreCase
    )) {
    continue
  }

  if (Test-Path -LiteralPath $destination -PathType Leaf) {
    $sourceHash = (Get-FileHash -LiteralPath $file.FullName -Algorithm SHA256).Hash
    $destinationHash = (Get-FileHash -LiteralPath $destination -Algorithm SHA256).Hash
    if ($sourceHash -ne $destinationHash) {
      $name = [System.IO.Path]::GetFileNameWithoutExtension($file.Name)
      $extension = $file.Extension
      $destination = Join-Path $characterDirectory "${name}_archive_duplicate_${sourceHash}${extension}"
    }
  }

  if (Test-Path -LiteralPath $destination -PathType Leaf) {
    Remove-Item -LiteralPath $file.FullName
  }
  else {
    Move-Item -LiteralPath $file.FullName -Destination $destination
  }
  $archivedCount++
}

$rootRosterFiles = @(
  Get-ChildItem -LiteralPath $OutputDirectory -File |
    Where-Object { $_.Name -match $rosterNamePattern }
)

foreach ($file in $rootRosterFiles) {
  $rosterInfo = Get-RosterFileInfo -File $file
  $characterDirectory = Join-Path $archiveDirectory $rosterInfo.Character
  New-Item -ItemType Directory -Force -Path $characterDirectory | Out-Null
  $destination = Join-Path $characterDirectory $file.Name

  # A root copy is authoritative when a file with the same name is already archived.
  Copy-Item -LiteralPath $file.FullName -Destination $destination -Force
  $archivedCount++

  if ($rosterInfo.Date -lt $cutoffDate) {
    Remove-Item -LiteralPath $file.FullName
    $prunedCount++
  }
}

Write-Host "Archived roster copies: $archivedCount"
Write-Host "Removed old rosters from output root: $prunedCount"
Write-Host "Root retention cutoff: $($cutoffDate.ToString('yyyy-MM-dd'))"
