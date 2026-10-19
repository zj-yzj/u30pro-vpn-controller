# 固定布局 Android 网关参考

[English](README.en.md) · [项目首页](../../README.md)

这里提供固定布局 Android 网关的 shell 源码，包括控制入口、Mihomo supervisor 和 Magisk 启动模板。部署时需准备 root、Mihomo 与私有配置，并填写核心和配置的校验摘要。

## 文件选择

| 文件 | 用途 |
| --- | --- |
| `control.sh` | 固定路径控制入口，提供 start、stop、status 和 disable，包含 supervisor 身份核验、显式停止标记和保守清理条件。 |
| `vpn-service.sh.in` | 支持 file provider 的 supervisor，包含进程与启动身份核验、摘要校验、专用路由/防火墙规则、SFP 控制、故障阻断和显式停止后的恢复。核心及配置摘要使用下述参数。 |
| `service.sh.in` | Magisk 启动模板，检查 root、模块 disable 状态及可选控制桥的所有权和权限，有限等待网络管理能力就绪后调用 `webctl boot`。 |

首次安装使用[安装与恢复](../../docs/install.md)中的 ZIP/APK。发行模块入口位于 `release/module/`，安装恢复逻辑位于 `web-manager/internal/installer/`。本目录供开发和移植参考，不要直接复制到设备执行；已有私有部署的升级需要单独设计迁移过程。

## 摘要参数

`vpn-service.sh.in` 必须渲染后才能作为运行脚本使用：

| 参数 | 取值规则 |
| --- | --- |
| `@@MIHOMO_SHA256@@` | 部署方单独取得、核验并准备安装的 Mihomo 可执行文件的 SHA-256，恰好 64 个小写十六进制字符。 |
| `@@CONFIG_SHA256@@` | 最终部署的固定 `config.yaml` 原始字节的 SHA-256，同样为 64 个小写十六进制字符。 |

每个参数在模板中恰好出现一次。用本地工具计算摘要，按字面替换，拒绝格式不符的值和任何残留的 `@@...@@`。然后检查输出语法、逐项审查差异，并通过带私有备份和断电恢复能力的部署事务安装。不要将变量改成“始终通过”的判断，也不要删除摘要校验。配置或核心发生变化时，应重新验证功能并在同一个可恢复事务中更新对应摘要。

`service.sh.in` 是集成模板，安装事务恢复由发行模块的 `release/module/service.sh` 调用 `u30setup boot` 完成，随后再按保存状态启动。其他移植也需要安排这一恢复阶段。

## 已保留的设备假设

- Android root shell 和 Magisk 模块启动；运行目录固定为 `/data/adb/u30pro-vpn`，模块目录固定为 `/data/adb/modules/codex_u30pro_vpn`。
- LAN 为 `br0`、`192.168.0.0/24`，网关为 `192.168.0.1`；系统已开启 IPv4 forwarding。
- SFP 开关位于 `/proc/net/sfp/enable`，正常退出时预期恢复值为 `1`。其他硬件、固件、加速实现或原始开关值需要重新审查。
- TUN 名为 `U30VPN`，路由表为 `20131`，规则优先级为 `9490`、`9491`，防火墙使用源码中命名的专用链。初始部署必须确认这些对象没有被其他功能占用，并记录归属。
- 系统具备脚本所用的 `ip`、`iptables`、`ip6tables`、`stat`、`sha256sum`、`nohup`、`setsid`、`timeout` 等命令及相应行为。
- 已建立 root 私有的 runtime、cache、log 和 web 管理目录、配置、file provider、状态和密钥。新安装器首先创建空节点/订阅与关闭状态；只有用户录入自己的订阅并主动开启后才使用 supervisor。公开模板不包含可用节点。
- 可选本机控制桥位于 `native-ui/webbridge`，其父目录与可执行文件须符合源码的 root 所有权和权限要求；Android 应用签名与信任记录由部署方创建。

脚本按以上固定值工作。移植到其他固件、随身 Wi-Fi 或 OpenWrt 时，需要逐项检查并适配。

## 运行保护与测速

supervisor 的 `healthy()` 只检查进程身份、TUN ifindex、SFP 与专用路由；它不做节点延迟测试或后台连通性测速。故障时保留阻断和受控重启，只有明确停止入口才授权清理本功能拥有的网络对象。它不会对未知同启动周期锁或不明对象进行全局清理。

配合 `../../web-manager/device/config.yaml` 使用时，provider 没有自动 health-check，`Auto` 是由管理后端根据手动测速结果设置的内层 select 组。不要重新加入周期 url-test 或 provider health-check，从而改变此包的手动测速语义。

测试记录见[测试与设备验证](../../docs/testing.md)。冷启动有网络保护尚未就绪的窗口；启动、延后清理和私有备份保留的具体行为见[安装与恢复](../../docs/install.md)。
