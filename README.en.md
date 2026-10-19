# U30 Pro VPN Controller

[中文](README.md) · [English](README.en.md) · [Download](https://github.com/zj-yzj/u30pro-vpn-controller/releases/tag/v1.3.0)

VPN controls for a rooted ZTE U30 Pro. Switch VPN on or off, pick a node and test latency on the device's round touchscreen, or open the UFI web panel from a phone or computer to manage subscriptions. Devices connected to the U30 Pro share its VPN exit.

This started with what I wanted from my own U30 Pro, where I have tested it and use it day to day. It keeps stock Android and uses [Mihomo](https://github.com/MetaCubeX/mihomo) for proxy traffic.

![Round-screen control and web management illustration](docs/assets/overview.svg)

## Features

- **On-device control:** switch VPN on or off and see its running state and current exit.
- **Node selection:** tap a node to select it; hold an ordinary row and swipe left to reveal its Test button.
- **Manual tests:** use Refresh at the bottom of Nodes to test the whole list, or test one node. Results appear when the task finishes; there is no periodic node probing.
- **Subscriptions:** add or change a URL in the web panel, then update the saved subscription from either interface.
- **Home shortcut:** enable a draggable VPN button to reopen controls from the unlocked stock Home screen.
- **Saved settings:** keep the VPN power state and node selection for later boots.

Tests measure latency in milliseconds. New nodes have no results until you run a test. Tests require VPN already running; a single-node test leaves your selection unchanged. A complete manual round can choose an inner `Auto` exit while keeping the outer manual selection.

## Installation

Setting up a device for the first time? Start with [Device preparation to working VPN](docs/setup.en.md), covering firmware checks, the B23 rooting record, UFI, installation and your subscription. If Magisk and UFI already work, check the B23 requirements below before installing.

For a matching B23 device that still needs root, read the [B23 rooting record](docs/root-b23.en.md) before downloading the [third-party toolkit archive](https://github.com/zj-yzj/u30pro-vpn-controller/releases/download/b23-rootkit-archive-20261019/U30Pro-B23-RootKit-Archive-20261019.zip). It is archived separately from the VPN installation routes below; skip it if root already works. Its image attempts at boot to enable TCP `5555` ADB, relax authentication and disable update components. The project's MIT license does not cover the toolkit, and it provides no complete stock backup or recovery guarantee. Read the full record before use.

If UFI is missing, download the [UFI 4.1.2 APK](https://github.com/zj-yzj/u30pro-vpn-controller/releases/download/ufi-tools-archive-20261019/UFI-TOOLS_WEB_V4.1.2_20260819_1854.apk), or the [complete archive with Windows installer 1.6, the manual and ADB tools](https://github.com/zj-yzj/u30pro-vpn-controller/releases/download/ufi-tools-archive-20261019/U30Pro-UFI-Tools-Archive-20261019.zip). This separately archives Minikano / kanoqwq's UFI-TOOLS and the supplied materials. Follow [step 3 of preparation](docs/setup.en.md#3-install-ufi-and-check-its-advanced-features); skip installation if UFI already works.

Current release: **[v1.3.0](https://github.com/zj-yzj/u30pro-vpn-controller/releases/tag/v1.3.0)**. Choose one installation entry; both install the same module.

| File | How to use it |
| --- | --- |
| [Magisk module ZIP](https://github.com/zj-yzj/u30pro-vpn-controller/releases/download/v1.3.0/U30ProVPN-1.3.0-Magisk.zip) | Install from local storage in Magisk on the U30 Pro |
| [Android installer APK](https://github.com/zj-yzj/u30pro-vpn-controller/releases/download/v1.3.0/U30ProVPN-1.3.0-Installer.apk) | Install on the U30 Pro, open it and grant root to install the module |
| [UFI web plugin](https://github.com/zj-yzj/u30pro-vpn-controller/releases/download/v1.3.0/U30ProVPN.txt) | Import into existing UFI-TOOLS to enter your subscription |

1. Check the device requirements below, then download one installation package and the plugin.
2. Install the module and reboot the U30 Pro.
3. Import the plugin into your existing UFI page and save your own subscription. The first installation starts with VPN off and no nodes or saved URL.
4. Once nodes appear, enable VPN and use either the round screen or the web panel.

Preparation downloads about **21.9 MB** from official Mihomo upstream on the device; the core is not bundled and first installation needs Internet access. No subscription is included. This version is for a fresh installation; see [Installation and recovery](docs/install.en.md) if you already have a deployment.

### Compatibility and prerequisites

| Item | Required profile |
| --- | --- |
| Device | U30 Pro; device and model properties `MU5358` |
| Firmware | `U30ProV1.0.0B23` |
| System | Stock Android API 35, `arm64-v8a`, SELinux already `Permissive` |
| Root | Existing Magisk 28.1 or newer, version code at least `28100` |
| Web management | Existing UFI-TOOLS with authenticated root access |

The installer checks the device and network environment. It does not obtain root or flash firmware; other firmware needs separate integration. Full steps, troubleshooting and uninstall instructions are in the [English installation guide](docs/install.en.md) / [中文安装说明](docs/install.md).

## Interface previews

### Round-screen control

![Home, a swiped node and a single-node test](docs/assets/native-ui.svg)

Round-screen UI illustration. The app is currently Chinese, with the main controls centered on the 240 × 240 screen.

### Web management

![Offline UFI VPN panel preview](docs/assets/web-preview.png)

Offline web preview with sample nodes.

## A few usage tips

- Exiting the controller leaves VPN running. Tap Disable VPN to stop it.
- Add or change subscription URLs in the web panel; the round screen updates a saved URL.
- Allow display over other apps in system settings when enabling the floating shortcut.
- Web Refresh reads status; Refresh at the bottom of the round-screen Nodes page starts a full-list test.
- After reboot, wait for VPN to report running before starting tasks that need it.

See [Usage](docs/usage.en.md) for the controls and the [FAQ](docs/faq.en.md) for common questions.

## Development

The project consists of a native Android app, a Go manager, a UFI web plugin and Magisk gateway scripts. Both interfaces share node and subscription state. Subscriptions are processed on the device, and Mihomo's controller uses a local Unix socket.

- [Building](docs/building.en.md): build the web interface, Android apps and both installation packages.
- [Architecture](docs/architecture.en.md): how management requests and client traffic pass through the components.
- [Device integration](docs/device-integration.en.md): fixed network layout, trust records and porting requirements.
- [Testing](docs/testing.en.md): device use and automated checks.
- [Contributing](CONTRIBUTING.md): report problems or propose changes.

To [report a problem](https://github.com/zj-yzj/u30pro-vpn-controller/issues), include the model, firmware, version and reproduction steps. Remove subscription URLs and node credentials before sharing logs or screenshots.

## License

Original code and documentation use the [MIT License](LICENSE). Mihomo, Magisk, UFI-TOOLS and other upstream components retain their own licenses; see [Third-party notices](THIRD_PARTY_NOTICES.md) for dependencies and license texts.
