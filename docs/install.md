# 安装与恢复

[English](install.en.md) · [项目首页](../README.md) · [使用说明](usage.md) · [常见问题](faq.md)

当前版本为 **`v1.3.0-alpha.1`，仅用于首次安装**。可以选择 Magisk 模块 ZIP 或 Android 安装器 APK，两者安装同一个模块，都需要 U30 Pro 已有兼容的 Magisk。

已在自用 U30 Pro 实测，日常使用正常，详见[测试记录](testing.md)。本项目不获取 root、不刷写固件，也不覆盖升级已有的私有部署。

## 安装前确认

请逐项核对。安装器遇到不匹配或未知状态会拒绝继续，不要修改属性、关闭 SELinux 或删除已有数据来绕过检查。

| 项目 | 本版要求 |
| --- | --- |
| 设备与型号属性 | U30 Pro；device/model 均为 `MU5358` |
| 系统与架构 | 原厂 Android，API **35**，`arm64-v8a` |
| 固件 display | 精确为 `U30ProV1.0.0B23` |
| root 管理 | 已安装并正常工作的 **Magisk 28.1 或更新版本**，版本码至少 `28100` |
| SELinux | 系统本来就为 `Permissive`；安装器不修改 SELinux 级别 |
| 热点网络 | `br0`，`192.168.0.1/24`，IPv4 forwarding 已开启，SFP 基线为 `1` |
| 首次部署 | 没有 `/data/adb/u30pro-vpn`、同包控制应用或冲突的专用网络对象；`/data/adb/modules/codex_u30pro_vpn` 与 `/data/adb/modules_update/codex_u30pro_vpn` 均不存在 |
| 订阅录入 | 已安装 UFI-TOOLS，能够登录并使用其认证 root 接口 |
| 网络与空间 | U30 Pro 自身可访问 GitHub 官方发布资源，且安装器空间检查通过 |

其他固件、KernelSU、APatch、其他热点和 OpenWrt 暂不支持。APK 的 `minSdk 26` 是编译声明，不扩大上表范围。安装前请保留现有可用网络和设备恢复入口。

**手动安装 ZIP 前，确认 Magisk 没有同 ID `codex_u30pro_vpn` 的已安装或待重启模块。** Magisk 可能在项目检查前替换同 ID 的 `modules_update` 暂存内容；APK 入口会提前检查这两个目录。目录归属不明时先停止核查，安装期间也不要同时使用两个入口。

## 下载哪些文件

在 [v1.3.0-alpha.1 Release](https://github.com/zj-yzj/u30pro-vpn-controller/releases/tag/v1.3.0-alpha.1) 的 Assets 下载：

| 文件 | 用途 |
| --- | --- |
| `U30ProVPN-1.3.0-alpha.1-Magisk.zip` | 用 Magisk 安装模块 |
| `U30ProVPN-1.3.0-alpha.1-Installer.apk` | 可选的 U30 Pro 本机安装入口，内嵌同一个 ZIP |
| `U30ProVPN.txt` | 首次录入订阅需要的 UFI 网页插件，两种入口都需手动导入 |
| `SHA256SUMS` | 下载文件的 SHA-256 校验清单 |
| `APK-CERTIFICATE.txt` | APK 签名信息与证书指纹 |
| `release-manifest.json` | 版本、文件身份与构建来源 |

ZIP 和安装器选一种即可。GitHub 自动生成的 Source code 压缩包不是 Magisk 安装包，也不要将 ZIP 解压后复制到设备运行目录。

在电脑上可用 `Get-FileHash -Algorithm SHA256 -LiteralPath "下载的文件名"` 核对同一 Release 的 `SHA256SUMS`。安装器还会校验内嵌 ZIP 的大小和摘要。

## 核心通过网络下载

ZIP 和 APK **不内嵌 Mihomo 核心，不支持离线首次安装**。准备时，设备会从 [Mihomo 官方 v1.19.32 Release](https://github.com/MetaCubeX/mihomo/releases/tag/v1.19.32) 下载约 **21.9 MB** 的 `mihomo-android-arm64-v8-v1.19.32.gz`，校验 HTTPS 证书、大小和 SHA-256 后解压。

U30 Pro 需要能访问 GitHub 和固定 IPv4 DNS `223.5.5.5:53`。下载失败时先检查设备原厂联网、系统时间和可用空间，保留官方来源与校验。安装器不附带订阅，核心下载与订阅下载各自进行。

<details>
<summary>核心文件的校验值</summary>

| 内容 | 字节数 | SHA-256 |
| --- | ---: | --- |
| gzip 下载 | `21866463` | `42b1f2be8702484d3ae6238daab392622f54bd0d264966700f1f28f52596e326` |
| 解压后的核心，约 64.0 MB | `64045272` | `bb9afbd48717f92d59e07da35fb694b83d40d6edb63eb9033c2ee4174137d195` |

</details>

## 方式一：使用 Magisk ZIP

1. 把模块 ZIP 传到 **U30 Pro 本机**，保持设备联网。
2. 在 Magisk 的“模块”中选择从本地存储安装，选中 `U30ProVPN-1.3.0-alpha.1-Magisk.zip`。
3. 等待核心下载与校验，确认 Magisk 显示模块已暂存后重启。这一步只暂存模块，不开启 VPN，也不改动热点路由或安装控制 APK。
4. 重启并等初始化完成，在 Magisk 模块卡片点“操作 / Action”，打开“VPN 控制”。首次应显示 **VPN 已关闭、无节点、未保存订阅**。
5. 按下文导入 UFI 插件并录入订阅。

## 方式二：使用 Android 安装器 APK

安装器要装在 **U30 Pro 上**。它的包名为 `com.codex.u30vpn.installer`；日常圆屏控制应用为 `com.codex.u30vpn`。

1. 通过已有的设备管理方式安装 `U30ProVPN-1.3.0-alpha.1-Installer.apk`，打开“U30 Pro VPN 安装”。如需安装来源权限，按系统提示授权。
2. 点“安装模块”，在 Magisk 提示中允许本次 root 授权。
3. 等待“模块已暂存”，再点“现在重启”，或稍后自行重启。应用不会自动重启。
4. 重启后打开安装器，点“打开控制”，也可使用 Magisk 模块 Action。首次应为 VPN 关闭、未配置订阅，然后继续下文。

安装器超时、被关闭或显示“状态未知”时，先查看 Magisk 是否已暂存模块，再决定下一步；超时后 Magisk 可能仍在工作，不要连续点击或反复重装。

<details>
<summary>通过电脑 ADB 安装 APK</summary>

确认连接目标是 U30 Pro 后，在下载目录执行；有多个连接时先选择正确的设备：

```sh
adb install "U30ProVPN-1.3.0-alpha.1-Installer.apk"
adb shell am start --user 0 -n com.codex.u30vpn.installer/.MainActivity
```

ADB 用于安装和打开 APK，模块安装仍需设备已有 Magisk 和本次 root 授权。安装器通过 `magisk --install-module` 安装内嵌 ZIP，不接受自定义模块 URL；在线核心下载由 root helper 完成。没有 APK 安装入口时，可使用 Magisk ZIP。

</details>

## 首次录入订阅并开启

1. 用手机或电脑连接 U30 Pro 网络，打开现有的 UFI-TOOLS 网页，登录并确认 root 高级功能可用。
2. 进入“插件功能 → 添加插件(.txt文件) → 提交”，导入 Release 中的 `U30ProVPN.txt`。
3. 打开 VPN 插件，在“订阅”中粘贴自己的 HTTPS 订阅 URL，点保存/更新，等节点出现。**保存订阅后 VPN 仍保持关闭。**
4. 在网页或圆屏主动开启 VPN，等待实际运行状态，再用热点客户端检查 DNS 与 HTTPS 联网。
5. 需要延迟结果时，在运行中的圆屏节点页点底部“刷新”，或左滑普通节点后点“测速”。没有后台全节点定时测速，首次没有延迟值是正常的。

圆屏“订阅”页只更新已保存链接；新增或更换 URL 使用 UFI 网页。后续操作见[使用说明](usage.md)。

## 圆屏快捷入口

除了 Magisk Action 和安装器，“更多 → 快捷入口”还能启用悬浮按钮，需在系统设置中授权显示在其他应用上层。权限页不可用或原厂首页未识别时，继续使用前两个入口即可。

退出应用或关闭快捷入口不会关闭 VPN。要停止代理，请在控制页点“关闭 VPN”，等待实际状态并确认原厂联网恢复。

## 重启、重复安装与恢复

- 首次启动会生成本机管理密钥，创建空 provider 和节点列表，VPN 关闭。
- 后续重启读取已保存的 VPN 开关状态。上次为开启时，需等 Android 和网关就绪；这段冷启动时间不保证所有客户端流量受隧道或阻断保护，有此要求时请等 VPN 运行后再接入客户端。
- 同版本安装中断后，只继续可核验的原安装事务，保留已有密钥和已提交配置。不要复制其他设备的 trust 来补齐安装。
- 已有私有部署、未知同包应用会被拒绝。本版不提供覆盖升级、签名迁移或自动卸载重装，请保留现状另行规划迁移。
- Magisk 禁用模块影响后续启动。禁用或卸载前，先在控制页关闭 VPN，确认原厂联网恢复。

## 卸载与保留的数据

先关闭 VPN，确认热点客户端恢复原厂联网，再在 Magisk 中移除模块并按提示重启。卸载只清理归属明确的本项目对象；控制 APK 也需与发行物身份匹配。

私有运行数据会保留在 root 私有备份中，可能含订阅、节点凭据、状态和密钥，请勿上传。需要永久删除时，先核实备份归属和联网恢复，再由熟悉设备的维护者处理。只卸载安装器或控制 APK，不会移除 Magisk 模块或停止 VPN。

如果提示 **cleanup deferred / 清理延后**，说明 PackageManager 或网络管理等条件尚未就绪。请保留 `/data/adb/u30pro-vpn`、安装 journal、私有备份和可用恢复入口，按下文处理。

## 常见安装问题

| 现象 | 处理方法 |
| --- | --- |
| 型号、固件、API、Magisk 或 SELinux 不匹配 | 停止安装，核对[安装前确认](#安装前确认) |
| 发现已有目录、模块、同包应用或网络对象 | 保留现状，检查归属；不要删数据让检查通过 |
| 核心下载、TLS、摘要或空间检查失败 | 检查设备原厂联网、`223.5.5.5:53`、系统时间和空间，保留官方来源与校验 |
| root 被拒绝或找不到 Magisk | 检查 U30 Pro 本机的 Magisk 和本次授权 |
| 安装器结果未知或超时 | 先看 Magisk 是否已暂存模块，避免重复安装 |
| 重启后没有控制应用或“本机服务未连接” | 等待初始化，检查模块是否启用；持续失败时提供脱敏错误类别 |
| “请先添加有效订阅，再开启 VPN” | 导入 UFI 插件，保存订阅，等节点出现后再开启 |
| 网页没有 VPN 插件 | 手动导入 `U30ProVPN.txt`，确认 UFI 已登录且 root 接口可用 |
| 卸载清理延后或联网异常 | 保留私有数据，检查 VPN、模块和网络管理状态；不要全局清空规则 |

其他问题见 [FAQ](faq.md)，反馈方式见[使用说明](usage.md#故障反馈与部署验收)。

## 模块已删除但清理延期时的人工恢复

<details>
<summary>供熟悉 root 和文件清单的维护者使用</summary>

Magisk 可能在清理未完成时仍删除模块目录，此时无法自动补完。可恢复**同一版本的清理工具**：

1. 保留 `/data/adb/u30pro-vpn`、安装 journal 和所有私有备份，确认 Android、PackageManager 与网络管理已就绪。
2. 取得同版本 Magisk ZIP 并核对 Release 的 SHA-256。已安装和待安装的同 ID 模块目录必须都不存在；若存在，先检查归属。
3. 在可信 root shell 新建 `/data/adb/modules/codex_u30pro_vpn`，立即创建其中的 `disable` 标记，再恢复已核验 ZIP 的原始文件。不要执行 `customize.sh` 或 `magisk --install-module`；清理无需再次下载核心。
4. 设置 root:root 所有权、目录 `0700`，文件按 `manifest.json` 的 `mode` 设置：数据 `0600`，列明的脚本与程序 `0700`。拒绝符号链接和额外 hook，保留 `disable` 标记。
5. 在同一 root shell 执行：

   ```sh
   /data/adb/modules/codex_u30pro_vpn/u30setup uninstall --module /data/adb/modules/codex_u30pro_vpn
   ```

6. 确认输出 `ok: true`、`complete: true` 和备份路径，检查客户端原厂联网，再在 Magisk 中完成移除与重启。若仍延期或归属错误，保留现状继续排查，不全局清空规则或删除私有目录。

此流程只恢复本版清理入口，不适用于其他版本、未知部署或无法启动的设备。

</details>

## 开发者与许可

源码构建与签名见[构建说明](building.md)，网关接入见[设备集成](device-integration.md)。原创代码和文档采用 [MIT](../LICENSE)，上游组件许可见[第三方说明](../THIRD_PARTY_NOTICES.md)。
