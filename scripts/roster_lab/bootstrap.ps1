[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
$inputDir = Join-Path $repoRoot 'input\roster_lab'
$examplesDir = Join-Path $inputDir 'examples'

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
  throw 'Required command not found in PATH: go'
}

New-Item -ItemType Directory -Force -Path $inputDir | Out-Null

foreach ($pair in @(
  @{ Source = 'roster_lab.example.yaml'; Target = 'roster_lab.yaml' },
  @{ Source = 'sim_config.example.dart'; Target = 'sim_config.dart' }
)) {
  $source = Join-Path $examplesDir $pair.Source
  $target = Join-Path $inputDir $pair.Target
  if (-not (Test-Path -LiteralPath $target)) {
    Copy-Item -LiteralPath $source -Destination $target
    Write-Host "Created $target from examples."
  }
}

Push-Location (Join-Path $repoRoot 'apps\roster_lab')
try {
  go mod download
  if ($LASTEXITCODE -ne 0) { throw 'go mod download failed' }
  go build -o roster_lab.exe .\cmd\roster_lab
  if ($LASTEXITCODE -ne 0) { throw 'go build failed' }
}
finally {
  Pop-Location
}

Write-Host 'Done.'
