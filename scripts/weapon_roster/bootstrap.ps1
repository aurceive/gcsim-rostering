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

Assert-Command -Name 'git'
Assert-Command -Name 'go'

$submodulesDir = Join-Path (Join-Path $PSScriptRoot '..') 'submodules'
$updateSubmodulesScript = Join-Path $submodulesDir 'update-submodules.ps1'
$buildEngineClisScript = Join-Path $submodulesDir 'build-engine-clis.ps1'

$weaponRosterInputDir = Join-Path $repoRoot (Join-Path 'input' 'weapon_roster')
$weaponRosterExamplesDir = Join-Path $weaponRosterInputDir 'examples'

function Ensure-WeaponRosterConfigs {
  [CmdletBinding()]
  param()

  New-Item -ItemType Directory -Force -Path $weaponRosterInputDir | Out-Null

  $pairs = @(
    @{ Name = 'config.txt'; Example = (Join-Path $weaponRosterExamplesDir 'config.exemple.txt') },
    @{ Name = 'roster_config.yaml'; Example = (Join-Path $weaponRosterExamplesDir 'roster_config.exemple.yaml') }
  )

  foreach ($p in $pairs) {
    $dst = Join-Path $weaponRosterInputDir $p.Name
    $src = $p.Example

    if (-not (Test-Path -LiteralPath $dst)) {
      if (-not (Test-Path -LiteralPath $src)) {
        throw "Example config missing: $src"
      }
      Copy-Item -LiteralPath $src -Destination $dst
      Write-Host "Created $dst from examples."
    }
  }
}

Push-Location $repoRoot
try {
  Invoke-Step -Title 'Update submodules' -Action {
    & $updateSubmodulesScript
  }

  Invoke-Step -Title 'Ensure weapon_roster configs exist' -Action {
    Ensure-WeaponRosterConfigs
  }

  Invoke-Step -Title 'Download Go modules' -Action {
    Invoke-InDir -Directory (Join-Path $repoRoot (Join-Path 'apps' 'weapon_roster')) -Action { & go mod download }

    Invoke-InDir -Directory (Join-Path $repoRoot (Join-Path 'engines' 'gcsim')) -Action { & go mod download }
    Invoke-InDir -Directory (Join-Path $repoRoot (Join-Path 'engines' 'wfpsim')) -Action { & go mod download }
    Invoke-InDir -Directory (Join-Path $repoRoot (Join-Path 'engines' 'custom')) -Action { & go mod download }
  }

  Invoke-Step -Title 'Build roster.exe' -Action {
    $appDir = Join-Path $repoRoot (Join-Path 'apps' 'weapon_roster')
    Invoke-InDir -Directory $appDir -Action { & go build -o 'roster.exe' './cmd/weapon_roster' }
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
