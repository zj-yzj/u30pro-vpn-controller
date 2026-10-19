# 网关参考源码

[English](README.en.md) · [项目首页](../README.md)

`reference/` 保留此控制器所配合的 Android/Magisk 网关逻辑，供审查和后续集成。它包括固定路径控制入口、带故障保护的 Mihomo supervisor，以及原生控制桥和管理后端的启动模板。

这些文件没有组成从零安装器。它们依赖既有 root 环境、设备网络布局、预先保留的规则名称与路由表、私有运行目录和另外取得的 Mihomo。阅读 [reference/README.md](reference/README.md) 中的参数及前置条件后，再为具体部署设计可恢复安装流程。不要直接把参考目录复制到设备并执行。
