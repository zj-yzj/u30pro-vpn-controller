[CmdletBinding()]
param(
    [Parameter(Mandatory)][string]$Go,
    [Parameter(Mandatory)][string]$JavaHome,
    [Parameter(Mandatory)][string]$AndroidJar,
    [Parameter(Mandatory)][string]$BuildTools,
    [Parameter(Mandatory)][string]$OutputDir,
    [Parameter(Mandatory)][string]$Keystore,
    [Parameter(Mandatory)][string]$KeystorePasswordFile,
    [Parameter(Mandatory)][string]$KeyAlias,
    [string]$Node = 'node'
)
$ErrorActionPreference = 'Stop'
$root = Split-Path $PSScriptRoot -Parent
$version = '1.3.0'
$coreSHA = 'bb9afbd48717f92d59e07da35fb694b83d40d6edb63eb9033c2ee4174137d195'
$utf8 = [Text.UTF8Encoding]::new($false)
function Checked([string]$Exe,[string[]]$Arguments) {
    & $Exe @Arguments
    if ($LASTEXITCODE -ne 0) { throw "Build command failed: $(Split-Path $Exe -Leaf) (exit $LASTEXITCODE)." }
}
function Put([string]$File,[string]$Text) { [IO.File]::WriteAllText($File,$Text.Replace("`r`n","`n"),$utf8) }
function Hash([string]$File) { (Get-FileHash -LiteralPath $File -Algorithm SHA256).Hash.ToLowerInvariant() }
$out = $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath($OutputDir)
if ($out.TrimEnd('\','/') -eq $root -or $out.StartsWith($root + [IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase)) {
    throw 'Release output must be outside the source checkout to prevent publishing build products or signing data.'
}
New-Item -ItemType Directory -Path $out -Force | Out-Null
if (Get-ChildItem -LiteralPath $out -Force | Select-Object -First 1) { throw 'OutputDir must be empty; use a fresh destination.' }
$work = Join-Path $out 'work'
$stage = Join-Path $work 'module'
$payload = Join-Path $stage 'payload'
New-Item -ItemType Directory -Path $payload -Force | Out-Null
$previous = @{}
foreach ($name in @('GOOS','GOARCH','CGO_ENABLED')) { $previous[$name] = [Environment]::GetEnvironmentVariable($name) }
Push-Location (Join-Path $root 'web-manager')
try {
    $env:GOOS='linux'; $env:GOARCH='arm64'; $env:CGO_ENABLED='0'
    foreach ($pair in @(@('u30setup','u30setup'),@('webctl','webctl'),@('webbridge','nativebridge'))) {
        $destination = if ($pair[0] -eq 'u30setup') { Join-Path $stage $pair[0] } else { Join-Path $payload $pair[0] }
        Checked $Go @('build','-trimpath','-buildvcs=false','-ldflags=-s -w','-o',$destination,('./cmd/' + $pair[1]))
    }
} finally {
    Pop-Location
    foreach ($name in $previous.Keys) { [Environment]::SetEnvironmentVariable($name,$previous[$name]) }
}
$nativeOut=Join-Path $work 'native'
& (Join-Path $root 'native-manager/build.ps1') -JavaHome $JavaHome -AndroidJar $AndroidJar -BuildTools $BuildTools -OutputDir $nativeOut -Keystore $Keystore -KeystorePasswordFile $KeystorePasswordFile -KeyAlias $KeyAlias
if (!$?) { throw 'Control APK build failed.' }
Copy-Item -LiteralPath (Join-Path $nativeOut 'U30ProVPN.apk') -Destination $payload
Checked $Node @((Join-Path $root 'web-manager/ui/assemble.mjs'),'--out',(Join-Path $work 'web-ui'))
Copy-Item -LiteralPath (Join-Path $work 'web-ui/U30ProVPN.txt') -Destination $payload
Copy-Item -LiteralPath (Join-Path $root 'web-manager/device/config.yaml') -Destination $payload
Copy-Item -LiteralPath (Join-Path $root 'gateway/reference/control.sh') -Destination $payload
$configSHA = Hash (Join-Path $payload 'config.yaml')
$supervisor = [IO.File]::ReadAllText((Join-Path $root 'gateway/reference/vpn-service.sh.in'))
foreach ($pin in @('MIHOMO_SHA256','CONFIG_SHA256')) {
    if ([regex]::Matches($supervisor,('@@' + $pin + '@@')).Count -ne 1) { throw "Expected one $pin placeholder." }
}
$supervisor=$supervisor.Replace('@@MIHOMO_SHA256@@',$coreSHA).Replace('@@CONFIG_SHA256@@',$configSHA)
if ($supervisor -match '@@[A-Z_]+@@') { throw 'Unrendered supervisor placeholder.' }
Put (Join-Path $payload 'vpn-service.sh') $supervisor
foreach ($file in @('customize.sh','service.sh','action.sh','uninstall.sh')) {
    Put (Join-Path $stage $file) ([IO.File]::ReadAllText((Join-Path $PSScriptRoot ('module/' + $file))))
}
Put (Join-Path $stage 'module.prop') ([IO.File]::ReadAllText((Join-Path $PSScriptRoot 'module/module.prop.in')).Replace('@@VERSION@@',$version))
$licenses=Join-Path $stage 'licenses'
New-Item -ItemType Directory -Path $licenses -Force | Out-Null
Copy-Item -LiteralPath (Join-Path $root 'LICENSE') -Destination (Join-Path $licenses 'PROJECT-LICENSE')
Copy-Item -LiteralPath (Join-Path $root 'THIRD_PARTY_NOTICES.md') -Destination $licenses
Copy-Item -LiteralPath (Join-Path $root 'third_party/licenses/APACHE-2.0.txt') -Destination $licenses
Copy-Item -LiteralPath (Join-Path $root 'third_party/licenses/go') -Destination $licenses -Recurse
Copy-Item -LiteralPath (Join-Path $root 'third_party/licenses/yaml.v3') -Destination $licenses -Recurse
Copy-Item -LiteralPath (Join-Path $root 'web-manager/ui/FORGE-LICENSE.txt') -Destination $licenses
$zip = Join-Path $out "U30ProVPN-$version-Magisk.zip"
Checked $Node @((Join-Path $PSScriptRoot 'package.mjs'),$stage,$zip)
$installerOut=Join-Path $work 'installer'
& (Join-Path $root 'installer-app/build.ps1') -JavaHome $JavaHome -AndroidJar $AndroidJar -BuildTools $BuildTools -OutputDir $installerOut -ModuleZip $zip -Keystore $Keystore -KeystorePasswordFile $KeystorePasswordFile -KeyAlias $KeyAlias
if (!$?) { throw 'Installer APK build failed.' }
$apk=Join-Path $out "U30ProVPN-$version-Installer.apk"
Copy-Item -LiteralPath (Join-Path $installerOut 'U30ProVPNInstaller.apk') -Destination $apk
Copy-Item -LiteralPath (Join-Path $payload 'U30ProVPN.txt') -Destination $out
Copy-Item -LiteralPath (Join-Path $nativeOut 'signature-verification.txt') -Destination (Join-Path $out 'APK-CERTIFICATE.txt')
$commit = & git -C $root rev-parse HEAD
if ($LASTEXITCODE -ne 0) { throw 'Source commit unavailable.' }
$dirty = !!(& git -C $root status --porcelain)
$goVersion = & $Go version
$assets=@(Get-ChildItem -LiteralPath $out -File | Sort-Object Name | ForEach-Object {
    [ordered]@{file=$_.Name;bytes=$_.Length;sha256=(Hash $_.FullName)}
})
$report=[ordered]@{
    schema=1;version=$version;sourceRepository='https://github.com/zj-yzj/u30pro-vpn-controller';sourceCommit=$commit.Trim();sourceDirty=$dirty
    moduleId='codex_u30pro_vpn';magiskMinimum=28100
    profile=[ordered]@{model='MU5358';device='MU5358';firmware='U30ProV1.0.0B23';abi='arm64-v8a';api=35;selinux='Permissive'}
    initialState='VPN OFF; empty subscription and nodes';freshInstallOnly=$true
    core=[ordered]@{bundled=$false;version='1.19.32';url='https://github.com/MetaCubeX/mihomo/releases/download/v1.19.32/mihomo-android-arm64-v8-v1.19.32.gz';gzipBytes=21866463;gzipSHA256='42b1f2be8702484d3ae6238daab392622f54bd0d264966700f1f28f52596e326';bytes=64045272;sha256=$coreSHA}
    validation='Host tests/build/signatures/asset integrity; no clean-device end-to-end installation acceptance.'
    go=$goVersion;assets=$assets
}
Put (Join-Path $out 'release-manifest.json') (($report | ConvertTo-Json -Depth 8) + "`n")
$sums=@(Get-ChildItem -LiteralPath $out -File | Sort-Object Name | ForEach-Object { (Hash $_.FullName) + '  ' + $_.Name })
Put (Join-Path $out 'SHA256SUMS') (($sums -join "`n") + "`n")
Write-Output "RELEASE_BUILD_OK: $version (sourceDirty=$dirty); only the six top-level assets are intended for upload."
