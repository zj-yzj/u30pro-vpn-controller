# Thin Android installer

This separate `com.codex.u30vpn.installer` application embeds the exact release
Magisk ZIP supplied at build time. It requires existing, compatible Magisk 28.1
or later and user-approved `su` access. It calls the documented
`magisk --install-module` command; it does not write Magisk module directories
itself. Both the ZIP and APK entry points require a reboot.

The APK requests **no INTERNET permission**. The module's root helper downloads
the fixed official Mihomo asset during preparation, verifies HTTPS certificates,
and checks compressed/decompressed sizes and SHA-256 hashes. First installation
therefore requires working device Internet access. The APK cannot substitute a
network ZIP or a user-supplied module URL for its embedded asset.

The central 164 x 164 pixel area of the 240 x 240 circular screen contains all
content and controls. Installation runs on one process-wide worker; closing the
activity only disconnects its UI observer. A 10-minute module-command deadline,
lost process, missing project stage marker or incomplete output produces an
unknown outcome. That state blocks repeat installation and instructs the user to
inspect Magisk. It does not assert that Magisk stopped or rolled anything back.
Root output is never displayed verbatim or written to app logs.

A successful zero exit plus the exact `U30PRO_MODULE_STAGED` output line means
only that the module was staged. The explicit reboot button is enabled only in
that state. Opening an available control application does not prove that module
boot reconciliation completed. After a detected reboot the installer asks the
user to verify the module and open the controller.

Build without Gradle using PowerShell 7, JDK 17+, Android Platform 35 and Android
Build Tools 35. Supply explicit local toolchain paths or standard SDK environment
variables:

```powershell
./installer-app/tests.ps1 -JavaHome $env:JAVA_HOME -OutputDir /absolute/external/host-tests
./installer-app/build.ps1 -JavaHome $env:JAVA_HOME -AndroidJar /path/android.jar -BuildTools /path/build-tools -ModuleZip /path/U30ProVPN.zip -OutputDir /absolute/external/apk-output
```

The module must contain one root `module.prop` with version `1.3.0-alpha.1`.
The builder generates `BuildConfig` with that version and the exact copied ZIP
size/hash, runs host tests, and writes `U30ProVPNInstaller-unsigned.apk` and
`build-manifest.json`. Android source minSdk is 26, targetSdk is 35 and the
versionCode is 130001. Only build-time Java constants can select the embedded ZIP.

Signing is opt-in: pass all of `-Keystore`, `-KeystorePasswordFile`, `-KeyAlias`.
The script never searches for or creates an identity. Signed output is
`U30ProVPNInstaller.apk`; `signature-verification.txt` contains public APK
signature verification and certificate fingerprints. The build manifest hashes
the final signed bytes and records the embedded module's identity. Release
licenses and notices travel inside the embedded module ZIP.

Host tests check actual file preservation on asset corruption/truncation/size
overflow, argument quoting, Magisk version/CLI validation, stage-result rules,
repeat/timeout state safety, a real subprocess output flood and deadline. They
do not establish physical fresh-device installation compatibility.
