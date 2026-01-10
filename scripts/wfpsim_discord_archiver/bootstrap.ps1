[CmdletBinding()]
param()

# PSScriptAnalyzer sometimes reports a stale PSUseApprovedVerbs warning for this file.
# Existing repo scripts use the same pattern; suppress to keep Problems panel clean.
# PSScriptAnalyzerDisable PSUseApprovedVerbs

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

$inputDir = Join-Path $repoRoot (Join-Path 'input' 'wfpsim_discord_archiver')
$examplesDir = Join-Path $inputDir 'examples'

function Initialize-WfpsimDiscordArchiverConfigs {
  [CmdletBinding()]
  param()

  New-Item -ItemType Directory -Force -Path $inputDir | Out-Null

  $dst = Join-Path $inputDir 'config.yaml'
  $src = Join-Path $examplesDir 'config.yaml'

  if (-not (Test-Path -LiteralPath $dst)) {
    if (-not (Test-Path -LiteralPath $src)) {
      throw "Example config missing: $src"
    }
    Copy-Item -LiteralPath $src -Destination $dst
    Write-Host "Created $dst from examples."
  }
}

Push-Location $repoRoot
try {
  Invoke-Step -Title 'Ensure wfpsim_discord_archiver configs exist' -Action {
    Initialize-WfpsimDiscordArchiverConfigs
  }

  Invoke-Step -Title 'Download Go modules' -Action {
    Invoke-InDir -Directory (Join-Path $repoRoot (Join-Path 'apps' 'wfpsim_discord_archiver')) -Action { & go mod download }
  }

  Invoke-Step -Title 'Build wfpsim_discord_archiver.exe' -Action {
    $appDir = Join-Path $repoRoot (Join-Path 'apps' 'wfpsim_discord_archiver')
    Invoke-InDir -Directory $appDir -Action { & go build -o 'wfpsim_discord_archiver.exe' './cmd/wfpsim_discord_archiver' }
  }

  Write-Host 'Done.'
}
finally {
  Pop-Location
}
