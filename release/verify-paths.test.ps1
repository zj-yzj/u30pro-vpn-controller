$ErrorActionPreference='Stop'
. (Join-Path $PSScriptRoot 'verify-paths.ps1')
foreach ($bad in @('../oops','payload/../oops','payload\oops','\..\oops','/absolute','a//b','./x','x/')) {
    $rejected=$false
    try { Assert-SafeRelativePath $bad } catch { $rejected=$true }
    if (!$rejected) { throw "Unsafe path accepted: $bad" }
}
Assert-SafeRelativePath 'payload/webctl'
Assert-ManifestInventory @('payload/webctl','module.prop','manifest.json') @('module.prop','payload/webctl')
foreach ($records in @(@('module.prop','module.prop'),@('module.prop','unlisted'))) {
    $rejected=$false
    try { Assert-ManifestInventory @('module.prop','payload/webctl','manifest.json') $records } catch { $rejected=$true }
    if (!$rejected) { throw 'Invalid manifest inventory accepted.' }
}
Write-Output 'VERIFY_PATH_TESTS_OK: traversal, duplicate and unlisted records refused.'
$hash='a'*64
$null=Read-ChecksumInventory @("$hash  a.apk","$hash  b.zip") @('a.apk','b.zip')
foreach ($lines in @(@(),@("$hash  a.apk"),@("$hash  a.apk","$hash  a.apk"),@("$hash  a.apk","$hash  b.zip","$hash  extra.txt"))) {
    $rejected=$false
    try { $null=Read-ChecksumInventory $lines @('a.apk','b.zip') } catch { $rejected=$true }
    if (!$rejected) { throw 'Incomplete or duplicated checksum inventory accepted.' }
}
Write-Output 'VERIFY_CHECKSUM_TESTS_OK: empty, truncated, duplicate and extra checksum records refused.'
