# Contributing / 参与贡献

[中文首页](README.md) · [English home](README.en.md)

欢迎针对 U30 Pro 的问题报告、文档改进和范围明确的代码贡献。提交问题时请说明
硬件/固件版本、受影响组件、复现步骤、预期与实际表现，以及不包含私人信息的错误类别。
不要提交订阅链接、节点凭据、签名材料、设备私钥或未经处理的运行日志。

Welcome U30 Pro issue reports, documentation improvements and focused code
contributions. Include the hardware/firmware version, affected component,
reproduction steps, expected and observed behavior, and sanitized error
categories. Never post subscription URLs, node credentials, signing identities,
device private keys or unreviewed runtime logs.

## Changes / 修改

- 保留手动测速语义；普通状态读取不应发起外部节点测试。
- 保留 root 私有目录、请求校验、客户端身份、APK 摘要和恢复边界。
- 不将设备专属的 UID、摘要、私有配置或历史事务复制进通用源码。
- 更改界面时检查物理圆屏边界；截图使用模拟数据，并注明示意或实机来源。
- 修改构建入口时使用可配置工具链路径，并让输出留在忽略目录中。

Preserve manual-only node measurement semantics, private runtime permissions,
request validation, client identity, APK pinning and recovery boundaries.
Avoid copying per-device identities or historical transactions into generic
source. Check circular bounds when changing UI. Use synthetic screenshot data,
identify illustrations accurately, and keep build paths configurable.

## Validation / 验证

Follow [构建说明](docs/building.md) / [building instructions](docs/building.en.md).
Run the relevant existing tests, publication scan and local-link checker.
Distinguish automated host verification from physical-device results in your
change description. Do not claim a new firmware is supported solely because
the code compiles.

原创代码和文档贡献按本仓库 MIT 许可提供；引入第三方内容时需要保留其准确来源与许可。
Original code and documentation contributions are provided under this
repository's MIT license. Preserve accurate provenance and licenses for any
third-party content you introduce.
