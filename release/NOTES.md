## 中文

这是适用于**已经 root 且已安装兼容 Magisk 的 U30 Pro** 的首个一键安装预发布版。提供两个入口，使用同一份模块 ZIP：

| 下载文件 | 用途 |
| --- | --- |
| `U30ProVPN-1.3.0-alpha.1-Magisk.zip` | 在 Magisk 的模块页面选择从本地安装 |
| `U30ProVPN-1.3.0-alpha.1-Installer.apk` | 独立 Android 安装器；取得 root 后调用 Magisk 安装内嵌 ZIP |
| `U30ProVPN.txt` | 导入设备已有 UFI-TOOLS 的项目网页插件，用于首次录入订阅 |
| `SHA256SUMS` | 下载文件的 SHA-256 清单 |
| `APK-CERTIFICATE.txt` | 公开版 APK 的签名验证与证书指纹 |
| `release-manifest.json` | 源码提交、适配范围、资产摘要与核心来源 |

**适配范围：** MU5358/U30 Pro，显示固件 `U30ProV1.0.0B23`，Android API 35，arm64-v8a，SELinux Permissive，Magisk 版本码至少 28100，以及安装器要求的原厂网络布局。仅有 root 并不足以保证兼容。安装器不会修改 SELinux 模式。

**全新安装流程：** 先阅读[中文安装说明](https://github.com/zj-yzj/u30pro-vpn-controller/blob/v1.3.0-alpha.1/docs/install.md)，任选 ZIP 或 APK 入口安装，成功暂存后明确重启。首次初始化保持 **VPN 关闭、节点与订阅为空**。随后将 `U30ProVPN.txt` 导入已有 UFI-TOOLS，录入自己的订阅，再开启 VPN。UFI-TOOLS、Magisk、固件和任何订阅均不随包提供。

安装准备需要设备可直接联网：从 Mihomo 官方 Release 获取固定版本核心，校验 HTTPS 证书、压缩与解压后的精确大小和 SHA-256。ZIP 与 APK **不内嵌 Mihomo 核心**，也不支持离线首次安装。下载失败会停止准备，不改用 HTTP 或忽略证书。核心获取会产生一次约 21.9 MB 的下载；正常使用和手动测速另计流量。

本版仅接受全新安装，拒绝未知旧运行目录、同名控制应用和冲突的网络对象，不是现有私有部署的升级包。安装器的“已暂存”与重启后的“已安装”是不同阶段。断电重入只恢复本版自己记录的事务，不能代替设备救援。卸载采用受控停止并保留 root 私有备份；环境未就绪时可能延期，需要按说明核查。

APK 入口在调用 Magisk 前还会检查已安装和待安装的同 ID 模块。直接 ZIP 入口须先在 Magisk 中确认没有同 ID 的待安装模块：Magisk 自身可能在运行本项目检查之前替换其暂存目录，因此不能承诺保留旧的同 ID 暂存内容。

**验证边界：** 主机测试、双 APK 构建与签名验证、模块文件与权限校验、APK 内嵌 ZIP 一致性检查；尚未在空白匹配设备上完成全流程安装、订阅、转发及重启验收。故标为 alpha 预发布。开启状态下的冷启动存在服务建立时间，不承诺从上电起零直连或网络不可识别。请勿把下载量、root 或混淆视为匿名保证。

本版保持按需测速：没有定时全节点测速或 provider health-check；用户点击刷新或单节点“测速”才启动相应测量。状态读取与服务健康检查仍用于界面和运行维护。

原创代码使用 MIT；第三方声明与许可证保留在源码和模块 `licenses/`，安装器内嵌同一模块，因此一并保留。

## English

This is the first **alpha one-click installer for an already-rooted U30 Pro with compatible Magisk**. Choose either the Magisk ZIP or the separate Android installer APK; the APK embeds the identical ZIP and invokes Magisk after explicit root authorization. Reboot is an explicit action after successful staging.

The strict profile is MU5358/U30 Pro, firmware display `U30ProV1.0.0B23`, Android API 35, arm64-v8a, SELinux Permissive, Magisk version code 28100 or later, and the expected stock network layout. Root alone does not establish compatibility. The installer does not change SELinux mode.

Read the [English installation guide](https://github.com/zj-yzj/u30pro-vpn-controller/blob/v1.3.0-alpha.1/docs/install.en.md) before installing. A fresh installation starts with **VPN OFF, no nodes and no subscription**. Import the separate `U30ProVPN.txt` plugin into an existing UFI-TOOLS installation, add your own subscription, and then enable VPN. UFI-TOOLS, Magisk, firmware and subscriptions are not bundled.

Preparation requires direct device Internet access to download the pinned Mihomo asset from its official Release. HTTPS certificates and both compressed/decompressed sizes and SHA-256 digests are checked. Neither downloadable package embeds the core. Offline first installation is unavailable; download failure does not trigger an insecure fallback. The one-time compressed core download is about 21.9 MB; ordinary use and manual latency checks consume additional traffic.

This is **fresh-install only**, not an upgrade for existing private deployments. Unknown runtime directories, same-name control applications and reserved network collisions are refused. Staging is distinct from boot initialization. Recovery resumes only transactions owned by this exact release. Uninstall uses controlled shutdown and retains a root-private backup; cleanup can be deferred if the required system services are unavailable.

The APK additionally checks both active and staged module namespaces before invoking Magisk. For the direct ZIP entry, first ensure that no same-ID staged module exists: Magisk itself can replace that staging directory before this project's checks run, so preservation of previous same-ID staging contents is not guaranteed.

Validation covers host tests, both APK builds/signatures, module inventory/modes and embedded ZIP identity. **No clean-device end-to-end installation, subscription, forwarding and reboot acceptance has been completed.** This is why the Release is marked prerelease. An enabled cold boot has a service startup window; there is no zero-direct-traffic-from-power-on or undetectability guarantee.

There are no periodic all-node probes or provider health checks. List refresh and single-node latency tests remain user-triggered. Local status reads and service health checks remain part of normal operation.

`SHA256SUMS` verifies downloads; `APK-CERTIFICATE.txt` records the public release signing certificate; `release-manifest.json` records the source commit, supported profile and asset/core digests. Original code is MIT licensed. Required third-party licenses and notices are retained in the source, module and embedded module inside the installer APK.
