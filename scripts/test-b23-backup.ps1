# Offline negative checks: synthetic files only; no device tools are invoked.
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$checker = Join-Path $PSScriptRoot '../support/b23-backup/Test-U30ProBackup.ps1'
$tempRoot = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
$testRoot = Join-Path $tempRoot ('u30pro-backup-tests-' + [guid]::NewGuid().ToString('N'))
$null = New-Item -ItemType Directory -Path $testRoot
$names = @('init_boot_a', 'init_boot_b', 'trustos_a', 'trustos_b', 'misc')
$count = 0
function New-Fixture([string]$Name) {
    $root = Join-Path $testRoot $Name
    $a = Join-Path $root 'pass1'; $b = Join-Path $root 'pass2'
    $null = New-Item -ItemType Directory -Path $a, $b
    $xml = '<Partitions>' + (($names | ForEach-Object {'<Partition id="' + $_ + '" size="1" />'}) -join '') + '</Partitions>'
    foreach ($folder in @($a, $b)) {
        [IO.File]::WriteAllText((Join-Path $folder 'partition.xml'), $xml)
        for ($i = 0; $i -lt $names.Count; $i++) {
            $data = [byte[]]::new(4096)
            $data[0] = [byte]($i + 1)
            [IO.File]::WriteAllBytes((Join-Path $folder ($names[$i] + '.bin')), $data)
        }
    }
    $csv = Join-Path $root 'sizes.csv'
    [IO.File]::WriteAllText($csv, "partition,bytes`n" + (($names | ForEach-Object {$_ + ',4096'}) -join "`n") + "`n")
    return @{BackupDirectory = $a; SecondReadDirectory = $b; SizesFile = $csv}
}
function Expect-Rejection([hashtable]$Fixture, [string]$Category) {
    $rejected = $false
    try { $null = & $checker @Fixture } catch {
        if ($_.Exception.Message -notlike ($Category + '*')) { throw }
        $rejected = $true
    }
    if (-not $rejected) { throw ('Expected rejection: ' + $Category) }
}
try {
    $f = New-Fixture 'consistent'
    $result = & $checker @f | ConvertFrom-Json
    if (-not $result.ok -or $result.partitions.Count -ne 5 -or $result.verifiesStockOrigin -or $result.verifiesFullDeviceBackup -or $result.verifiesRestoreSuccess) { throw 'Incorrect result scope' }
    $count++

    $f = New-Fixture 'changed'
    [IO.File]::WriteAllBytes((Join-Path $f.SecondReadDirectory 'init_boot_a.bin'), [byte[]]::new(4096))
    Expect-Rejection $f 'PARTITION_COPIES_DIFFER'; $count++

    $f = New-Fixture 'short-read'
    [IO.File]::WriteAllBytes((Join-Path $f.BackupDirectory 'trustos_b.bin'), [byte[]]::new(512))
    Expect-Rejection $f 'INCOMPLETE_PARTITION'; $count++

    $f = New-Fixture 'same-directory'
    $f.SecondReadDirectory = $f.BackupDirectory
    Expect-Rejection $f 'TWO_DIFFERENT_READ_DIRECTORIES_REQUIRED'; $count++

    $f = New-Fixture 'placeholder'
    [IO.File]::WriteAllText($f.SizesFile, ([IO.File]::ReadAllText($f.SizesFile).Replace('4096', 'REPLACE_FROM_SIZE_PART')))
    Expect-Rejection $f 'EXACT_BYTE_SIZE_REQUIRED'; $count++

    $f = New-Fixture 'duplicate-size'
    [IO.File]::WriteAllText($f.SizesFile, ([IO.File]::ReadAllText($f.SizesFile).Replace('trustos_b,', 'trustos_a,')))
    Expect-Rejection $f 'INVALID_SIZE_PARTITION'; $count++

    $f = New-Fixture 'force-map'
    $table = Join-Path $f.BackupDirectory 'partition.xml'
    [IO.File]::WriteAllText($table, ([IO.File]::ReadAllText($table).Replace('</Partitions>', '<Partition id="w_force" size="8" /></Partitions>')))
    Expect-Rejection $f 'TEMPORARY_FORCE_PARTITION_FOUND'; $count++

    $f = New-Fixture 'different-map'
    $table = Join-Path $f.SecondReadDirectory 'partition.xml'
    [IO.File]::WriteAllText($table, ([IO.File]::ReadAllText($table).Replace('size="1"', 'size="2"')))
    Expect-Rejection $f 'PARTITION_TABLE_COPIES_DIFFER'; $count++

    [ordered]@{ok = $true; offlineChecks = $count; deviceCommandsExecuted = $false} | ConvertTo-Json
} finally {
    $resolved = [IO.Path]::GetFullPath($testRoot)
    $relative = $resolved.Substring($tempRoot.TrimEnd('\', '/').Length).TrimStart('\', '/')
    if (-not $resolved.StartsWith($tempRoot, [StringComparison]::OrdinalIgnoreCase) -or $relative -notmatch '^u30pro-backup-tests-[0-9a-f]{32}$') { throw 'Unsafe test cleanup path' }
    Remove-Item -LiteralPath $resolved -Recurse -Force
}
