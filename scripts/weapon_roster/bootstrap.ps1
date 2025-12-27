[CmdletBinding()]
param()

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

Assert-Command -Name 'git'
Assert-Command -Name 'go'

$submodulesDir = Join-Path (Join-Path $PSScriptRoot '..') 'submodules'
$updateSubmodulesScript = Join-Path $submodulesDir 'update-submodules.ps1'
$buildEngineClisScript = Join-Path $submodulesDir 'build-engine-clis.ps1'

Push-Location $repoRoot
try {
  Invoke-Step -Title 'Update submodules' -Action {
    & $updateSubmodulesScript
  }

  Invoke-Step -Title 'Download Go modules' -Action {
    & go -C (Join-Path $repoRoot 'apps' 'weapon_roster') mod download

    & go -C (Join-Path $repoRoot 'engines' 'gcsim') mod download
    & go -C (Join-Path $repoRoot 'engines' 'wfpsim') mod download
    & go -C (Join-Path $repoRoot 'engines' 'custom') mod download
  }

  Invoke-Step -Title 'Build roster.exe' -Action {
    $appDir = Join-Path $repoRoot (Join-Path 'apps' 'weapon_roster')
    & go -C $appDir build -o 'roster.exe' './cmd/weapon_roster'
  }

  Invoke-Step -Title 'Build engine CLIs' -Action {
    # weapon_roster only requires gcsim.exe; repl.exe/server.exe are not used.
    & $buildEngineClisScript -Engine 'all' -Targets @('gcsim')
  }

  Write-Host 'Done.'
}
finally {
  Pop-Location
}
