# Gateway reference source

[简体中文](README.md) · [Project home](../README.en.md)

`reference/` contains the Android/Magisk gateway logic paired with this
controller: a fixed-path control entry, a Mihomo supervisor with failure
protection, and a native bridge/manager startup template.

For installation, use the release ZIP or APK and follow
[Installation and recovery](../docs/install.en.md). For development or porting,
start with [reference/README.en.md](reference/README.en.md). The scripts require
existing root, the specified device network layout, reserved rule and route-table
names, private runtime directories and a separately obtained Mihomo executable.
Do not copy and execute the reference directory directly on a hotspot.
