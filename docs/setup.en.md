# From device preparation to a working VPN

[简体中文](setup.md) · [Project home](../README.en.md) · [Installation and recovery](install.en.md)

If this is your first time using the project, work through these steps: **check the firmware → back up and verify your device's critical partitions → get working Magisk root → install UFI and enable its advanced features → install the VPN module → add your own subscription → test from a hotspot client**. Skip any steps you have already completed.

The current v1.3.0 release supports **MU5358 / `U30ProV1.0.0B23` / Android API 35**. Rooting the device and installing the VPN are separate steps: downloading this project's APK or ZIP does not root a stock device.

Download these before powering off the device:

| Material | Download and purpose |
| --- | --- |
| Backup support package | [Pinned reader, two-pass checker and guide](https://github.com/zj-yzj/u30pro-vpn-controller/releases/download/b23-backup-tools-20261019/U30Pro-B23-Backup-Tools-20261019.zip) |
| B23 root kit | [Flashing tools, download-mode driver and matching Magisk](https://github.com/zj-yzj/u30pro-vpn-controller/releases/download/b23-rootkit-archive-20261019/U30Pro-B23-RootKit-Archive-20261019.zip) |
| UFI installation materials | [Full UFI, ADB tools and manual](https://github.com/zj-yzj/u30pro-vpn-controller/releases/download/ufi-tools-archive-20261019/U30Pro-UFI-Tools-Archive-20261019.zip) |
| VPN installation files | Choose the module ZIP or installer APK from [v1.3.0](https://github.com/zj-yzj/u30pro-vpn-controller/releases/tag/v1.3.0), plus `U30ProVPN.txt` |

Provide your own Windows computer, data cable, working network and subscription. The device also downloads the official Mihomo core during first installation. Source-reference packages are optional and are not needed to run the tools.

## 1. Find your starting point

| Current state | Where to continue |
| --- | --- |
| B23, with working Magisk and UFI advanced features, and this project not yet installed | Check the [installation requirements](install.en.md#before-installation), then continue at step 4 |
| B23, with Magisk root but no UFI yet | Prepare UFI in step 3 |
| B23, not yet rooted | Read step 2 and the B23 rooting record first; prepare matching tools, your own device backups and a recovery method |
| B24, U30 Pro AI, MU5353 or another firmware/model | The current installer rejects these; do not continue with the B23 installation steps |
| An older version, alpha or another deployment already installed | Read [Installation and recovery](install.en.md) first; v1.3.0 has no upgrade path over an existing deployment |

Record the full model, hardware version and firmware version from the stock screen or management page. `MU5358` is only the model name; devices with the same model can still differ in boot partitions, firmware and rooting method.

### Read the full properties if USB ADB is already available

On the computer, use the already extracted `adb-fastboot` directory in the [complete UFI archive](https://github.com/zj-yzj/u30pro-vpn-controller/releases/download/ufi-tools-archive-20261019/U30Pro-UFI-Tools-Archive-20261019.zip), or download Google's [Android Platform-Tools](https://developer.android.com/tools/releases/platform-tools). Keep `adb.exe` and its DLLs together, then connect the powered-on U30 Pro with a data-capable USB cable. Open a terminal in that tools directory. In Windows PowerShell, use `./adb.exe` in place of `adb` below, or add that directory to PATH first.

```sh
adb devices -l
```

Confirm that the listed target is the U30 Pro and its state is `device`. Replace `SERIAL` below with its actual serial number before entering the shell. When exactly one USB Android device is connected, you can also use `adb -d shell`. An `unauthorized` state requires ADB authorization on the device. If no device appears, check the cable, drivers and existing USB debugging settings first.

```sh
adb -s SERIAL shell
```

In the Android shell, read these values one line at a time:

```sh
getprop ro.product.model
getprop ro.product.device
getprop ro.build.display.id
getprop ro.build.version.sdk
getprop ro.product.cpu.abi
getprop ro.boot.slot_suffix
getenforce
pm list packages com.minikano.f50_sms
```

The current installer requires the first five values to be `MU5358`, `MU5358`, `U30ProV1.0.0B23`, `35` and `arm64-v8a`, respectively, with SELinux already `Permissive`. The boot slot is information to check against the rooting method, not an instruction to switch slots manually. Once the full UFI version is installed, the package query should show `package:com.minikano.f50_sms`.

No ADB access does not necessarily mean the device lacks root. You can inspect its information through an existing trusted management interface instead. These commands do not enable ADB or obtain root. Stop if the information does not match; do not change system properties or SELinux to pass the installer's checks.

## 2. Root a stock device first

The maintainer rooted the tested B23 device. The kit and steps are recorded separately in **[B23 root: tools, steps and verification](root-b23.en.md)**, including the Windows download-mode driver, direct-connection procedure, supplied Magisk `28103`, environment repair and reboot checks.

Before a first flash, complete the [two-read critical-partition backup and conditional recovery preparation](backup-b23.en.md). Download the support package in advance. Backup uses download mode and does not require root or ADB first; the original one-click root script does not back up for you. This prepares limited recovery of your device's critical partitions, not complete stock firmware or a guaranteed unbrick route.

If you need the same kit, download the [third-party B23 toolkit archive ZIP](https://github.com/zj-yzj/u30pro-vpn-controller/releases/download/b23-rootkit-archive-20261019/U30Pro-B23-RootKit-Archive-20261019.zip) ([separate archive Release](https://github.com/zj-yzj/u30pro-vpn-controller/releases/tag/b23-rootkit-archive-20261019)). This historical mirror is separate from the VPN controller's v1.3.0 installation packages. Skip it if root already works; do not reflash to install the VPN.

Read the root record before proceeding. This third-party package includes a script that attempts at every boot to enable wireless ADB on TCP `5555`, relax authentication and disable update components. Its automatic script has no backup step. The archive preserves the supplied tool bytes, and no fresh flash test was performed. Apart from the supplied Magisk APK whose official file identity has been confirmed, other components' original authors and redistribution licenses remain unconfirmed. The project's MIT license does not cover the toolkit. A first flash still requires your own stock backups and a matching recovery method; the archive provides no complete recovery guarantee.

The [public B24 rooting material](https://github.com/SAMYUBA/u30pro-b24-root-kit) covers different firmware and cannot replace the B23 kit. Installing only the Magisk APK does not complete rooting either. See the [official Magisk guide](https://topjohnwu.github.io/Magisk/install.html) for the general process, but do not apply generic fastboot commands directly to the U30 Pro.

### Already rooted: check that root actually works

Run the following two read-only commands in an existing trusted root terminal. If using ADB, first run `su` inside `adb -d shell`, grant the shell root through the device's existing Magisk management flow, then run:

```sh
id
magisk -V
```

The `id` output should contain `uid=0(root)`, and `magisk -V` should return a numeric version code of at least `28100`. Magisk should report that it is installed, rather than showing only the app version or “not installed.” If the device already has root, check whether its current version meets the requirements before changing any boot image for this project.

If `su` or `magisk` reports “command not found,” the device may still have root. Some environments have existing entry points at `/debug_ramdisk/su` and `/debug_ramdisk/magisk`. Use those absolute paths only after confirming they belong to this device's installed Magisk. The APK installation route also needs ordinary `su` to be callable; a working absolute path does not establish that the APK route is ready.

If shell authorization is denied, check root authorization for that caller first. A working UFI root terminal does not automatically authorize the ADB shell. Check again after a reboot to confirm that root and stock Wi-Fi connectivity still work, then continue with UFI. Use `exit` to leave the root shell and ADB shell when finished.

## 3. Install UFI and check its advanced features

Entering the first subscription URL requires the **full UFI-TOOLS version on the device**, with package name `com.minikano.f50_sms`. The PE version installed on a phone cannot replace it.

Download the [recommended UFI 4.1.2 APK](https://github.com/zj-yzj/u30pro-vpn-controller/releases/download/ufi-tools-archive-20261019/UFI-TOOLS_WEB_V4.1.2_20260819_1854.apk), or the [complete installation-material ZIP](https://github.com/zj-yzj/u30pro-vpn-controller/releases/download/ufi-tools-archive-20261019/U30Pro-UFI-Tools-Archive-20261019.zip) ([separate archive Release](https://github.com/zj-yzj/u30pro-vpn-controller/releases/tag/ufi-tools-archive-20261019)). The ZIP's `UFI-Tools/` directory retains all original files, including Windows installer 1.6, the 4.1.1/4.1.2 APKs, the original manual and helper tools. Documentation, a manifest and checksums are added separately. If UFI already works, skip installation rather than reinstalling over it.

UFI-TOOLS is developed by **Minikano / kanoqwq** and uses the MIT license. Other helper programs in the archive retain their own author and license information; the archive is not given a blanket MIT license. See the [UFI-TOOLS repository](https://github.com/kanoqwq/UFI-TOOLS) and [official manual](https://github.com/kanoqwq/UFI-TOOLS/blob/http-server-version/User_Doc.md) for the author's instructions. UFI's installer and this project's VPN installer are separate tools.

| Installation route | Files to use |
| --- | --- |
| Windows one-click installer | Extract the complete ZIP, open `UFI-TOOLS一键安装器1.6.exe` inside `UFI-Tools/`, connect the powered-on U30 Pro, and follow the author's installer prompts to confirm the target and install |
| Manual ADB, pinned to 4.1.2 | Download the standalone 4.1.2 APK, use the archive's `adb-fastboot` directory or official Platform-Tools, and follow the commands below |

1. Keep the U30 Pro's stock connection working, and use one route above to install UFI on the **U30 Pro itself**.
2. Complete the permissions and service setup in the UFI instructions. Enter the device's actual gateway address; this project's supported address is `192.168.0.1`.
3. Connect a phone or computer to the U30 Pro network and open `http://192.168.0.1:2333/`. Log in with your own UFI password and the stock management password requested by the page.
4. If advanced features are not enabled yet, open “高级功能” (Advanced features), choose “添加高级功能” (Add advanced features) and wait for completion. Keep the file-sharing feature it depends on enabled as directed by UFI.
5. Confirm that the **authenticated UFI RootShell used by plugins** can run `id` and that its output contains `uid=0(root)`. After a reboot, confirm that the UFI service and this root channel still work.

<details>
<summary>Install and open UFI through USB ADB</summary>

Static inspection of the directly downloadable `UFI-TOOLS_WEB_V4.1.2_20260819_1854.apk` confirmed package name `com.minikano.f50_sms`, version `4.1.2`, version code `20260819` and size `22936046` bytes. The commands below apply only to this APK; follow the author's installer prompts for the Windows one-click route. The APK's SHA-256 is `4831924e9a7cea2184dc2b515d816c9182e4e6fee44111ea62b7d8e9acf28347`. This checks the archived file's identity; it does not mean this project has verified the original author's signature.

Confirm the target device first and replace `SERIAL` with its serial number. If UFI is already working, skip installation rather than reinstalling over it:

If your terminal is currently in the ADB tools directory, copy the downloaded APK there first, or replace the quoted filename below with the APK's full path.

```sh
adb -s SERIAL install "UFI-TOOLS_WEB_V4.1.2_20260819_1854.apk"
adb -s SERIAL shell am start -n com.minikano.f50_sms/.MainActivity
```

This opens UFI's own interface. Set the gateway and start the service as directed by its author. If the stock round screen is difficult to use, use an existing trusted ADB screen-mirroring tool. Do not try to start UFI's unexported background services directly from outside the app.

Complete notification, usage-access and battery-optimization settings as needed on your device. For the inspected 4.1.2/API 35 environment, the corresponding ADB operations are:

```sh
adb -s SERIAL shell pm grant com.minikano.f50_sms android.permission.POST_NOTIFICATIONS
adb -s SERIAL shell appops set com.minikano.f50_sms GET_USAGE_STATS allow
adb -s SERIAL shell dumpsys deviceidle whitelist +com.minikano.f50_sms
```

These set notification permission, the usage-access app-op and the battery-optimization whitelist, respectively. Do not use `pm grant` in place of the last two. SMS permission is not needed for the VPN features. These commands were compiled from the package declarations and the author's deployment instructions; installation was not rerun on a new device for this documentation.

</details>

<details>
<summary>Check the RootShell channel actually used by plugins</summary>

Open UFI in a computer browser and log in. After completing Add advanced features, press F12 on **that page**, open Console and run this read-only check:

```javascript
await runShellWithRoot('id', 10000)
```

The result must contain both `success: true` and an actual `uid=0(root)` in `content`. It reuses the current UFI login; you do not need to read or paste a token. If the function is missing, check the page and version first. For socket or permission errors, check advanced features and file-sharing state. `success: true` alone does not establish UID 0.

See [UFI's upstream requests.js](https://github.com/kanoqwq/UFI-TOOLS/blob/http-server-version/app/frontEnd/public/script/requests.js#L559) for the function and authenticated request.

</details>

Opening the UFI page confirms only that the web service is reachable. A working standalone TTYD web terminal also does not replace step 5. Check the plugin channel using [UFI's RootShell interface documentation](https://github.com/kanoqwq/UFI-TOOLS/blob/http-server-version/User_Doc.md#123-功能内容). Check UFI advanced features and Magisk separately: the current VPN installer needs both working. Resolve missing UFI root access before installing the VPN. The UFI author also states that the Pro series must be rooted first; see [issue #144](https://github.com/kanoqwq/UFI-TOOLS/issues/144).

The VPN controller's installation packages provide this project's plugin, retain UFI's author attribution and do not embed a UFI installer or rooting tools. The UFI installation materials and B23 toolkit are provided as separate third-party historical archives.

## 4. Install v1.3.0

Check the [installation requirements](install.en.md#before-installation), then open the [v1.3.0 Release](https://github.com/zj-yzj/u30pro-vpn-controller/releases/tag/v1.3.0). Download `U30ProVPN.txt` and choose one installation route:

| Route | What to do |
| --- | --- |
| Magisk ZIP | Download `U30ProVPN-1.3.0-Magisk.zip`, transfer it to the U30 Pro, and select it in Magisk under Modules → Install from storage |
| Android APK | Install `U30ProVPN-1.3.0-Installer.apk` on the U30 Pro, open “U30 Pro VPN 安装,” tap “安装模块” (Install module) and grant root |

Before manually selecting the ZIP, confirm that Magisk has no installed or pending-reboot module with the same ID: both `/data/adb/modules/codex_u30pro_vpn` and `/data/adb/modules_update/codex_u30pro_vpn` must be absent. Preserve uncertain directories and review them first. Magisk can replace same-ID staged content before this project's checks run; see [Before installation](install.en.md#before-installation). Do not run both installation routes at the same time.

During first preparation, the U30 Pro downloads the official Mihomo core, about **21.9 MB**. The device must be able to reach the official GitHub assets. Wait for “模块已暂存” (Module staged), then reboot. Open the installer's “打开控制” (Open control), or use “操作 / Action” on the Magisk module card.

The first launch should show **VPN off, no nodes and no saved subscription**. This means installation is waiting for configuration; it is not an installation failure. See [Installation and recovery](install.en.md) for both routes, download verification and troubleshooting.

## 5. Add your subscription and start using it

1. In UFI, use “插件功能 → 添加插件(.txt文件) → 提交” (Plugins → Add .txt plugin → Submit) to import the downloaded `U30ProVPN.txt`.
2. Open the VPN plugin's Subscription tab, enter your own working HTTPS subscription URL, choose Save and update, and wait for nodes to appear.
3. Select a node in the round-screen app or web panel and enable VPN yourself. Saving a subscription does not enable VPN automatically.
4. To check latency, tap Refresh at the bottom of the round-screen Nodes page for a full-list test, or hold an ordinary node, swipe left and tap Test for that node. There are no scheduled all-node tests.

You can then use the device's round screen to switch VPN on or off, select nodes and update a saved subscription. Change the URL in the UFI web panel. More → Shortcut enables an optional floating button; once authorized, it can reopen controls after you leave the controller.

## 6. Check from a device connected to the hotspot

Connect a phone to the U30 Pro's Wi-Fi, temporarily disabling the phone's own VPN and mobile data. First confirm that the stock management page opens. Then enable VPN on the U30 Pro and visit both a site that needs the selected proxy node and a site you normally use. Both must work to confirm actual hotspot-client connectivity.

Next, disable VPN and confirm that stock connectivity returns. Enable it again, select a node and reboot the U30 Pro. Wait for VPN to report running, then repeat the client checks. Later boots read the saved power state; do not mistake a client's direct connection during startup for a working VPN.

No recorded latency does not mean a node is unusable; run a manual test first. Exiting the controller also leaves VPN running. See [Usage](usage.en.md) for daily controls and [Installation and recovery](install.en.md) for failed installation, uninstalling and retained data.
