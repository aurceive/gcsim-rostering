[CmdletBinding()]
param(
  [Parameter(Mandatory = $true)]
  [string]$TablePath
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
if (-not [System.IO.Path]::IsPathRooted($TablePath)) {
  $TablePath = Join-Path (Get-Location).Path $TablePath
}
$TablePath = [System.IO.Path]::GetFullPath($TablePath)
if (-not (Test-Path -LiteralPath $TablePath -PathType Leaf)) {
  throw "Roster table not found: $TablePath"
}

Add-Type -AssemblyName System.IO.Compression.FileSystem

function Read-ZipXml {
  param(
    [Parameter(Mandatory = $true)][System.IO.Compression.ZipArchive]$Archive,
    [Parameter(Mandatory = $true)][string]$EntryPath
  )

  $entry = $Archive.GetEntry($EntryPath)
  if ($null -eq $entry) {
    throw "Invalid XLSX: missing $EntryPath"
  }
  $stream = $entry.Open()
  try {
    $reader = [System.IO.StreamReader]::new($stream, [System.Text.Encoding]::UTF8, $true)
    try {
      return [System.Xml.Linq.XDocument]::Parse($reader.ReadToEnd())
    }
    finally {
      $reader.Dispose()
    }
  }
  finally {
    $stream.Dispose()
  }
}

function Get-XlsxCellText {
  param(
    [Parameter(Mandatory = $true)][System.Xml.Linq.XElement]$Cell,
    [Parameter(Mandatory = $true)][System.Xml.Linq.XNamespace]$Namespace,
    [Parameter(Mandatory = $true)][AllowEmptyCollection()][System.Collections.Generic.List[string]]$SharedStrings
  )

  $cellType = [string]$Cell.Attribute('t').Value
  if ($cellType -eq 's') {
    $value = $Cell.Element($Namespace + 'v')
    if ($null -eq $value) {
      return ''
    }
    $index = [int]$value.Value
    if ($index -lt 0 -or $index -ge $SharedStrings.Count) {
      throw "Invalid XLSX: shared string index out of range: $index"
    }
    return $SharedStrings[$index]
  }

  if ($cellType -eq 'inlineStr') {
    $inline = $Cell.Element($Namespace + 'is')
    if ($null -eq $inline) {
      return ''
    }
    return (($inline.Descendants($Namespace + 't') | ForEach-Object { [string]$_.Value }) -join '')
  }

  $value = $Cell.Element($Namespace + 'v')
  if ($null -eq $value) {
    return ''
  }
  return [string]$value.Value
}

$spreadsheetNamespace = [System.Xml.Linq.XNamespace]::Get('http://schemas.openxmlformats.org/spreadsheetml/2006/main')
$documentRelationshipNamespace = [System.Xml.Linq.XNamespace]::Get('http://schemas.openxmlformats.org/officeDocument/2006/relationships')
$packageRelationshipNamespace = [System.Xml.Linq.XNamespace]::Get('http://schemas.openxmlformats.org/package/2006/relationships')
$archive = [System.IO.Compression.ZipFile]::OpenRead($TablePath)
try {
  $workbook = Read-ZipXml -Archive $archive -EntryPath 'xl/workbook.xml'
  $workbookRelationships = Read-ZipXml -Archive $archive -EntryPath 'xl/_rels/workbook.xml.rels'
  $runConfigSheet = $workbook.Root.Element($spreadsheetNamespace + 'sheets').Elements($spreadsheetNamespace + 'sheet') |
    Where-Object { [string]$_.Attribute('name').Value -eq 'Run Config' } |
    Select-Object -First 1
  if ($null -eq $runConfigSheet) {
    throw "XLSX does not contain the 'Run Config' sheet: $TablePath"
  }

  $relationshipId = [string]$runConfigSheet.Attribute($documentRelationshipNamespace + 'id').Value
  $relationship = $workbookRelationships.Root.Elements($packageRelationshipNamespace + 'Relationship') |
    Where-Object { [string]$_.Attribute('Id').Value -eq $relationshipId } |
    Select-Object -First 1
  if ($null -eq $relationship) {
    throw "Invalid XLSX: cannot resolve the 'Run Config' sheet"
  }
  $worksheetPath = [string]$relationship.Attribute('Target').Value
  if ($worksheetPath.StartsWith('/')) {
    $worksheetPath = $worksheetPath.TrimStart('/')
  }
  elseif (-not $worksheetPath.StartsWith('xl/')) {
    $worksheetPath = "xl/$worksheetPath"
  }
  $worksheet = Read-ZipXml -Archive $archive -EntryPath $worksheetPath

  $sharedStrings = [System.Collections.Generic.List[string]]::new()
  if ($null -ne $archive.GetEntry('xl/sharedStrings.xml')) {
    $sharedStringsXml = Read-ZipXml -Archive $archive -EntryPath 'xl/sharedStrings.xml'
    foreach ($item in $sharedStringsXml.Root.Elements($spreadsheetNamespace + 'si')) {
      $parts = @($item.Descendants($spreadsheetNamespace + 't') | ForEach-Object { [string]$_.Value })
      $sharedStrings.Add(($parts -join ''))
    }
  }

  $configs = @{}
  foreach ($row in $worksheet.Root.Element($spreadsheetNamespace + 'sheetData').Elements($spreadsheetNamespace + 'row')) {
    $cells = @{}
    foreach ($cell in $row.Elements($spreadsheetNamespace + 'c')) {
      $reference = [string]$cell.Attribute('r').Value
      if ($reference -notmatch '^([A-Z]+)\d+$') {
        continue
      }
      $column = $Matches[1]
      $cells[$column] = Get-XlsxCellText -Cell $cell -Namespace $spreadsheetNamespace -SharedStrings $sharedStrings
    }

    $rowNumber = [int]$row.Attribute('r').Value
    if ($rowNumber -eq 1) {
      if ($cells['A'] -ne 'Config file' -or $cells['B'] -ne 'Setting' -or $cells['C'] -ne 'Content') {
        throw "Invalid 'Run Config' sheet header in $TablePath"
      }
      continue
    }

    $fileName = [System.IO.Path]::GetFileName([string]$cells['A'])
    if ([string]::IsNullOrWhiteSpace($fileName) -or -not [string]::IsNullOrWhiteSpace([string]$cells['B'])) {
      continue
    }
    $key = $fileName.ToLowerInvariant()
    switch ($key) {
      'config.example.txt' { $key = 'config.txt' }
      'roster_config.example.yaml' { $key = 'roster_config.yaml' }
    }
    if ($key -ne 'config.txt' -and $key -ne 'roster_config.yaml') {
      continue
    }
    if ($configs.ContainsKey($key)) {
      throw "Duplicate full contents for $fileName in the 'Run Config' sheet"
    }
    $configs[$key] = [string]$cells['C']
  }

  foreach ($requiredName in @('config.txt', 'roster_config.yaml')) {
    if (-not $configs.ContainsKey($requiredName)) {
      throw "The 'Run Config' sheet does not contain full contents for $requiredName"
    }
  }
}
finally {
  $archive.Dispose()
}

$inputDirectory = Join-Path $repoRoot (Join-Path 'input' 'weapon_roster')
New-Item -ItemType Directory -Force -Path $inputDirectory | Out-Null
$utf8WithoutBom = [System.Text.UTF8Encoding]::new($false)
foreach ($fileName in @('config.txt', 'roster_config.yaml')) {
  $destination = Join-Path $inputDirectory $fileName
  [System.IO.File]::WriteAllText($destination, $configs[$fileName], $utf8WithoutBom)
  Write-Host "Copied $fileName from $TablePath to $destination"
}
