# Installation and recovery

[简体中文](install.md) · [Project home](../README.en.md) · [Usage](usage.en.md) · [FAQ](faq.en.md)

**Current release: `v1.3.0-alpha.1`, a first-install prerelease.** Choose either the Magisk module ZIP or the Android installer APK. Both install the same module and require compatible Magisk already working on the U30 Pro. This project does not obtain root, flash firmware or upgrade an existing private deployment.

A complete fresh-device sequence—install, reboot, enter a subscription, verify client networking, stop and uninstall—has **not** been validated on physical hardware for this alpha. Host tests, APK signature checks and file hashes do not establish that result. See the [Release notes](https://github.com/zj-yzj/u30pro-vpn-controller/releases/tag/v1.3.0-alpha.1) for the actual verification scope.

## Before installation

All conditions below must hold. The installer is expected to reject mismatches and unknown state. Do not spoof properties, lower SELinux protection or delete existing data to bypass a rejection.

| Item | Fixed scope of this alpha |
| --- | --- |
| Device and model properties | U30 Pro; both device/model must be `MU5358` |
| System and architecture | Stock Android, API **35**, `arm64-v8a` |
| Firmware display | Exactly `U30ProV1.0.0B23` |
| Root manager | Working **Magisk 28.1 or newer**, version code at least `28100` |
| SELinux | Already `Permissive`; the installer does not lower enforcement |
| Hotspot network | `br0`, `192.168.0.1/24`, IPv4 forwarding enabled, SFP baseline `1` |
| Initial deployment | No `/data/adb/u30pro-vpn` or same-package control app; neither `/data/adb/modules/codex_u30pro_vpn` nor `/data/adb/modules_update/codex_u30pro_vpn` exists; no conflicting reserved network objects |
| Subscription entry | Existing UFI-TOOLS with working login and its authenticated root API |
| Connectivity and storage | The U30 Pro itself can reach official GitHub release assets and passes the installer's space checks |

The APK's `minSdk 26` declaration does not mean the installer supports every Android 8–15 device. Other root implementations without compatible Magisk, other U30 Pro firmware, KernelSU, APatch, other hotspots and OpenWrt are outside the declared scope. Keep a working network and the device's existing recovery access available; read the recovery sections before proceeding.

**Magisk staging boundary:** the module ID is `codex_u30pro_vpn`. Before manual ZIP installation, verify that Magisk has neither an installed nor a pending-reboot staged module with that ID. If uncertain, stop and have a maintainer inspect both directories above without changing them. Magisk itself can replace same-ID `modules_update` content before project `customize.sh` checks run. The project therefore promises neither zero prior writes nor preservation of an existing stage in that namespace. The APK entry prechecks both directories before invoking the CLI; manual ZIP installation still follows Magisk's own behavior. Do not run module installation concurrently from another entry.

## Files to download

Open the [v1.3.0-alpha.1 Release](https://github.com/zj-yzj/u30pro-vpn-controller/releases/tag/v1.3.0-alpha.1) and use its Assets section. GitHub's automatically generated Source code archives are for developers and are not Magisk modules.

| File | Purpose |
| --- | --- |
| `U30ProVPN-1.3.0-alpha.1-Magisk.zip` | Install through the Magisk app on the U30 Pro |
| `U30ProVPN-1.3.0-alpha.1-Installer.apk` | Optional Android entry on the U30 Pro, embedding the exact same ZIP |
| `U30ProVPN.txt` | UFI web plugin needed to enter the first subscription; import it manually with either route |
| `SHA256SUMS` | SHA-256 checksums of the downloads |
| `APK-CERTIFICATE.txt` | Public APK signature verification and certificate fingerprints |
| `release-manifest.json` | Version, file identities and build provenance |

Choose one installation route. Do not unpack the ZIP and copy its files directly into the live runtime, and do not feed Source code.zip to Magisk.

On a computer, `Get-FileHash -Algorithm SHA256 -LiteralPath "downloaded-file"` can be compared with `SHA256SUMS` from the same Release. The Android installer also checks the exact size/hash of its embedded ZIP; that does not replace checking where the APK came from.

## The core is downloaded online

Neither this project's ZIP nor its APK **embeds a Mihomo executable**. During preparation, the device's root helper downloads the fixed `mihomo-android-arm64-v8-v1.19.32.gz` from the [official Mihomo v1.19.32 Release](https://github.com/MetaCubeX/mihomo/releases/tag/v1.19.32), verifies HTTPS certificates, and checks both compressed and decompressed size and SHA-256.

| Content | Bytes | SHA-256 |
| --- | ---: | --- |
| gzip download, about 21.9 MB | `21866463` | `42b1f2be8702484d3ae6238daab392622f54bd0d264966700f1f28f52596e326` |
| Decompressed executable, about 64.0 MB | `64045272` | `bb9afbd48717f92d59e07da35fb694b83d40d6edb63eb9033c2ee4174137d195` |

**Offline first installation is unavailable**, even with the APK already on the device. An unreachable GitHub download, TLS failure, insufficient space or mismatched digest stops preparation. Do not switch to an unknown mirror, disable certificate checks or substitute an unverified executable. Core acquisition is separate from subscription acquisition; no subscription is supplied.

Core acquisition resolves upstream addresses through the fixed IPv4 DNS endpoint `223.5.5.5:53`, matching the existing subscription downloader. The device must reach that DNS service and the GitHub download path using its existing connection; installation does not rely on the VPN being installed to connect itself.

## Route 1: Magisk ZIP

1. Transfer the module ZIP to the **U30 Pro itself**, keeping its own Internet connection working.
2. Confirm no installed or pending-reboot module with the same ID, then open Modules in Magisk, choose installation from local storage and select `U30ProVPN-1.3.0-alpha.1-Magisk.zip`.
3. Wait for preparation, including the core download and verification. Preparation stages module files; it should not enable VPN, alter hotspot routes or install the control APK. For an error or unknown outcome, follow the recovery guidance instead of repeatedly reinstalling.
4. Once Magisk confirms staging, explicitly choose to reboot. Successful staging does not mean first-boot installation has finished.
5. After Android/module initialization, use the module card's **Action** in Magisk to open the circular “VPN 控制” app. Initial state should be **VPN off, no nodes and no saved subscription**. If the control app is temporarily unavailable, allow startup to finish and check module state.
6. Continue with UFI plugin import and subscription entry below.

## Route 2: Android installer APK

Install this APK on the **U30 Pro**, not on the phone connected to its hotspot. It is a separate `com.codex.u30vpn.installer` app; the everyday round-screen controller is `com.codex.u30vpn`.

1. Use your existing device-management method to install `U30ProVPN-1.3.0-alpha.1-Installer.apk` on the U30 Pro and open “U30 Pro VPN 安装”. Use normal Android consent if it asks permission to install APKs from that source.
2. Tap “安装模块” (Install module). Approve the installation's root request in Magisk. The app verifies its embedded ZIP and invokes `magisk --install-module`; it accepts no module URL supplied by the user.
3. Wait for the result. “现在重启” (Reboot now) becomes available only after confirmed staging. Choose that button or reboot yourself later; reboot is never automatic.
4. After reboot, reopen the installer and use “打开控制” (Open control), or use the Magisk module's Action. Verify that the controller reports VPN off and an unconfigured subscription before continuing.

If using ADB on a computer, first confirm that its single connected target is the U30 Pro, then run from the download directory:

```sh
adb install "U30ProVPN-1.3.0-alpha.1-Installer.apk"
adb shell am start --user 0 -n com.codex.u30vpn.installer/.MainActivity
```

Select the correct target first if multiple devices are connected. ADB only transfers and opens the APK; module installation still needs existing Magisk and user root consent on the U30 Pro. Use the Magisk ZIP route if there is no usable APK installation/launch entry.

The installer APK has no `INTERNET` permission; the root helper performs the online core download. A timeout, killed process, lost result or **unknown outcome** does not establish that Magisk stopped or rolled back. Inspect Magisk's module state before deciding what to do next; do not keep clicking or reinstalling to guess the result.

## Enter your first subscription and enable VPN

1. Connect a phone or computer to the U30 Pro network, open your existing UFI-TOOLS page, log in normally and confirm its authenticated root features work.
2. In “插件功能 → 添加插件(.txt文件) → 提交” (Plugins → Add .txt plugin → Submit), import `U30ProVPN.txt` from the same Release. Module installation neither imports this plugin nor installs UFI itself.
3. Open the VPN plugin and read status. It should initially be off with an empty node list. Paste your own HTTPS subscription URL in Subscription, save/update it and wait for success.
4. Confirm nodes appear. **Saving the first subscription keeps VPN off.** The circular Subscription page only refreshes an already saved URL; add/change URLs in the web panel.
5. Explicitly enable VPN from the web panel or round-screen controller when ready. Wait for actual running state, then check DNS and HTTPS from a hotspot client. An enabled switch or an exit name alone does not verify client connectivity.
6. For latency results, manually use the circular Nodes page's bottom Refresh while VPN is running, or swipe an ordinary node left and tap Test. There are no scheduled batch node tests, so initial empty measurement fields are expected.

See [Usage](usage.en.md) for download restrictions, selection and subscription-failure recovery. Do not post your URL, node credentials or device private key in issues.

## Round-screen shortcut

Magisk's module Action and the installer's Open control button open the control page. More → Shortcut offers an optional floating button, which needs your explicit system consent to display over other apps. The installer does not grant that consent or modify the stock Home app. If the permission page or Home detection does not work on your firmware, use the other launch entries.

Disabling the shortcut or exiting an app does not stop VPN. Explicitly disable VPN, wait for actual state and check stock client connectivity when you want to stop proxying.

## Reboots, repeated installation and recovery

- **First boot** creates an empty provider, empty nodes and the device's own management key with VPN off. No development-device subscription, UID, trust or signing private key is imported.
- **Interrupted same-release installation** resumes only a verifiable owned transaction, preserving its key and committed configuration. Android APK installation and trust creation can straddle a reboot; do not copy a trust file from another device.
- **Existing private deployment or unknown same-package APK** is refused by project checks. This alpha provides no old-deployment upgrade, overwrite, signer migration or forced uninstall/reinstall. Preserve the working installation and plan migration separately. Magisk same-ID staging also needs the prior directory check; do not rely on project checks to preserve it. Do not delete data to pass checks.
- **Subsequent normal boots** read your saved desired VPN state. When it was enabled, a cold-boot window exists before Android/gateway readiness. This version does not promise tunnel or blocking protection for every client packet from power-on until VPN is ready. Wait for actual VPN readiness before connecting workload clients if you need coverage of that period.
- **Disabling a module differs from stopping the live gateway.** Magisk's disable setting mainly controls later startup; do not assume it immediately stops running VPN. Disable VPN and verify stock networking before disabling or removing the module.

## Uninstalling and retained data

First disable VPN and confirm stock connectivity from hotspot clients, then remove this module in Magisk and reboot as prompted. Teardown acts only on objects whose project ownership it can verify. It removes a control APK only if its identity matches this release; it does not uninstall unknown same-package apps or globally flush firewall/routing rules.

Private runtime data is retained in a root-private backup. It may contain subscription URLs, node credentials, state and keys; do not upload it. Uninstall is not private-data erasure. Permanent deletion requires a separate ownership/recovery check by someone who understands the device.

If PackageManager or network administration is unavailable during teardown, it may report **cleanup deferred**. Data remains preserved and cleanup must not be treated as complete. Follow the table below and retain recovery access. Uninstalling only the Android installer APK or control APK also does not remove the Magisk module or stop VPN that is already running.

## Installation troubleshooting

| Symptom | Next step |
| --- | --- |
| Model, firmware, API, Magisk or SELinux mismatch | Stop; this alpha supports only the fixed profile above. Do not lower security settings or spoof properties |
| Existing runtime, module, same-package app or network objects | Preserve them; this release does not overwrite unknown installations. Do not delete data to make checks pass |
| Core download, TLS, checksum or space failure | Check the U30 Pro's own stock connection, fixed DNS `223.5.5.5:53`, clock and free space. Keep official acquisition and verification; failed preparation should leave live runtime/networking untouched |
| Root denied or Magisk missing | Check existing Magisk and this installation's consent on the U30 Pro; root on a connected phone is irrelevant |
| Installer result unknown, timed out or interrupted | Inspect whether Magisk has staged the module and preserve state. Timeout does not prove rollback; do not repeatedly reinstall |
| No control app or local service connection shortly after reboot | Allow Android/PM startup and check that the module is enabled. Persistent failure needs sanitized error categories and maintainer review of the transaction/APK identity |
| Prompt to add a valid subscription before enabling VPN | Import the UFI plugin, save your own valid subscription and wait for nodes to appear |
| Controller opens but no VPN web panel appears | Manually import Release `U30ProVPN.txt` and verify UFI login/authenticated root availability |
| Cleanup deferred or unexpected connectivity after removal | Keep private data/recovery access; verify actual VPN stop, module state and network readiness, then have a maintainer restore it without global rule flushing |

Report version, model/firmware, entry route, desired/actual state and sanitized error categories in [project issues](https://github.com/zj-yzj/u30pro-vpn-controller/issues). Do not upload raw root output, subscriptions, providers/state/trust, private keys, backups or unchecked screenshots.

## Manual recovery after Magisk removed a deferred module

Magisk may delete the module directory even when its uninstall hook reports an error. Deferred cleanup preserves private data; it cannot automatically finish after its module has disappeared. A maintainer can restore **the same release's cleanup tools**:

1. Preserve `/data/adb/u30pro-vpn`, its installation journal and all private backups. Wait for Android, PackageManager and networking tools to be ready.
2. Obtain this exact Release's Magisk ZIP and verify its SHA-256. Both active and staged directories for this module ID must be absent. Investigate existing directories instead of replacing them.
3. In a trusted root shell, create a new `/data/adb/modules/codex_u30pro_vpn`, immediately create its `disable` marker, then restore the verified ZIP's original files into it. Do not run `customize.sh` or invoke `magisk --install-module`. This cleanup path requires no new core download.
4. Require root:root ownership, `0700` directories and the file modes recorded in `manifest.json`: `0600` data, `0700` listed scripts/tools. Reject symlinks and extra hooks. Keep the `disable` marker so restored tools cannot start as a normal boot module.
5. Execute in the same root shell:

   ```sh
   /data/adb/modules/codex_u30pro_vpn/u30setup uninstall --module /data/adb/modules/codex_u30pro_vpn
   ```

6. Treat cleanup as complete only after `ok: true`, `complete: true` and a backup path. Check stock client connectivity, then finish module removal/reboot in Magisk. For deferred or ownership errors, preserve current data and investigate the fixed error category; do not flush global rules or delete private runtime directories.

This recovery requires an experienced maintainer and has not received clean-device acceptance. It restores only this release's cleanup entry point, not other versions, unknown deployments or an unbootable device.

## Developers and licenses

See [Building](building.en.md) for source builds, separate release signing and artifacts, and [Device integration](device-integration.en.md) for the fixed baseline/trust boundary. Original project code/documentation uses MIT. Mihomo, Magisk, UFI, Android, Go, yaml.v3 and node-forge retain their respective upstream licenses; see [Third-party notices](../THIRD_PARTY_NOTICES.md). This project makes no complete corresponding-source distribution claim for the separately downloaded upstream Mihomo binary.
