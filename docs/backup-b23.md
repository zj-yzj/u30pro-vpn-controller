# B23 root 前备份与有条件恢复

[English](backup-b23.en.md) · [B23 root 记录](root-b23.md) · [完整准备流程](setup.md)

本页为原版 **MU5358 / U30ProHW1.0 / `U30ProV1.0.0B23`** 提供五个关键分区的双读备份，以及分区表仍完整时的定向恢复方法。目标是在 root 前留下本机的完整原始分区文件；它不覆盖整机固件、NV、用户数据或所有救砖情形。

命令依据固定的 `spd_dump` 源码提交 **`23c5ab211ca1e58e2fb8c70108d2bccfd0110925`** 审阅。该版本没有新版的自动 GPT 修复路径，但这不等于重新验证了每台 B23 的下载模式入口或原厂恢复。本次没有重新连接、刷写或恢复设备。

## 1. 在关机前下载和准备

下载 [B23 备份支持包](https://github.com/zj-yzj/u30pro-vpn-controller/releases/download/b23-backup-tools-20261019/U30Pro-B23-Backup-Tools-20261019.zip)（[独立 Release](https://github.com/zj-yzj/u30pro-vpn-controller/releases/tag/b23-backup-tools-20261019)），完整解压。包内 `tools/spd_dump.exe` 是本页固定的 reader；根目录提供 `Test-U30ProBackup.ps1` 和 `expected-sizes.example.csv`，仓库中的检查器源码见 [support/b23-backup](../support/b23-backup/Test-U30ProBackup.ps1)。不要将 reader 与历史 root 套件中同名的 EXE 混用。

在电脑先准备好 [B23 套件中的下载模式驱动](root-b23.md#1-准备工具和恢复材料)、可传数据的 USB 线和足够的本地空间。下载模式连接与 Android 的 ADB 是两条通道：本页不要求设备已经 root、安装 Magisk、开启普通 ADB 或无线 ADB。**不要先运行一键 root BAT 来进入备份**，那些脚本会继续写入镜像。

备份保存在支持包之外的新目录。下面路径仅作示例；目录已存在时换一个新名字，不覆盖旧备份。在支持包解压根目录打开 PowerShell，执行：

```powershell
New-Item -ItemType Directory -Path 'C:\U30Pro-B23-Backup' -ErrorAction Stop
New-Item -ItemType Directory -Path 'C:\U30Pro-B23-Backup\pass1','C:\U30Pro-B23-Backup\pass2','C:\U30Pro-B23-Backup\readback' -ErrorAction Stop
Copy-Item -LiteralPath '.\expected-sizes.example.csv' -Destination 'C:\U30Pro-B23-Backup\expected-sizes.csv'
```

保留设备型号、完整固件、硬件版本与备份发生时间。已经 root 后读出的文件只能标为当前状态，不能补称为 root 前原厂备份。

## 2. 只进入已记录的 B23 下载模式

断开 USB，将 U30 Pro 关机。在支持包根目录运行：

```powershell
.\tools\spd_dump.exe --wait 300 --kickto 2 exec
```

看到等待设备后，连接关机的 U30 Pro。这里沿用 B23 原脚本的入口，已经去掉后续 root 写入命令。必须等到 **`FDL2 >`** 才进行下文；出现超时、错误模式或无法识别时停止检查连接和驱动，不添加其他 ROM 的 FDL、猜地址、切槽或转入短接。

此后无前缀的命令都输入 reader 的 `FDL2 >` 提示符，不能输入 PowerShell。先查看分区与大小：

```text
p
size_part init_boot_a
size_part init_boot_b
size_part trustos_a
size_part trustos_b
size_part misc
```

要求分区表中五个名字都真实存在，并有正常的 A/B 检测和当前槽信息。记录实际槽位，不默认 B 槽，也不用 `slot` 或 `set_active` 补造检测结果。存在 `w_force` 临时分区、名字/布局不匹配、大小为零或读取报错时，停在这里。

`size_part` 给出字节数。逐项记录并检查它们与本机分区布局一致；`p` 和生成的 XML 使用 MiB 显示，不能直接当成所有分区的精确字节数。将 `expected-sizes.csv` 的每个占位值替换为对应分区实际返回的正整数字节数：

```csv
partition,bytes
init_boot_a,REPLACE_FROM_SIZE_PART
init_boot_b,REPLACE_FROM_SIZE_PART
trustos_a,REPLACE_FROM_SIZE_PART
trustos_b,REPLACE_FROM_SIZE_PART
misc,REPLACE_FROM_SIZE_PART
```

模板不能原样用于检查。不要从其他设备抄大小，也不要把 `tos.bin` 等补丁文件大小当成原分区大小。

## 3. 五个分区分别读两遍

以下所有 **`SIZE` 都是占位符**：每一行必须换成 CSV 中该分区对应的实际字节数。保持偏移为 `0`、名字带明确槽后缀；`misc` 没有槽后缀。不要用 `0xffffffff` 自动探测代替已经核对的长度。

第一遍输入：

```text
path C:\U30Pro-B23-Backup\pass1
partition_list partition.xml
read_part init_boot_a 0 SIZE init_boot_a.bin
read_part init_boot_b 0 SIZE init_boot_b.bin
read_part trustos_a 0 SIZE trustos_a.bin
read_part trustos_b 0 SIZE trustos_b.bin
read_part misc 0 SIZE misc.bin
```

确认每项输出的 `target` 与 `read` 都等于预期长度，再读取到另一个空目录：

```text
path C:\U30Pro-B23-Backup\pass2
partition_list partition.xml
read_part init_boot_a 0 SIZE init_boot_a.bin
read_part init_boot_b 0 SIZE init_boot_b.bin
read_part trustos_a 0 SIZE trustos_a.bin
read_part trustos_b 0 SIZE trustos_b.bin
read_part misc 0 SIZE misc.bin
```

`path` 只设置保存目录，不创建目录，也不改变恢复时输入文件的位置；不设置时输出到当前工作目录。有 `path` 时，reader 按输出文件的基本名保存。每遍使用独立目录，避免第二遍覆盖第一遍。对应实现见固定版本的[输出路径处理](https://github.com/TomKing062/spreadtrum_flash/blob/23c5ab211ca1e58e2fb8c70108d2bccfd0110925/common.c#L715)和[读取命令](https://github.com/TomKing062/spreadtrum_flash/blob/23c5ab211ca1e58e2fb8c70108d2bccfd0110925/spd_dump.c#L890)。

启动时可能已在支持包根目录产生 `pgpt.bin`、`sprdpart.bin` 或带时间的分区 XML。另行保留这些文件和本次输出记录；它们不会因为后来设置 `path` 而自动搬到备份目录。`pgpt.bin` 只是 `user_partition` 开头的 32 KiB，不能标为完整 GPT/整盘备份，XML 也不能直接当作自动恢复脚本。

## 4. 离线核对后再继续 root

完成两遍读取后，可在另一个 PowerShell 窗口、支持包根目录执行以下检查，不向设备发命令：

```powershell
powershell.exe -NoProfile -File .\Test-U30ProBackup.ps1 `
  -BackupDirectory 'C:\U30Pro-B23-Backup\pass1' `
  -SecondReadDirectory 'C:\U30Pro-B23-Backup\pass2' `
  -SizesFile 'C:\U30Pro-B23-Backup\expected-sizes.csv'
```

检查器核对两遍各自的五个分区文件及完整长度、两遍 SHA-256 一致性，并检查每遍的 `partition.xml` 都包含五个名字、没有 `w_force` 且两份表的摘要一致。成功时输出 `ok: true`、`scope: critical-partition-file-consistency-only` 的 JSON；出现错误时不继续 root。reader 的退出码或“Read Part Done”本身不够：读取中断可能留下短文件，见[读取循环](https://github.com/TomKing062/spreadtrum_flash/blob/23c5ab211ca1e58e2fb8c70108d2bccfd0110925/common.c#L867)。

该命令使用 Windows 自带的 PowerShell，无需另装 PowerShell 7。若本机策略阻止脚本运行，按现有策略核验并处理这份脚本，不永久放宽全局执行策略。

JSON 通过只表示这两组文件满足离线检查条件；它不能证明两组文件来自独立的物理读取，也不证明硬件兼容、内容是原厂状态、完整克隆或恢复必定成功。A/B 检测仍由操作时人工核对。把原始备份、大小表和检查结果另存一份，不上传含本机状态的备份。确认保留完成后，在 `FDL2 >` 输入 `reset`，等设备正常启动，再决定是否按 [B23 root 记录](root-b23.md)继续。

## 5. 仅在条件满足时恢复实际改动的分区

定向恢复要求：备份来自这台设备、在 root 前取得且已双读核验；当前仍能进入同一 FDL2 环境；分区表、名字和大小与备份时相同；A/B 检测正常，没有 `w_force` 临时名字；清楚实际改动了哪个槽的 `init_boot`、`trustos`。未知部署、损坏分区表、不同固件和无法进入下载模式不适用下面的模板。

重新进入同一会话后，先重复 `p`、实际目标的 `size_part` 和槽信息核对。**只恢复确实被改动的分区**，不为了“完整”而写另一槽。下面的 `PARTITION`、`SIZE` 必须换成实际名字和长度，路径也必须指向对应原始文件；这是模板，不能原样运行：

```text
skip_confirm 0
w PARTITION C:\U30Pro-B23-Backup\pass1\PARTITION.bin
path C:\U30Pro-B23-Backup\readback
read_part PARTITION 0 SIZE PARTITION.bin
```

`skip_confirm 0` 打开写入确认。固定版本在 A/B 已识别且使用存在的精确槽后缀名字时，普通 `w` 直接写该目标；没有这个条件，普通写入也可能进入 `_bak` 联动逻辑。源码见[名字解析](https://github.com/TomKing062/spreadtrum_flash/blob/23c5ab211ca1e58e2fb8c70108d2bccfd0110925/common.c#L1659)和[普通写入分派](https://github.com/TomKing062/spreadtrum_flash/blob/23c5ab211ca1e58e2fb8c70108d2bccfd0110925/common.c#L2178)。不要用不带后缀的 `init_boot`、`trustos` 代替实际目标。

每次写完立即读回，在另一个 PowerShell 窗口核对原件与读回件的完整长度及 SHA-256：

```powershell
Get-Item -LiteralPath 'C:\U30Pro-B23-Backup\pass1\PARTITION.bin','C:\U30Pro-B23-Backup\readback\PARTITION.bin' | Select-Object Name,Length
Get-FileHash -Algorithm SHA256 -LiteralPath 'C:\U30Pro-B23-Backup\pass1\PARTITION.bin','C:\U30Pro-B23-Backup\readback\PARTITION.bin'
```

两份长度必须等于记录的 `SIZE`，摘要必须一致。全部实际目标都通过后，才在 reader 输入 `reset`。启动后检查原厂首页、Wi-Fi 和联网，再核对预期的 root 状态。写入被拒绝或读回不一致时保留结果，不升级成强制写入、分区重建或切槽；本页没有证明这些故障的完整恢复路线。

`misc` 与分区表在本流程中用于留证和核对，**不自动写回**。不要把整个备份目录交给 `write_parts`，也不使用 `w_force`、`repartition`、NV 写入或整盘写入。旧版 `w_force` 实际会临时重建分区表后再改回，并非普通单分区写入，见[实现](https://github.com/TomKing062/spreadtrum_flash/blob/23c5ab211ca1e58e2fb8c70108d2bccfd0110925/common.c#L1364)。

## 这份备份没有覆盖什么

`r all` 会跳过 `blackbox`、`cache` 和 `userdata`；`all_lite` 还跳过另一槽。该旧版本把 `splloader` 读取长度固定为 256 KiB。NV 读取还存在改名、跳过头部等专用处理，批量恢复会推断槽位并可能改动其他对象。因此本页只提供上述五分区的有限备份与定向恢复，不将它称为完整原厂镜像或一键救砖。上游的[固定版本说明](https://github.com/TomKing062/spreadtrum_flash/blob/23c5ab211ca1e58e2fb8c70108d2bccfd0110925/README.md)也列明批量读取的排除项。

2026-10-19 核查时，[中兴 U30 Pro 官方支持页](https://www.ztedevices.com/cn/supports/u30-pro/)的 SD 卡升级和软件升级栏目均显示没有可用软件；本次未找到可核验的官方 B23 完整恢复 PAC。因此 Release 不提供来源不明的“原厂恢复包”，也不将 B24 镜像用作 B23 恢复材料。
