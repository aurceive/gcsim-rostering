[CmdletBinding()]
param(
  [ValidateSet('gcsim','wfpsim','custom','wfpsim-custom','all')]
  [string]$Engine = 'all',

  [string[]]$Targets = @('gcsim','repl','server')
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path

$binsRoot = Join-Path $repoRoot (Join-Path 'engines' 'bins')

function Get-EngineDirs {
  param([string]$engine)

  $map = @{
    'gcsim'  = @('gcsim')
    'wfpsim' = @('wfpsim')
    'custom' = @('custom')
    'wfpsim-custom' = @('wfpsim-custom')
    'all'    = @('gcsim','wfpsim','custom','wfpsim-custom')
  }

  return $map[$engine] | ForEach-Object { Join-Path $repoRoot (Join-Path 'engines' $_) }
}

function Build-Target {
  param(
    [string]$engineDir,
    [string]$target
  )

  $pkgPath = "./cmd/$target"
  $mainGo = Join-Path $engineDir (Join-Path 'cmd' (Join-Path $target 'main.go'))
  if (-not (Test-Path $mainGo)) {
    Write-Host "[skip] ${engineDir}: $pkgPath (no main.go)"
    return
  }

  $engineName = Split-Path $engineDir -Leaf
  $outDir = Join-Path $binsRoot $engineName
  New-Item -ItemType Directory -Force -Path $outDir | Out-Null

  $outExe = Join-Path $outDir ("$target.exe")

  Write-Host "[build] ${engineDir}: $pkgPath -> $outExe"
  & go -C $engineDir build -o $outExe $pkgPath
}

$engineDirs = Get-EngineDirs -engine $Engine
foreach ($engineDir in $engineDirs) {
  if (-not (Test-Path $engineDir)) {
    Write-Host "[skip] missing dir: $engineDir"
    continue
  }

  foreach ($t in $Targets) {
    Build-Target -engineDir $engineDir -target $t
  }
}

Write-Host 'Engine CLIs built.'
