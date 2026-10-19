[CmdletBinding()]
param([string]$JavaHome = $env:JAVA_HOME, [string]$OutputDir)
$ErrorActionPreference = 'Stop'
if (!$JavaHome) { throw 'Pass -JavaHome with JDK 17 or later.' }
if (!$OutputDir) { $OutputDir = Join-Path $PSScriptRoot 'build/host-tests' }
New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
$suffix = if ($env:OS -eq 'Windows_NT') { '.exe' } else { '' }
$javac = Join-Path $JavaHome "bin/javac$suffix"
$java = Join-Path $JavaHome "bin/java$suffix"
$source = Join-Path $PSScriptRoot 'android'
$inputs = @(
    (Join-Path $source 'src/com/codex/u30vpn/installer/InstallerCore.java'),
    (Join-Path $source 'src/com/codex/u30vpn/installer/ProcessRunner.java'),
    (Join-Path $source 'tests/InstallerCoreTest.java'),
    (Join-Path $source 'tests/ProcessRunnerTest.java')
)
& $javac -encoding UTF-8 --release 8 -Xlint:unchecked -d $OutputDir @inputs
if ($LASTEXITCODE -ne 0) { throw 'Installer host tests did not compile.' }
foreach ($test in @('InstallerCoreTest','ProcessRunnerTest')) {
    & $java -cp $OutputDir "com.codex.u30vpn.installer.$test"
    if ($LASTEXITCODE -ne 0) { throw "Installer host test failed: $test" }
}
