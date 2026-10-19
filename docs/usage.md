# 使用说明

[中文](usage.md) · [English](usage.en.md) · [返回主页](../README.md) · [FAQ](faq.md)

本页面向两类读者：想在电脑查看源码界面的人，以及已经完成设备集成的使用者。原生应用为 **1.2.0 / version code 6**，当前界面文字为中文。初次部署者请先读[设备集成边界](device-integration.md)和[网关参考](../gateway/reference/README.md)；本页操作不能替代设备安装和恢复流程。

## 目录

- [在电脑上预览](#在电脑上预览)
- [使用前确认](#使用前确认)
- [圆屏主页与退出](#圆屏主页与退出)
- [节点选择与手动测速](#节点选择与手动测速)
- [订阅管理](#订阅管理)
- [首页悬浮快捷入口](#首页悬浮快捷入口)
- [UFI 网页面板](#ufi-网页面板)
- [测速、自动选择与状态读取](#测速自动选择与状态读取)
- [构建自己的应用与后端](#构建自己的应用与后端)
- [故障反馈与部署验收](#故障反馈与部署验收)

## 在电脑上预览

需要 Node.js 18+ 和 npm。从仓库根目录运行：

```sh
npm ci
npm test
npm run build
npm run test:preview
npm run preview
```

打开终端打印的本机地址，默认 `http://127.0.0.1:18771/`。结束预览在终端按 `Ctrl+C`。预览服务只监听 `127.0.0.1`，使用模拟节点、临时模拟密钥和内存状态；不会连设备、读取真实订阅或改变网关。网页构建产物不加载 CDN。

![使用模拟数据的网页离线预览，非实机截图](assets/web-preview.png)

预览中的订阅框可以使用示例 `https://fail.example/test` 检查模拟更新失败后的反馈。这是 mock 专用场景，不是实际可用订阅。不要在 mock 中输入真实链接。

默认输出在 `build/web-ui/`。可为构建、HTTP 测试和预览统一指定输出位置：

```powershell
npm run build:ui -- --out C:/Temp/u30pro-ui-preview
npm run test:preview -- --out C:/Temp/u30pro-ui-preview
npm run preview -- --out C:/Temp/u30pro-ui-preview --port 18771
```

也支持 `U30PRO_UI_OUT`。更详细的产物和依赖说明见[网页 README](../web-manager/ui/README.md)。

## 使用前确认

真实设备上的操作要求以下条件已经由维护者完成：

1. 已 root 的 U30 Pro 原厂 Android 环境，已审查的启动和恢复机制。
2. 独立取得并核验的 Android arm64 Mihomo、匹配网络基线的网关配置和脚本。
3. root 私有的 `/data/adb/u30pro-vpn`，有效 provider、管理状态和密钥。
4. 正确安装的 `webctl`；原生控制还需 `webbridge`、签名 APK，以及该设备当前包名、UID、安装路径和 APK SHA-256 对应的信任记录。
5. 网页控制还需已安装 UFI-TOOLS、可用登录和已认证 `runShellWithRoot` 接口。

公开模板没有任何可用节点。`gateway/reference/` 不能直接复制执行：其中摘要占位符须按最终部署内容填入，固定网络假设须验证，启动前的安装事务恢复须另行实现。不要为消除错误而放宽目录权限、删除 APK 核验、禁用摘要检查或改成任意 root 命令入口。

## 圆屏主页与退出

打开“VPN 控制”，等待本机状态读取。主页展示开关、实际运行状态、当前出口，以及“节点”“更多”。

- 点“开启 VPN”或“关闭 VPN”，等待操作结束并确认实际状态。目标开关状态与真实进程状态可能暂时不同；按钮文字不能单独证明已联网。
- 节点名称对应当前出口。状态未连接或启动尚未完成时，会显示等待/尚未确定等提示。
- “更多 → 退出”关闭当前 Activity；主页按返回也会退出。其他页面按返回回到主页。

**退出界面不关闭 VPN。** 若要恢复原厂联网方式，在主页明确点“关闭 VPN”，等待并核实设备实际联网。悬浮快捷入口有独立设置；退出界面不等于关闭该入口。

![原生主页、左滑节点与单节点测速示意，非实机截图](assets/native-ui.svg)

## 节点选择与手动测速

1. 主页点“节点”。列表包含“自动选择”和订阅内普通节点，支持上下滚动。
2. 点击普通节点行选择它；选中行显示“已选择”。点击“自动选择”将外层 `VPN` 切回 `Auto`。这是更改选择的操作，不是测速。
3. **整批测速：** 确认 VPN 已运行，点页面底部“刷新”。按钮变为“测速中…”，任务结束后列表显示结果。它测试当前订阅的整个节点列表，不仅是可见行。
4. **单节点测速：** 按住普通节点向左滑动，展开右侧独立“测速”按钮，再点该按钮。右滑收起。当前选中节点也可左滑；`Auto` 行无独立测速按钮。无需等待固定长按时长。
5. 单节点测速只测目标，不切换当前节点或内外选择组。整批测速会按完整一轮的成功结果优选内层 `Auto`，不会覆盖外层手动选择；若当前外层已选择 `Auto`，内层变化会影响实际出口。

两种测速都要求目标开关已开且核心实际运行，不能通过点击测速自动启动 VPN。处理中控件会暂时禁用，避免重叠操作。没有结果显示“未测速”；历史失败显示“上次测速未通过”。它们不是持续可用性的证明。

## 订阅管理

### 圆屏更新已保存链接

进入“更多 → 订阅”，查看保存状态和上次更新时间，点“更新已保存订阅”。尚未保存链接时按钮不可用。圆屏不提供 URL 键盘输入；新增或更换链接请使用网页。

### 网页新增、更换或复用

在 UFI 面板进入“订阅”：

1. 在密码样式的输入框粘贴自己的 HTTPS 链接，点“保存并更新”。输入框遮挡文字不等于链接可以公开分享。
2. 已有链接时可点“更新已保存的订阅”；此时不用重新粘贴。
3. 等待任务完成，检查节点数、更新时间、实际出口和提示。更换列表后，若原稳定节点 ID 已不存在，后端退回 `Auto` 并提示。

输入支持 Clash/Mihomo 风格的 inline `proxies` 内容，或节点映射列表；不是任意客户端格式或远程 provider 的通用转换器。规范化限制输入体积、展开深度和节点数，拒绝危险外部文件引用等内容；最终还需设备 Mihomo 校验所用节点协议与字段。

下载器要求 HTTPS、默认 443 或显式 443、无 URL userinfo 和 fragment，使用受限制的公网 IPv4 解析/连接并校验 TLS。内网地址、纯 IPv6 源、非标准端口或不兼容格式可能失败。完整 URL 仍可能包含凭据，必须私下保管。

下载/解析失败会保留旧配置；应用失败进入恢复流程。若出现“恢复未完成”，请核实实际状态并交由维护者检查，不能从失败提示推断恢复一定成功。VPN 开启时更新还会验证配置、当前核心节点与出口 HTTPS；该一次性检查不是定时节点测速。

## 首页悬浮快捷入口

“更多 → 快捷入口”可启用或关闭可拖动的 VPN 入口。

1. 未授权时点“授权快捷入口”，进入系统设置，明确允许应用显示在其他应用上层。
2. 返回应用检查设置。已有权限时，可直接点“开启快捷入口”或“关闭快捷入口”。若系统授权页面不可用，需要按该固件实际系统设置处理，应用不能绕过权限。
3. 回到解锁的原厂首页，入口在实现识别到首页时显示；点击打开控制页，拖动可调整位置。

入口在应用本身、其他页面、锁屏与息屏时隐藏，并会进行本地状态/可见性轮询。原厂首页识别和系统前台服务限制依赖固件；未知系统不能仅凭 overlay 权限认定兼容。关闭入口不关闭 VPN。它是本机控制快捷方式，不是 VPN 网络隧道服务。

## UFI 网页面板

先完成网页构建，再把 `build/web-ui/U30ProVPN.txt` 通过已有 UFI 的“插件功能 → 添加插件(.txt文件) → 提交”导入。导入只提供界面，不安装 Go 后端、网关、核心或 root。

登录已有 UFI 后，面板可以：

- 点击电源开关，并读取目标与实际运行状态。
- 搜索节点名称、点击选择普通节点或 `Auto`。
- 管理 HTTPS 订阅 URL、复用已保存 URL。
- 点右上角“刷新”读取状态和已有测速结果。

**网页“刷新”不是圆屏节点页的整批测速。** 当前网页没有单独的批量/单节点测速按钮。关闭网页也不会关闭 VPN。UFI 或原厂 WebUI 登录可能触发设备原有屏幕锁定提示；原生应用不需要登录该 WebUI，项目不修改原厂屏幕锁定机制。

## 测速、自动选择与状态读取

| 动作 | 会访问外部目标吗 | 会更改选择吗 |
| --- | --- | --- |
| 前台/悬浮状态轮询、网页“刷新” | 不发起外部节点测速 | 不会 |
| 圆屏单节点测速 | 对指定节点发起固定目标测量 | 不会 |
| 圆屏整批手动测速 | 测当前 provider 节点 | 成功完整一轮可优选内层 `Auto`；保留外层 `VPN` |
| 应用订阅，VPN 已开启 | 下载订阅、验证当前出口 HTTPS | 保留仍存在的稳定选择，否则回到 `Auto` |
| 网关 supervisor `healthy()` | 检查本地运行对象，无外部测速 | 不负责自动节点选择 |

固定测量目标为 `https://www.gstatic.com/generate_204`，预期 HTTP 204。结果受该目标、当前网络与节点出口影响，是一次延迟样本，不是带宽、整网可达性或服务等级保证。状态轮询持续显示已有结果，不会主动刷新其测量时间。

参考配置中 provider 为 `file`，无 `health-check`；`VPN`、`Auto` 都是 `select`，无 `url-test` 和周期 `interval`。不要把组名 `Auto` 理解为后台持续巡检。正常转发、DNS、订阅下载和明确操作仍会使用网络。

## 构建自己的应用与后端

完整说明见[构建](building.md)、[原生 README](../native-manager/README.md)和[后端 README](../web-manager/README.md)。以下均为主机构建，不执行设备安装。

### Android

准备 PowerShell、JDK 17+、Android API 35 `android.jar` 和 Build Tools 35 或兼容版本。可设置 `JAVA_HOME`、`ANDROID_SDK_ROOT`（缺省时接受 `ANDROID_HOME`），然后从仓库根运行：

```powershell
./native-manager/build.ps1 -TestOnly
./native-manager/build.ps1
node ./native-manager/android/tests/round-screen-geometry.mjs
```

`TestOnly` 编译 Android 源码并运行加密、手势主机测试，不生成 APK。完整构建默认生成未签名 APK 和 `build-manifest.json`。工具链路径也可显式指定；下列变量须事先赋为你自己的路径：

```powershell
./native-manager/build.ps1 -JavaHome $jdkHome -AndroidJar $androidJar `
  -BuildTools $buildTools -OutputDir $outputDir
```

签名是可选的独立步骤，三个参数须一起提供：

```powershell
./native-manager/build.ps1 -JavaHome $jdkHome -AndroidJar $androidJar `
  -BuildTools $buildTools -OutputDir $outputDir `
  -Keystore $keystore -KeystorePasswordFile $passwordFile -KeyAlias $alias
```

脚本不会自动查找或创建签名身份。私钥和密码文件留在源码树之外；使用密钥密码与库密码相同的 keystore。已安装同包应用通常需相同签名才能更新；新的 APK 摘要还需同步 root 信任记录。不要复用其他设备 UID 或旧 APK pin。未签名构建、签名构建和信任配置是不同环节。

### Go

主机测试前清除交叉编译环境，确保 Go 1.27+、`java`、`javac` 可用：

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

Linux/arm64 设备程序不需 CGO。以下从仓库根运行；`$outputDir` 须是你准备的源码树外输出目录：

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

`diagnostic` 是额外固定目标诊断构建标签，普通发布构建不启用。`normalize` 工具输入和输出可能含节点凭据，须在私有目录使用，不能把其产物当公开示例提交。

## 故障反馈与部署验收

常见提示见[FAQ](faq.md)。反馈包含版本、固件、操作顺序、目标/实际状态和固定错误类别即可；先脱敏再提供。不要上传私钥、完整订阅、provider、trust、state、原始日志或备份。

维护者还需对真实设备验收：客户端 DNS/TLS/HTTPS、转发与 TUN、显式关闭后的原厂联网、重启恢复、非法订阅后的保留/恢复、故障阻断和安装断电恢复。公开源码主机测试不代替这些检查；`gateway/reference/` 不包含原开发设备的历史安装事务或可直接重跑安装包。

原创代码与本说明采用 [MIT](../LICENSE)，第三方许可见[第三方说明](../THIRD_PARTY_NOTICES.md)。
