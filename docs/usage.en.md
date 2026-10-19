# Usage

[中文](usage.md) · [English](usage.en.md) · [Home](../README.en.md) · [FAQ](faq.en.md)

This guide covers **v1.3.0-alpha.1**, with round-screen controller version code `7`. The app interface is currently Chinese. For a new installation, start with [Installation and recovery](install.en.md).

## Before using a device

After installing the module and rebooting, open “VPN 控制” from the Magisk module's **Action** or the installer's **Open control** button. The first installation has VPN off and no subscription. Import `U30ProVPN.txt` into your existing UFI-TOOLS, save your own URL and wait for nodes to appear.

## Native Home and exiting

Home shows the VPN switch, running state, current exit, Nodes (“节点”) and More (“更多”).

- Tap **Enable VPN** (“开启 VPN”) or **Disable VPN** (“关闭 VPN”), then wait for the running state to update.
- Open **Nodes** to choose a node or test latency.
- Open **More** to update a saved subscription, set up the Home shortcut or exit the app.
- Back from a secondary page returns Home; Back from Home exits.

Exiting the app leaves VPN running. To stop proxying, use **Disable VPN** and check connectivity from a hotspot client.

![Round-screen Home, a swiped node and a single-node test](assets/native-ui.svg)

Round-screen UI illustration.

## Node selection and manual measurements

1. Open **Nodes** (“节点”). Tap a node to select it, or choose **Auto** (“自动选择”).
2. To test the entire list, tap **Refresh** (“刷新”) at the bottom. “测速中…” appears while it runs, and results appear when it finishes.
3. To test one node, hold its row and swipe left, then tap **Test** (“测速”). Swipe right to close the button. This also works on the selected node; `Auto` has no individual Test button.

Both tests require VPN enabled and the core running; neither starts VPN automatically. A single-node test leaves your selection unchanged. A complete full-list round can choose an inner `Auto` exit from successful results while preserving your outer manual choice. If you have selected `Auto`, that inner change becomes the active exit.

Results are latency in milliseconds, not download speed. “未测速” means no result yet; “上次测速未通过” means the last check failed. See [Measurements, Auto and status reads](#measurements-auto-and-status-reads) for the details.

## Subscriptions

### Update a saved URL on the round screen

Open **More → Subscription** (“订阅”) and tap **Update saved subscription** (“更新已保存订阅”). The page also shows the saved state and update time. Add or change the URL in the web panel first; the round screen has no URL keyboard.

### Add, change or reuse a URL in the web panel

1. Open UFI's **Subscription** tab, enter your HTTPS URL and choose **Save and update** (“保存并更新”).
2. For later updates, use **Update saved subscription** (“更新已保存的订阅”) without entering it again.
3. Wait for completion and check the node count and notices. The first subscription keeps VPN off. If a previously selected node disappears, selection falls back to `Auto` with a notice.

Subscriptions can contain Clash/Mihomo-style inline `proxies` or a list of node mappings. Downloads require HTTPS on port 443, valid TLS and a reachable public IPv4 source. Keep the URL private; it may contain credentials. More restrictions and update troubleshooting are in the [FAQ](faq.en.md).

Download or parse failures keep the previous configuration. If applying an update fails, the manager attempts recovery; refresh and check the actual running state if it reports incomplete recovery. An update with VPN enabled also checks the current exit's HTTPS connectivity once.

## Home overlay shortcut

1. Open **More → Shortcut** (“快捷入口”).
2. Tap **Authorize shortcut** (“授权快捷入口”) if needed, then allow display over other apps in system settings.
3. Return and tap **Enable shortcut** (“开启快捷入口”).
4. Return to unlocked stock Home. Tap the floating button to open controls, or drag it to move it.

The shortcut hides in the controller, on other pages, on the lock screen and when the display is off. It uses a foreground service and notification. If it does not appear, check overlay permission and Home recognition in the [FAQ](faq.en.md#9-why-is-the-shortcut-missing-are-system-permission-and-a-notification-required). Disabling the shortcut leaves VPN running.

## UFI web panel

Import the Release's `U30ProVPN.txt` through **插件功能 → 添加插件(.txt文件) → 提交** (Plugins → Add .txt plugin → Submit), then open it after logging into UFI. The panel lets you:

- Switch VPN on or off and read its running state.
- Search and select nodes or `Auto`.
- Add, change or update a subscription.
- Read status and existing latency results with top-right **Refresh**.

Web Refresh reads status; the round-screen Nodes Refresh starts a full-list test. The web panel has no separate full-list or one-node Test button. Closing the page leaves VPN running. UFI still uses its existing login and authenticated root interface.

## Measurements, Auto and status reads

| Action | Network check | Selection |
| --- | --- | --- |
| Foreground/shortcut status polls, web Refresh | Reads local state | Unchanged |
| One-node test | Measures that node against the fixed target | Unchanged |
| Full manual test | Measures current subscription nodes | A complete round can choose inner `Auto`; outer `VPN` is preserved |
| Apply subscription with VPN enabled | Downloads subscription and checks current exit HTTPS | Keeps a surviving selection, otherwise falls back to `Auto` |
| Gateway supervisor | Checks local process, TUN and routing state | Unchanged |

The test target is `https://www.gstatic.com/generate_204`, expecting HTTP 204. A result describes that check at that time. Reading status does not refresh its timestamp.

There are **no periodic external node tests**. The provider has no `health-check`; both `VPN` and `Auto` are `select` groups, with no `url-test` or periodic `interval`. `Auto` uses the results of a complete manual round. Normal VPN traffic, DNS, downloads and requested actions still use the network.

## Preview on a computer

<details>
<summary>Run the offline web preview</summary>

Requires Node.js 18+ and npm. From the repository root:

```sh
npm ci
npm run build
npm run preview
```

Open the printed address, normally `http://127.0.0.1:18771/`. Press `Ctrl+C` to stop the server. It listens only on `127.0.0.1` and uses mock nodes, temporary keys and in-memory state, with no device connection or CDN resources.

![Offline UFI web panel preview](assets/web-preview.png)

Offline preview with sample nodes.

Use `https://fail.example/test` to try a simulated subscription failure. Keep real subscription URLs out of the preview. Output defaults to `build/web-ui/`; see the [web guide](../web-manager/ui/README.md) for output-directory and port options, including `U30PRO_UI_OUT`.

</details>

## Build your app and backend

See [Building](building.en.md) for web checks, Android toolchains and signing, Go helpers and release packaging. The [Native guide](../native-manager/README.md) and [Backend guide](../web-manager/README.md) cover their individual components. Building or importing a plugin alone does not install the device gateway.

## Reporting problems and device acceptance

Include the app/source version, firmware, steps, running state and error category in [GitHub issues](https://github.com/zj-yzj/u30pro-vpn-controller/issues). Remove subscription URLs, node credentials, keys, provider/trust/state files, raw logs and backups before sharing material. Device use and recorded checks are in [Testing](testing.en.md).
