# B23 root: tools, steps and verification

[简体中文](root-b23.md) · [Full preparation guide](setup.en.md) · [VPN installation](install.en.md)

This record describes the third-party kit **`U30Pro_B23_一键Root 永久无线ADB 永久禁用系统更新`** used by the maintainer. Its target is the original **MU5358 / U30ProHW1.0 / `U30ProV1.0.0B23`**, not B24, U30 Pro AI or F50 Pro.

This project inspected the package's scripts. Its `magisk.img` matches the readback of `init_boot_b` from the tested device, and its `tos.bin` matches the corresponding prefix of the `trustos_b` readback. This establishes the relationship between the package and the tested device; it does not guarantee a matching boot chain on another device with the same model. No device was reflashed while preparing this documentation.

**[Download the third-party B23 toolkit archive ZIP](https://github.com/zj-yzj/u30pro-vpn-controller/releases/download/b23-rootkit-archive-20261019/U30Pro-B23-RootKit-Archive-20261019.zip)** · [Archive Release](https://github.com/zj-yzj/u30pro-vpn-controller/releases/tag/b23-rootkit-archive-20261019). This mirrors the historical kit supplied by the maintainer. The date `20261019` identifies this archive snapshot, not an upstream version or official release. It is separate from the VPN controller's v1.3.0 installation packages. **Skip this archive if root already works**, and continue checking Magisk and UFI.

The archive contains **98 file entries: 91 kit files with their original bytes preserved under `B23-RootKit/`, plus 7 archive documentation and metadata files**. Two runtime logs, one PDB debug-symbol file and one empty marker file are omitted. After extraction, use the original directory layout inside `B23-RootKit/`. The archive contains no stock backup from your device and adds no complete recovery procedure.

| Archive file | Bytes | SHA-256 |
| --- | ---: | --- |
| `U30Pro-B23-RootKit-Archive-20261019.zip` | `52334651` | `87ee5dc281ea897f9861b1ba8afb16243cf57abf653870648900f8c24fa554bf` |

The supplied Magisk APK's SHA-256 matches the [official Magisk Canary 28103 release](https://github.com/topjohnwu/Magisk/releases/tag/canary-28103). A separate [Magisk source companion](https://github.com/zj-yzj/u30pro-vpn-controller/releases/download/b23-rootkit-archive-20261019/Magisk-Canary-28103-Source.zip), including the GPL license text, is provided in the same archive Release. This match establishes the APK's file provenance only; it does not establish the provenance of other images, loaders, drivers or programs. Their original authors and redistribution licenses remain unconfirmed. **This project's MIT license does not cover the third-party toolkit.**

In a [public reply on June 4, 2026](https://github.com/kanoqwq/UFI-TOOLS/issues/97#issuecomment-4624994150), the UFI-TOOLS author also provided a [historical U30 Pro one-click root resource link](https://www.123pan.com/s/7oa5Vv-h0yD3?pwd=xk9q), with public access code `xk9q`. The reply does not specify B23, and the cloud directory could not be loaded during this review, so its current contents have not been matched to this archive. Providing the link does not establish authorship of every bundled file. Check the firmware and file identities below after obtaining the archive; do not begin flashing without matching tools and your own device backups. The [public B24 tools](https://github.com/SAMYUBA/u30pro-b24-root-kit) cannot replace this B23 package.

## Understand what it changes

| Item | Behavior confirmed from the kit or tested device |
| --- | --- |
| TrustOS | The automatic script writes `bin/tos.bin` to the `trustos` target |
| Root boot image | It then forcibly writes `bin/magisk.img` to the `init_boot` target and reboots |
| Magisk | The package requires the supplied `Magisk_28103_面具.apk` and the environment repair requested by the app |
| Wireless ADB and system updates | These changes are already built into the image and cannot be deselected individually during flashing |
| Backup and recovery | The instructions require a backup first, but the automatic script does not make one; no matching stock recovery image or recovery script is supplied |

This image includes a script that attempts at every boot to configure ADB on TCP `5555`, relax ADB and system security properties, and disable update-related components. **The VPN controller does not depend on wireless ADB or require system updates to be disabled.** If you do not accept these additional changes, choose a separately verified rooting method; this package does more than add root.

Leaving wireless ADB enabled adds a device-management entry point. In particular, if ADB authentication is relaxed, other devices that can reach the port may gain debugging access. Do not expose it to the public Internet or forward it to untrusted networks. This documentation has not verified port reachability on every interface, and an attempt to set `ro.adb.secure` does not establish that authentication is disabled on every device.

## 1. Prepare tools and recovery material

You need a Windows computer, a cable confirmed to carry data, and the following:

| Material | Purpose |
| --- | --- |
| Matching B23 root kit | Supplies the flashing program, loaders, images and matching Magisk APK |
| The kit's `刷机驱动_SPD_Driver_R4.20.4201` | UNISOC download-mode driver; use the installer for the corresponding Windows version |
| [Google Platform-Tools](https://developer.android.com/tools/releases/platform-tools) | Checks USB ADB and installs APKs after flashing; verify this separately from the download-mode driver |
| Original backups from your own device and a recovery plan | Required before flashing; retain the original files, partition table, active slot, sizes and SHA-256 values |

The supplied driver directory has Win7, Win8 and Win10 directories. This does not confirm Windows 11, Windows on ARM or the signature status of every driver. Device Manager should recognize the corresponding UNISOC/SPRD download device; check ADB separately after Android starts. A working USB network connection does not establish that either of these drivers is ready.

`搞机助手_V4.9.1` and “一键安装安卓驱动” are separate helper tools, not required parts of rooting itself. This project does not distribute their EXEs or repeat their advice to disable security software.

Backups must come from **your own device before flashing**. Readbacks of `init_boot` or `trustos` after rooting contain the modified state, not stock recovery images. The package's `tos.bin` is `5,195,528` bytes, while the corresponding TrustOS partition on the tested device is `6,291,456` bytes. They are not interchangeable full backups.

The original package does not provide a complete recovery procedure. This project separately provides **[B23 backup before root and conditional recovery](backup-b23.en.md)**: use the pinned reader to read both slots' `init_boot` and `trustos`, plus `misc`, twice and check them offline before considering recovery of the actual changed partitions with an intact table and the other stated conditions. Complete that process before a first flash. It does not supply whole-device firmware, NV or damaged-table recovery.

### Check the kit's identity

On the computer, use `Get-FileHash -Algorithm SHA256 -LiteralPath "file-path"`. This table identifies only the files inspected for this record; it is not a stock signature or a compatibility guarantee:

| File | Bytes | SHA-256 |
| --- | ---: | --- |
| `bin/magisk.img` | `8388608` | `26acbb673a93a5ed7f5de91d6025f2aa1300fc64f04815f6e64aef0ca5792161` |
| `bin/tos.bin` | `5195528` | `827de96f2124d678337c48f351293296551deffcd48d0bc53dd6800c7b35eb70` |
| `Magisk_28103_面具.apk` | `11475322` | `fa7e64af14e57194e2463d959e809812e9bdd48f4cc996565706f658e78694de` |
| `[直接插电]一键Root.bat` | `219` | `4cca43edf831bb96a668b01cdd8123fc159e6c2e679cba4fb04391d2ce56b2ad` |

Check the other programs, loaders and APKs against the tool provider's sources and checksums too. Matching the two images does not prove the entire download is trustworthy.

## 2. The direct-connection procedure

Continue only after confirming that the model, full firmware, boot chain, your own backups and recovery method all match. The package's automatic script does not check these conditions for you.

1. Extract the complete kit, preserving the relative locations of the BAT file, `bin` and other files. Install the download-mode driver appropriate for the computer first.
2. Disconnect the data cable and power off the U30 Pro.
3. Open **`[直接插电]一键Root.bat`** in the kit. Once it is waiting for a device, connect the powered-off U30 Pro to the data cable.
4. Wait for the script to finish and inspect the complete result. Do not unplug the cable during the process. The main script connects in download mode, writes TrustOS, forcibly writes init_boot and reboots.
5. Wait for the device to boot normally, then confirm that stock Home, Wi-Fi and stock Internet access still work. If Wi-Fi capabilities change or startup is abnormal, stop further installation and use your own device's recovery plan.

The package also contains “备用2” (Alternative 2), “备用3” (Alternative 3), “手动Root” (Manual root) and “短接模式” (Test-point mode) scripts. They use different boot/download modes; they are not a sequence to run one after another. If you see `kick reboot timeout` or `wrong command or wrong mode`, preserve the error and check USB mode, drivers and device state before proceeding. Do not keep reflashing repeatedly.

The automatic script uses `w_force init_boot`, while the original manual instructions use `w init_boot`. The material does not explain the difference, so these routes cannot be described as verified equivalents. The script also does not specify an `_a` or `_b` suffix; current evidence confirms only the tested device's B-slot images. Do not add a suffix, switch slots or assume other slots are compatible. The complete internal behavior of the force-write tool has not been audited, and this page does not claim it affects only two physical partitions.

Test-point mode requires opening the device and identifying the points on the actual hardware. The local material lacks a test-point diagram and complete recovery steps sufficient for independent verification, so this page does not give shorting locations or instructions to resume flashing automatically.

## 3. Install the matching Magisk and complete repair

The original package explicitly requires **`Magisk_28103_面具.apk`** and warns against substituting another version, which may break its runtime environment and update patches. The VPN installer checks for a version code of at least `28100`. That is a minimum check, not an instruction to upgrade this root environment.

1. Install the matching Magisk APK from the same kit on the U30 Pro itself, after confirming the target. Installing Magisk on the computer or a phone does not replace this step.
2. Open Magisk on the U30 Pro, complete the environment repair it requests, and reboot as directed. Do not start installing the VPN module just because the icon appears.
3. Reopen Magisk and confirm that it reports installed, the repair prompt is gone and module management is available.
4. In a trusted root terminal, confirm that `id` returns `uid=0(root)` and `magisk -V` returns the required version code, including after a reboot. See the [preparation guide](setup.en.md#already-rooted-check-that-root-actually-works) for the read-only checks.

Static inspection of the supplied APK confirmed package name `com.topjohnwu.magisk`, versionCode `28103` and versionName `1e3edb88`, rather than relying on its filename. If USB ADB is available, replace `SERIAL` with the target device's serial number to install and open the manager:

```sh
adb -s SERIAL install "Magisk_28103_面具.apk"
adb -s SERIAL shell am start -n com.topjohnwu.magisk/com.topjohnwu.magisk.ui.MainActivity
```

If the matching app is already installed and working, skip the first line rather than reinstalling over it. This APK's SHA-256 has been matched to the official Canary 28103 release file: `fa7e64af14e57194e2463d959e809812e9bdd48f4cc996565706f658e78694de`, as listed above. A matching package name alone does not establish that file identity.

If modules later fail to start at boot, check the Magisk environment first. The tested device once lacked `/data/adb/magisk/busybox`; service startup recovered only after matching-version runtime files were restored. That history does not mean every device should receive a manual BusyBox copy. Check the local repair result first, and do not fill the gap with a binary of unknown origin.

## 4. Continue with UFI and VPN

Once root and Magisk are ready, return to [step 3 of the full preparation guide](setup.en.md#3-install-ufi-and-check-its-advanced-features): install full UFI, confirm that the authenticated RootShell used by plugins works, then install v1.3.0 and add your own subscription.

UFI installation materials are available as a [complete archive ZIP](https://github.com/zj-yzj/u30pro-vpn-controller/releases/download/ufi-tools-archive-20261019/U30Pro-UFI-Tools-Archive-20261019.zip) and [standalone 4.1.2 APK](https://github.com/zj-yzj/u30pro-vpn-controller/releases/download/ufi-tools-archive-20261019/UFI-TOOLS_WEB_V4.1.2_20260819_1854.apk), so no local folder needs to be found separately. The full archive includes Windows installer 1.6, the manual and helper tools; it does not replace the root preparation described here. Follow [step 3 of preparation](setup.en.md#3-install-ufi-and-check-its-advanced-features), or skip installation if UFI already works. UFI-TOOLS is authored by Minikano / kanoqwq. Each component retains its own license; this project's MIT license does not cover the entire collection of external files.
