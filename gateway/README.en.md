# Gateway reference source

[简体中文](README.md) · [Project home](../README.en.md)

`reference/` contains the Android/Magisk gateway logic paired with this
controller: a fixed-path control entry, a Mihomo supervisor with failure
protection, and a native bridge/manager startup template.

These files are not a from-scratch installer. They require existing root,
the specified device network layout, reserved rule and route-table names,
private runtime directories and a separately obtained Mihomo executable.
Read the parameters and preconditions in
[reference/README.en.md](reference/README.en.md), then design a recoverable
installation process for the actual device. Do not copy and execute the
reference directory directly on a hotspot.
