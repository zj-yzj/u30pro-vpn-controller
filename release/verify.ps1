[CmdletBinding()]
param([Parameter(Mandatory)][string]$Assets)
$ErrorActionPreference='Stop'
. (Join-Path $PSScriptRoot 'verify-paths.ps1')
Add-Type -AssemblyName System.IO.Compression
function Digest([byte[]]$Bytes) { [Convert]::ToHexString([Security.Cryptography.SHA256]::HashData($Bytes)).ToLowerInvariant() }
function EntryBytes($Entry) {
    if (!$Entry -or $Entry.Length -gt 128MB) { throw 'Missing or oversized ZIP entry.' }
    $stream=$Entry.Open(); $memory=[IO.MemoryStream]::new()
    try { $stream.CopyTo($memory); return ,$memory.ToArray() } finally { $stream.Dispose(); $memory.Dispose() }
}
$assetsRoot=(Resolve-Path -LiteralPath $Assets).Path
$report=Get-Content -LiteralPath (Join-Path $assetsRoot 'release-manifest.json') -Raw | ConvertFrom-Json
if ($report.sourceDirty -or $report.version -ne '1.3.0-alpha.1' -or $report.core.bundled) { throw 'Unexpected release provenance.' }
$payloadAssets=@('U30ProVPN-1.3.0-alpha.1-Magisk.zip','U30ProVPN-1.3.0-alpha.1-Installer.apk','U30ProVPN.txt','APK-CERTIFICATE.txt')
$checksums=Read-ChecksumInventory @(Get-Content -LiteralPath (Join-Path $assetsRoot 'SHA256SUMS')) @($payloadAssets + 'release-manifest.json')
foreach ($name in $checksums.Keys) {
    $expected=$checksums[$name]
    $file=Join-Path $assetsRoot $name
    if ((Get-FileHash -LiteralPath $file -Algorithm SHA256).Hash.ToLowerInvariant() -ne $expected) { throw "Checksum mismatch: $name" }
}
$declared=[Collections.Generic.HashSet[string]]::new([StringComparer]::Ordinal)
foreach ($asset in $report.assets) {
    if (!$declared.Add($asset.file) -or $asset.file -notin $payloadAssets) { throw 'Invalid release asset record.' }
    $actual=Get-Item -LiteralPath (Join-Path $assetsRoot $asset.file)
    if ($actual.Length -ne $asset.bytes -or $asset.sha256 -cne $checksums[$asset.file]) { throw 'Release asset record mismatch.' }
}
if (!$declared.SetEquals($payloadAssets)) { throw 'Missing release asset records.' }
$zipPath=Join-Path $assetsRoot ('U30ProVPN-' + $report.version + '-Magisk.zip')
$apkPath=Join-Path $assetsRoot ('U30ProVPN-' + $report.version + '-Installer.apk')
$zip=[IO.Compression.ZipFile]::OpenRead($zipPath)
$apk=[IO.Compression.ZipFile]::OpenRead($apkPath)
try {
    $names=@($zip.Entries | ForEach-Object FullName)
    if (($names | Select-Object -Unique).Count -ne $names.Count) { throw 'Duplicate module ZIP entry.' }
    foreach ($name in $names) {
        Assert-SafeRelativePath $name
        if ($name -match '(^|/)(mihomo|private-key.pem|state.json|provider.yaml)$') { throw 'Private or upstream core module entry.' }
    }
    $manifest=[Text.Encoding]::UTF8.GetString((EntryBytes ($zip.GetEntry('manifest.json')))) | ConvertFrom-Json
    if ($manifest.schema -ne 1 -or $manifest.version -ne $report.version -or $manifest.moduleId -ne $report.moduleId) { throw 'Manifest identity mismatch.' }
    if ($manifest.files.Count + 1 -ne $names.Count) { throw 'Module inventory mismatch.' }
    Assert-ManifestInventory $names @($manifest.files | ForEach-Object path)
    foreach ($record in $manifest.files) {
        $entry=$zip.GetEntry($record.path);$bytes=EntryBytes $entry
        if ($bytes.Length -ne $record.bytes -or (Digest $bytes) -ne $record.sha256) { throw ('Payload mismatch: ' + $record.path) }
        if ((($entry.ExternalAttributes -shr 16) -band 511) -ne $record.mode) { throw ('ZIP mode mismatch: ' + $record.path) }
    }
    foreach ($name in @('u30setup','payload/webctl','payload/webbridge')) {
        $bytes=EntryBytes ($zip.GetEntry($name))
        if ($bytes[0] -ne 127 -or [Text.Encoding]::ASCII.GetString($bytes,1,3) -ne 'ELF' -or $bytes[4] -ne 2 -or $bytes[18] -ne 183) { throw "Wrong executable target: $name" }
    }
    $embedded=EntryBytes ($apk.GetEntry('assets/u30pro-module.zip'))
    if ((Digest $embedded) -ne (Get-FileHash -LiteralPath $zipPath -Algorithm SHA256).Hash.ToLowerInvariant()) { throw 'Installer APK embeds a different ZIP.' }
    $plugin=EntryBytes ($zip.GetEntry('payload/U30ProVPN.txt'))
    if ((Digest $plugin) -ne (Get-FileHash -LiteralPath (Join-Path $assetsRoot 'U30ProVPN.txt') -Algorithm SHA256).Hash.ToLowerInvariant()) { throw 'Standalone plugin differs from module.' }
    $supervisor=[Text.Encoding]::UTF8.GetString((EntryBytes ($zip.GetEntry('payload/vpn-service.sh'))))
    $configDigest=Digest (EntryBytes ($zip.GetEntry('payload/config.yaml')))
    if ($supervisor -match '@@[A-Z_]+@@' -or !$supervisor.Contains($report.core.sha256) -or !$supervisor.Contains($configDigest)) { throw 'Supervisor pins missing.' }
    foreach ($license in @('licenses/PROJECT-LICENSE','licenses/FORGE-LICENSE.txt','licenses/APACHE-2.0.txt','licenses/go/LICENSE','licenses/yaml.v3/LICENSE','licenses/yaml.v3/NOTICE')) {
        if (!$zip.GetEntry($license)) { throw "License missing: $license" }
    }
    Write-Output ('RELEASE_ASSETS_VERIFIED: ' + $manifest.files.Count + ' payload entries, all checksums/modes, arm64 ELF, embedded ZIP identity and licenses.')
} finally { $zip.Dispose(); $apk.Dispose() }
