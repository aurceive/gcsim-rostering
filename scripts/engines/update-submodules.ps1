[CmdletBinding()]
param(
  [switch]$Init,
  [switch]$Remote,
  [switch]$Shallow,
  [switch]$Recursive
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
Push-Location $repoRoot

try {
  $isGitRepo = Test-Path (Join-Path $repoRoot '.git')
  if (-not $isGitRepo) {
    throw "Not a git repository: $repoRoot"
  }

  if (-not $PSBoundParameters.ContainsKey('Init')) { $Init = $true }
  if (-not $PSBoundParameters.ContainsKey('Remote')) { $Remote = $true }
  if (-not $PSBoundParameters.ContainsKey('Shallow')) { $Shallow = $true }
  if (-not $PSBoundParameters.ContainsKey('Recursive')) { $Recursive = $true }

  function Invoke-Git {
    param([Parameter(ValueFromRemainingArguments = $true)][string[]]$Args)
    & git @Args
    if ($LASTEXITCODE -ne 0) {
      throw "git failed: git $($Args -join ' ')"
    }
  }

  $commonArgs = @()
  if ($Recursive) { $commonArgs += '--recursive' }

  # Ensure submodule URLs/branches match .gitmodules
  Invoke-Git submodule sync @commonArgs

  if ($Init) {
    $initArgs = @('submodule','update','--init') + $commonArgs
    if ($Shallow) {
      $initArgs += @('--depth','1','--recommend-shallow')
    }
    Invoke-Git @initArgs
  }

  if ($Remote) {
    function Invoke-GitOutput {
      param(
        [Parameter(Mandatory = $true)][string]$WorkingDirectory,
        [Parameter(ValueFromRemainingArguments = $true)][string[]]$Args
      )

      $output = & git -C $WorkingDirectory @Args 2>$null
      if ($LASTEXITCODE -ne 0) {
        return $null
      }
      return ($output -join "`n").Trim()
    }

    function Get-SubmodulePaths {
      $lines = & git submodule status --recursive
      if ($LASTEXITCODE -ne 0) {
        throw 'git failed: git submodule status --recursive'
      }

      $paths = New-Object System.Collections.Generic.List[string]
      foreach ($line in $lines) {
        $m = [regex]::Match($line, '^[ \+\-U]?([0-9a-f]{7,40})\s+(.+?)(\s+\(.*\))?$')
        if ($m.Success) {
          $paths.Add($m.Groups[2].Value)
        }
      }

      return ($paths | Sort-Object -Unique)
    }

    $submodulePaths = Get-SubmodulePaths
    foreach ($subPath in $submodulePaths) {
      $fullPath = Join-Path $repoRoot $subPath
      if (-not (Test-Path -LiteralPath $fullPath)) {
        continue
      }

      # Read configured branch from root .gitmodules (same source as `git submodule foreach` variables).
      $want = Invoke-GitOutput -WorkingDirectory $repoRoot config -f (Join-Path $repoRoot '.gitmodules') --get "submodule.$subPath.branch"

      # Keep it shallow by default; safe for local "latest" usage.
      $fetchArgs = @('-C', $fullPath, 'fetch', '-q', 'origin', '--prune')
      if ($Shallow) { $fetchArgs += @('--depth', '1') }
      & git @fetchArgs | Out-Null
      # Ignore fetch errors (mirrors previous behavior).

      $ref = $null
      if ($want) {
        & git -C $fullPath show-ref --verify --quiet "refs/remotes/origin/$want"
        if ($LASTEXITCODE -eq 0) {
          $ref = $want
        }
      }

      if (-not $ref) {
        $head = Invoke-GitOutput -WorkingDirectory $fullPath symbolic-ref -q 'refs/remotes/origin/HEAD'
        if ($head) {
          $ref = $head -replace '^refs/remotes/origin/', ''
        }
      }

      if (-not $ref) {
        Write-Host "[warn] ${subPath}: cannot determine origin/HEAD; skipping"
        continue
      }

      & git -C $fullPath checkout -q -B $ref --track "origin/$ref" 2>$null
      if ($LASTEXITCODE -ne 0) {
        & git -C $fullPath checkout -q -B $ref | Out-Null
      }

      & git -C $fullPath reset -q --hard "origin/$ref" | Out-Null
      # Ignore reset errors (mirrors previous behavior).

      Write-Host "[ok] $subPath -> origin/$ref"
    }
  }

  Write-Host 'Submodules are up to date.'
}
finally {
  Pop-Location
}
