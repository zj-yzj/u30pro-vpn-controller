# Checks local files only. It never opens USB/ADB or invokes a flash tool.
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)][string]$BackupDirectory,
    [Parameter(Mandatory = $true)][string]$SecondReadDirectory,
    [Parameter(Mandatory = $true)][string]$SizesFile
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$required = @('init_boot_a', 'init_boot_b', 'trustos_a', 'trustos_b', 'misc')

function Get-RegularItem([string]$ItemPath, [bool]$Directory) {
    $item = Get-Item -LiteralPath $ItemPath -Force
    if (($item.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) {
        throw 'REPARSE_POINT_NOT_ACCEPTED'
    }
    if ([bool]$item.PSIsContainer -ne $Directory) { throw 'UNEXPECTED_ITEM_TYPE' }
    return $item
}

function Read-PartitionTable([string]$Directory) {
    $item = Get-RegularItem (Join-Path $Directory 'partition.xml') $false
    if ($item.Length -eq 0 -or $item.Length -gt 1048576) { throw 'INVALID_TABLE_LENGTH' }
    $settings = [Xml.XmlReaderSettings]::new()
    $settings.DtdProcessing = [Xml.DtdProcessing]::Prohibit
    $settings.XmlResolver = $null
    $settings.MaxCharactersInDocument = 1048576
    $reader = [Xml.XmlReader]::Create($item.FullName, $settings)
    try {
        $document = [Xml.XmlDocument]::new()
        $document.XmlResolver = $null
        $document.Load($reader)
    } finally { $reader.Dispose() }
    $names = [Collections.Generic.HashSet[string]]::new([StringComparer]::Ordinal)
    foreach ($partition in $document.SelectNodes('//Partition')) {
        $name = $partition.GetAttribute('id')
        if (-not $name -or -not $names.Add($name)) { throw 'INVALID_TABLE_PARTITION' }
        if ($name -match '(?i)w_force') { throw 'TEMPORARY_FORCE_PARTITION_FOUND' }
    }
    foreach ($name in $required) {
        if (-not $names.Contains($name)) { throw ('REQUIRED_PARTITION_MISSING: ' + $name) }
    }
    # XML confirms names only. Its MiB sizes are not used as exact byte lengths.
    return (Get-FileHash -LiteralPath $item.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
}

$first = (Get-RegularItem $BackupDirectory $true).FullName
$second = (Get-RegularItem $SecondReadDirectory $true).FullName
if ([string]::Equals($first.TrimEnd('\', '/'), $second.TrimEnd('\', '/'), [StringComparison]::OrdinalIgnoreCase)) {
    throw 'TWO_DIFFERENT_READ_DIRECTORIES_REQUIRED'
}
$csv = Get-RegularItem $SizesFile $false
if ($csv.Length -eq 0 -or $csv.Length -gt 65536) { throw 'INVALID_SIZES_FILE_LENGTH' }
$rows = @(Import-Csv -LiteralPath $csv.FullName)
if ($rows.Count -ne $required.Count) { throw 'EXACTLY_FIVE_SIZE_ROWS_REQUIRED' }
$headers = @($rows[0].PSObject.Properties.Name)
if ($headers.Count -ne 2 -or $headers[0] -cne 'partition' -or $headers[1] -cne 'bytes') {
    throw 'EXPECTED_CSV_COLUMNS_PARTITION_BYTES'
}
$sizes = [Collections.Generic.Dictionary[string, long]]::new([StringComparer]::Ordinal)
foreach ($row in $rows) {
    if ($row.partition -cnotin $required -or $sizes.ContainsKey($row.partition)) { throw 'INVALID_SIZE_PARTITION' }
    [long]$length = 0
    if ($row.bytes -cnotmatch '^[0-9]+$' -or -not [long]::TryParse($row.bytes, [ref]$length) -or $length -le 0 -or $length -gt 2147483648) {
        throw ('EXACT_BYTE_SIZE_REQUIRED: ' + $row.partition)
    }
    $sizes.Add($row.partition, $length)
}
$firstTableHash = Read-PartitionTable $first
$secondTableHash = Read-PartitionTable $second
if ($firstTableHash -cne $secondTableHash) { throw 'PARTITION_TABLE_COPIES_DIFFER' }
$files = @()
foreach ($name in $required) {
    $a = Get-RegularItem (Join-Path $first ($name + '.bin')) $false
    $b = Get-RegularItem (Join-Path $second ($name + '.bin')) $false
    if ($a.Length -ne $sizes[$name] -or $b.Length -ne $sizes[$name]) { throw ('INCOMPLETE_PARTITION: ' + $name) }
    $aHash = (Get-FileHash -LiteralPath $a.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
    $bHash = (Get-FileHash -LiteralPath $b.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($aHash -cne $bHash) { throw ('PARTITION_COPIES_DIFFER: ' + $name) }
    $files += [ordered]@{partition = $name; bytes = $sizes[$name]; sha256 = $aHash}
}
[ordered]@{
    ok = $true
    scope = 'critical-partition-file-consistency-only'
    partitionTableSHA256 = $firstTableHash
    partitions = $files
    verifiesStockOrigin = $false
    verifiesIndependentHardwareReads = $false
    verifiesDeviceCompatibility = $false
    verifiesFullDeviceBackup = $false
    verifiesRestoreSuccess = $false
} | ConvertTo-Json -Depth 5
