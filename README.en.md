# U30 Pro VPN Controller

[中文](README.md) · [English](README.en.md)

Manage VPN on a rooted ZTE U30 Pro running stock Android, either from its round touchscreen or through an existing UFI web interface. This project contains an **Android controller, Go backend, UFI web plugin and first-install alpha**. A [Mihomo](https://github.com/MetaCubeX/mihomo) executable downloaded separately on the device from official upstream handles proxy traffic.

![Illustration of the round-screen app, web panel and device manager](docs/assets/overview.svg)

> The native UI, overview and architecture images on this page are explanatory illustrations. The web image uses offline mock data. None is an on-device screenshot or evidence of a new device deployment test.

**Current release: `v1.3.0-alpha.1`; round-screen control app Android version code 7.** The layout targets a 240 × 240 circular framebuffer, with main controls in a centered 164 × 164 safe area.

**[Download the Magisk ZIP or Android installer APK](https://github.com/zj-yzj/u30pro-vpn-controller/releases/tag/v1.3.0-alpha.1) → [follow installation and recovery](docs/install.en.md).** Both routes use the same Magisk module and require existing Magisk 28.1+, MU5358, arm64, API 35, firmware display `U30ProV1.0.0B23`, and SELinux already Permissive. They do not obtain root, flash firmware or overwrite existing private deployments. First install is empty with VPN off; existing UFI-TOOLS is needed to enter your subscription, and core acquisition requires Internet access.

**This is a first-install prerelease without complete fresh-device installation, reboot, client-networking and uninstall acceptance.** Host verification and illustrations do not establish physical compatibility; see Release notes for actual tests. A cold-boot window exists before a previously enabled gateway becomes ready; continuous tunnel/blocking protection from power-on is not promised.

Before manual ZIP installation, verify that Magisk has no installed or pending-reboot `codex_u30pro_vpn` module. Magisk can replace same-ID staging before project checks; preservation of that prior stage is not promised. The APK route additionally prechecks directories before the CLI. See [prerequisites](docs/install.en.md#before-installation).

## Contents

- [Features and action semantics](#features-and-action-semantics)
- [Interface previews](#interface-previews)
- [Compatibility and prerequisites](#compatibility-and-prerequisites)
- [Getting started](#getting-started)
- [Installation and recovery](docs/install.en.md)
- [Architecture and source layout](#architecture-and-source-layout)
- [Building and verification](#building-and-verification)
- [Privacy and security boundaries](#privacy-and-security-boundaries)
- [Questions and support](#questions-and-support)
- [License and upstream projects](#license-and-upstream-projects)

## Features and action semantics

| Entry point | Existing capability | Boundary |
| --- | --- | --- |
| Native Home | Toggle VPN; read actual running state and current exit | Exiting the app leaves VPN running; explicitly use the power control to stop it |
| Native Nodes | Tap to select; bottom Refresh manually tests the whole list | VPN must already be running; tests never enable it implicitly |
| Per-node action | Hold an ordinary row and swipe left to reveal Test; swipe right to close | Tests only that node without changing nodes/groups; `Auto` has no individual Test button |
| Auto selection | Prefers an inner `Auto` exit from successful results in a complete manual round | No periodic `url-test` or provider `health-check`; preserves the outer `VPN` manual selection |
| Native More | Update the saved subscription, configure a draggable Home shortcut, exit | Add/change URLs in the web panel; overlay needs system authorization |
| UFI web panel | Toggle VPN, search/select nodes, save/update or reuse a saved subscription | Requires UFI's existing authenticated root interface; top-right Refresh only reads status |

“Test” measures **latency in milliseconds** to a fixed HTTPS connectivity target, not download bandwidth. A failed check does not prove the node is unusable for every application. Delay/connectivity fields may be empty before a measurement exists.

Foreground interfaces and the optional shortcut poll local status. The gateway supervisor checks process, TUN, routing and protection state. These operations do not automatically test the entire node list. Applying a subscription with VPN enabled still checks the current exit's HTTPS connectivity once; normal VPN traffic also uses the network. See [Usage](docs/usage.en.md#measurements-auto-and-status-reads).

## Interface previews

### Native round screen

![Native UI illustration: Home, a swiped node and a per-node test](docs/assets/native-ui.svg)

*Drawn from the source layout, not an on-device screenshot. The current interface is Chinese; English documentation does not imply an English app interface.*

### UFI web panel

![Offline web preview showing mock state and nodes](docs/assets/web-preview.png)

*Offline mock preview: nodes, keys and state are simulated; it does not connect to a device or real subscription.*

## Compatibility and prerequisites

| Item | Current scope | Integrator must still verify |
| --- | --- | --- |
| Device | device/model `MU5358`, display `U30ProV1.0.0B23` | No other model/firmware support claim or fresh-device complete acceptance |
| System | Stock API 35, arm64, Magisk version code at least 28100, SELinux already Permissive | Does not lower SELinux or install Magisk; no generic support for other root implementations |
| Android app | `minSdk 26`, `targetSdk 35`, 240 × 240 round screen | SDK declarations do not expand the installer's fixed profile; Home detection and overlay depend on the system |
| Backend | Linux/arm64 Go helpers; fixed `/data/adb/u30pro-vpn` | Root ownership, private permissions, configuration, state, keys and startup recovery |
| Core | Fixed official Mihomo v1.19.32 downloaded during preparation with size/hash verification | No core embedded in ZIP/APK; no offline first install; see [core acquisition](docs/install.en.md#the-core-is-downloaded-online) |
| Web | Existing UFI-TOOLS with root management enabled | Existing authentication and `runShellWithRoot`; UFI and authentication assets are not included |
| Network | Reference fixes objects including `br0`, `192.168.0.0/24` and `U30VPN` | SFP, DNS, ownership of rules/tables, IPv4 forwarding and recovery; see [reference assumptions](gateway/reference/README.en.md#explicit-device-assumptions) |
| Install/upgrade | Magisk ZIP and Android entry embedding the same ZIP, first-install alpha | Refuses private runtimes, unknown same-package applications and conflicts; no old-deployment upgrade |

Acquire UFI-TOOLS, Magisk and build tools separately; the device obtains Mihomo directly from official upstream. This repository/project installers do not bundle those programs, firmware, SDKs, authentication assets, private keys or original device configuration. Their respective licenses remain applicable.

## Getting started

**First install:** download `U30ProVPN-1.3.0-alpha.1-Magisk.zip` or `U30ProVPN-1.3.0-alpha.1-Installer.apk`, plus `U30ProVPN.txt`, from [Release Assets](https://github.com/zj-yzj/u30pro-vpn-controller/releases/tag/v1.3.0-alpha.1). Follow [Installation and recovery](docs/install.en.md) to check prerequisites/checksums and use one route. After reboot VPN starts off; manually import the plugin into existing UFI, enter your subscription, then explicitly enable VPN. The Android installer belongs on the U30 Pro itself.

**Inspect the interface:** run the offline preview on a development computer. No device, root or subscription is needed.

```sh
npm ci
npm test
npm run build
npm run test:preview
npm run preview
```

Open the printed address, normally `http://127.0.0.1:18771/`. Outputs default to `build/web-ui/`; see the [web plugin guide](web-manager/ui/README.md) for output options and UFI import.

**Already integrated device:** open “VPN 控制” on the round screen, check actual state, then toggle VPN or select a node. Nodes → Refresh runs a full manual test; swipe an ordinary node left and tap Test for one node. More → Subscription updates a saved URL; use UFI to add a new one. See [Usage](docs/usage.en.md).

**Porting source:** read [device integration boundaries](docs/device-integration.en.md), the [gateway overview](gateway/README.en.md) and [fixed-layout reference](gateway/reference/README.en.md). The reference directory itself is not the installation entry. Host tests cannot verify actual DNS, forwarding, failure blocking, stock-network restoration after stopping, or power-loss recovery.

## Architecture and source layout

![Architecture illustration: two control interfaces cross trust boundaries to the manager, which controls independent Mihomo and gateway components](docs/assets/architecture.svg)

*Explanatory diagram, not a device-state screenshot.*

The web panel calls `webctl` through UFI's authenticated root interface. The native app connects over a local Unix socket to `webbridge`, which validates callers and invokes restricted commands. `webctl` handles state, subscriptions and operation transactions. Independent Mihomo accepts control over a local Unix HTTP controller; gateway scripts manage device network objects. See [Architecture](docs/architecture.en.md).

| Path | Contents |
| --- | --- |
| `native-manager/android/src/` | Native Java app, model, encryption, shortcut and boot receiver |
| `installer-app/` | Separate Android installer: fixed embedded ZIP, user root consent and explicit reboot |
| `release/` | Magisk wrappers and two-entry release build |
| `web-manager/cmd/u30setup/`, `web-manager/internal/installer/` | Fixed-profile checks, core acquisition, installation recovery and owned teardown |
| `native-manager/build.ps1` | Explicit-toolchain build, host tests, optional signing and manifest |
| `native-manager/android/tests/` | Crypto/gesture tests, geometry and local screenshot inspection |
| `web-manager/cmd/webctl/` | Root management program |
| `web-manager/cmd/nativebridge/` | Native bridge, built as `webbridge` |
| `web-manager/cmd/normalize/` | Converts private inline `proxies` into an initial provider, metadata and validation configuration |
| `web-manager/internal/` | Management, storage, download, normalization, protocols and tests |
| `web-manager/device/config.yaml` | Configuration template with no subscription credentials or usable nodes |
| `web-manager/ui/` | Original web interface, assembly, mock preview and logic tests |
| `gateway/reference/` | Fixed-layout control, supervisor and startup templates requiring integration |
| `docs/` | Bilingual usage/FAQ, build, architecture, integration guidance and illustrations |
| `third_party/licenses/` | Separately applicable third-party license texts |

## Building and verification

Web requires Node.js 18+ and npm; the lockfile pins `node-forge@1.4.0`. The Go module requires 1.27.0 or later; source preparation used 1.27.2. Full interoperability tests need `java` and `javac` in `PATH`. Android requires PowerShell, JDK 17+, Android Platform 35 `android.jar`, and Build Tools 35 or a compatible later version. It has no Gradle/Maven dependency or build-time Android toolchain downloads.

After the web commands from the repository root, run:

```sh
cd web-manager
go test -count=1 ./...
go test -count=1 -tags diagnostic ./...
go vet ./...
go vet -tags diagnostic ./...
```

For Android, set `JAVA_HOME` and `ANDROID_SDK_ROOT` in PowerShell and run from the repository root:

```powershell
./native-manager/build.ps1 -TestOnly
./native-manager/build.ps1
node ./native-manager/android/tests/round-screen-geometry.mjs
```

Defaults produce `native-manager/build/U30ProVPN-unsigned.apk` and a build manifest. Unsigned APKs cannot be installed normally. See [Building](docs/building.en.md), [Native](native-manager/README.md) and [Backend](web-manager/README.md) for signing, cross-compilation, output options and smoke tests. Keep signing identities outside the source tree; the script does not locate or generate an existing project's identity.

Verification scope includes host manager/installer-engine tests, web logic, Java crypto/gesture and installer subprocess tests, Android compilation, and signature/file checks. Use the matching Release notes for actual passes and skips; old fixed test counts do not establish that this release is entirely passing. Mock controllers, temporary keys and synthetic nodes do not replace fresh-device acceptance. Missing JDK may skip interoperability checks, so inspect output. See [Building](docs/building.en.md) for the two-entry release build.

```sh
# Run on a publication source copy without node_modules, build output or caches
npm run check:publication
node scripts/check-doc-links.mjs
```

The publication check reports relative paths, line numbers and categories for common sensitive content without printing matched values. It does not replace manual review or prove arbitrary inputs have been sanitized. Public source should contain no generated outputs, caches or runtime data.

## Privacy and security boundaries

- **Subscriptions are processed on the device**, without a public conversion service. URLs, node credentials, providers, state, backups and logs are private runtime data and belong outside repositories and public issues.
- **The circular control app has no `INTERNET` permission and does not request root directly.** It invokes fixed operations through the local bridge. RSA-OAEP-SHA256/AES-256-GCM requests accompany peer UID, package and pinned APK checks. The separate installer APK also has no `INTERNET` permission, but requests user root consent for module installation; its root helper downloads the fixed core online.
- **Web relies on existing UFI trust.** Encryption, freshness and replay checks do not replace UFI authentication or defend against an attacker controlling root or the host web environment. Mihomo uses a local Unix controller; this project adds no LAN TCP management listener.
- **The shortcut needs user permission.** The implementation displays it only on unlocked stock Home and hides it when the display is off, locked or another page is active. Its local reads do not imply periodic external node tests.
- **Subscription updates have operation-level transactions.** Failures preserve/attempt to restore prior configuration; recovery can fail, so inspect actual state. Separate first-install journals do not imply support for old-deployment upgrades. Teardown can defer and private backups remain; see [recovery](docs/install.en.md#uninstalling-and-retained-data).
- **The gateway assumes a fixed baseline.** Reference source retains failure blocking, identity checks and controlled cleanup, whose effect needs validation on the actual system.

## Questions and support

The [FAQ](docs/faq.en.md) covers local services, trust, subscriptions, measurements, overlay permissions, exiting versus stopping, stock WebUI lock prompts and ports. In [repository issues](https://github.com/zj-yzj/u30pro-vpn-controller/issues), provide model, firmware, source/app version, reproduction steps and sanitized error categories. Do not upload real subscriptions, node configuration, private keys, trust/state/provider files, raw logs or unchecked backups.

English: [Install](docs/install.en.md) · [Usage](docs/usage.en.md) · [FAQ](docs/faq.en.md) · [Building](docs/building.en.md) · [Architecture](docs/architecture.en.md) · [Integration](docs/device-integration.en.md). Chinese: [安装](docs/install.md) · [使用](docs/usage.md) · [FAQ](docs/faq.md) · [构建](docs/building.md) · [架构](docs/architecture.md) · [集成](docs/device-integration.md).

## License and upstream projects

Original code and documentation use the [MIT License](LICENSE), under its published terms and without suitability or fault-free-operation guarantees. Third parties retain their licenses; see [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md): node-forge uses the BSD-3-Clause option; yaml.v3 retains file-specific MIT/Apache-2.0 texts; Go binary distributions also require the Go license.

Mihomo, UFI-TOOLS, Magisk and Android are independent upstream projects/integration environments. Their programs, firmware, SDKs, authentication assets and original device signing key are not bundled. Future distributions adding components must meet each one's license obligations. This project is independently maintained and does not represent ZTE, Mihomo, UFI or other upstream projects.
