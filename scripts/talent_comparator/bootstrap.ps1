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

$inputDir = Join-Path $repoRoot (Join-Path 'input' 'talent_comparator')
$examplesDir = Join-Path $inputDir 'examples'

function Ensure-TalentComparatorConfigs {
  [CmdletBinding()]
  param()

  New-Item -ItemType Directory -Force -Path $inputDir | Out-Null

  $pairs = @(
    @{ Name = 'config.txt'; Example = (Join-Path $examplesDir 'config.example.txt') },
    @{ Name = 'talent_config.yaml'; Example = (Join-Path $examplesDir 'talent_config.example.yaml') }
  )

  foreach ($p in $pairs) {
    $dst = Join-Path $inputDir $p.Name
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
  Invoke-Step -Title 'Ensure talent_comparator configs exist' -Action {
    Ensure-TalentComparatorConfigs
  }

  Invoke-Step -Title 'Download Go modules' -Action {
    Invoke-InDir -Directory (Join-Path $repoRoot (Join-Path 'apps' 'talent_comparator')) -Action { & go mod download }
  }

  Invoke-Step -Title 'Build talent_comparator.exe' -Action {
    $appDir = Join-Path $repoRoot (Join-Path 'apps' 'talent_comparator')
    Invoke-InDir -Directory $appDir -Action { & go build -o 'talent_comparator.exe' './cmd/talent_comparator' }
  }

  Write-Host 'Done.'
}
finally {
  Pop-Location
}
