# U30 Pro VPN Controller

[中文](README.md) · [English](README.en.md)

让已取得 root、运行原厂 Android 的中兴 U30 Pro 可以在设备圆屏上管理 VPN，也可以通过已有 UFI 网页面板管理节点与订阅。项目包含 **Android 原生应用、Go 管理后端、UFI 网页插件和首次安装 alpha**。代理转发由设备从官方上游单独下载的 [Mihomo](https://github.com/MetaCubeX/mihomo) 完成。

![项目组件与使用方式示意：圆屏应用、网页面板和设备端管理器](docs/assets/overview.svg)

> 本页圆屏图、总览图和架构图均为说明性示意，网页图使用离线 mock 数据。它们不是实机截图，也不代表一次新的设备部署验收。

**当前版本：`v1.3.0-alpha.1`，圆屏控制应用 Android version code 7。** 面向 240 × 240 圆屏 framebuffer，主要控件放在居中的 164 × 164 安全区域。

**[下载 Magisk ZIP 或 Android 安装器 APK](https://github.com/zj-yzj/u30pro-vpn-controller/releases/tag/v1.3.0-alpha.1) → [按安装说明操作](docs/install.md)。** 两种入口使用同一个 Magisk 模块，要求已安装 Magisk 28.1+、MU5358、arm64、API 35、固件 display `U30ProV1.0.0B23`，且 SELinux 原本为 Permissive。它不获取 root、不刷机、不覆盖已有私有部署。首次 VPN 关闭、无订阅；已有 UFI-TOOLS 用于录入自己的订阅，核心需联网下载。

**这是首次安装预发布，尚未完成干净设备上的安装、重启、客户端联网和卸载全流程实机验收。** 主机验证和示意图不代表实机兼容结论；具体测试范围见 Release 说明。后续已开启状态的冷启动存在网关就绪前窗口，不承诺从通电起全程隧道/阻断保护。

手动 ZIP 安装前须确认 Magisk 没有同 ID `codex_u30pro_vpn` 的已安装或待重启模块。Magisk 可能在项目检查前替换同 ID 暂存目录；本项目不承诺保留该目录旧内容。APK 入口另有调用 CLI 前的目录预检。见[安装前确认](docs/install.md#安装前确认)。

## 目录

- [功能与操作语义](#功能与操作语义)
- [界面预览](#界面预览)
- [兼容范围与前置条件](#兼容范围与前置条件)
- [开始使用](#开始使用)
- [安装与恢复](docs/install.md)
- [架构与源码目录](#架构与源码目录)
- [构建与验证](#构建与验证)
- [隐私与安全边界](#隐私与安全边界)
- [问题反馈](#问题反馈)
- [许可与上游](#许可与上游)

## 功能与操作语义

| 入口 | 现有功能 | 边界 |
| --- | --- | --- |
| 原生主页 | 开关 VPN，查看实际运行状态和当前出口 | 退出应用不关闭 VPN；关闭需显式操作开关 |
| 原生节点页 | 点击选择节点；底部“刷新”手动测试整个列表 | 测速要求 VPN 已运行，不会隐式开启 VPN |
| 单节点操作 | 按住普通节点向左滑动，展开“测速”；右滑收起 | 只测试该节点，不切换节点或选择组；`Auto` 无单独测速按钮 |
| 自动选择 | 按完整一轮手动测速的成功结果优选内层 `Auto` | 没有周期 `url-test` 或 provider `health-check`；保留外层 `VPN` 手动选择 |
| 原生“更多” | 更新已保存订阅、设置可拖动的首页快捷入口、退出 | 新增或更换 URL 使用网页；悬浮入口需系统授权 |
| UFI 网页面板 | 开关 VPN、搜索/选择节点、保存并更新订阅、更新已保存订阅 | 依赖现有已认证 UFI root 接口；右上角“刷新”只读取状态 |

“测速”测量固定 HTTPS 连通性目标的**延迟，单位毫秒**，不是下载带宽测试。一次失败不能证明节点对所有业务均不可用；没有历史测量时，延迟和连通性可以为空。

前台界面和可选悬浮入口存在本地状态轮询；网关 supervisor 检查进程、TUN、路由和保护状态。这些操作不会自动测试整个节点列表。VPN 开启时应用订阅仍会验证一次当前出口的 HTTPS 联网，正常 VPN 流量也会访问网络。详见[使用说明](docs/usage.md#测速自动选择与状态读取)。

## 界面预览

### 原生圆屏

![原生界面示意：主页、左滑节点和单节点测速](docs/assets/native-ui.svg)

*根据源码布局绘制，非实机截图。现有界面为中文；英文文档不表示应用已提供英文界面。*

### UFI 网页面板

![网页离线预览，展示模拟状态与节点列表](docs/assets/web-preview.png)

*离线 mock 预览：节点、密钥与状态为模拟数据，不连接设备或真实订阅。*

## 兼容范围与前置条件

| 项目 | 当前范围 | 集成方仍需确认 |
| --- | --- | --- |
| 设备 | device/model `MU5358`，display `U30ProV1.0.0B23` | alpha 不声明支持其他型号/固件；尚无干净设备全流程验收 |
| 系统 | 原厂 API 35、arm64、Magisk 版本码至少 28100、已为 SELinux Permissive | 不降低 SELinux，不安装 Magisk，不支持其他 root 方案的通用模块安装 |
| Android 应用 | `minSdk 26`、`targetSdk 35`，240 × 240 圆屏 | SDK 声明不扩大安装器固定范围；原厂首页识别与悬浮行为依赖系统 |
| 后端 | Linux/arm64 Go helper，固定 `/data/adb/u30pro-vpn` | root 所有权、私有权限、配置、状态、密钥和启动恢复 |
| 代理核心 | 准备时从官方在线下载固定 Mihomo v1.19.32，校验大小与摘要 | ZIP/APK 不内嵌核心，不支持离线首次安装；见[下载说明](docs/install.md#核心通过网络下载) |
| 网页 | 已安装并启用 root 管理的 UFI-TOOLS | 既有登录与 `runShellWithRoot`；仓库不含 UFI 本体和认证资产 |
| 网络 | 参考固定使用 `br0`、`192.168.0.0/24`、`U30VPN` 等对象 | SFP、DNS、规则/路由表归属、IPv4 转发与故障恢复；见[参考前置条件](gateway/reference/README.md#已保留的设备假设) |
| 安装/升级 | Magisk ZIP 与内嵌同 ZIP 的 Android 安装器，首次安装 alpha | 拒绝已有私有运行目录、未知同包应用与冲突对象；不提供旧部署升级 |

UFI-TOOLS、Magisk 和构建工具需另外取得。Mihomo 由设备直接从官方上游下载。仓库与本项目安装包不捆绑这些项目的程序、固件、SDK、认证资产、私钥或原设备配置；分别适用的许可见第三方说明。

## 开始使用

**首次安装：** 从 [Release Assets](https://github.com/zj-yzj/u30pro-vpn-controller/releases/tag/v1.3.0-alpha.1) 下载 `U30ProVPN-1.3.0-alpha.1-Magisk.zip` 或 `U30ProVPN-1.3.0-alpha.1-Installer.apk`，另取 `U30ProVPN.txt`。按[安装与恢复](docs/install.md)确认固定前提、校验下载文件并选择一个入口安装。重启后首次状态为关闭；在已有 UFI 中手动导入插件、输入订阅后再主动开启。Android 安装器 APK 应安装在 U30 Pro 本机。

**查看界面：** 在开发电脑运行离线预览，无需设备、root 或订阅。

```sh
npm ci
npm test
npm run build
npm run test:preview
npm run preview
```

打开终端显示的地址，默认 `http://127.0.0.1:18771/`。产物默认写入 `build/web-ui/`；输出目录和 UFI 导入步骤见[网页插件说明](web-manager/ui/README.md)。

**已有正确集成的设备：** 打开圆屏“VPN 控制”，确认实际状态后开关或选择节点。节点页“刷新”进行整批测速；左滑普通节点，再点“测速”进行单节点测试。在“更多 → 订阅”更新已保存链接，新增链接通过 UFI 面板完成。详见[使用说明](docs/usage.md)。

**源码移植：** 先读[设备集成边界](docs/device-integration.md)、[网关总览](gateway/README.md)和[固定布局参考](gateway/reference/README.md)。参考目录本身不是安装入口。主机测试不能验证实际设备的 DNS、转发、断网保护、关闭后的原厂联网恢复或断电恢复。

## 架构与源码目录

![架构示意：两种控制入口经信任边界到达管理器，管理器控制独立 Mihomo 与网关](docs/assets/architecture.svg)

*架构说明图，非设备状态截图。*

网页经 UFI 已认证 root 接口调用 `webctl`；原生应用经本地 Unix socket 连接 `webbridge`，由其校验调用者并调用受限命令。`webctl` 处理状态、订阅与操作事务，独立 Mihomo 通过本地 Unix HTTP controller 接受控制，网关脚本管理设备网络对象。详见[架构](docs/architecture.md)。

| 路径 | 内容 |
| --- | --- |
| `native-manager/android/src/` | 原生 Java 应用、模型、加密封装、悬浮入口与启动接收器 |
| `installer-app/` | 独立 Android 安装入口；内嵌固定 ZIP、用户授权 root、显式重启 |
| `release/` | Magisk 包装脚本与双入口发布构建 |
| `web-manager/cmd/u30setup/`、`web-manager/internal/installer/` | 固定设备检查、核心获取、安装恢复与归属卸载 |
| `native-manager/build.ps1` | 显式工具链构建、主机测试、可选签名与构建清单 |
| `native-manager/android/tests/` | 加密/手势测试、圆屏几何、本地截图检查工具 |
| `web-manager/cmd/webctl/` | root 管理程序 |
| `web-manager/cmd/nativebridge/` | 原生控制桥，编译产物名 `webbridge` |
| `web-manager/cmd/normalize/` | 私有 inline `proxies` 转为初始 provider、元数据和验证配置 |
| `web-manager/internal/` | 管理、存储、下载、订阅规范化、协议和测试 |
| `web-manager/device/config.yaml` | 没有订阅凭据或可用节点的配置模板 |
| `web-manager/ui/` | 原创网页界面、构建、mock 预览和逻辑测试 |
| `gateway/reference/` | 固定布局控制、supervisor 与启动模板，需另行集成 |
| `docs/` | 中英使用/FAQ、构建、架构、集成说明与示意图 |
| `third_party/licenses/` | 分别适用的第三方许可文本 |

## 构建与验证

网页需要 Node.js 18+ 和 npm，lockfile 固定 `node-forge@1.4.0`。Go 模块最低版本为 1.27.0，源码准备时使用 1.27.2；完整互操作测试需要 `java`、`javac` 在 `PATH` 中。Android 需要 PowerShell、JDK 17+、Android Platform 35 的 `android.jar`、Build Tools 35 或兼容后续版本，无 Gradle/Maven 依赖，也不在构建时下载 Android 工具链。

从仓库根目录完成上述网页命令后，运行：

```sh
cd web-manager
go test -count=1 ./...
go test -count=1 -tags diagnostic ./...
go vet ./...
go vet -tags diagnostic ./...
```

Android 在已设置 `JAVA_HOME`、`ANDROID_SDK_ROOT` 的 PowerShell 中，从仓库根目录运行：

```powershell
./native-manager/build.ps1 -TestOnly
./native-manager/build.ps1
node ./native-manager/android/tests/round-screen-geometry.mjs
```

默认生成 `native-manager/build/U30ProVPN-unsigned.apk` 和构建清单；未签名 APK 不能正常安装。显式签名、交叉编译、输出目录和 smoke test 见[构建说明](docs/building.md)、[原生说明](native-manager/README.md)、[后端说明](web-manager/README.md)。签名身份在源码树之外管理，脚本不寻找或生成现有项目身份。

验证范围包括 Go 管理器/安装引擎的主机测试、网页逻辑、Java 加密/手势及安装器子进程测试、Android 编译与签名/文件检查。具体通过与跳过项目以同版本 Release 说明为准；不以旧版本固定测试数量代表本次全部通过。模拟控制器、临时密钥与合成节点不能替代新设备验收；缺少 JDK 可能导致互操作测试跳过，需检查输出。双入口发布构建见[构建说明](docs/building.md)。

```sh
# 在不含 node_modules、构建产物或缓存的发布源码副本中运行
npm run check:publication
node scripts/check-doc-links.mjs
```

发布检查输出常见敏感内容的相对路径、行号与类别，不打印匹配值；它不能替代人工审查，也不证明任意输入均已脱敏。公开源码不应包含生成物、缓存或运行数据。

## 隐私与安全边界

- **订阅在设备端处理。** 不提交到公共转换服务。URL、节点凭据、provider、状态、备份与日志属于私有运行数据，不应进入仓库或公开 issue。
- **圆屏控制应用没有 `INTERNET` 权限，不直接申请 root。** 通过本机桥执行固定操作；请求采用 RSA-OAEP-SHA256/AES-256-GCM，桥校验 peer UID、包身份和固定 APK 摘要。独立安装器 APK 也无 `INTERNET` 权限，但安装模块时会申请用户 root 授权，由 root helper 在线下载固定核心。
- **网页依赖现有 UFI 的信任。** 加密、时效和重放检查不能替代 UFI 登录，也不能抵御已控制 root 或网页宿主的攻击者。Mihomo controller 使用本地 Unix socket，本项目不新增局域网 TCP 管理入口。
- **悬浮入口需用户授权。** 现有实现仅在解锁的原厂首页显示，在息屏、锁屏或其他页面隐藏；它读取本地状态，不代表外部周期测速。
- **订阅更新有操作级事务。** 失败时保留/尝试恢复旧配置；恢复也可能失败，需查看实际状态。安装器另有首次安装恢复记录，不能据此推断支持旧部署升级。卸载可延后，私有数据备份会保留，见[恢复说明](docs/install.md#卸载与保留的数据)。
- **网关依赖固定网络基线。** 参考保留故障阻断、身份核验和受控清理，但效果需在具体系统上验证，不能从主机测试推断。

## 问题反馈

[FAQ](docs/faq.md) 涵盖本机服务、信任失败、订阅更新、测速、悬浮授权、退出和开关、WebUI 锁屏提示与移植边界。可在[仓库 issue](https://github.com/zj-yzj/u30pro-vpn-controller/issues)提供型号、固件、源码/应用版本、复现步骤和脱敏错误类别。不要上传真实订阅、节点配置、私钥、trust/state/provider 文件、原始日志或未审查备份。

中文：[安装](docs/install.md) · [使用](docs/usage.md) · [FAQ](docs/faq.md) · [构建](docs/building.md) · [架构](docs/architecture.md) · [集成](docs/device-integration.md)。英文：[Install](docs/install.en.md) · [Usage](docs/usage.en.md) · [FAQ](docs/faq.en.md) · [Building](docs/building.en.md) · [Architecture](docs/architecture.en.md) · [Integration](docs/device-integration.en.md)。

## 许可与上游

原创代码和文档采用 [MIT License](LICENSE)，按原文提供，不附带适用性或无故障保证。第三方继续适用各自许可，见 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)：node-forge 使用 BSD-3-Clause 选项；yaml.v3 保留按文件适用的 MIT/Apache-2.0 文本；分发 Go 二进制还需携带 Go 许可。

Mihomo、UFI-TOOLS、Magisk 和 Android 为独立上游/集成环境。仓库不捆绑其程序、固件、SDK、认证资产或原设备签名私钥；未来分发其他组件需分别履行许可义务。本项目独立维护，不代表中兴、Mihomo、UFI 或其他上游。
