[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)][string]$JavaHome,
    [Parameter(Mandatory = $true)][string]$AndroidJar,
    [Parameter(Mandatory = $true)][string]$BuildTools,
    [Parameter(Mandatory = $true)][string]$OutputDir
)
$ErrorActionPreference = 'Stop'
$project = Split-Path $PSScriptRoot -Parent
$builder = Join-Path $project 'build.ps1'
if (!(Test-Path -LiteralPath $builder -PathType Leaf)) {
    throw 'Portable build.ps1 is missing.'
}
$testOnlyOutput = Join-Path $OutputDir 'test-only'
$fullOutput = Join-Path $OutputDir 'unsigned'
foreach ($destination in @($testOnlyOutput, $fullOutput)) {
    if (Test-Path -LiteralPath $destination) { throw 'Use a fresh OutputDir for this smoke test.' }
}
& $builder -JavaHome $JavaHome -AndroidJar $AndroidJar -BuildTools $BuildTools -OutputDir $testOnlyOutput -TestOnly
if (@(Get-ChildItem -LiteralPath $testOnlyOutput -Recurse -Filter '*.apk').Count -ne 0) {
    throw 'TestOnly must not produce an APK.'
}
& $builder -JavaHome $JavaHome -AndroidJar $AndroidJar -BuildTools $BuildTools -OutputDir $fullOutput
$report = Get-Content -Raw -LiteralPath (Join-Path $fullOutput 'build-manifest.json') | ConvertFrom-Json
if ($report.signed -ne $false) { throw 'Default build must report an unsigned APK.' }
$apk = Join-Path $fullOutput $report.artifact
if (!(Test-Path -LiteralPath $apk -PathType Leaf)) { throw 'Unsigned APK is missing.' }
if ((Get-FileHash -Algorithm SHA256 -LiteralPath $apk).Hash.ToLowerInvariant() -ne $report.sha256) {
    throw 'APK does not match its build manifest.'
}
$javaName = if ($env:OS -eq 'Windows_NT') { 'java.exe' } else { 'java' }
$signatureCheck = & (Join-Path $JavaHome "bin/$javaName") -jar (Join-Path $BuildTools 'lib/apksigner.jar') verify $apk 2>&1
if ($LASTEXITCODE -eq 0) { throw 'Default APK unexpectedly has a valid signature.' }
if (($signatureCheck -join "`n") -notmatch 'Missing META-INF/MANIFEST.MF|No signatures|DOES NOT VERIFY') {
    throw 'Signature check did not establish that the APK is unsigned.'
}
if (Test-Path -LiteralPath (Join-Path $project '.signing')) { throw 'Build must not create a signing identity.' }
Write-Output 'BUILD_SCRIPT_SMOKE_OK: TestOnly, real unsigned APK, manifest checksum, no generated signing identity'
