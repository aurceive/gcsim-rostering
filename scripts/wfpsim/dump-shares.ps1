param(
  [Parameter(Mandatory = $false)]
  [string] $UrlFile = (Join-Path $PSScriptRoot "..\..\input\wfpsim\share_urls.txt"),

  [Parameter(Mandatory = $false)]
  [string[]] $Urls,

  [Parameter(Mandatory = $false)]
  [string] $OutFile = (Join-Path $PSScriptRoot "..\..\work\wfpsim_share_dump.json")
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Read-UrlsFromFile {
  param([Parameter(Mandatory = $true)][string] $Path)

  if (-not (Test-Path -LiteralPath $Path)) {
    throw "URL file not found: $Path"
  }

  return Get-Content -LiteralPath $Path | ForEach-Object { $_.Trim() } | Where-Object {
    $_ -ne '' -and -not $_.StartsWith('#') -and -not $_.StartsWith('//')
  }
}

function Get-RawHttp {
  param(
    [Parameter(Mandatory = $true)]
    [string] $Url
  )

  $handler = [System.Net.Http.HttpClientHandler]::new()
  $handler.AutomaticDecompression = [System.Net.DecompressionMethods]::None

  $client = [System.Net.Http.HttpClient]::new($handler)
  try {
    $request = [System.Net.Http.HttpRequestMessage]::new([System.Net.Http.HttpMethod]::Get, $Url)
    $request.Headers.TryAddWithoutValidation('User-Agent', 'gcsim-rostering dump-shares script') | Out-Null
    $request.Headers.TryAddWithoutValidation('Accept', '*/*') | Out-Null

    $response = $client.SendAsync($request).GetAwaiter().GetResult()
    $statusCode = [int]$response.StatusCode

    $headers = @{}
    foreach ($h in $response.Headers) {
      $headers[$h.Key] = ($h.Value -join ', ')
    }
    foreach ($h in $response.Content.Headers) {
      $headers[$h.Key] = ($h.Value -join ', ')
    }

    $bytes = $response.Content.ReadAsByteArrayAsync().GetAwaiter().GetResult()

    $enc = @()
    try {
      foreach ($e in $response.Content.Headers.ContentEncoding) {
        $enc += [string]$e
      }
    } catch {
      $enc = @()
    }

    return [pscustomobject]@{
      StatusCode = $statusCode
      Headers    = $headers
      Encoding   = $enc
      Bytes      = $bytes
    }
  }
  finally {
    $client.Dispose()
    $handler.Dispose()
  }
}

function Try-Decompress {
  param(
    [Parameter(Mandatory = $true)]
    [byte[]] $Bytes,

    [string[]] $Encoding = @()
  )

  $encLower = @($Encoding | ForEach-Object { $_.ToLowerInvariant() })

  if ($encLower -contains 'gzip') {
    try {
      $msIn = [System.IO.MemoryStream]::new($Bytes)
      $gzip = [System.IO.Compression.GZipStream]::new($msIn, [System.IO.Compression.CompressionMode]::Decompress)
      $msOut = [System.IO.MemoryStream]::new()
      $gzip.CopyTo($msOut)
      $gzip.Dispose(); $msIn.Dispose()
      $out = $msOut.ToArray(); $msOut.Dispose()
      return [pscustomobject]@{ Bytes = $out; Used = 'gzip' }
    } catch {
      # fall through
    }
  }

  if ($encLower -contains 'deflate') {
    try {
      $msIn = [System.IO.MemoryStream]::new($Bytes)
      $def = [System.IO.Compression.DeflateStream]::new($msIn, [System.IO.Compression.CompressionMode]::Decompress)
      $msOut = [System.IO.MemoryStream]::new()
      $def.CopyTo($msOut)
      $def.Dispose(); $msIn.Dispose()
      $out = $msOut.ToArray(); $msOut.Dispose()
      return [pscustomobject]@{ Bytes = $out; Used = 'deflate' }
    } catch {
      # fall through
    }
  }

  return [pscustomobject]@{ Bytes = $Bytes; Used = 'none' }
}

function Extract-Key {
  param([Parameter(Mandatory = $true)][string] $Url)

  $m = [regex]::Match($Url, "\/sh\/(?<key>[0-9a-fA-F-]{36})")
  if ($m.Success) { return $m.Groups['key'].Value }

  # allow api/share/<key> as input too
  $m2 = [regex]::Match($Url, "\/api\/share\/(?<key>[^\/?#]+)")
  if ($m2.Success) { return $m2.Groups['key'].Value }

  throw "Could not extract share key from URL: $Url"
}

$results = @()

$effectiveUrls = @()
if ($Urls -and $Urls.Count -gt 0) {
  $effectiveUrls += $Urls
}
if ($UrlFile -and $UrlFile.Trim() -ne '') {
  $effectiveUrls += Read-UrlsFromFile -Path $UrlFile
}

$effectiveUrls = @(
  $effectiveUrls |
    ForEach-Object { $_.Trim() } |
    Where-Object { $_ -ne '' } |
    Select-Object -Unique
)

if (-not $effectiveUrls -or $effectiveUrls.Count -eq 0) {
  throw "No URLs provided. Add URLs to $UrlFile or pass -Urls."
}

foreach ($url in $effectiveUrls) {
  $key = Extract-Key -Url $url
  $apiUrl = "https://wfpsim.com/api/share/$key"

  $fetchedAt = (Get-Date).ToString('o')
  $item = [ordered]@{
    sourceUrl = $url
    key       = $key
    apiUrl    = $apiUrl
    fetchedAt = $fetchedAt
  }

  try {
    $raw = Get-RawHttp -Url $apiUrl
    $item.statusCode = $raw.StatusCode
    $item.headers = $raw.Headers
    $item.contentEncoding = $raw.Encoding

    $dec = Try-Decompress -Bytes $raw.Bytes -Encoding $raw.Encoding
    $item.decompressedUsing = $dec.Used

    # try interpret as UTF-8 text
    $text = [System.Text.Encoding]::UTF8.GetString($dec.Bytes)
    $item.bodyText = $text

    try {
      $parsed = $text | ConvertFrom-Json -Depth 100
      $item.parsedJson = $parsed
      $item.parseOk = $true
    } catch {
      $item.parseOk = $false
      $item.parseError = $_.Exception.Message
    }
  } catch {
    $item.error = $_.Exception.Message
  }

  $results += [pscustomobject]$item
}

$dir = Split-Path -Parent $OutFile
if (-not (Test-Path $dir)) {
  New-Item -ItemType Directory -Path $dir | Out-Null
}

$results | ConvertTo-Json -Depth 100 | Set-Content -Path $OutFile -Encoding UTF8
Write-Host "Wrote $($results.Count) items to $OutFile"