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

$enkaImportInputDir = Join-Path $repoRoot (Join-Path 'input' 'enka_import')
$enkaImportExamplesDir = Join-Path $enkaImportInputDir 'examples'

function Ensure-EnkaImportConfigs {
  [CmdletBinding()]
  param()

  New-Item -ItemType Directory -Force -Path $enkaImportInputDir | Out-Null

  $dst = Join-Path $enkaImportInputDir 'config.yaml'
  $src = Join-Path $enkaImportExamplesDir 'config.example.yaml'

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
  Invoke-Step -Title 'Ensure enka_import config exists' -Action {
    Ensure-EnkaImportConfigs
  }

  Invoke-Step -Title 'Download Go modules' -Action {
    Invoke-InDir -Directory (Join-Path $repoRoot (Join-Path 'apps' 'enka_import')) -Action { & go mod download }
  }

  Invoke-Step -Title 'Build enka_import.exe' -Action {
    $appDir = Join-Path $repoRoot (Join-Path 'apps' 'enka_import')
    Invoke-InDir -Directory $appDir -Action { & go build -o 'enka_import.exe' './cmd/enka_import' }
  }

  Write-Host 'Done.'
}
finally {
  Pop-Location
}
