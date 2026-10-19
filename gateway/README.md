# 网关参考源码

[English](README.en.md) · [项目首页](../README.md)

`reference/` 包含此控制器使用的 Android/Magisk 网关逻辑：固定路径控制入口、带故障保护的 Mihomo supervisor，以及原生控制桥和管理后端的启动模板。

安装请使用发行 ZIP/APK，步骤见[安装与恢复](../docs/install.md)。开发和移植时，先阅读 [reference/README.md](reference/README.md)：参考脚本依赖既有 root、固定设备网络布局、保留的规则名称与路由表、私有运行目录和另外取得的 Mihomo。不要直接把参考目录复制到设备并执行。
