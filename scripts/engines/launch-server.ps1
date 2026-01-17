[CmdletBinding()]
param(
  [Parameter(Mandatory = $true)]
  [ValidateSet('gcsim','wfpsim','custom','wfpsim-custom')]
  [string]$Engine,

  [string]$ListenHost = '127.0.0.1',
  [int]$Port = 54321,
  [int]$Workers = 10,
  [int]$TimeoutSec = 300,
  [string]$ShareKey = '',

  [switch]$Update,

  # Optional helpers
  [switch]$UpdateSubmodules,
  [switch]$DownloadModules,
  [switch]$NoBuild
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path

function Assert-Command {
  param([Parameter(Mandatory = $true)][string]$Name)
  if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
    throw "Required command not found in PATH: $Name"
  }
}

Assert-Command -Name 'go'

$engineDir = Join-Path $repoRoot (Join-Path 'engines' $Engine)
if (-not (Test-Path -LiteralPath $engineDir)) {
  throw "Engine directory not found: $engineDir"
}

$updateSubmodulesScript = Join-Path $PSScriptRoot 'update-submodules.ps1'
$buildEngineClisScript = Join-Path $PSScriptRoot 'build-engine-clis.ps1'

if ($UpdateSubmodules) {
  Assert-Command -Name 'git'
  Write-Host "==> Update submodules"
  & $updateSubmodulesScript
}

if ($DownloadModules) {
  Write-Host "==> Download Go modules (${Engine})"
  & go -C $engineDir mod download
}

if (-not $NoBuild) {
  Write-Host "==> Build server.exe (${Engine})"
  & $buildEngineClisScript -Engine $Engine -Targets @('server')
}

$exePath = Join-Path $repoRoot (Join-Path 'engines' (Join-Path 'bins' (Join-Path $Engine 'server.exe')))
if (-not (Test-Path -LiteralPath $exePath)) {
  throw "Server binary not found: $exePath (try without -NoBuild)"
}

$serverArgs = @(
  '-host', $ListenHost,
  '-port', "$Port",
  '-workers', "$Workers",
  '-timeout', "$TimeoutSec"
)

if ($ShareKey -ne '') {
  $serverArgs += @('-sharekey', $ShareKey)
}

if ($Update) {
  $serverArgs += @('-update')
}

Write-Host "==> Run: $exePath $($serverArgs -join ' ')"
& $exePath @serverArgs
