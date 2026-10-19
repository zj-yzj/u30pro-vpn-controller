# 安装与恢复

[English](install.en.md) · [项目首页](../README.md) · [使用说明](usage.md) · [常见问题](faq.md)

**当前版本：`v1.3.0-alpha.1`，首次安装预发布。** 安装入口为 Magisk 模块 ZIP 或 Android 安装器 APK；两者安装的是同一个模块，均要求 U30 Pro 已有兼容的 Magisk。项目不获取 root、不刷写固件，也不负责升级已有的私有部署。

本版本尚未在一台干净设备上完成“安装 → 重启 → 录入订阅 → 客户端联网 → 关闭 → 卸载”的完整实机验收。主机测试、签名检查和文件摘要核对不能替代这个过程。具体发布验证范围见 [Release 说明](https://github.com/zj-yzj/u30pro-vpn-controller/releases/tag/v1.3.0-alpha.1)。

## 安装前确认

以下条件必须全部满足。安装器对不匹配或未知状态应拒绝继续；不要通过修改属性、关闭 SELinux 或删掉现有文件绕过检查。

| 项目 | 本 alpha 固定范围 |
| --- | --- |
| 设备与型号属性 | U30 Pro；device/model 均为 `MU5358` |
| 系统与架构 | 原厂 Android，API **35**，`arm64-v8a` |
| 固件 display | 精确为 `U30ProV1.0.0B23` |
| root 管理 | 已安装并正常工作的 **Magisk 28.1 或更新版本**，版本码至少 `28100` |
| SELinux | 系统本来就为 `Permissive`；安装器不降低 SELinux 级别 |
| 热点网络 | `br0`，`192.168.0.1/24`，IPv4 forwarding 已开启，SFP 基线为 `1` |
| 首次部署 | 没有现有 `/data/adb/u30pro-vpn` 或同包控制应用；`/data/adb/modules/codex_u30pro_vpn` 与 `/data/adb/modules_update/codex_u30pro_vpn` 均不存在，也没有冲突的专用网络对象 |
| 订阅录入 | 已安装 UFI-TOOLS，能够登录并使用其认证 root 接口 |
| 网络与空间 | U30 Pro 自身可访问 GitHub 官方发布资源，且安装器空间检查通过 |

`minSdk 26` 是 APK 的编译声明，不表示安装器支持 Android 8 至 15 的所有系统。仅有其他 root 实现而没有兼容 Magisk，不在本版本范围内；其他 U30 Pro 固件、KernelSU、APatch、其他热点和 OpenWrt 也未声明支持。保留现有可用网络和设备原有恢复入口，安装前先阅读本页的恢复部分。

**Magisk 暂存目录边界：** 模块 ID 为 `codex_u30pro_vpn`。手动 ZIP 安装前，须确认 Magisk 没有该 ID 的已安装模块或等待重启的暂存模块；不确定时停止并让维护者只读核查上面的两个目录。Magisk 自己可能在执行项目的 `customize.sh` 检查之前就替换同 ID 的 `modules_update` 内容，因此项目不承诺该命名空间在检查前完全无写入或可保留旧暂存内容。APK 入口会在调用 Magisk CLI 前预检两个目录；这不扩大到手动 ZIP 的框架行为。安装期间不要同时从另一个入口执行模块安装。

## 下载哪些文件

打开 [v1.3.0-alpha.1 Release](https://github.com/zj-yzj/u30pro-vpn-controller/releases/tag/v1.3.0-alpha.1)，从 Assets 下载下列文件。GitHub 自动生成的 Source code 压缩包供开发者使用，不是 Magisk 安装包。

| 文件 | 用途 |
| --- | --- |
| `U30ProVPN-1.3.0-alpha.1-Magisk.zip` | 在 U30 Pro 的 Magisk App 中安装模块 |
| `U30ProVPN-1.3.0-alpha.1-Installer.apk` | 可选的 U30 Pro 本机 Android 安装入口，内嵌与上面完全相同的 ZIP |
| `U30ProVPN.txt` | 首次录入订阅需要的 UFI 网页插件；两种入口都要手动导入 |
| `SHA256SUMS` | 下载文件的 SHA-256 校验清单 |
| `APK-CERTIFICATE.txt` | 公开的 APK 签名验证信息与证书指纹 |
| `release-manifest.json` | 版本、文件身份与构建来源信息 |

两种安装入口选一种即可。不要将 ZIP 解压后直接复制到设备运行目录，不要把“Source code.zip”交给 Magisk。

在电脑核对下载摘要时，可使用 `Get-FileHash -Algorithm SHA256 -LiteralPath "下载的文件名"`，与同一 Release 的 `SHA256SUMS` 比较。Android 安装器还会校验自己内嵌 ZIP 的大小和摘要；这不代替核对下载来源。

## 核心通过网络下载

项目 ZIP 和 APK **不含 Mihomo 核心二进制**。准备阶段由设备 root helper 从 [Mihomo 官方 v1.19.32 Release](https://github.com/MetaCubeX/mihomo/releases/tag/v1.19.32) 下载固定的 `mihomo-android-arm64-v8-v1.19.32.gz`，验证 HTTPS 证书，并核对下载前后固定的大小和 SHA-256。

| 内容 | 字节数 | SHA-256 |
| --- | ---: | --- |
| gzip 下载，约 21.9 MB | `21866463` | `42b1f2be8702484d3ae6238daab392622f54bd0d264966700f1f28f52596e326` |
| 解压后的核心，约 64.0 MB | `64045272` | `bb9afbd48717f92d59e07da35fb694b83d40d6edb63eb9033c2ee4174137d195` |

因此**不支持离线首次安装**，即使 APK 已经下载到设备也一样。GitHub 下载不可达、TLS 证书校验失败、空间不足或任何摘要不符都会使准备失败。不要切换到不明镜像、关闭证书校验或改用未经核对的核心。下载核心与下载你的订阅是两件事；安装器不附带订阅。

核心下载使用固定 IPv4 DNS `223.5.5.5:53` 解析上游地址，与现有订阅下载的解析方式一致。设备需要能够访问该 DNS 和 GitHub 下载链路；这不依赖安装中的 VPN 为自己提供连接。

## 方式一：使用 Magisk ZIP

1. 把模块 ZIP 传到 **U30 Pro 本机**，保持设备自身可以联网。
2. 确认 Magisk 中没有同 ID 的已安装或待重启模块，再进入“模块”，选择从本地存储安装，选中 `U30ProVPN-1.3.0-alpha.1-Magisk.zip`。
3. 等待准备结束，包括核心下载与校验。准备只暂存模块，不应开启 VPN、改动热点路由或安装控制 APK。出现失败或未知状态时，按下文检查，不要反复覆盖安装。
4. Magisk 确认模块已暂存后，主动选择重启。模块文件暂存成功不等于启动安装已经完成。
5. 重启后等 Android 与模块初始化完成。在 Magisk 模块卡片使用“操作 / Action”打开圆屏“VPN 控制”。初始应为 **VPN 已关闭、无节点、未保存订阅**。控制应用暂时不可用时，等待启动并检查模块状态。
6. 继续完成下文的 UFI 插件导入和订阅录入。

## 方式二：使用 Android 安装器 APK

这个 APK 要安装在 **U30 Pro 上**，不是连接热点的手机上。它是单独的 `com.codex.u30vpn.installer` 应用；日常圆屏控制应用为 `com.codex.u30vpn`。

1. 通过你已有的设备管理方式，将 `U30ProVPN-1.3.0-alpha.1-Installer.apk` 安装到 U30 Pro，并打开“U30 Pro VPN 安装”。若系统询问 APK 安装来源权限，使用正常系统授权流程。
2. 点“安装模块”。在 Magisk 的 root 授权提示中允许这次安装。安装器校验内嵌 ZIP，然后通过 `magisk --install-module` 安装它；不接受用户提供的模块 URL。
3. 等待结果。只有明确显示模块已暂存时，“现在重启”才可使用。你可以主动点击它，或稍后自己重启；应用不会自动重启。
4. 重启后重新打开安装器，使用“打开控制”，或使用 Magisk 模块的 Action。核实控制页显示 VPN 关闭且未配置订阅，再继续下面的步骤。

如果你使用电脑上的 ADB，且已确认连接的唯一目标就是 U30 Pro，可以在下载目录执行：

```sh
adb install "U30ProVPN-1.3.0-alpha.1-Installer.apk"
adb shell am start --user 0 -n com.codex.u30vpn.installer/.MainActivity
```

设备有多个 ADB 连接时先选择正确的目标。ADB 只负责传入并打开 APK，安装模块仍需设备已有 Magisk 和用户 root 授权。没有可用的 APK 安装/启动入口时，使用 Magisk ZIP 方式。

安装器本身没有 `INTERNET` 权限，在线核心下载由 root helper 执行。安装器超时、进程被杀、结果丢失或显示“状态未知”时，不代表 Magisk 已停止或已经回滚。检查 Magisk 的模块状态后再决定下一步；不要连续点击或重新安装来猜测结果。

## 首次录入订阅并开启

1. 在手机或电脑连接 U30 Pro 的网络，打开你现有的 UFI-TOOLS 网页，正常登录，并确认 root 高级功能可用。
2. 进入“插件功能 → 添加插件(.txt文件) → 提交”，导入同一 Release 的 `U30ProVPN.txt`。安装模块不会自动导入插件，也不安装 UFI 本体。
3. 打开 VPN 插件，读取状态。初始应为关闭、节点列表为空。在“订阅”中粘贴你自己的 HTTPS 订阅 URL，执行保存/更新，等待成功。
4. 核实节点已经出现。**录入订阅仍保持 VPN 关闭**；圆屏“订阅”页只能更新已保存 URL，新增和更换 URL 使用这个网页。
5. 你准备好使用时，再在网页或圆屏主动开启 VPN。等待实际运行状态，随后从连接热点的客户端检查 DNS 与 HTTPS 联网。单纯看到开关已开或节点名称不等于实际联网已验证。
6. 需要延迟结果时，在 VPN 已运行的圆屏节点页手动点底部“刷新”，或左滑普通节点后点“测速”。没有定时批量测速，初始没有测速值是正常情况。

订阅下载限制、节点选择和失败后的恢复语义见[使用说明](usage.md)。不要把 URL、节点凭据或设备私钥贴进 issue。

## 圆屏快捷入口

Magisk 模块 Action 和安装器的“打开控制”可进入控制页。“更多 → 快捷入口”提供可选悬浮按钮，必须由你在系统设置中授权显示在其他应用上层。安装器不代替你授权，也不修改原厂首页。权限页不可用或原厂首页识别不匹配时先继续使用其他打开入口。

关闭快捷入口或退出应用不会关闭 VPN；停止网络代理要明确点“关闭 VPN”，等待实际状态并检查原厂联网恢复。

## 重启、重复安装与恢复

- **首次启动**创建空 provider、空节点状态和设备自己的管理密钥，VPN 关闭。安装器不会带入开发设备的订阅、UID、trust 或签名私钥。
- **同版本中断重入**只对可核验的本安装事务继续处理，保留已有密钥和已提交配置。Android APK 安装与 trust 写入可能跨越重启，不能手工复制别台设备的 trust 来补齐。
- **已有私有部署或未知同包 APK**会被项目检查拒绝。本 alpha 不提供旧部署升级、覆盖、签名迁移或自动卸载重装；请保留原安装并单独规划迁移。对 Magisk 同 ID 暂存内容还须遵守安装前目录检查，不能依赖项目检查保存它。不要为通过检查而删除目录或强卸载原应用。
- **后续正常重启**会读取你保存的 VPN 目标状态。若上次开启，Android/网关就绪前存在冷启动窗口；本版不承诺从通电到 VPN 就绪期间所有客户端流量均受隧道或阻断保护。需要严格覆盖这段时间时，应等待 VPN 实际运行后再接入业务客户端。
- **模块禁用与运行中的关闭不同。** Magisk 中禁用模块主要控制后续启动，不应假定点击禁用立刻停止当前网关。先在控制页关闭 VPN，确认原厂联网恢复，再进行模块禁用或卸载。

## 卸载与保留的数据

先关闭 VPN 并确认热点客户端恢复原厂联网，再在 Magisk 中移除本模块并按提示重启。卸载流程只处理清单可证明归属的本项目对象；控制 APK 也只有身份与本发行物匹配时才允许移除。它不会卸载未知同包应用，也不全局清空防火墙或路由规则。

私有运行数据会保留在 root 私有备份中，可能含订阅 URL、节点凭据、状态和密钥。不要上传该备份。卸载不等于销毁私密数据；如需永久删除，应先核实备份归属和实际联网恢复，再由了解该设备的维护者处理。

如果卸载时 PackageManager 或网络管理尚不可用，可能报告 **cleanup deferred / 清理延后**。此时数据被保留，不能视作卸载已完整完成；按下表处理，保留可用的恢复入口。只卸载 Android 安装器 APK 或圆屏控制 APK，也不会自动移除 Magisk 模块或停止已运行的 VPN。

## 常见安装问题

| 现象 | 应做什么 |
| --- | --- |
| 型号、固件、API、Magisk 或 SELinux 不匹配 | 停止安装；本 alpha 只支持上面的固定范围，不降低系统安全设置或伪造属性 |
| 发现已有运行目录、模块、同包应用或网络对象 | 保留现状，本版本不覆盖未知安装；不要以删除数据的方法让检查通过 |
| 核心下载、TLS、摘要或空间检查失败 | 检查 U30 Pro 自身原厂联网、固定 DNS `223.5.5.5:53`、时间和可用空间；保持官方来源与校验，准备失败不应修改 live 运行目录或网络 |
| root 被拒绝或找不到 Magisk | 检查已有 Magisk 和本次授权；root 在连接热点的手机上无效，必须是 U30 Pro 本机 |
| APK 安装器显示未知结果、超时或被中断 | 查看 Magisk 是否已经暂存本模块，保留当前状态；不把超时当作回滚成功，也不反复重新安装 |
| 重启后暂时没有控制应用或“本机服务未连接” | 给 Android/PM 初始化留出时间，检查模块是否启用；持续失败时记录脱敏错误类别，由维护者检查安装事务与实际 APK 身份 |
| “请先添加有效订阅，再开启 VPN” | 导入 UFI 插件并保存自己的有效订阅，等节点出现后再开启 |
| 控制页能打开但网页没有 VPN 插件 | 手动导入 Release 的 `U30ProVPN.txt`，确认 UFI 已登录且认证 root 接口可用 |
| 卸载报告清理延后或仍有异常联网 | 保留私有数据和恢复入口；核实 VPN 实际停止、模块状态与网络管理就绪，再由维护者恢复，不手工全局清空规则 |

反馈可提供版本、型号/固件、所用入口、期望/实际状态和脱敏错误类别，提交到[项目 issue](https://github.com/zj-yzj/u30pro-vpn-controller/issues)。不要上传原始 root 输出、订阅、provider/state/trust、私钥、备份或未经审查的截图。

## 模块已删除但清理延期时的人工恢复

Magisk 不保证因卸载 hook 返回错误而保留模块目录。`cleanup deferred` 会保留私有数据；模块已删除后，它不能自动补完清理。维护者可恢复**同一版本的清理工具**，而不是重新安装来覆盖运行数据：

1. 保留 `/data/adb/u30pro-vpn`、本版安装 journal 和所有私有备份，先确认 Android、PackageManager 与网络管理已就绪。
2. 重新取得本页同版本的 Magisk ZIP，核对 Release 的 SHA-256。已安装和待安装的同 ID 模块目录都必须不存在；若存在，先检查归属，不覆盖。
3. 在可信 root shell 中新建 `/data/adb/modules/codex_u30pro_vpn`，立即创建其中的 `disable` 标记，再把已核验 ZIP 的原始文件恢复到这个新目录。不要执行 `customize.sh` 或 `magisk --install-module`；这条清理路径不需要再次下载核心。
4. 要求 root:root 所有权、目录 `0700`，文件按 `manifest.json` 的 `mode` 设置：数据 `0600`，列明的脚本与程序 `0700`。拒绝符号链接和额外 hook，保留 `disable` 标记，避免恢复工具被当作正常启动模块运行。
5. 在同一 root shell 执行：

   ```sh
   /data/adb/modules/codex_u30pro_vpn/u30setup uninstall --module /data/adb/modules/codex_u30pro_vpn
   ```

6. 只有输出 `ok: true`、`complete: true` 和备份路径后，才可将本工具的清理视作完成；再核实客户端原厂联网，并在 Magisk 中完成移除与重启。若仍返回延期或归属错误，保留现状按错误类别排查，不全局清空规则或删除私有目录。

该流程需要了解 root 和文件清单的维护者，尚未完成干净实机验收；它只恢复本版自己的清理入口，不能处理其他版本、未知部署或设备无法启动的情况。

## 开发者与许可

源码构建、独立签名和发布文件组成见[构建说明](building.md)；固定网关基线与信任边界见[设备集成](device-integration.md)。原项目代码与文档采用 MIT；Mihomo、Magisk、UFI、Android、Go、yaml.v3 和 node-forge 分别保留上游许可，见[第三方说明](../THIRD_PARTY_NOTICES.md)。本项目没有对另行从上游下载的 Mihomo 二进制作完整对应源码分发承诺。
