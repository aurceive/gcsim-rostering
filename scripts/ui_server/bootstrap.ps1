[CmdletBinding()]
param(
  [string]$Engine = 'gcsim',
  [switch]$SkipEngines
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
$kqmRoot = Join-Path $repoRoot 'engines\kqmsim'
$appRoot = Join-Path $repoRoot 'apps\ui_server'

foreach ($name in @('git', 'go', 'node', 'corepack', 'tar')) {
  if (-not (Get-Command $name -ErrorAction SilentlyContinue)) {
    throw "Required command not found in PATH: $name"
  }
}

function Assert-NativeSuccess {
  param([string]$Step)
  if ($LASTEXITCODE -ne 0) {
    throw "$Step failed with exit code $LASTEXITCODE"
  }
}

if (-not (Test-Path -LiteralPath (Join-Path $kqmRoot 'ui\pnpm-lock.yaml'))) {
  throw 'KQM submodule is not initialized. Run git submodule update --init engines/kqmsim.'
}
$status = & git -C $kqmRoot status --porcelain --untracked-files=no
Assert-NativeSuccess 'Checking KQM submodule'
if ($status) {
  throw 'KQM submodule has uncommitted tracked changes. Commit them or build from a clean checkout; the UI is staged from the checked-out commit.'
}

if (-not $SkipEngines) {
  & (Join-Path $repoRoot 'scripts\engines\build-engine-clis.ps1') -Engine $Engine -Targets @('server')
}

$stage = Join-Path $repoRoot ("work\ui_server\build-" + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Force -Path $stage | Out-Null
$archive = Join-Path $stage 'kqm-ui.tar'
Write-Host "==> Stage KQM UI from the checked-out commit: $stage"
& git -C $kqmRoot archive --format=tar -o $archive HEAD ui
Assert-NativeSuccess 'Archiving KQM UI'
& tar -xf $archive -C $stage
Assert-NativeSuccess 'Extracting KQM UI'
$uiRoot = Join-Path $stage 'ui'
$source = Join-Path $uiRoot 'packages\kqmsim\src'
Copy-Item -Path (Join-Path $appRoot 'web\src\*.tsx') -Destination $source -Force
Copy-Item -LiteralPath (Join-Path $appRoot 'web\components\Commit.tsx') -Destination (Join-Path $uiRoot 'packages\components\src\Metadata\Commit.tsx') -Force
Copy-Item -LiteralPath (Join-Path $appRoot 'web\index.html') -Destination (Join-Path $uiRoot 'packages\kqmsim\index.html') -Force
$executorFile = Join-Path $uiRoot 'packages\executors\src\ServerExecutor.ts'
$executorSource = [IO.File]::ReadAllText($executorFile)
$oldId = 'this.id = "id" + new Date().getTime();'
if (-not $executorSource.Contains($oldId)) {
  throw 'KQM ServerExecutor ID initialization changed; update the ui_server build adapter.'
}
$executorSource = $executorSource.Replace($oldId, 'this.id = "id" + crypto.randomUUID();')
[IO.File]::WriteAllText($executorFile, $executorSource, [Text.UTF8Encoding]::new($false))
$providerFile = Join-Path $uiRoot 'packages\components\src\Editor\ExecutorProvider.tsx'
$providerSource = [IO.File]::ReadAllText($providerFile)
$oldHandler = '}, noop);'
if ([regex]::Matches($providerSource, [regex]::Escape($oldHandler)).Count -ne 1) {
  throw 'KQM ExecutorProvider validation handler changed; update the ui_server build adapter.'
}
$providerSource = $providerSource.Replace($oldHandler, '}, (err: unknown) => setRunResult((prev) => ({ ...prev, config, error: asError(err) })));')
$pattern = '(\(validated\.errors && validated\.errors\.length > 0\) \|\|\s*executor\.running\(\)\s*\) \{)'
if ([regex]::Matches($providerSource, $pattern).Count -ne 1) {
  throw 'KQM ExecutorProvider validation result handling changed; update the ui_server build adapter.'
}
$providerSource = [regex]::Replace($providerSource, $pattern, '$1' + "`n`t`t`t`t`tif (validated.errors?.length) setRunResult((prev) => ({ ...prev, config, error: validated.errors?.join(`"\n`") ?? null }));")
[IO.File]::WriteAllText($providerFile, $providerSource, [Text.UTF8Encoding]::new($false))
Copy-Item -LiteralPath (Join-Path $appRoot 'web\tests\ProviderValidation.test.tsx') -Destination (Join-Path $uiRoot 'packages\components\src\Editor\ProviderValidation.test.tsx') -Force

Write-Host '==> Build KQM browser WebAssembly'
$oldGoos = $env:GOOS
$oldGoarch = $env:GOARCH
try {
  $env:GOOS = 'js'
  $env:GOARCH = 'wasm'
  & go -C $kqmRoot build -trimpath -o (Join-Path $uiRoot 'packages\kqmsim\public\main.wasm') './cmd/wasm'
  Assert-NativeSuccess 'Building KQM WebAssembly'
}
finally {
  $env:GOOS = $oldGoos
  $env:GOARCH = $oldGoarch
}
$goRoot = (& go env GOROOT).Trim()
Assert-NativeSuccess 'Finding Go runtime'
Copy-Item -LiteralPath (Join-Path $goRoot 'lib\wasm\wasm_exec.js') -Destination (Join-Path $uiRoot 'packages\kqmsim\public\wasm_exec.js') -Force

Push-Location $uiRoot
try {
  Write-Host '==> Install KQM UI dependencies'
  & corepack pnpm install --frozen-lockfile
  Assert-NativeSuccess 'Installing UI dependencies'
  Write-Host '==> Type-check ui_server UI'
  & corepack pnpm --filter '@gcsim/kqmsim' typecheck
  Assert-NativeSuccess 'Type-checking UI'
  Write-Host '==> Build ui_server UI'
  & corepack pnpm --filter '@gcsim/kqmsim' build
  Assert-NativeSuccess 'Building UI'
}
finally {
  Pop-Location
}

$dist = Join-Path $appRoot 'web\dist'
New-Item -ItemType Directory -Force -Path $dist | Out-Null
Copy-Item -Path (Join-Path $uiRoot 'packages\kqmsim\dist\*') -Destination $dist -Recurse -Force
Write-Host '==> Build ui_server.exe'
& go -C $appRoot build -o (Join-Path $appRoot 'ui_server.exe') './cmd/ui_server'
Assert-NativeSuccess 'Building ui_server'
Write-Host "Done. Run apps\ui_server\ui_server.exe. Staged UI remains in $stage."
