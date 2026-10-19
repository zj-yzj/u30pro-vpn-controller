# U30 Pro native VPN controller

Version 1.3.0-alpha.1 (Android version code 7) is a small Java touchscreen companion for the root-side controller in this repository. It targets the U30 Pro's physically circular 240 × 240 panel. Interactive controls fit inside a centered 164 × 164 pixel safe area.

This directory contains the everyday control app and its standalone build. The separate [Android installer](../installer-app/README.md) and Magisk ZIP are published as a fixed-profile first-install alpha. Follow [Installation and recovery](../docs/install.en.md) ([中文](../docs/install.md)); neither roots/flashes a device or upgrades a private deployment. No signing private key or real subscription is supplied.

Use Magisk's module Action or the installer's Open control button after reboot. First installation has an empty node list and VPN off. Enter your own URL through the existing UFI plugin before enabling; the circular Subscription page updates only an already saved URL. Optional overlay consent is still a separate user action. This alpha has no complete fresh-device physical installation/network/teardown acceptance.

## On-device controls

- Home: enable or disable VPN, read its actual state and current exit, and open Nodes or More.
- Nodes: tap a row to select it. Left-swipe an ordinary node to reveal its separate test button; right-swipe closes it. Only that node is tested. Auto uses the most recent complete manual measurement to prefer an exit and has no individual test button.
- Bottom Refresh tests the whole list. Test actions require VPN to be running and never start it implicitly. Progress and delay results appear in the list.
- More: update the subscription already saved by the root-side manager, configure the optional Home-screen shortcut, or exit. Enter a new subscription URL through the existing web manager.

External node measurements are initiated manually. Foreground refreshes and the optional shortcut poll local status; they do not initiate measurements, change a node or download a subscription. Normal VPN traffic still uses the network. Exiting the activity leaves the gateway's selected state unchanged.

## Permissions and trust

The APK requests Android overlay, foreground-service, special-use foreground-service and boot-completed permissions. It has no INTERNET permission and does not request direct root access. The optional shortcut requires explicit user authorization, appears only on the unlocked factory Home screen, and hides while the screen is off or another page is focused.

Commands use the local abstract Unix socket and existing encrypted request protocol. The app requires a root peer. The root-side bridge must independently validate the caller's UID, package identity and pinned APK. A new build or signing identity therefore needs corresponding root-side trust provisioning. Copying an APK onto an arbitrary rooted hotspot is insufficient.

Android updates also require the same signing certificate as the installed app. No previously used private key is supplied. Release signing uses a separate identity outside the checkout. The first installer accepts only its fixed release APK and provisions the actual UID/path plus pinned digest; it refuses unknown existing same-package apps and does not migrate the old device signer. Coordinate any later signer/APK-pin change with a reviewed installation/recovery procedure.

## Build

Requirements: PowerShell, JDK 17 or later, Android API 35 `android.jar`, and Android build-tools 35 or a compatible later version. Node.js 18 or later is optional for geometry checks. No Gradle, Maven, npm dependencies or build-time downloads are required.

With `JAVA_HOME` and `ANDROID_SDK_ROOT` set to your own JDK and SDK directories, run from this directory:

```powershell
./build.ps1
./build.ps1 -TestOnly
```

`ANDROID_HOME` is accepted when `ANDROID_SDK_ROOT` is absent. SDK discovery uses `platforms/android-35/android.jar` and the highest installed stable build-tools version. All paths can instead be explicit:

```powershell
./build.ps1 -JavaHome $jdkHome -AndroidJar $androidJar -BuildTools $buildTools -OutputDir $outputDir
```

The default output directory is `build/`. Parent directories are created automatically. Each run uses a fresh work directory to prevent stale classes from entering an APK. `TestOnly` compiles Android sources and runs the crypto and swipe host tests without creating an APK.

The default full build produces **`U30ProVPN-unsigned.apk`** and `build-manifest.json`. It does not discover, generate or read a signing identity. An unsigned APK cannot be installed normally. To sign using an identity you supply explicitly:

```powershell
./build.ps1 -JavaHome $jdkHome -AndroidJar $androidJar -BuildTools $buildTools `
  -Keystore $keystore -KeystorePasswordFile $passwordFile -KeyAlias $alias
```

All three signing parameters are required together. The password file is passed directly to Android's APK signer and is not printed by this script. Use a keystore whose key password matches its store password. Explicit signing additionally produces `U30ProVPN.apk` and a public signature-verification report. Keep your keystore and password file outside the published source tree.

## Verification

```powershell
node ./android/tests/round-screen-geometry.mjs
./tests/BuildScriptSmokeTest.ps1 -JavaHome $jdkHome -AndroidJar $androidJar `
  -BuildTools $buildTools -OutputDir $freshOutputDir
```

The smoke test requires a fresh output directory, runs both TestOnly and a full unsigned build, checks the artifact checksum, verifies the absence of an APK signature and checks that no signing directory was created. Host tests cover real Java encryption and the swipe intent state machine. Geometry checks bound the round-screen controls and shortcut corners; they do not substitute for Android lifecycle, peer-credential or physical touch validation.

`android/tests/round-screen-preview.html` contains no screenshot or embedded image. This offline viewer lets you select your own local screenshot, apply a circular mask and inspect safe-area guides without uploading the image.

Original project code is provided under the repository's MIT license. See repository third-party notices for separately licensed components.
