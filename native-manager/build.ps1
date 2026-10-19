[CmdletBinding()]
param(
    [string]$JavaHome = $env:JAVA_HOME,
    [string]$AndroidJar,
    [string]$BuildTools,
    [string]$OutputDir,
    [string]$Keystore,
    [string]$KeystorePasswordFile,
    [string]$KeyAlias,
    [switch]$TestOnly
)
$ErrorActionPreference = 'Stop'
$project = $PSScriptRoot
$source = Join-Path $project 'android'
$executableSuffix = if ($env:OS -eq 'Windows_NT') { '.exe' } else { '' }

function Resolve-File([string]$Path, [string]$Description) {
    if (!$Path -or !(Test-Path -LiteralPath $Path -PathType Leaf)) {
        throw "$Description was not found. Supply its explicit build parameter."
    }
    return (Get-Item -LiteralPath $Path).FullName
}
function Invoke-Checked([string]$Executable, [string[]]$Arguments) {
    & $Executable @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "Build command failed: $(Split-Path $Executable -Leaf) (exit $LASTEXITCODE)."
    }
}

# Signing is opt-in. Never locate, create or read an existing project's identity.
$signingRequested = !!($Keystore -or $KeystorePasswordFile -or $KeyAlias)
if ($signingRequested -and (!$Keystore -or !$KeystorePasswordFile -or !$KeyAlias)) {
    throw 'Signing requires -Keystore, -KeystorePasswordFile and -KeyAlias together.'
}
if ($signingRequested) {
    $Keystore = Resolve-File $Keystore 'Keystore'
    $KeystorePasswordFile = Resolve-File $KeystorePasswordFile 'Keystore password file'
}
if (!$JavaHome) { throw 'Set JAVA_HOME or pass -JavaHome pointing to a JDK 17 or later.' }
$javaBin = Join-Path $JavaHome 'bin'
$java = Resolve-File (Join-Path $javaBin "java$executableSuffix") 'Java executable'
$javac = Resolve-File (Join-Path $javaBin "javac$executableSuffix") 'Java compiler'
$jar = Resolve-File (Join-Path $javaBin "jar$executableSuffix") 'Java archive tool'

$sdkRoot = $env:ANDROID_SDK_ROOT
if (!$sdkRoot) { $sdkRoot = $env:ANDROID_HOME }
if (!$AndroidJar) {
    if (!$sdkRoot) { throw 'Pass -AndroidJar or set ANDROID_SDK_ROOT with Android platform 35 installed.' }
    $AndroidJar = Join-Path $sdkRoot 'platforms/android-35/android.jar'
}
$AndroidJar = Resolve-File $AndroidJar 'Android API 35 android.jar'
if (!$BuildTools) {
    if (!$sdkRoot) { throw 'Pass -BuildTools or set ANDROID_SDK_ROOT with Android build-tools installed.' }
    $toolsRoot = Join-Path $sdkRoot 'build-tools'
    if (!(Test-Path -LiteralPath $toolsRoot -PathType Container)) { throw 'Android SDK build-tools directory is missing.' }
    $versions = @(Get-ChildItem -LiteralPath $toolsRoot -Directory |
        Where-Object { $_.Name -match '^\d+\.\d+\.\d+$' } |
        Sort-Object { [version]$_.Name } -Descending)
    if (!$versions.Count) { throw 'No stable Android build-tools found; pass -BuildTools explicitly.' }
    $BuildTools = $versions[0].FullName
}
$aapt2 = Resolve-File (Join-Path $BuildTools "aapt2$executableSuffix") 'aapt2'
$zipalign = Resolve-File (Join-Path $BuildTools "zipalign$executableSuffix") 'zipalign'
$d8 = Resolve-File (Join-Path $BuildTools 'lib/d8.jar') 'D8 compiler'
if ($signingRequested) { $apksigner = Resolve-File (Join-Path $BuildTools 'lib/apksigner.jar') 'APK signer' }

if (!$OutputDir) { $OutputDir = Join-Path $project 'build' }
$OutputDir = $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath($OutputDir)
New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
# A fresh working directory prevents deleted source classes from entering a later APK.
$work = Join-Path $OutputDir ('work-' + [guid]::NewGuid().ToString('N'))
$classes = Join-Path $work 'classes'
$dex = Join-Path $work 'dex'
$testOut = Join-Path $work 'host-tests'
foreach ($directory in @($classes, $dex, $testOut)) {
    New-Item -ItemType Directory -Path $directory -Force | Out-Null
}
$javaSources = @(Get-ChildItem -LiteralPath (Join-Path $source 'src') -Recurse -Filter '*.java' |
    Sort-Object FullName | ForEach-Object FullName)
if (!$javaSources.Count) { throw 'Android Java sources are missing.' }
Invoke-Checked $javac (@('-encoding','UTF-8','--release','8','-classpath',$AndroidJar,'-d',$classes) + $javaSources)
Invoke-Checked $javac @('-encoding','UTF-8','--release','8','-d',$testOut,
    (Join-Path $source 'src/com/codex/u30vpn/CryptoEnvelope.java'),
    (Join-Path $source 'tests/CryptoEnvelopeTest.java'),
    (Join-Path $source 'src/com/codex/u30vpn/SwipeGesture.java'),
    (Join-Path $source 'tests/SwipeGestureTest.java'))
Invoke-Checked $java @('-cp',$testOut,'com.codex.u30vpn.CryptoEnvelopeTest')
Invoke-Checked $java @('-cp',$testOut,'com.codex.u30vpn.SwipeGestureTest')
if ($TestOnly) {
    Write-Output 'TEST_ONLY_OK: Android sources compiled; crypto and swipe tests passed; no APK produced.'
    return
}

$classJar = Join-Path $work 'classes.jar'
Invoke-Checked $jar @('cf',$classJar,'-C',$classes,'.')
Invoke-Checked $java @('-cp',$d8,'com.android.tools.r8.D8','--release','--min-api','26',
    '--lib',$AndroidJar,'--output',$dex,$classJar)
$unpacked = Join-Path $work 'unaligned.apk'
Invoke-Checked $aapt2 @('link','-I',$AndroidJar,'--manifest',(Join-Path $source 'AndroidManifest.xml'),
    '--min-sdk-version','26','--target-sdk-version','35','-o',$unpacked)
Invoke-Checked $jar @('uf',$unpacked,'-C',$dex,'classes.dex')
$unsigned = Join-Path $OutputDir 'U30ProVPN-unsigned.apk'
Invoke-Checked $zipalign @('-f','-p','4',$unpacked,$unsigned)
Invoke-Checked $zipalign @('-c','-p','4',$unsigned)
$artifact = $unsigned
if ($signingRequested) {
    $artifact = Join-Path $OutputDir 'U30ProVPN.apk'
    Invoke-Checked $java @('-jar',$apksigner,'sign','--ks',$Keystore,'--ks-key-alias',$KeyAlias,
        '--ks-pass',"file:$KeystorePasswordFile",'--out',$artifact,$unsigned)
    $signature = & $java -jar $apksigner verify --verbose --print-certs $artifact
    if ($LASTEXITCODE -ne 0) { throw 'APK signature verification failed.' }
    $signature | Set-Content -LiteralPath (Join-Path $OutputDir 'signature-verification.txt') -Encoding utf8
    Invoke-Checked $zipalign @('-c','-p','4',$artifact)
}
[xml]$appManifest = Get-Content -Raw -LiteralPath (Join-Path $source 'AndroidManifest.xml')
$androidNamespace = 'http://schemas.android.com/apk/res/android'
$report = [ordered]@{
    schema = 1
    packageName = $appManifest.manifest.GetAttribute('package')
    versionName = $appManifest.manifest.GetAttribute('versionName', $androidNamespace)
    versionCode = [int]$appManifest.manifest.GetAttribute('versionCode', $androidNamespace)
    minSdk = 26
    targetSdk = 35
    signed = $signingRequested
    artifact = Split-Path $artifact -Leaf
    sha256 = (Get-FileHash -Algorithm SHA256 -LiteralPath $artifact).Hash.ToLowerInvariant()
    bytes = (Get-Item -LiteralPath $artifact).Length
}
$report | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $OutputDir 'build-manifest.json') -Encoding utf8
$report | ConvertTo-Json
