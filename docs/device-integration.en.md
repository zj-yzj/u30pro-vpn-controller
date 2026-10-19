# Device integration boundary

[简体中文](device-integration.md) · [Project home](../README.en.md)

This repository contains a buildable management layer and gateway reference
source. It does not bundle firmware, root acquisition tools or a universal
installer. A successful host build does not make the output ready to deploy
on every U30 Pro firmware.

## Required environment

- A stock Android device with existing root access and a reviewed startup and
  recovery mechanism.
- A separately obtained Android arm64 Mihomo executable whose provenance and
  digest can be verified.
- The fixed `/data/adb/u30pro-vpn` runtime root, owned by root and protected
  against access by other users.
- Network baselines and restoration data matching that actual device, plus
  private provider, state and key material.
- An existing UFI installation with its authenticated root API for the web
  panel. UFI's authentication implementation is not bundled here.

`web-manager/device/config.yaml` is a credential-free reference configuration.
It points at a private runtime provider; that file and usable nodes are not
included. Both `Auto` and `VPN` are select groups, with no scheduled tests.

## Native application trust

The default package is `com.codex.u30vpn`. Provision the root-side trust record
from the installed application's actual schema, package name, UID, APK path
and SHA-256. Do not reuse another device's UID or a previous APK's digest.
The runtime trust record is not distributed in this repository.

The Android build defaults to an unsigned APK. Keep your signing identity
outside the repository. Same-package Android updates require the original
signer, while the helper's exact APK check additionally requires a deliberate
trust update. The original development device's signing material and automatic
trust-update transactions are not published.

## Gateway adaptation

Read the fixed assumptions in [gateway/reference/README.md](../gateway/reference/README.md):
they include `br0`, the LAN subnet, SFP control, TUN name, route table, rule
priorities and protected directories. The two digest parameters in the supervisor
template must be rendered from the actual reviewed core and final configuration.
Installation transaction recovery must be designed by the integrator before
using the boot template. No device-discovery or universal installation logic is implied.

## Verification scope

Automated tests use mocked controllers and synthetic nodes. They do not need
an actual subscription or perform node tests on the hotspot. Coverage includes
envelope cryptography, input limits, transaction recovery, node operations and
screen geometry.

Deployment still requires device-specific verification of forwarding, DNS,
stock-network restoration when disabled, reboot startup and failure recovery.
Host tests do not establish those results on a new firmware. Initial integration
should be performed by a maintainer who understands the device's recovery path.
