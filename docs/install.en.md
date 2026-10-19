# Installation and recovery

[简体中文](install.md) · [Project home](../README.en.md) · [Usage](usage.en.md) · [FAQ](faq.en.md)

For **v1.3.0-alpha.1**, choose the Magisk ZIP or the Android installer APK. Both install the same module on the U30 Pro. Magisk and UFI-TOOLS must already be working; this release prepares a fresh installation rather than upgrading an existing private deployment.

## Before installation

Check that your device matches this profile:

| Item | Required profile |
| --- | --- |
| Device and model properties | U30 Pro; both `MU5358` |
| System | Stock Android, API `35`, `arm64-v8a` |
| Firmware display | `U30ProV1.0.0B23` |
| Root manager | Magisk 28.1+; version code at least `28100` |
| SELinux | Already `Permissive` |
| Hotspot network | `br0`, `192.168.0.1/24`, IPv4 forwarding enabled, SFP baseline `1` |
| Initial state | No existing project runtime, module, same-package control app or conflicting reserved network objects |
| Web interface | Existing UFI-TOOLS with working login and authenticated root API |
| Connection and storage | The U30 Pro can reach official GitHub release assets and passes the installer's space checks |

The installer checks this profile and does not obtain root, flash firmware or lower SELinux enforcement. Other firmware, KernelSU, APatch, other hotspots and OpenWrt need separate integration. Keep your usual device-management and recovery access available.

<details>
<summary>Existing installations and Magisk staging</summary>

Before installing, check that `/data/adb/u30pro-vpn`, `/data/adb/modules/codex_u30pro_vpn` and `/data/adb/modules_update/codex_u30pro_vpn` are absent. Also check for the control app package `com.codex.u30vpn`. Preserve existing installations and have their state reviewed rather than deleting them to pass checks.

For manual ZIP installation, Magisk can replace same-ID staged content before this project's checks run. An existing `codex_u30pro_vpn` stage is therefore not protected by those checks. The APK route checks both module directories before invoking Magisk's CLI. Use one route at a time.

</details>

## Files to download

Open the [v1.3.0-alpha.1 Release](https://github.com/zj-yzj/u30pro-vpn-controller/releases/tag/v1.3.0-alpha.1) and download from **Assets**:

| File | Purpose |
| --- | --- |
| `U30ProVPN-1.3.0-alpha.1-Magisk.zip` | Install through Magisk on the U30 Pro |
| `U30ProVPN-1.3.0-alpha.1-Installer.apk` | Android entry on the U30 Pro, containing the same ZIP |
| `U30ProVPN.txt` | Import into UFI to enter your first subscription |
| `SHA256SUMS` | Checksums for the downloads |

Choose one installation package and download the plugin. Compare the package's SHA-256 with `SHA256SUMS` from the same Release; in PowerShell use `Get-FileHash -Algorithm SHA256 -LiteralPath "downloaded-file"`. The Release also includes `APK-CERTIFICATE.txt` and `release-manifest.json` for signature and build details. GitHub's **Source code** archives are source snapshots, not installable modules.

## The core is downloaded online

During preparation, the device downloads `mihomo-android-arm64-v8-v1.19.32.gz` from the [official Mihomo v1.19.32 Release](https://github.com/MetaCubeX/mihomo/releases/tag/v1.19.32). The download is about **21.9 MB**, expanding to about **64.0 MB**. Neither the ZIP nor the APK bundles the core, so first installation needs the U30 Pro's own Internet connection.

<details>
<summary>Download verification and network details</summary>

The root helper verifies TLS and the compressed and decompressed sizes and SHA-256 values:

| Content | Bytes | SHA-256 |
| --- | ---: | --- |
| gzip download | `21866463` | `42b1f2be8702484d3ae6238daab392622f54bd0d264966700f1f28f52596e326` |
| Executable | `64045272` | `bb9afbd48717f92d59e07da35fb694b83d40d6edb63eb9033c2ee4174137d195` |

Upstream addresses are resolved through IPv4 DNS `223.5.5.5:53`. That service and the GitHub download path must be reachable over the existing connection. A failed download, TLS check, digest or space check stops preparation. Keep the official download and verification enabled.

</details>

## Route 1: Magisk ZIP

1. Transfer `U30ProVPN-1.3.0-alpha.1-Magisk.zip` to the **U30 Pro itself**.
2. In Magisk, open **Modules**, install from local storage and select the ZIP.
3. Wait for preparation and the core download to finish. When Magisk confirms staging, reboot.
4. After startup, use the module card's **Action** to open “VPN 控制”. Allow initialization to finish if it is not ready yet.
5. Confirm **VPN off, no nodes and no saved subscription**, then continue below.

## Route 2: Android installer APK

Install this APK on the **U30 Pro**, using your usual device-management method.

1. Install `U30ProVPN-1.3.0-alpha.1-Installer.apk` and open “U30 Pro VPN 安装”.
2. Tap **安装模块** (Install module) and approve its root request in Magisk.
3. Wait for confirmed staging, then tap **现在重启** (Reboot now), or reboot later yourself.
4. After reboot, reopen the installer and tap **打开控制** (Open control), or use the Magisk module's Action.
5. Confirm the controller starts with VPN off and an unconfigured subscription.

<details>
<summary>Installing with ADB and handling an interrupted result</summary>

With the U30 Pro selected as your ADB target, run from the download directory:

```sh
adb install "U30ProVPN-1.3.0-alpha.1-Installer.apk"
adb shell am start --user 0 -n com.codex.u30vpn.installer/.MainActivity
```

Select the correct target first if multiple devices are connected. The installer package is `com.codex.u30vpn.installer`; the everyday controller is `com.codex.u30vpn`. ADB transfers and opens the installer; module installation still needs Magisk and root consent on the U30 Pro.

The installer verifies its embedded ZIP and calls `magisk --install-module`. It has no `INTERNET` permission; the root helper downloads the core. Reboot is explicit. If the result times out or is unknown, inspect Magisk's module state before retrying: a lost result does not mean installation rolled back.

</details>

## Enter your first subscription and enable VPN

1. Connect a phone or computer to the U30 Pro network and log into your existing UFI-TOOLS.
2. Use **插件功能 → 添加插件(.txt文件) → 提交** (Plugins → Add .txt plugin → Submit) to import `U30ProVPN.txt`.
3. Open the VPN plugin's Subscription tab, enter your own HTTPS subscription URL and choose **Save and update**.
4. Wait for nodes to appear. Saving the first subscription keeps VPN off.
5. Enable VPN from the web panel or round-screen app, wait for the running state, then check DNS and HTTPS from a hotspot client.

The round-screen Subscription page can update a saved URL; use UFI to add or change it. Once VPN is running, use Nodes → Refresh for a full manual latency test, or swipe an ordinary node left and tap Test. See [Usage](usage.en.md) for the controls.

## Round-screen shortcut

More → Shortcut adds an optional floating button to the unlocked stock Home screen. Allow display over other apps in system settings, then enable the shortcut. You can also keep using Magisk's Action or the installer's Open control button.

Exiting the app or disabling the shortcut leaves VPN running. Use **Disable VPN** to stop it and check that clients return to stock networking.

## Reboots, repeated installation and recovery

Normal boots use your saved desired VPN state. If it was enabled, the gateway starts after Android is ready; there is a cold-boot window before the tunnel and its protection are ready.

<details>
<summary>Installation state and recovery limits</summary>

The first boot creates empty nodes, a device-generated management key and trust for the installed control APK, with VPN off. It does not import another device's subscription, UID, trust or private signing key.

An interrupted same-release installation resumes only a verifiable transaction owned by this installer. It preserves its key and committed configuration. Existing private deployments and unknown same-package apps are refused; this alpha has no upgrade, signer-migration or forced-reinstall path. Do not copy trust files between devices or delete existing data to force installation.

Magisk's disable setting controls later module startup; it need not stop a gateway already running. Disable VPN and check stock connectivity before disabling or removing the module.

Everyday use has been tested on the author's U30 Pro. A complete clean-device installation, power-loss recovery and uninstall still need separate checks; see [Testing](testing.en.md).

</details>

## Uninstalling and retained data

Disable VPN, confirm stock connectivity from hotspot clients, then remove the module in Magisk and reboot as prompted. Removing only an APK leaves the module installed.

Uninstall retains private runtime data in a root-private backup, including possible subscription URLs, credentials, state and keys. Keep it private. Cleanup touches only objects whose project ownership it can verify and removes the control APK only when its identity matches this release.

If teardown reports **cleanup deferred**, preserve the data and recovery access. Cleanup still needs to finish; the manual recovery section below covers a module directory already removed by Magisk.

## Installation troubleshooting

| Symptom | Next step |
| --- | --- |
| Device, firmware, Magisk or SELinux mismatch | Check the required profile; preserve the current settings |
| Existing runtime, module, app or network objects | Preserve them and review the existing deployment |
| Core download, TLS, checksum or space failure | Check the device's own connection, DNS `223.5.5.5:53`, clock and free space |
| Root denied | Check Magisk and approve this installation's request on the U30 Pro |
| Installer timed out or result is unknown | Inspect Magisk's module state before retrying |
| Controller or local service unavailable after reboot | Allow startup to finish, then check that the module is enabled |
| Subscription required or node list empty | Import the UFI plugin and save your own supported subscription |
| No web panel | Import `U30ProVPN.txt` and check UFI login and root access |
| Cleanup deferred or networking unexpected after removal | Keep recovery access and review actual VPN/module state before further changes |

Report your version, firmware, installation route and error category in [project issues](https://github.com/zj-yzj/u30pro-vpn-controller/issues). Remove private data from logs and screenshots before sharing.

## Manual recovery after Magisk removed a deferred module

<details>
<summary>Maintainer procedure for restoring this release's cleanup tools</summary>

Magisk may delete a module directory even when its uninstall hook reports an error. To finish deferred cleanup, an experienced maintainer can restore the same release's tools:

1. Preserve `/data/adb/u30pro-vpn`, its installation journal and private backups. Wait for Android, PackageManager and networking tools to be ready.
2. Obtain this exact Release's Magisk ZIP and verify its SHA-256. Both active and staged `codex_u30pro_vpn` directories must be absent; review existing directories instead of replacing them.
3. In a trusted root shell, create `/data/adb/modules/codex_u30pro_vpn`, immediately create its `disable` marker, then restore the verified ZIP's original files into it. Do not run `customize.sh` or `magisk --install-module`; no new core download is needed.
4. Set root:root ownership, `0700` directories and the file modes in `manifest.json`: `0600` data and `0700` listed scripts/tools. Reject symlinks and extra hooks. Keep `disable` present.
5. Run in the same root shell:

   ```sh
   /data/adb/modules/codex_u30pro_vpn/u30setup uninstall --module /data/adb/modules/codex_u30pro_vpn
   ```

6. Require `ok: true`, `complete: true` and a backup path. Check stock client connectivity, then finish module removal and reboot in Magisk. On deferred or ownership errors, preserve current data and investigate the error category. Do not flush global rules or delete private runtime directories.

This restores this release's cleanup entry point. It has not had clean-device acceptance and is not a recovery method for other versions, unknown deployments or an unbootable device.

</details>

## Developers and licenses

See [Building](building.en.md) for source builds and release signing, and [Device integration](device-integration.en.md) for the fixed baseline and trust setup. Original project code and documentation use [MIT](../LICENSE); upstream components retain their own licenses in [Third-party notices](../THIRD_PARTY_NOTICES.md). The separately downloaded Mihomo binary is obtained from its upstream release.
