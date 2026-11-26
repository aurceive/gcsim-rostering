[CmdletBinding()]
param(
  [Parameter(Mandatory = $true)]
  [string]$TablePath
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
if (-not [System.IO.Path]::IsPathRooted($TablePath)) {
  $TablePath = Join-Path (Get-Location).Path $TablePath
}
$TablePath = [System.IO.Path]::GetFullPath($TablePath)
if (-not (Test-Path -LiteralPath $TablePath -PathType Leaf)) {
  throw "Config table not found: $TablePath"
}
Push-Location (Join-Path $repoRoot 'apps\roster_lab')
try {
  go run .\cmd\roster_lab -configTable $TablePath -copyConfigs
  if ($LASTEXITCODE -ne 0) {
    throw 'copy configs failed'
  }
}
finally {
  Pop-Location
}
