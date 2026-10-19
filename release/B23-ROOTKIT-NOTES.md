# U30 Pro B23 Root 工具包存档 / Root toolkit archive

## 中文

把维护者提供并实际使用过的 B23 第三方 Root 套件整理成一份可直接下载的存档，方便按项目的准备指南继续安装 UFI 和 VPN。日期 `20261019` 表示本次存档，不是上游版本或官方发布。

**已经有可用的 Magisk root 和 UFI，不需要再刷这份包。** 直接按 [VPN 安装说明](https://github.com/zj-yzj/u30pro-vpn-controller/blob/main/docs/install.md)安装控制器即可。

### 下载

| 文件 | 用途 |
| --- | --- |
| [U30Pro-B23-RootKit-Archive-20261019.zip](https://github.com/zj-yzj/u30pro-vpn-controller/releases/download/b23-rootkit-archive-20261019/U30Pro-B23-RootKit-Archive-20261019.zip) | Root 套件主包，约 52.3 MB；完整解压后先读 README |
| `Magisk-Canary-28103-Source.zip` | 可选源码包，包含对应 Magisk 提交及十个锁定子模块；使用工具不需要下载 |
| `rootkit-file-manifest.json` | 91 个保留原件的路径、大小与 SHA-256 |
| `Magisk-Canary-28103-Source-Receipt.json` | 源码来源、提交和逐文件记录 |
| `SHA256SUMS` | 本次存档资产的 SHA-256 校验清单 |

主包保留了 91 个原始运行文件，另附中英文说明、许可和清单。镜像、BAT、APK、刷写程序、DLL 和驱动字节不变；只排除了两份历史通信日志、一个 PDB 调试文件和一个空提示文件。`B23-RootKit/` 内的原始相对路径保持不变。

### 从哪里开始

1. 阅读 [B23 Root 步骤与核验](https://github.com/zj-yzj/u30pro-vpn-controller/blob/main/docs/root-b23.md)，确认设备、固件、启动链，以及本机备份和恢复方法。
2. 从主 ZIP 解压出的 README 开始，再按 Root 记录操作；配套 Magisk 为 `28103`，需要完成环境修复并重启核验。
3. 按 [设备准备到 VPN 可用](https://github.com/zj-yzj/u30pro-vpn-controller/blob/main/docs/setup.md)准备 UFI、安装 v1.3.0 并导入自己的订阅。

仅面向本文记录的 **MU5358 / U30ProHW1.0 / U30ProV1.0.0B23**。当前证据来自一台实测设备及其 B 槽镜像，不扩大到其他槽位、B24、AI 版或所有同型号机器。自动脚本不会核对这些条件，也不会备份；本包没有原厂恢复镜像，本次没有重新刷机或验证完整救砖。

镜像内置脚本会在每次开机尝试开启 TCP `5555` 无线 ADB、放宽 ADB/系统安全属性，并禁用更新相关组件。这些改动不能在刷入时分别取消，VPN 控制器不要求它们。不要把无线 ADB 暴露到不可信网络。未知写入状态时先停止核查，不并行启动另一个刷机脚本。

### 来源与许可

这是第三方材料的存档，本项目 MIT 不覆盖其中的工具、镜像和驱动。Magisk APK 已核对与[官方 Canary 28103](https://github.com/topjohnwu/Magisk/releases/tag/canary-28103)相同，GPL v3、对应源码和原始许可随存档提供。其他组件的精确来源与适用许可尚未全部确认，按 `NOASSERTION` 如实记录，保留已有版权声明，不宣称本项目原创或官方发布。

[UFI 作者提供过的历史 Root 资源入口](https://github.com/kanoqwq/UFI-TOOLS/issues/97#issuecomment-4624994150)作为来源参考保留；尚未确认该网盘当前内容就是这份本地 B23 包。内容筛查和文件摘要核对不是完整二进制安全审计。

此存档是独立的准备材料，**VPN 控制器的最新正式版仍为 [v1.3.0](https://github.com/zj-yzj/u30pro-vpn-controller/releases/tag/v1.3.0)**。

## English

This archives the third-party B23 root kit supplied and used by the maintainer, so users can download it alongside the project's UFI and VPN setup instructions. `20261019` is the archive snapshot date, not an upstream version or an official release.

**If Magisk root and UFI already work, do not root again for this controller.** Follow [VPN installation](https://github.com/zj-yzj/u30pro-vpn-controller/blob/main/docs/install.en.md) instead.

### Downloads

| File | Purpose |
| --- | --- |
| [U30Pro-B23-RootKit-Archive-20261019.zip](https://github.com/zj-yzj/u30pro-vpn-controller/releases/download/b23-rootkit-archive-20261019/U30Pro-B23-RootKit-Archive-20261019.zip) | Main toolkit, about 52.3 MB; extract it completely and read the README first |
| `Magisk-Canary-28103-Source.zip` | Optional corresponding Magisk source tree and ten pinned submodules; not needed to operate the tools |
| `rootkit-file-manifest.json` | Paths, sizes and SHA-256 values for the 91 retained originals |
| `Magisk-Canary-28103-Source-Receipt.json` | Source repositories, commits and per-file records |
| `SHA256SUMS` | SHA-256 checksums for this archive's assets |

The main ZIP contains 91 original operational files plus bilingual instructions, licensing information and manifests. Images, BATs, APK, flashing program, DLL and driver bytes are unchanged. Only two historical communication logs, one PDB debug file and one empty marker are omitted. Original relative paths inside `B23-RootKit/` are preserved.

### Setup

1. Read the [B23 rooting record](https://github.com/zj-yzj/u30pro-vpn-controller/blob/main/docs/root-b23.en.md) and check the device, firmware, boot chain, original backups and recovery method.
2. Start with the extracted README and follow the root record. The matching Magisk is `28103`; complete its environment repair and reboot checks.
3. Continue with [Device preparation to working VPN](https://github.com/zj-yzj/u30pro-vpn-controller/blob/main/docs/setup.en.md) for UFI, v1.3.0 and your own subscription.

For the recorded **MU5358 / U30ProHW1.0 / U30ProV1.0.0B23** setup only. Existing evidence comes from one tested device and its B-slot images, not other slots, B24, AI firmware or every device of the same model. Automatic scripts do not verify these conditions or create backups. This kit contains no stock recovery images; no fresh flash or complete recovery test was performed for this archive.

The image includes a script that attempts at every boot to enable wireless ADB on TCP `5555`, relax ADB/system security properties, and disable update-related components. These changes cannot be individually deselected while flashing, and the VPN controller does not require them. Keep wireless ADB away from untrusted networks. Resolve an unknown write state before another attempt; do not run flashing scripts in parallel.

### Provenance and licensing

This is an archive of third-party material. The project's MIT license does not cover its tools, images or drivers. The Magisk APK matches [official Canary 28103](https://github.com/topjohnwu/Magisk/releases/tag/canary-28103); GPL v3, corresponding source and original licenses are provided. Exact provenance and applicable licenses for the other components remain partly unconfirmed and are recorded as `NOASSERTION`. Original notices are retained; this is not presented as original project code or an official vendor release.

The [historical root resource linked by the UFI author](https://github.com/kanoqwq/UFI-TOOLS/issues/97#issuecomment-4624994150) remains a provenance reference. Its current cloud contents have not been matched to this local B23 kit. Content screening and checksum verification are not a complete binary security audit.

This is a separate preparation archive. **[v1.3.0](https://github.com/zj-yzj/u30pro-vpn-controller/releases/tag/v1.3.0) remains the latest stable VPN controller release.**
