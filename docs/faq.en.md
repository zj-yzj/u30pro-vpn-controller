# Frequently asked questions

[中文](faq.md) · [English](faq.en.md) · [Home](../README.en.md) · [Usage](usage.en.md)

These answers describe `v1.3.0-alpha.1` (circular control version code 7). First installation requires MU5358, arm64, API 35, firmware display `U30ProV1.0.0B23`, existing Magisk version code at least 28100 and SELinux already Permissive. Images remain illustrations/mock previews; no complete fresh-device acceptance has been performed for this alpha.

## 1. Can I use it immediately after downloading source, building an APK or importing the plugin?

Downloaders can use a Magisk ZIP or Android installer APK from the [Release](https://github.com/zj-yzj/u30pro-vpn-controller/releases/tag/v1.3.0-alpha.1), following [Installation and recovery](install.en.md). Both require compatible Magisk on the U30 Pro; neither obtains root, flashes firmware nor upgrades private deployments. Install the installer APK on the U30 Pro itself.

The installer prepares project components, a device-generated key and current APK trust. The core is acquired online from official upstream, not embedded; offline first install is unavailable. First boot is empty with VPN off. Existing UFI needs manual `U30ProVPN.txt` import and your own URL. Building an unsigned APK or importing the plugin alone does not install the gateway; reference scripts are not the installer entry. See [integration boundaries](device-integration.en.md).

To inspect the interface, run the [offline preview](usage.en.md#preview-on-a-computer) without a real device or subscription.

## 2. Does it support other U30 Pro firmware, hotspots or OpenWrt?

This alpha accepts only the [fixed installation profile](install.en.md#before-installation). Other firmware/hardware, root implementations and OpenWrt are outside the declared scope. Android `minSdk 26` does not expand installer compatibility. Do not lower SELinux or spoof properties to bypass checks; stock Home, SFP, interfaces and rules still have fixed system assumptions.

The gateway reference fixes `br0`, `192.168.0.0/24`, `U30VPN`, particular routing tables/rules and `/proc/net/sfp/enable`. A port requires reviewing object ownership, DNS, stop restoration, startup and power-loss recovery. See the [compatibility matrix](../README.en.md#compatibility-and-prerequisites).

## 3. What if the local service is disconnected, trust fails or controls are disabled?

Distinguish connection from trust: a connection failure may mean `webbridge` is not running, the backend is uninitialized or local endpoints do not match. Trust failures require checking the current package, UID, installed APK path, SHA-256 record and directory ownership/permissions. Rebuilding changes the APK digest; changing signer also affects Android package updates.

Wait for fresh local state and retry. Busy operations or stale state temporarily disable controls. Do not work around failures by exposing TCP management, removing UID/APK validation or loosening private permissions. Installing an APK does not provision the required backend on an arbitrary rooted device.

## 4. Does exiting the app, closing the page or disabling the shortcut stop VPN?

No. These actions affect the interface. Explicitly tap Disable VPN (“关闭 VPN”) on Home, wait for actual-state confirmation, then verify stock networking. Desired state, actual running state and real client connectivity are different observations.

Power state and node selection are persisted and later boots use saved desired state. Enabled boots have a cold-start window before Android/gateway readiness, without a continuous-protection promise from power-on. A saved setting is not evidence of reboot acceptance.

## 5. How do swipe tests work, and why are the two Refresh buttons different?

In native Nodes, hold an ordinary row and swipe left, then tap its separate Test (“测速”) button; swipe right to close. There is no fixed long-press duration. Selected rows work too; `Auto` has no individual test button. A one-node test changes no node/group selection.

Native Nodes → bottom Refresh manually tests the current subscription list. Web top-right Refresh only reads existing state; the current web panel has no full-list or one-node test button. Both native tests require VPN enabled and actually running and never enable it implicitly. See [node actions](usage.en.md#node-selection-and-manual-measurements).

## 6. Are nodes tested automatically? How does Auto work, and what do missing/failed results mean?

There are no periodic external node tests. The provider has no `health-check`; `VPN` and `Auto` are `select`. A complete manual round can prefer an inner `Auto` exit from that round's successful results, while preserving the outer manual choice. If outer selection is already `Auto`, inner changes affect the active exit.

Foreground, web and optional shortcut interfaces still poll local state; the supervisor checks process/TUN/routes. Subscription application retains necessary checks. Normal VPN, DNS and downloads still use the network; absence of automatic tests does not mean absence of network access.

Results are latency samples in ms against fixed `https://www.gstatic.com/generate_204`, expecting HTTP 204, not bandwidth. Missing means unmeasured; a historical failure describes that check and does not prove all current applications fail. Status polling does not renew old measurement timestamps.

## 7. Which subscriptions work? Why can my browser open a URL while device updates fail?

Normalization supports Clash/Mihomo-style inline `proxies` or node-mapping lists, not arbitrary client formats/remote providers. It bounds size, depth and node count and rejects dangerous file references. Device Mihomo must still validate protocol/field support.

Downloads require HTTPS, port 443, no URL userinfo/fragment, restricted public IPv4 connections and valid TLS. Device DNS, certificate stores, network paths, redirects or formats can differ from a computer browser; browser access does not establish successful device application.

The round screen only updates a saved URL; add/change one in UFI's Subscription tab. URLs may contain credentials and must not appear in public reports. See [Subscriptions](usage.en.md#subscriptions).

## 8. Will an update failure lose nodes or stop VPN?

Download/parse/validation failures preserve the prior configuration. Application has backup, transaction digest, recovery and lock checks and attempts restoration. If a selected node disappears from a successfully updated list, selection falls back to `Auto` with a notice.

Recovery itself may fail. After incomplete-recovery/application errors, refresh and inspect actual running state/exit; do not assume VPN stayed active or restoration succeeded. Updating with VPN enabled performs a one-time exit HTTPS check without enabling periodic measurements. First installation has separate recovery records; subscription transactions provide no old-deployment upgrade capability.

## 9. Why is the shortcut missing? Are system permission and a notification required?

Enable it under More → Shortcut and explicitly allow display over other apps in system settings. The implementation displays it only on unlocked stock Home and hides it inside the control app, other pages, lock screen or with the display off. Failed Home recognition or unavailable local services also hide it. Allow time for a local read after returning Home.

The shortcut uses an Android foreground service and its notification, polling local state/visibility. An unavailable permission page, firmware restrictions or changed Home detection need review; the app cannot bypass system authorization. Disabling the shortcut leaves VPN unchanged. See [overlay usage](usage.en.md#home-overlay-shortcut).

## 10. Why does UFI login trigger a screen-lock prompt? Does the native app need it?

UFI/stock WebUI login may trigger the device's existing screen-lock prompt. This panel does not implement the outer authentication or stock page. The project neither modifies nor bypasses stock locking.

The native app uses a local Unix socket and root bridge without WebUI login. It has no `INTERNET` permission and does not request root directly. The web panel still needs UFI's existing authenticated root interface; encrypted requests cannot replace login.

## 11. What tools are required, and why can't an unsigned or newly rebuilt APK be used directly?

Web: Node.js 18+ and npm. Backend: Go 1.27+; full interoperability tests need `java`/`javac`. Native: PowerShell, JDK 17+, Android Platform 35 and Build Tools 35 or a compatible later version. No Gradle/Maven is required; toolchains are not bundled.

`build.ps1` defaults to an unsigned APK and manifest. Signing explicitly requires keystore, password file and alias together, kept outside the repository. Android package updates normally need the same certificate; root bridge trust separately pins the exact APK digest. Both requirements apply. See [Building](building.en.md) and the [native build guide](../native-manager/README.md). Host/geometry tests do not establish real touch, Android lifecycle or network acceptance.

## 12. How should I report or redistribute safely? Does MIT cover the core and all dependencies?

Reports can include model/firmware, source/app version, reproduction, desired/actual state and sanitized error categories. Do not publish URLs, node credentials, private keys, providers, state, trust, raw logs, backups or unchecked screenshots. Publication scanning catches common patterns and does not replace manual review. Report through [GitHub issues](https://github.com/zj-yzj/u30pro-vpn-controller/issues).

Original code/documentation use [MIT](../LICENSE). node-forge, yaml.v3, Go and others keep their own licenses. Mihomo, UFI-TOOLS, Magisk and Android are separately acquired projects/environments whose programs are not bundled. MIT does not replace third-party terms; future bundling/redistribution must meet each obligation. See [third-party notices](../THIRD_PARTY_NOTICES.md). This project is independent of those upstreams.
