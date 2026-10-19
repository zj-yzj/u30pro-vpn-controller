# Testing

[中文](testing.md) · [Project home](../README.en.md) · [Building](building.en.md)

## Device use

The project has been tested on my own U30 Pro and works normally in daily use. This describes that device; other firmware, a fresh-device installation, power-loss recovery and a complete uninstall still need separate checks.

See [Installation](install.en.md) for the supported setup. On a new device, check client connectivity, stock connectivity after switching VPN off, and saved settings after a reboot.

## v1.3.0-alpha.1 release checks

These results were recorded when the installer packages were built. The Release's `release-manifest.json` records their source commit and file digests. Later documentation changes do not change that build provenance.

| Check | Result |
| --- | --- |
| Go normal tests | 233 passing test events |
| Go diagnostic tests | 234 passing test events |
| Go vet and static Linux/arm64 builds | Passed |
| Web logic | 12 checks passed |
| Module and packaging | 10 tests passed |
| Installer | 37 core checks plus subprocess output-limit and timeout tests passed |
| Round-screen controller | Crypto interoperability, gestures and circular geometry passed |
| Release assets | APK signatures, ZIP inventory/modes and embedded ZIP identity passed |

Both Go runs skipped one POSIX file-permission test on Windows: `TestBootstrapRejectsUnsafeProductionFileModes`. Installer simulations also cover interrupted transactions, unknown applications, integrity failures, concurrent requests and deferred cleanup.

All six uploaded assets were downloaded from GitHub again and checked against their digests. Verification covered the module's 20 payload files, executable architecture, configuration pins and required licenses.

Commands are in [Building](building.en.md). Use `SHA256SUMS` from the same Release when checking downloads.
