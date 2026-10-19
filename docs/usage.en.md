# Usage

[中文](usage.md) · [English](usage.en.md) · [Home](../README.en.md) · [FAQ](faq.en.md)

This guide covers inspecting the interface on a computer and using a device whose integration is already complete. Native app version is **1.2.0 / version code 6**, with Chinese interface text. For first deployment, read [device integration boundaries](device-integration.en.md) and the [gateway reference](../gateway/reference/README.en.md). These usage steps do not replace installation and recovery procedures.

## Contents

- [Preview on a computer](#preview-on-a-computer)
- [Before using a device](#before-using-a-device)
- [Native Home and exiting](#native-home-and-exiting)
- [Node selection and manual measurements](#node-selection-and-manual-measurements)
- [Subscriptions](#subscriptions)
- [Home overlay shortcut](#home-overlay-shortcut)
- [UFI web panel](#ufi-web-panel)
- [Measurements, Auto and status reads](#measurements-auto-and-status-reads)
- [Build your app and backend](#build-your-app-and-backend)
- [Reporting problems and device acceptance](#reporting-problems-and-device-acceptance)

## Preview on a computer

Requires Node.js 18+ and npm. From the repository root:

```sh
npm ci
npm test
npm run build
npm run test:preview
npm run preview
```

Open the printed local address, normally `http://127.0.0.1:18771/`. Press `Ctrl+C` in the terminal to stop it. The server listens only on `127.0.0.1` and uses mock nodes, temporary mock keys and in-memory state. It does not connect to a device, read real subscriptions or change a gateway. Web outputs load no CDN resources.

![Offline web preview using mock data, not an on-device screenshot](assets/web-preview.png)

Enter `https://fail.example/test` in the preview subscription field to inspect a simulated update failure. This is a mock-only scenario, not a usable subscription. Do not enter real URLs into the mock.

Outputs default to `build/web-ui/`. You can select the same output directory for assembly, HTTP tests and preview:

```powershell
npm run build:ui -- --out C:/Temp/u30pro-ui-preview
npm run test:preview -- --out C:/Temp/u30pro-ui-preview
npm run preview -- --out C:/Temp/u30pro-ui-preview --port 18771
```

`U30PRO_UI_OUT` is also supported. See the [web README](../web-manager/ui/README.md) for outputs and dependencies.

## Before using a device

A maintainer must already have provided:

1. A rooted U30 Pro stock Android environment with reviewed startup/recovery mechanisms.
2. A separately obtained and verified Android arm64 Mihomo, plus gateway configuration/scripts matching the network baseline.
3. Private root-owned `/data/adb/u30pro-vpn` with a valid provider, management state and keys.
4. Installed `webctl`; native control also needs `webbridge`, a signed APK, and trust records matching that device's current package, UID, installed APK path and SHA-256.
5. For web control, existing UFI-TOOLS with working authentication and an authenticated `runShellWithRoot` interface.

The public template has no usable nodes. Do not copy and execute `gateway/reference/` directly: render digest placeholders from final deployment content, verify fixed network assumptions, and implement installation-transaction recovery before startup. Do not suppress errors by loosening directory permissions, removing APK/digest checks or exposing arbitrary root commands.

## Native Home and exiting

Open “VPN 控制” and wait for local status. Home shows power, actual running state, current exit, Nodes (“节点”) and More (“更多”).

- Tap Enable VPN (“开启 VPN”) or Disable VPN (“关闭 VPN”), wait, and verify actual state. Desired state and real process state may temporarily differ; button text alone does not establish connectivity.
- The exit name describes the active exit. Pending startup or unavailable state can display waiting/unknown messages.
- More → Exit (“退出”) closes the Activity. Back from Home also exits; Back on other pages returns Home.

**Exiting the interface does not stop VPN.** To restore stock networking, explicitly tap Disable VPN, wait and verify actual connectivity. The overlay has its own setting; exiting the Activity does not disable that shortcut.

![Illustration of native Home, a swiped node and a per-node test; not a device screenshot](assets/native-ui.svg)

## Node selection and manual measurements

1. Open Nodes (“节点”). The scrollable list contains Auto (“自动选择”) and subscription nodes.
2. Tap an ordinary row to select it; it shows “已选择”. Tap Auto to select outer `VPN` → `Auto`. Selection changes do not initiate measurements.
3. **Full-list measurement:** with VPN running, tap bottom Refresh (“刷新”). It becomes “测速中…” and the list displays results when the task finishes. This tests the current subscription list, not just visible rows.
4. **One-node measurement:** hold an ordinary row and swipe left, then tap the separate Test (“测速”) button. Swipe right to close. The selected node can also be swiped; `Auto` has no individual test button. No fixed long-press duration is required.
5. A one-node test changes neither the selected node nor either group. A complete full-list round can prefer an inner `Auto` exit from successful results while preserving the outer manual selection. If the outer selection is already `Auto`, an inner change affects the actual exit.

Both actions require desired VPN enabled and the core actually running; neither starts VPN implicitly. Controls are temporarily disabled while busy. “未测速” means no measurement; “上次测速未通过” describes a failed prior measurement. Neither is a continuous availability guarantee.

## Subscriptions

### Update a saved URL on the round screen

Open More → Subscription (“订阅”), inspect saved state and update time, then tap Update saved subscription (“更新已保存订阅”). It is disabled when no URL is saved. The round screen has no URL keyboard; add/change URLs in the web panel.

### Add, change or reuse a URL in the web panel

Open UFI's Subscription tab:

1. Paste your HTTPS URL into the password-style field and select Save and update (“保存并更新”). Masking does not make the URL safe to share publicly.
2. For an existing URL, use Update saved subscription (“更新已保存的订阅”) without pasting it again.
3. Wait for completion and inspect node count, update time, actual exit and notices. If the old stable node ID no longer exists, the backend falls back to `Auto` and reports it.

Input supports Clash/Mihomo-style inline `proxies` content or a list of node mappings. It is not a universal converter for arbitrary client formats or remote providers. Normalization limits size, expansion depth and node count and rejects dangerous external file references. Device Mihomo must still validate protocol/field support.

The downloader requires HTTPS, port 443 or its default, no URL userinfo or fragment, restricted public IPv4 resolution/connections, and valid TLS. Private addresses, IPv6-only sources, unusual ports or incompatible formats can fail. Full URLs may contain credentials and must remain private.

Download/parse failures preserve the previous configuration; application failures enter recovery. If recovery is incomplete, verify actual state and involve the maintainer; failure feedback does not prove recovery succeeded. Updates with VPN enabled also validate configuration, current core nodes and exit HTTPS connectivity. This one-time check is not scheduled node measurement.

## Home overlay shortcut

More → Shortcut (“快捷入口”) enables/disables a draggable VPN entry.

1. Without permission, tap Authorize shortcut (“授权快捷入口”), enter system settings and explicitly allow display over other apps.
2. Return and check the setting. With permission, tap Enable/Disable shortcut (“开启快捷入口” / “关闭快捷入口”). If the system permission page is unavailable, use the firmware's actual settings; the app cannot bypass permission.
3. Return to unlocked stock Home. The shortcut appears when the implementation recognizes Home. Tap to open controls or drag to reposition it.

It hides in the app, on other pages, on the lock screen and with the display off, while polling local status/visibility. Stock Home recognition and foreground-service restrictions depend on firmware; overlay permission alone does not establish compatibility. Disabling the shortcut does not stop VPN. It is a local control shortcut, not the network tunnel service.

## UFI web panel

Build the web output, then import `build/web-ui/U30ProVPN.txt` through existing UFI: “插件功能 → 添加插件(.txt文件) → 提交” (Plugins → Add .txt plugin → Submit). Import supplies the interface only, not Go helpers, gateway, core or root.

After logging into existing UFI, the panel can:

- Toggle power and read desired/actual running state.
- Search names and select ordinary nodes or `Auto`.
- Manage HTTPS subscription URLs and reuse the saved URL.
- Read status and existing measurement results with top-right Refresh.

**Web Refresh differs from native Nodes Refresh.** The current web panel has no separate full-list or per-node test button. Closing the web page does not stop VPN. UFI/stock WebUI login may trigger the device's existing screen-lock prompt; the native app does not need that login. This project does not modify stock screen locking.

## Measurements, Auto and status reads

| Action | External access | Selection changes |
| --- | --- | --- |
| Foreground/shortcut polls, web Refresh | No external node measurement | None |
| Native one-node test | Fixed-target measurement through that node | None |
| Native full manual test | Measures current provider nodes | A successful complete round can prefer inner `Auto`; outer `VPN` is preserved |
| Apply subscription with VPN enabled | Downloads subscription and checks current exit HTTPS | Keeps a surviving stable selection, otherwise falls back to `Auto` |
| Gateway supervisor `healthy()` | Local runtime checks, no external measurement | Does not select nodes automatically |

The fixed measurement target is `https://www.gstatic.com/generate_204`, expecting HTTP 204. Results depend on the target, current network and exit. They are latency samples, not bandwidth, universal reachability or a service guarantee. Status polling displays existing results without renewing their measurement time.

The reference provider is `file` with no `health-check`; both `VPN` and `Auto` are `select`, without `url-test` or periodic `interval`. The name `Auto` does not mean continuous background probing. Normal forwarding, DNS, downloads and explicit actions still use the network.

## Build your app and backend

See [Building](building.en.md), [Native README](../native-manager/README.md) and [Backend README](../web-manager/README.md). The following host builds do not install anything on a device.

### Android

Prepare PowerShell, JDK 17+, Android API 35 `android.jar` and Build Tools 35 or a compatible version. Set `JAVA_HOME` and `ANDROID_SDK_ROOT` (`ANDROID_HOME` is accepted if the latter is absent), then run from the repository root:

```powershell
./native-manager/build.ps1 -TestOnly
./native-manager/build.ps1
node ./native-manager/android/tests/round-screen-geometry.mjs
```

`TestOnly` compiles Android source and runs crypto/gesture host tests without an APK. A full build defaults to an unsigned APK and `build-manifest.json`. Explicit paths are also supported; assign these variables to your own paths first:

```powershell
./native-manager/build.ps1 -JavaHome $jdkHome -AndroidJar $androidJar `
  -BuildTools $buildTools -OutputDir $outputDir
```

Signing is optional and requires all three signing parameters together:

```powershell
./native-manager/build.ps1 -JavaHome $jdkHome -AndroidJar $androidJar `
  -BuildTools $buildTools -OutputDir $outputDir `
  -Keystore $keystore -KeystorePasswordFile $passwordFile -KeyAlias $alias
```

The script does not find/create a signing identity. Keep keys/password files outside the source tree. Use a keystore with matching key/store passwords. Updating an installed package normally requires the same signing certificate; the new APK digest also needs matching root trust. Do not reuse another device's UID or an old APK pin. Unsigned output, signing and trust provisioning are separate steps.

### Go

Before host tests, clear cross-compilation settings and provide Go 1.27+, `java` and `javac`:

```powershell
Remove-Item Env:GOOS -ErrorAction SilentlyContinue
Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
Push-Location ./web-manager
go test -count=1 ./...
go test -count=1 -tags diagnostic ./...
go vet ./...
go vet -tags diagnostic ./...
Pop-Location
```

Linux/arm64 helpers require no CGO. Run from the repository root with `$outputDir` set to a prepared directory outside the source tree:

```powershell
$env:CGO_ENABLED = '0'
$env:GOOS = 'linux'
$env:GOARCH = 'arm64'
Push-Location ./web-manager
go build -trimpath -ldflags='-s -w' -o (Join-Path $outputDir 'webctl') ./cmd/webctl
go build -trimpath -ldflags='-s -w' -o (Join-Path $outputDir 'webbridge') ./cmd/nativebridge
Pop-Location
Remove-Item Env:GOOS
Remove-Item Env:GOARCH
Remove-Item Env:CGO_ENABLED
```

`diagnostic` enables extra fixed-target diagnostic code; ordinary release builds omit it. `normalize` inputs/outputs may contain node credentials: use private directories and do not commit its artifacts as examples.

## Reporting problems and device acceptance

See the [FAQ](faq.en.md). Reports can include versions, firmware, action order, desired/actual state and fixed error categories after sanitization. Do not upload keys, full subscriptions, provider/trust/state files, raw logs or backups.

A maintainer still needs actual-device acceptance: client DNS/TLS/HTTPS, forwarding/TUN, stock networking after explicit stop, reboot restoration, invalid-subscription retention/recovery, failure blocking and installation power-loss recovery. Host tests do not replace these checks. `gateway/reference/` includes neither the original device's historical installation transactions nor a directly rerunnable installer.

Original code and this guide use [MIT](../LICENSE); see [third-party notices](../THIRD_PARTY_NOTICES.md) for separate licenses.
