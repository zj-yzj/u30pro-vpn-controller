# 从设备准备到 VPN 可用

[English](setup.en.md) · [项目首页](../README.md) · [安装与恢复](install.md)

第一次用这个项目，可以按这条顺序准备：**核对固件 → 备份并校验本机关键分区 → 取得可用的 Magisk root → 安装并启用 UFI → 安装 VPN 模块 → 导入自己的订阅 → 测试热点客户端**。已经完成的步骤可以跳过。

当前 v1.3.0 适配的是 **MU5358 / `U30ProV1.0.0B23` / Android API 35**。Root 和 VPN 安装是两件事：下载本项目的 APK 或 ZIP 不会给原厂设备获取 root。

开始前可按这张表下载，避免设备关机后再找文件：

| 材料 | 下载与用途 |
| --- | --- |
| 备份支持包 | [固定 reader、双读校验脚本与教程](https://github.com/zj-yzj/u30pro-vpn-controller/releases/download/b23-backup-tools-20261019/U30Pro-B23-Backup-Tools-20261019.zip) |
| B23 root 套件 | [刷写工具、下载模式驱动与配套 Magisk](https://github.com/zj-yzj/u30pro-vpn-controller/releases/download/b23-rootkit-archive-20261019/U30Pro-B23-RootKit-Archive-20261019.zip) |
| UFI 安装材料 | [完整 UFI、ADB 工具和手册](https://github.com/zj-yzj/u30pro-vpn-controller/releases/download/ufi-tools-archive-20261019/U30Pro-UFI-Tools-Archive-20261019.zip) |
| VPN 安装文件 | [v1.3.0 Release](https://github.com/zj-yzj/u30pro-vpn-controller/releases/tag/v1.3.0) 中选模块 ZIP 或安装器 APK，并下载 `U30ProVPN.txt` |

自备 Windows 电脑、数据线、可用网络和自己的订阅。首次安装还会由设备联网下载官方 Mihomo 核心；源码参考包仅供查阅，不是运行必需文件。

## 1. 先确认自己从哪一步开始

| 当前状态 | 从哪里继续 |
| --- | --- |
| B23，Magisk 和 UFI 高级功能都可用，尚未安装本项目 | 核对[安装条件](install.md#安装前确认)，然后从第 4 步安装 |
| B23，已有 Magisk root，但还没有 UFI | 从第 3 步准备 UFI |
| B23，尚未 root | 先看第 2 步及 B23 root 记录，准备匹配的工具、本机备份和恢复办法 |
| B24、U30 Pro AI、MU5353 或其他固件/型号 | 当前安装包不接收；不要按 B23 的安装步骤继续 |
| 已有旧版、alpha 或其他部署 | 先看[安装与恢复](install.md)，v1.3.0 不提供覆盖升级 |

在原厂屏幕或管理页的设备信息中记录完整型号、硬件版本和固件版本。`MU5358` 只是型号，同型号不代表启动分区、固件和 root 方法相同。

### 已有 USB ADB 时，读取完整属性

电脑可使用 [UFI 完整存档](https://github.com/zj-yzj/u30pro-vpn-controller/releases/download/ufi-tools-archive-20261019/U30Pro-UFI-Tools-Archive-20261019.zip)中已解压的 `adb-fastboot` 目录，也可下载 [Android 官方 Platform-Tools](https://developer.android.com/tools/releases/platform-tools)。保持 `adb.exe` 与配套 DLL 在同一目录，用可传数据的数据线连接已经开机的 U30 Pro。在该工具目录打开终端；Windows PowerShell 可把下面的 `adb` 写成 `./adb.exe`，或者先将该目录加入 PATH。

```sh
adb devices -l
```

先确认列表中的目标是 U30 Pro，状态为 `device`。将下方的 `SERIAL` 换成目标的实际序列号，再进入 shell；有且仅有一台 USB Android 设备时，也可用 `adb -d shell`。`unauthorized` 需要在设备上完成 ADB 授权；没有设备时先检查数据线、驱动和已有的 USB 调试设置。

```sh
adb -s SERIAL shell
```

进入 Android shell 后逐行读取：

```sh
getprop ro.product.model
getprop ro.product.device
getprop ro.build.display.id
getprop ro.build.version.sdk
getprop ro.product.cpu.abi
getprop ro.boot.slot_suffix
getenforce
pm list packages com.minikano.f50_sms
```

当前安装包对应的结果依次为 `MU5358`、`MU5358`、`U30ProV1.0.0B23`、`35`、`arm64-v8a`；SELinux 已为 `Permissive`。启动槽供 root 方法核对，不是让你手动切槽；UFI 完整版安装后应能看到 `package:com.minikano.f50_sms`。

没有 ADB 不等于没有 root。可以先用已有的可信设备管理入口查看信息；这些命令也不会替你开启 ADB 或获取 root。信息不匹配时停止，不修改系统属性或 SELinux 来通过安装检查。

## 2. 原厂设备先完成 root

维护者给实测 B23 刷过 root，使用的套件和步骤已单独整理为 **[B23 root：工具、步骤与核验](root-b23.md)**。其中包括 Windows 下载模式驱动、直接插电流程、配套 Magisk `28103`、修复与重启核验。

第一次刷写前，先完成[关键分区双读备份与有条件恢复准备](backup-b23.md)。支持包需提前下载，备份通过下载模式进行，不要求先取得 root 或开启 ADB；原一键 root 脚本不会替你备份。该流程提供本机关键分区的有限恢复准备，不是完整原厂固件或救砖保证。

需要同一套件时，可下载[第三方 B23 工具包存档 ZIP](https://github.com/zj-yzj/u30pro-vpn-controller/releases/download/b23-rootkit-archive-20261019/U30Pro-B23-RootKit-Archive-20261019.zip)（[独立存档 Release](https://github.com/zj-yzj/u30pro-vpn-controller/releases/tag/b23-rootkit-archive-20261019)）。这是单独的历史镜像，不属于 VPN 控制器 v1.3.0 安装包；已有可用 root 时跳过，不为安装 VPN 重刷。

先读 root 记录再操作：这份第三方包内置了每次开机尝试开启 TCP `5555` 无线 ADB、放宽认证和禁用更新组件的脚本，自动脚本没有备份步骤。存档保留工具原始字节，未重新刷机验证；除已核对官方文件身份的配套 Magisk 外，其他组件的原作者与再分发许可仍未确认，工具包不适用本项目的 MIT 许可。第一次刷写仍需具备自己的原厂备份和匹配的恢复方法，存档不提供完整恢复保证。

[公开 B24 root 资料](https://github.com/SAMYUBA/u30pro-b24-root-kit)对应其他固件，不能替代 B23 套件。仅安装 Magisk APK 也没有完成 root；通用流程可参考 [Magisk 官方说明](https://topjohnwu.github.io/Magisk/install.html)，但不要把通用 fastboot 命令直接用于 U30 Pro。

### 已经 root：确认实际可用

在现有的可信 root 终端执行以下两条只读命令。通过 ADB 时，先在 `adb -d shell` 中执行 `su`，按设备现有的 Magisk 管理方式授予 shell root，再执行：

```sh
id
magisk -V
```

`id` 应包含 `uid=0(root)`，`magisk -V` 应返回至少 `28100` 的数值版本码。Magisk 应显示已安装，而不是只有应用版本或“未安装”。已有 root 也不要为本项目随意更新启动镜像；先确认现有版本是否已满足要求。

`su` 或 `magisk` 提示找不到命令，不一定表示没有 root。某些环境的既有入口位于 `/debug_ramdisk/su` 和 `/debug_ramdisk/magisk`；只在确认它们属于本机已安装 Magisk 时使用对应绝对路径检查。APK 安装入口本身还要求普通 `su` 可调用，绝对路径能执行不代表 APK 入口已可用。

如果 shell 授权被拒绝，先检查该调用方的 root 授权；UFI root 终端可用不代表 ADB shell 自动获准。重启后再检查一次，确认 root 和原厂 Wi-Fi 联网仍正常，再继续准备 UFI。读完可用 `exit` 退出 root shell 和 ADB shell。

## 3. 安装 UFI 并确认高级功能

订阅链接第一次录入需要设备上的 **UFI-TOOLS 完整版**，包名为 `com.minikano.f50_sms`。装在手机上的 PE 版不能替代它。

直接下载[推荐的 UFI 4.1.2 APK](https://github.com/zj-yzj/u30pro-vpn-controller/releases/download/ufi-tools-archive-20261019/UFI-TOOLS_WEB_V4.1.2_20260819_1854.apk)，或[完整安装材料 ZIP](https://github.com/zj-yzj/u30pro-vpn-controller/releases/download/ufi-tools-archive-20261019/U30Pro-UFI-Tools-Archive-20261019.zip)（[独立存档 Release](https://github.com/zj-yzj/u30pro-vpn-controller/releases/tag/ufi-tools-archive-20261019)）。ZIP 的 `UFI-Tools/` 目录保留 Windows 安装器 1.6、4.1.1/4.1.2 APK、原手册和辅助工具等全部原始文件，另外附上说明、清单与校验文件。已有可用 UFI 时跳过安装，不覆盖重装。

UFI-TOOLS 由 **Minikano / kanoqwq** 开发，原项目采用 MIT 许可；存档内的其他辅助程序分别保留原有作者与许可信息，不为整个存档统一授予 MIT。作者与使用说明见 [UFI-TOOLS 仓库](https://github.com/kanoqwq/UFI-TOOLS)和[官方手册](https://github.com/kanoqwq/UFI-TOOLS/blob/http-server-version/User_Doc.md)。UFI 安装器与本项目的 VPN 安装器是两个工具。

| 安装方式 | 使用文件 |
| --- | --- |
| Windows 一键安装 | 完整解压 ZIP，在 `UFI-Tools/` 内打开 `UFI-TOOLS一键安装器1.6.exe`，连接已开机的 U30 Pro，按作者安装器提示确认目标并安装 |
| 手动 ADB，固定使用 4.1.2 | 下载单独的 4.1.2 APK，使用存档中的 `adb-fastboot` 或官方 Platform-Tools，按下方命令安装 |

1. 保持 U30 Pro 原厂联网正常，用上面一种方式把 UFI 装到 **U30 Pro 本机**。
2. 按 UFI 说明完成所需权限和服务设置，设备网关填写实际地址。本项目适配的地址是 `192.168.0.1`。
3. 用手机或电脑连接 U30 Pro 网络，打开 `http://192.168.0.1:2333/`。使用自己的 UFI 口令和页面要求的原厂管理密码登录。
4. 如果尚未启用，在“高级功能”中选择“添加高级功能”，等待操作完成，并按 UFI 说明保持其依赖的文件共享功能开启。
5. 确认插件使用的 **UFI 认证 RootShell** 能执行 `id`，结果包含 `uid=0(root)`；重启后确认 UFI 服务和这个 root 通道仍可使用。

<details>
<summary>通过 USB ADB 安装并打开 UFI</summary>

上方可直接下载的 `UFI-TOOLS_WEB_V4.1.2_20260819_1854.apk` 已静态核对为包名 `com.minikano.f50_sms`、版本 `4.1.2`、版本码 `20260819`，大小 `22936046` 字节。以下命令只针对这份 APK；Windows 一键安装路线以作者安装器的提示为准。APK 的 SHA-256 为 `4831924e9a7cea2184dc2b515d816c9182e4e6fee44111ea62b7d8e9acf28347`，用于核对存档文件身份，不代表本项目已核验原作者签名。

先确认目标设备，将 `SERIAL` 替换为其序列号。已有可用 UFI 时跳过安装，不覆盖重装：

如果终端当前位于 ADB 工具目录，先把下载的 APK 复制到该目录，或把下面引号内的文件名替换为 APK 的完整路径。

```sh
adb -s SERIAL install "UFI-TOOLS_WEB_V4.1.2_20260819_1854.apk"
adb -s SERIAL shell am start -n com.minikano.f50_sms/.MainActivity
```

这会打开 UFI 自己的界面。按作者说明设置网关并启动服务；原厂圆屏操作不便时，可以使用已有的可信 ADB 投屏工具。不要尝试从外部直接启动 UFI 未导出的后台服务。

按本机需要完成通知、使用情况访问和电池优化设置。对已核对的 4.1.2/API 35 环境，对应的 ADB 操作是：

```sh
adb -s SERIAL shell pm grant com.minikano.f50_sms android.permission.POST_NOTIFICATIONS
adb -s SERIAL shell appops set com.minikano.f50_sms GET_USAGE_STATS allow
adb -s SERIAL shell dumpsys deviceidle whitelist +com.minikano.f50_sms
```

这三项分别是通知权限、使用情况 app-op 和电池优化白名单；不要用 `pm grant` 代替后两项，也不需要为 VPN 功能授予短信权限。以上命令依据包声明和作者部署说明整理，本次没有在新设备上重跑安装。

</details>

<details>
<summary>检查插件实际使用的 RootShell 通道</summary>

电脑浏览器打开并登录 UFI 页面，完成“添加高级功能”后，在**该页面**按 F12 打开 Console，执行这条只读检查：

```javascript
await runShellWithRoot('id', 10000)
```

结果应同时包含 `success: true` 和 `content` 中实际的 `uid=0(root)`。它复用当前 UFI 登录，不需要读取或粘贴 token。若函数不存在，先核对页面和版本；若返回 socket 或权限错误，回到高级功能与文件共享状态排查。`success: true` 本身不等于 UID 0。

函数与认证请求见 [UFI 上游 requests.js](https://github.com/kanoqwq/UFI-TOOLS/blob/http-server-version/app/frontEnd/public/script/requests.js#L559)。

</details>

能打开 UFI 页面，只说明网页服务可达；独立的 TTYD 网页终端可用也不能代替第 5 项。可按 [UFI 的 RootShell 接口说明](https://github.com/kanoqwq/UFI-TOOLS/blob/http-server-version/User_Doc.md#123-功能内容)检查插件通道。UFI 高级功能与 Magisk 分别检查：当前 VPN 安装包需要两者都可用。无法取得 UFI root 时先处理这一层，再安装 VPN；UFI 作者也说明 Pro 系列需先 root，见 [issue #144](https://github.com/kanoqwq/UFI-TOOLS/issues/144)。

VPN 控制器安装包提供本项目的插件，不修改 UFI 的作者信息，也不内嵌 UFI 安装器或 root 工具。UFI 安装材料和 B23 工具包分别作为独立的第三方历史存档提供。

## 4. 安装 v1.3.0

先核对[安装条件](install.md#安装前确认)，再打开 [v1.3.0 Release](https://github.com/zj-yzj/u30pro-vpn-controller/releases/tag/v1.3.0)。下载 `U30ProVPN.txt`，再选择一种安装入口：

| 入口 | 操作 |
| --- | --- |
| Magisk ZIP | 下载 `U30ProVPN-1.3.0-Magisk.zip`，传到 U30 Pro，在 Magisk“模块 → 从本地安装”中选中 ZIP |
| Android APK | 将 `U30ProVPN-1.3.0-Installer.apk` 安装到 U30 Pro，打开“U30 Pro VPN 安装”，点“安装模块”并授权 root |

手动选 ZIP 前，确认 Magisk 没有同 ID 的已安装或待重启模块：`/data/adb/modules/codex_u30pro_vpn` 和 `/data/adb/modules_update/codex_u30pro_vpn` 都应不存在；不确定的目录先保留核查。Magisk 可能在本项目检查前替换同 ID 的暂存内容，详见[安装前确认](install.md#安装前确认)。两个入口不要同时运行。

首次准备会由 U30 Pro 联网下载约 **21.9 MB** 的官方 Mihomo 核心，设备必须能访问官方 GitHub 资源。等待“模块已暂存”后重启，打开安装器的“打开控制”，或 Magisk 模块卡片的“操作 / Action”。

首次应看到 **VPN 关闭、节点为空、尚未保存订阅**。这表示安装完成后等待配置，不是安装失败。两种入口的逐步说明、下载校验和失败处理见[安装与恢复](install.md)。

## 5. 导入订阅并使用

1. 在 UFI 网页打开“插件功能 → 添加插件(.txt文件) → 提交”，导入刚下载的 `U30ProVPN.txt`。
2. 打开 VPN 插件的“订阅”页，填写自己可用的 HTTPS 订阅链接，保存/更新，等待节点出现。
3. 在圆屏或网页中选中节点，主动开启 VPN。保存订阅不会自动打开 VPN。
4. 需要看延迟时，在圆屏节点页底部点“刷新”测整批，或按住普通节点左滑后点“测速”测这一项；没有定时全节点测速。

以后可以在本机圆屏开关、切换节点、更新已保存的订阅。更换链接仍在 UFI 网页完成。“更多 → 快捷入口”可开启悬浮按钮，授权后退出控制页也能重新进入。

## 6. 用连接热点的设备验证一次

手机连接 U30 Pro 的 Wi-Fi，暂时关闭手机自己的 VPN 和移动数据。先确认原厂后台可打开，再开启 U30 Pro 的 VPN，访问一个需要通过所选节点连接的网站，以及平时常用的网站。两者都正常，才算热点客户端实际可用。

随后关闭 VPN，确认原厂联网恢复；再开启、选好节点并重启 U30 Pro，等待 VPN 显示运行后重复一次客户端检查。后续启动会读取保存的开关状态，等待启动期间不要把客户端的直连结果误认为 VPN 已生效。

“没有延迟记录”不是节点不可用，先手动测一次；退出控制页也不会关闭 VPN。日常操作见[使用说明](usage.md)，安装失败、卸载和数据保留见[安装与恢复](install.md)。
