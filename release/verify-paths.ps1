function Assert-SafeRelativePath([string]$Name) {
    if (!$Name -or $Name.Contains('\') -or $Name.StartsWith('/') -or $Name -cnotmatch '^[A-Za-z0-9_.\-/]+$') { throw 'Unsafe ZIP path.' }
    foreach ($segment in $Name.Split('/')) { if ($segment -in @('','.','..')) { throw 'Unsafe ZIP path segment.' } }
}
function Assert-ManifestInventory([string[]]$Entries,[string[]]$Records) {
    $seen=[Collections.Generic.HashSet[string]]::new([StringComparer]::Ordinal)
    foreach ($name in $Entries) { Assert-SafeRelativePath $name; if (!$seen.Add($name)) { throw 'Duplicate ZIP entry.' } }
    if (!$seen.Remove('manifest.json')) { throw 'Missing manifest.' }
    $declared=[Collections.Generic.HashSet[string]]::new([StringComparer]::Ordinal)
    foreach ($name in $Records) { Assert-SafeRelativePath $name; if (!$declared.Add($name)) { throw 'Duplicate manifest record.' } }
    if (!$seen.SetEquals($declared)) { throw 'Unlisted or missing payload file.' }
}
function Read-ChecksumInventory([string[]]$Lines,[string[]]$Expected) {
    $records=[Collections.Generic.Dictionary[string,string]]::new([StringComparer]::Ordinal)
    foreach ($line in $Lines) {
        if ($line -cnotmatch '^([a-f0-9]{64})  ([A-Za-z0-9_.-]+)$') { throw 'Invalid checksum line.' }
        if (!$records.TryAdd($Matches[2],$Matches[1])) { throw 'Duplicate checksum entry.' }
    }
    $wanted=[Collections.Generic.HashSet[string]]::new([StringComparer]::Ordinal)
    foreach ($name in $Expected) { $null=$wanted.Add($name) }
    if (!$wanted.SetEquals($records.Keys)) { throw 'Incomplete or extra checksum inventory.' }
    return $records
}
