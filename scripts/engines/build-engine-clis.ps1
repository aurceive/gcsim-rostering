[CmdletBinding()]
param(
  [string]$Engine = 'all',

  [string[]]$Targets = @('gcsim','repl','server')
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
. (Join-Path $PSScriptRoot 'engine-common.ps1')

$binsRoot = Join-Path $repoRoot (Join-Path 'engines' 'bins')

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
  if ($LASTEXITCODE -ne 0) {
    throw "go build failed for $engineName $pkgPath (exit $LASTEXITCODE)"
  }
}

$engineDirs = Get-EngineDirs -Engine $Engine -RepoRoot $repoRoot
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
