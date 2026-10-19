[CmdletBinding()]
param(
    [string]$JavaHome = $env:JAVA_HOME,
    [string]$AndroidJar,
    [string]$BuildTools,
    [string]$OutputDir,
    [string]$ModuleZip,
    [string]$Keystore,
    [string]$KeystorePasswordFile,
    [string]$KeyAlias
)
$ErrorActionPreference = 'Stop'
$source = Join-Path $PSScriptRoot 'android'
$suffix = if ($env:OS -eq 'Windows_NT') { '.exe' } else { '' }
function Resolve-Input([string]$Path, [string]$Description) {
    if (!$Path -or !(Test-Path -LiteralPath $Path -PathType Leaf)) { throw "$Description not found; supply its explicit build parameter." }
    return (Get-Item -LiteralPath $Path).FullName
}
function Invoke-Tool([string]$Tool, [string[]]$Arguments) {
    & $Tool @Arguments
    if ($LASTEXITCODE -ne 0) { throw "Installer build tool failed: $(Split-Path $Tool -Leaf) (exit $LASTEXITCODE)." }
}
$ModuleZip = Resolve-Input $ModuleZip 'Module ZIP (-ModuleZip)'
if (!$JavaHome) { throw 'Set JAVA_HOME or pass -JavaHome with JDK 17 or later.' }
$java = Resolve-Input (Join-Path $JavaHome "bin/java$suffix") 'Java runtime'
$javac = Resolve-Input (Join-Path $JavaHome "bin/javac$suffix") 'Java compiler'
$sdkRoot = $env:ANDROID_SDK_ROOT
if (!$sdkRoot) { $sdkRoot = $env:ANDROID_HOME }
if (!$AndroidJar -and $sdkRoot) { $AndroidJar = Join-Path $sdkRoot 'platforms/android-35/android.jar' }
$AndroidJar = Resolve-Input $AndroidJar 'Android API 35 android.jar'
if (!$BuildTools -and $sdkRoot) {
    $versions = @(Get-ChildItem -LiteralPath (Join-Path $sdkRoot 'build-tools') -Directory |
        Where-Object { $_.Name -match '^\d+\.\d+\.\d+$' } | Sort-Object { [version]$_.Name } -Descending)
    if ($versions.Count) { $BuildTools = $versions[0].FullName }
}
if (!$BuildTools) { throw 'Pass -BuildTools with Android Build Tools 35 or later.' }
$aapt2 = Resolve-Input (Join-Path $BuildTools "aapt2$suffix") 'aapt2'
$zipalign = Resolve-Input (Join-Path $BuildTools "zipalign$suffix") 'zipalign'
$d8 = Resolve-Input (Join-Path $BuildTools 'lib/d8.jar') 'D8 compiler'
$signingRequested = !!($Keystore -or $KeystorePasswordFile -or $KeyAlias)
if ($signingRequested -and (!$Keystore -or !$KeystorePasswordFile -or !$KeyAlias)) {
    throw 'Signing requires -Keystore, -KeystorePasswordFile and -KeyAlias together.'
}
if ($signingRequested) {
    # Only caller-supplied release signing inputs are resolved; never discover identities.
    $Keystore = Resolve-Input $Keystore 'Keystore'
    $KeystorePasswordFile = Resolve-Input $KeystorePasswordFile 'Keystore password file'
    $apksigner = Resolve-Input (Join-Path $BuildTools 'lib/apksigner.jar') 'APK signer'
}
[xml]$appManifest = Get-Content -Raw -LiteralPath (Join-Path $source 'AndroidManifest.xml')
$ns = 'http://schemas.android.com/apk/res/android'
$version = $appManifest.manifest.GetAttribute('versionName', $ns)
if ($version -ne '1.3.0') { throw 'Unexpected installer source version.' }
$inputHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $ModuleZip).Hash.ToLowerInvariant()
$zipReader = [IO.Compression.ZipFile]::OpenRead($ModuleZip)
try {
    $props = @($zipReader.Entries | Where-Object { $_.FullName -eq 'module.prop' })
    if ($props.Count -ne 1) { throw 'Module ZIP must have exactly one root module.prop.' }
    $reader = [IO.StreamReader]::new($props[0].Open())
    try { $propertyText = $reader.ReadToEnd() } finally { $reader.Dispose() }
    $versions = @([regex]::Matches($propertyText, '(?m)^version=([^\r\n]+)\r?$'))
    if ($versions.Count -ne 1 -or $versions[0].Groups[1].Value -ne $version) { throw 'Module ZIP version does not match this installer.' }
} finally { $zipReader.Dispose() }
if (!$OutputDir) { $OutputDir = Join-Path $PSScriptRoot 'build' }
$OutputDir = $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath($OutputDir)
New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
$work = Join-Path $OutputDir ('work-' + [guid]::NewGuid().ToString('N'))
$classes = Join-Path $work 'classes'
$dex = Join-Path $work 'dex'
$generated = Join-Path $work 'generated/com/codex/u30vpn/installer'
$assets = Join-Path $work 'assets'
foreach ($directory in @($classes,$dex,$generated,$assets)) { New-Item -ItemType Directory -Path $directory -Force | Out-Null }
$embeddedZip = Join-Path $assets 'u30pro-module.zip'
Copy-Item -LiteralPath $ModuleZip -Destination $embeddedZip
$zipHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $embeddedZip).Hash.ToLowerInvariant()
$zipBytes = (Get-Item -LiteralPath $embeddedZip).Length
if ($zipHash -ne $inputHash -or $zipHash -ne (Get-FileHash -Algorithm SHA256 -LiteralPath $ModuleZip).Hash.ToLowerInvariant()) { throw 'Module ZIP changed during build.' }
@"
package com.codex.u30vpn.installer;
public final class BuildConfig {
    private BuildConfig() { }
    public static final String MODULE_ZIP_SHA256 = "$zipHash";
    public static final long MODULE_ZIP_BYTES = ${zipBytes}L;
    public static final String MODULE_VERSION = "$version";
}
"@ | Set-Content -LiteralPath (Join-Path $generated 'BuildConfig.java') -Encoding utf8
& (Join-Path $PSScriptRoot 'tests.ps1') -JavaHome $JavaHome -OutputDir (Join-Path $work 'host-tests')
$javaSources = @(Get-ChildItem -LiteralPath (Join-Path $source 'src') -Recurse -Filter '*.java' | Sort-Object FullName | ForEach-Object FullName)
$javaSources += Join-Path $generated 'BuildConfig.java'
Invoke-Tool $javac (@('-encoding','UTF-8','--release','8','-classpath',$AndroidJar,'-d',$classes) + $javaSources)
# Stable archive timestamps remove host clock variation from the classes/APK inputs.
$stamp = [DateTimeOffset]::new(2000,1,1,0,0,0,[TimeSpan]::Zero)
$classJar = Join-Path $work 'classes.jar'
$archive = [IO.Compression.ZipFile]::Open($classJar, [IO.Compression.ZipArchiveMode]::Create)
try {
    foreach ($file in @(Get-ChildItem -LiteralPath $classes -Recurse -File | Sort-Object FullName)) {
        $relative = [IO.Path]::GetRelativePath($classes, $file.FullName).Replace('\','/')
        $entry = $archive.CreateEntry($relative, [IO.Compression.CompressionLevel]::Optimal)
        $entry.LastWriteTime = $stamp
        $inputStream = [IO.File]::OpenRead($file.FullName)
        $entryStream = $entry.Open()
        try { $inputStream.CopyTo($entryStream) } finally { $entryStream.Dispose(); $inputStream.Dispose() }
    }
} finally { $archive.Dispose() }
Invoke-Tool $java @('-cp',$d8,'com.android.tools.r8.D8','--release','--min-api','26','--lib',$AndroidJar,'--output',$dex,$classJar)
$unpacked = Join-Path $work 'unaligned.apk'
Invoke-Tool $aapt2 @('link','-I',$AndroidJar,'--manifest',(Join-Path $source 'AndroidManifest.xml'),'-A',$assets,'--min-sdk-version','26','--target-sdk-version','35','-o',$unpacked)
$archive = [IO.Compression.ZipFile]::Open($unpacked, [IO.Compression.ZipArchiveMode]::Update)
try {
    $entry = [IO.Compression.ZipFileExtensions]::CreateEntryFromFile($archive, (Join-Path $dex 'classes.dex'), 'classes.dex', [IO.Compression.CompressionLevel]::Optimal)
    foreach ($entry in $archive.Entries) { $entry.LastWriteTime = $stamp }
} finally { $archive.Dispose() }
$unsigned = Join-Path $OutputDir 'U30ProVPNInstaller-unsigned.apk'
Invoke-Tool $zipalign @('-f','-p','4',$unpacked,$unsigned)
Invoke-Tool $zipalign @('-c','-p','4',$unsigned)
$artifact = $unsigned
if ($signingRequested) {
    $artifact = Join-Path $OutputDir 'U30ProVPNInstaller.apk'
    Invoke-Tool $java @('-jar',$apksigner,'sign','--ks',$Keystore,'--ks-key-alias',$KeyAlias,'--ks-pass',"file:$KeystorePasswordFile",'--out',$artifact,$unsigned)
    $signature = & $java -jar $apksigner verify --verbose --print-certs $artifact
    if ($LASTEXITCODE -ne 0) { throw 'Installer APK signature verification failed.' }
    $signature | Set-Content -LiteralPath (Join-Path $OutputDir 'signature-verification.txt') -Encoding utf8
    Invoke-Tool $zipalign @('-c','-p','4',$artifact)
}
$report = [ordered]@{
    schema=1; packageName=$appManifest.manifest.GetAttribute('package'); versionName=$version
    versionCode=[int]$appManifest.manifest.GetAttribute('versionCode',$ns); minSdk=26; targetSdk=35
    signed=$signingRequested; artifact=(Split-Path $artifact -Leaf)
    sha256=(Get-FileHash -Algorithm SHA256 -LiteralPath $artifact).Hash.ToLowerInvariant()
    bytes=(Get-Item -LiteralPath $artifact).Length
    moduleZip=[ordered]@{asset='assets/u30pro-module.zip';sha256=$zipHash;bytes=$zipBytes;version=$version}
}
$report | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath (Join-Path $OutputDir 'build-manifest.json') -Encoding utf8
$report | ConvertTo-Json -Depth 4
