[CmdletBinding()]
param(
  [switch]$Shallow,
  [switch]$MaxThin
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
Push-Location $repoRoot

try {
  if (-not (Test-Path (Join-Path $repoRoot '.git'))) {
    throw "Not a git repository: $repoRoot"
  }

  if (-not $PSBoundParameters.ContainsKey('Shallow')) { $Shallow = $true }

  # Accepts a pre-built [string[]] to avoid PowerShell binding dash-prefixed git flags as parameter names.
  # Stdout and stderr flow directly to the terminal so git's own error messages are always visible.
  function Invoke-Git {
    param([string[]]$GitArgs)
    & git @GitArgs
    if ($LASTEXITCODE -ne 0) {
      throw "git failed (exit $LASTEXITCODE): git $($GitArgs -join ' ')"
    }
  }

  function Invoke-GitOutput {
    param(
      [Parameter(Mandatory)][string]$WorkingDirectory,
      [string[]]$GitArgs
    )
    $output = & git -C $WorkingDirectory @GitArgs 2>$null
    if ($LASTEXITCODE -ne 0) { return $null }
    return ($output -join "`n").Trim()
  }

  function Get-SubmodulePaths {
    $lines = & git submodule status --recursive
    if ($LASTEXITCODE -ne 0) { throw 'git failed: git submodule status --recursive' }

    $paths = [System.Collections.Generic.List[string]]::new()
    foreach ($line in $lines) {
      $m = [regex]::Match($line, '^[ \+\-U]?([0-9a-f]{7,40})\s+(.+?)(\s+\(.*\))?$')
      if ($m.Success) { $paths.Add($m.Groups[2].Value) }
    }
    return ($paths | Sort-Object -Unique)
  }

  # Ensure submodule URLs/branches match .gitmodules
  Invoke-Git @('submodule', 'sync', '--recursive')

  # Init any submodules not yet cloned
  $initArgs = @('submodule', 'update', '--init', '--recursive')
  if ($Shallow) { $initArgs += @('--depth', '1', '--recommend-shallow') }
  Invoke-Git $initArgs

  foreach ($subPath in (Get-SubmodulePaths)) {
    $fullPath = Join-Path $repoRoot $subPath
    if (-not (Test-Path -LiteralPath $fullPath)) { continue }

    $branch = Invoke-GitOutput -WorkingDirectory $repoRoot -GitArgs @('config', '-f', (Join-Path $repoRoot '.gitmodules'), '--get', "submodule.$subPath.branch")
    if (-not $branch) {
      throw "Submodule '$subPath' has no branch configured in .gitmodules. Add 'branch = <name>' to fix."
    }

    # A shallow clone is typically single-branch, so we must fetch the target branch explicitly.
    # The '+' prefix allows force-updating the remote-tracking ref, which is required for
    # shallow clones (each --depth 1 fetch produces a new root commit with no common ancestor).
    $fetchArgs = @('-C', $fullPath, 'fetch', 'origin', '--prune', "+refs/heads/${branch}:refs/remotes/origin/${branch}")
    if ($Shallow) { $fetchArgs += @('--depth', '1') }
    Invoke-Git $fetchArgs

    Invoke-Git @('-C', $fullPath, 'checkout', '-q', '-B', $branch, "origin/$branch")
    Invoke-Git @('-C', $fullPath, 'reset', '-q', '--hard', "origin/$branch")

    if ($MaxThin) {
      # Best-effort cleanup: keep submodules as small as practical.
      & git -C $fullPath clean -ffd -q | Out-Null
      & git -C $fullPath reflog expire --expire=now --all | Out-Null
      & git -C $fullPath gc --prune=now --quiet | Out-Null
    }

    Write-Host "[ok] $subPath -> origin/$branch"
  }

  Write-Host 'Submodules are up to date.'
}
finally {
  Pop-Location
}
