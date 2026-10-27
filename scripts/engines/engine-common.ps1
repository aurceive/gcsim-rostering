Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Get-ConfiguredEngineNames {
  param(
    [Parameter(Mandatory = $true)][string]$RepoRoot
  )

  $gitmodulesPath = Join-Path $RepoRoot '.gitmodules'
  if (-not (Test-Path -LiteralPath $gitmodulesPath)) {
    throw "Submodule configuration not found: $gitmodulesPath"
  }

  $configLines = & git -C $RepoRoot config --file $gitmodulesPath --get-regexp '^submodule\..+\.path$'
  if ($LASTEXITCODE -ne 0) {
    throw "Could not read submodule paths from $gitmodulesPath"
  }

  $engineNames = @(
    foreach ($line in $configLines) {
      if ($line -match '^submodule\..+\.path\s+engines/([^/]+)$') {
        $Matches[1]
      }
    }
  ) | Sort-Object -Unique

  if ($engineNames.Count -eq 0) {
    throw "No engine submodules configured under engines/ in $gitmodulesPath"
  }

  return $engineNames
}

function Get-EngineDirs {
  param(
    [Parameter(Mandatory = $true)][string]$Engine,
    [Parameter(Mandatory = $true)][string]$RepoRoot
  )

  $availableEngines = @(Get-ConfiguredEngineNames -RepoRoot $RepoRoot)
  if ($Engine -eq 'all') {
    $selectedEngines = $availableEngines
  }
  elseif ($availableEngines -contains $Engine) {
    $selectedEngines = @($Engine)
  }
  else {
    throw "Unknown engine '$Engine'. Configured engines: $($availableEngines -join ', '); use 'all' to select every engine."
  }

  return @(
    $selectedEngines | ForEach-Object {
      Join-Path $RepoRoot (Join-Path 'engines' $_)
    }
  )
}
