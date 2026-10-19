# B23 关键分区备份工具 / B23 critical-partition backup tools

给未 root 的原版 **MU5358 / U30ProHW1.0 / U30ProV1.0.0B23** 补上刷写前的关键分区备份步骤。先读[完整准备流程](https://github.com/zj-yzj/u30pro-vpn-controller/blob/main/docs/setup.md)和[备份与有条件恢复教程](https://github.com/zj-yzj/u30pro-vpn-controller/blob/main/docs/backup-b23.md)，确认适配后再操作。

**[下载备份支持包](https://github.com/zj-yzj/u30pro-vpn-controller/releases/download/b23-backup-tools-20261019/U30Pro-B23-Backup-Tools-20261019.zip)**

包内有：

- `tools/spd_dump.exe`：从固定的 `23c5ab` 源码构建的 Windows reader，附配套 DLL 和配置；无需另外安装 VC++ 运行库。
- `Test-U30ProBackup.ps1` 和尺寸表模板：用 Windows 自带 PowerShell 核对两次读取的五个关键分区文件、长度和 SHA-256。
- 中英文操作说明、构建记录、来源与许可说明、文件清单和校验值。

下载模式驱动仍使用 [B23 root 套件](https://github.com/zj-yzj/u30pro-vpn-controller/releases/tag/b23-rootkit-archive-20261019)中的版本。源码参考及构建来源记录在本 Release 单独提供，日常使用不必下载源码。

流程读取两槽 `init_boot`、`trustos` 和 `misc`，在 root 前保存本机原件并做双读核对。恢复仅限于分区表完整、下载通道和 A/B 检测正常时，使用本机原件写回实际改动的明确目标，再立即读回验证。没有一键批量回刷、自动修表、NV 回写或全盘写入。

这是依据固定源码和 B23 原套件入口整理的条件流程，**本次没有在另一台原厂 B23 上重新完成 root 或恢复演练**。本次未找到可核验的官方 B23 完整恢复固件；此包不提供原厂 PAC、整机克隆或通用救砖保证，也不含维护者的任何设备备份。工具和上游材料保留原有署名与各自许可状态，不统一适用本项目 MIT。

---

This package adds pre-root critical-partition backup steps for the original **MU5358 / U30ProHW1.0 / U30ProV1.0.0B23**. Read the [full setup guide](https://github.com/zj-yzj/u30pro-vpn-controller/blob/main/docs/setup.en.md) and [backup / conditional recovery guide](https://github.com/zj-yzj/u30pro-vpn-controller/blob/main/docs/backup-b23.en.md) before use.

The ZIP contains a locally built, pinned `23c5ab` Windows reader with its DLL/configuration, an offline PowerShell checker, a size-table template, bilingual instructions, build/provenance records and checksums. Static CRT linkage avoids a separate VC++ runtime installer. Use the download-mode driver from the [B23 root archive](https://github.com/zj-yzj/u30pro-vpn-controller/releases/tag/b23-rootkit-archive-20261019). The upstream source snapshot is an optional separate download.

Read both slots' `init_boot` and `trustos`, plus `misc`, twice before rooting. The checker verifies exact local lengths and matching SHA-256 values. It cannot establish stock origin, independent hardware reads or restore success. Conditional rollback requires the same device's originals, an intact matching map, working FDL2 access and positive A/B detection; restore only the actual changed targets and immediately verify complete readbacks.

There is no bulk flash-back, automatic repartitioning, NV restoration or whole-disk write procedure. **A fresh original B23 root/restore rehearsal was not performed in this work.** No verifiable official B23 recovery firmware was found in this research. This package contains no stock PAC or device dumps and makes no full-clone or universal unbrick claim. Third-party attribution and license status are retained separately from the controller's MIT license.
