[CmdletBinding()]
param(
  [switch]$Init,
  [switch]$Remote,
  [switch]$Shallow,
  [switch]$Recursive,
  [switch]$MaxThin
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
      # NOTE: A `--depth 1` clone is typically a *single-branch* clone, which means
      # `origin/<branch>` may not exist for non-default branches unless we fetch it explicitly.
      if ($want) {
        $fetchArgs = @(
          '-C', $fullPath,
          'fetch', '-q', 'origin', '--prune',
          "refs/heads/${want}:refs/remotes/origin/${want}"
        )
        if ($Shallow) { $fetchArgs += @('--depth', '1') }
        & git @fetchArgs | Out-Null
      }
      else {
        $fetchArgs = @('-C', $fullPath, 'fetch', '-q', 'origin', '--prune')
        if ($Shallow) { $fetchArgs += @('--depth', '1') }
        & git @fetchArgs | Out-Null
      }
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

      # Some git/Pwsh combinations can error out when `--track` can't be set up
      # (e.g. "starting point 'origin/<ref>' is not a branch"). Tracking is not
      # required for our workflow since we hard reset to origin/<ref> below.
      $oldNativeEap = $null
      $hasNativeEap = Get-Variable -Name 'PSNativeCommandUseErrorActionPreference' -Scope Global -ErrorAction SilentlyContinue
      if ($hasNativeEap) {
        $oldNativeEap = $global:PSNativeCommandUseErrorActionPreference
        $global:PSNativeCommandUseErrorActionPreference = $false
      }

      try {
        try {
          & git -C $fullPath checkout -q -B $ref "origin/$ref" 2>$null | Out-Null
        }
        catch {
          # Fall through to the next strategy.
        }

        if ($LASTEXITCODE -ne 0) {
          try {
            & git -C $fullPath checkout -q -B $ref | Out-Null
          }
          catch {
            # If we can't even create/switch the branch, let it surface.
            throw
          }
        }
      }
      finally {
        if ($hasNativeEap) {
          $global:PSNativeCommandUseErrorActionPreference = $oldNativeEap
        }
      }

      & git -C $fullPath reset -q --hard "origin/$ref" | Out-Null
      # Ignore reset errors (mirrors previous behavior).

      if ($MaxThin) {
        # Keep submodules as small as practical:
        # - remove untracked files
        # - expire reflogs so old commits become prunable
        # - prune unreachable objects immediately
        & git -C $fullPath clean -ffd -q | Out-Null
        & git -C $fullPath reflog expire --expire=now --all | Out-Null
        & git -C $fullPath gc --prune=now --quiet | Out-Null
      }

      Write-Host "[ok] $subPath -> origin/$ref"
    }
  }

  Write-Host 'Submodules are up to date.'
}
finally {
  Pop-Location
}
