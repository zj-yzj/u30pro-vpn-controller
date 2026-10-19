# Frequently asked questions

[中文](faq.md) · [English](faq.en.md) · [Home](../README.en.md) · [Usage](usage.en.md)

For everyday controls, see [Usage](usage.en.md). For the supported device profile and installation steps, see [Installation and recovery](install.en.md).

## 1. Can I use it immediately after downloading source, building an APK or importing the plugin?

Use one of the installation packages from the [Release](https://github.com/zj-yzj/u30pro-vpn-controller/releases/tag/v1.3.0): the Magisk ZIP or the installer APK. Both go on the U30 Pro and install the same module. The module prepares the backend and controller; importing `U30ProVPN.txt` adds the web interface to your existing UFI.

The device downloads the official Mihomo core during installation. After reboot, enter your own subscription through UFI and enable VPN. To explore the interface without a device, use the [offline preview](usage.en.md#preview-on-a-computer).

If UFI is missing, download the [complete installation materials](https://github.com/zj-yzj/u30pro-vpn-controller/releases/download/ufi-tools-archive-20261019/U30Pro-UFI-Tools-Archive-20261019.zip) or [standalone 4.1.2 APK](https://github.com/zj-yzj/u30pro-vpn-controller/releases/download/ufi-tools-archive-20261019/UFI-TOOLS_WEB_V4.1.2_20260819_1854.apk), then choose the Windows installer or manual ADB in the [preparation guide](setup.en.md#3-install-ufi-and-check-its-advanced-features). Skip installation if UFI already works.

## 2. Does it support other U30 Pro firmware, hotspots or OpenWrt?

This release supports the [fixed device profile](install.en.md#before-installation). Other firmware, root implementations, hotspots and OpenWrt need their own integration. The APK's `minSdk 26` is not the installer's supported Android range. See [Device integration](device-integration.en.md) for the network layout and porting requirements.

## 3. What if the local service is disconnected, trust fails or controls are disabled?

After reboot, allow startup to finish and check that the Magisk module is enabled. Controls are also temporarily disabled during an operation or while fresh status is unavailable.

For a persistent connection failure, check that `webbridge` is running and the backend is initialized. A trust failure needs a check of the installed package, UID, APK path and SHA-256 record, plus private-directory ownership and permissions. Rebuilding changes the APK digest; changing its signer also affects Android updates. Preserve the checks and report the error category instead of loosening permissions or exposing a TCP management port.

## 4. Does exiting the app, closing the page or disabling the shortcut stop VPN?

No. Use **Disable VPN** on Home or the web panel, wait for the running state to update, then check stock connectivity from a hotspot client.

The VPN setting and node choice are saved for later boots. When enabled, VPN starts after Android and the gateway are ready, leaving a cold-start window. Wait for the running state before starting tasks that need the VPN.

## 5. How do swipe tests work, and why are the two Refresh buttons different?

Hold an ordinary node row and swipe left, then tap **Test** (“测速”). Swipe right to close it. This also works on the selected node, while `Auto` has no individual Test button. A single-node test leaves the selection unchanged.

The bottom **Refresh** on the round-screen Nodes page tests the whole list. Top-right **Refresh** in the web panel reads existing state. Tests require VPN already running and do not enable it automatically. See [Node selection and manual measurements](usage.en.md#node-selection-and-manual-measurements).

## 6. Are nodes tested automatically? How does Auto work, and what do missing/failed results mean?

There are no periodic external node tests. A complete manual round can choose an inner `Auto` exit from successful results, while preserving your outer manual selection. If your outer selection is `Auto`, that choice affects the current exit. Local status polls and the gateway supervisor continue to check runtime state.

Results are latency in milliseconds to `https://www.gstatic.com/generate_204`, expecting HTTP 204. Missing means unmeasured; failed means that check did not pass. A failed check can be specific to the target or current network. Results are not bandwidth measurements, and refreshing status does not renew them.

## 7. Which subscriptions work? Why can my browser open a URL while device updates fail?

Supported input is Clash/Mihomo-style inline `proxies` or a list of node mappings. Other client formats and remote-provider configurations may need conversion first; Mihomo must also support the supplied protocols and fields.

The downloader requires HTTPS on port 443, valid TLS, public IPv4 resolution and connections, and no URL userinfo or fragment. Private addresses, IPv6-only sources, unusual ports, redirects or a different certificate/network path can explain why a URL works in a computer browser but fails on the device. Normalization also limits size, expansion depth and node count and rejects dangerous external file references.

Add or change the URL in UFI's Subscription tab. The round screen updates the saved URL. Keep URLs private because they may contain credentials.

## 8. Will an update failure lose nodes or stop VPN?

Download, parse and validation failures keep the previous configuration. If applying the configuration fails, the manager attempts to restore it. For an incomplete-recovery error, refresh and check the actual running state and exit before continuing.

If a selected node disappears from a successful update, selection falls back to `Auto` with a notice. Updating while VPN is enabled also checks the current exit's HTTPS connectivity once; this does not enable periodic tests.

## 9. Why is the shortcut missing? Are system permission and a notification required?

Open **More → Shortcut**, allow display over other apps in system settings and enable the shortcut. Return to unlocked stock Home and allow time for status to update. It hides in the controller, on other pages, on the lock screen and with the display off.

The shortcut uses an Android foreground service and notification. If permission is allowed but it still does not appear, Home detection, firmware restrictions or the local service may need checking. You can use Magisk's Action or the installer's Open control button meanwhile. See [Home overlay shortcut](usage.en.md#home-overlay-shortcut).

## 10. Why does UFI login trigger a screen-lock prompt? Does the native app need it?

UFI/stock WebUI login may trigger the device's existing screen-lock prompt. This project uses that login and leaves stock locking in place.

The native controller uses a local Unix socket and bridge, so it does not need WebUI login. It has no `INTERNET` permission and does not request root directly. The separate installer does request root for module installation.

## 11. What tools are required, and why can't an unsigned or newly rebuilt APK be used directly?

See [Building](building.en.md) for toolchain versions and commands. Web builds use Node.js/npm, backend builds use Go, and Android builds use PowerShell, JDK and Android platform/build tools. Full interoperability tests also need `java` and `javac`.

The native build defaults to an unsigned APK. Installation needs signing; updating an existing package normally needs the same certificate. The local bridge separately pins the exact APK digest, which changes when you rebuild. See the [Native guide](../native-manager/README.md) for signing and trust setup.

## 12. How should I report or redistribute safely? Does MIT cover the core and all dependencies?

[Report a problem](https://github.com/zj-yzj/u30pro-vpn-controller/issues) with the model, firmware, version, steps and error category. Review logs and screenshots before sharing: remove subscription URLs, credentials, keys, providers, state, trust and backups.

Original code and documentation use [MIT](../LICENSE). Mihomo, UFI-TOOLS, Magisk, Android and other dependencies retain their upstream licenses. The separate B23 root and UFI material archives are not given a blanket license under this project's MIT terms. UFI-TOOLS itself is published by Minikano / kanoqwq under MIT; other bundled components retain their own licenses. See [Third-party notices](../THIRD_PARTY_NOTICES.md) before redistribution.
