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
    $fetchDepthArg = ''
    if ($Shallow) { $fetchDepthArg = '--depth 1' }

    # Update each submodule to its configured branch.
    # If the configured branch doesn't exist on origin (common with forks/tags), fall back to origin/HEAD.
    $foreachScript = @'
set -e

want="$(git config -f "$toplevel/.gitmodules" submodule.$name.branch || true)"

# Keep it shallow by default; safe for local "latest" usage.
git fetch -q origin --prune __FETCH_DEPTH__ || true

if [ -n "$want" ] && git show-ref --verify --quiet "refs/remotes/origin/$want"; then
  ref="$want"
else
  ref="$(git symbolic-ref -q refs/remotes/origin/HEAD 2>/dev/null | sed 's#^refs/remotes/origin/##')"
fi

if [ -z "$ref" ]; then
  echo "[warn] $name: cannot determine origin/HEAD; skipping"
  exit 0
fi

git checkout -q -B "$ref" --track "origin/$ref" 2>/dev/null || git checkout -q -B "$ref"
git reset -q --hard "origin/$ref" || true

echo "[ok] $name -> origin/$ref"
'@

    $foreachScript = $foreachScript.Replace('__FETCH_DEPTH__', $fetchDepthArg)
    Invoke-Git submodule foreach @commonArgs $foreachScript
  }

  Write-Host 'Submodules are up to date.'
}
finally {
  Pop-Location
}
