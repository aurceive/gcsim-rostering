[CmdletBinding()]
param(
  [ValidateSet('gcsim','wfpsim','custom','all')]
  [string]$Engine = 'all',

  [string[]]$Targets = @('gcsim')
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path

function Assert-Command {
  param(
    [Parameter(Mandatory = $true)][string]$Name
  )

  if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
    throw "Required command not found in PATH: $Name"
  }
}

function Invoke-Step {
  param(
    [Parameter(Mandatory = $true)][string]$Title,
    [Parameter(Mandatory = $true)][scriptblock]$Action
  )

  Write-Host "==> $Title"
  & $Action
}

function Invoke-InDir {
  param(
    [Parameter(Mandatory = $true)][string]$Directory,
    [Parameter(Mandatory = $true)][scriptblock]$Action
  )

  Push-Location $Directory
  try {
    & $Action
  }
  finally {
    Pop-Location
  }
}

function Get-EngineDirs {
  param([string]$engine)

  $map = @{
    'gcsim'  = @('gcsim')
    'wfpsim' = @('wfpsim')
    'custom' = @('custom')
    'all'    = @('gcsim','wfpsim','custom')
  }

  return $map[$engine] | ForEach-Object { Join-Path $repoRoot (Join-Path 'engines' $_) }
}

Assert-Command -Name 'git'
Assert-Command -Name 'go'

$updateSubmodulesScript = Join-Path $PSScriptRoot 'update-submodules.ps1'
$buildEngineClisScript = Join-Path $PSScriptRoot 'build-engine-clis.ps1'

Push-Location $repoRoot
try {
  Invoke-Step -Title 'Update submodules' -Action {
    & $updateSubmodulesScript
  }

  Invoke-Step -Title 'Download engine Go modules' -Action {
    $engineDirs = Get-EngineDirs -engine $Engine
    foreach ($engineDir in $engineDirs) {
      if (-not (Test-Path -LiteralPath $engineDir)) {
        Write-Host "[skip] missing dir: $engineDir"
        continue
      }

      Invoke-InDir -Directory $engineDir -Action {
        & go mod download
      }
    }
  }

  Invoke-Step -Title 'Build engine CLIs' -Action {
    & $buildEngineClisScript -Engine $Engine -Targets $Targets
  }

  Write-Host 'Done.'
}
finally {
  Pop-Location
}
