# 构建与验证

[English](building.en.md) · [返回首页](../README.md)

所有命令从仓库根目录开始；需要 Node.js/npm、Go 1.27 或兼容版本、JDK 17。Android 构建使用 Platform 35 与 Build Tools 35，工具链由使用者自行安装。

## 网页插件与离线预览

```sh
npm ci
npm test
npm run build
npm run test:preview
```

具体输出目录与预览命令见 `web-manager/ui/README.md`。预览使用模拟数据，不读取设备订阅，也不连接真实网关。node-forge 版本由 lockfile 固定，构建同时核对其发行文件摘要并保留 BSD 许可头。

## Go 后端

确保 `javac` 和 `java` 位于 PATH，以运行 Java/Go 加密互操作测试；缺少 JDK 时该项可能跳过。

```sh
cd web-manager
go test -count=1 ./...
go vet ./...
```

设备程序使用 Linux/arm64，无需 CGO。以下是 PowerShell 示例：

```powershell
$env:CGO_ENABLED = '0'
$env:GOOS = 'linux'
$env:GOARCH = 'arm64'
go build -trimpath -ldflags='-s -w' -o build/webctl ./cmd/webctl
go build -trimpath -ldflags='-s -w' -o build/webbridge ./cmd/nativebridge
```

这些环境变量仅供交叉编译；运行主机测试前应清除 `GOOS`、`GOARCH` 的交叉编译设置。输出目录由 Git 忽略，源码发布包不包含这些二进制。

## Android 应用

构建入口为 `native-manager/build.ps1`，支持显式指定工具链和输出目录，默认不需要私钥。参数和签名选项见 `native-manager/README.md`。构建运行主机加密与手势测试；圆屏几何还可单独验证：

```sh
node native-manager/android/tests/round-screen-geometry.mjs
```

## 发布前检查

```sh
node scripts/check-publication.mjs
node scripts/check-doc-links.mjs
```

该检查只输出相对路径、行号和问题类别，不打印匹配到的敏感内容。它检查发布目录结构、敏感文件类型、个人路径、私钥文本和非示例订阅凭据等常见问题；它不能替代人工代码审查。构建测试完成后，发布目录中不应包含生成物或依赖缓存。
