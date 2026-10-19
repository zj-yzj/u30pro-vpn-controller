# Device integration boundary

[简体中文](device-integration.md) · [Project home](../README.en.md)

`v1.3.0` adds a fixed-profile first installer: a Magisk ZIP and an
Android installer APK embedding the same ZIP. Downloaders should start with
[Installation and recovery](install.en.md). Prepare the required firmware, root
and Magisk environment first. Existing private deployments need their own upgrade procedure.

## Required environment

- device/model `MU5358`, `arm64-v8a`, API 35 and firmware display exactly
  `U30ProV1.0.0B23`.
- Existing working Magisk, version code at least `28100`; SELinux must already
  be Permissive. The installer does not lower enforcement.
- `br0` at `192.168.0.1/24`, IPv4 forwarding enabled, SFP baseline `1` and no
  conflicts with the project's reserved network objects.
- Android arm64 Mihomo acquired online from official upstream during preparation
  and verified by fixed size/hash. No core is embedded; no offline first install.
- The fixed `/data/adb/u30pro-vpn` runtime root, owned by root and protected
  against access by other users.
- Runtime created by the installer: initially empty provider/nodes, VPN off and
  a management key generated on the device.
- An existing UFI installation with its authenticated root API for the web
  panel. UFI's authentication implementation is not bundled here.

`web-manager/device/config.yaml` is a credential-free reference configuration.
It points at a private runtime provider and supplies no usable nodes. Both `Auto`
and `VPN` are select groups, with no scheduled tests. `webctl bootstrap` creates
the empty state. With no nodes, enable returns `not_configured` before changing
desired state or calling gateway control. Legacy `webctl init` expects existing
node inputs and is not the empty-subscription installation path.

## Installation stages and recovery

Preparation validates the fixed profile, file manifest, space and core and stages
only module files. Boot waits for Android/PackageManager/network administration,
journals progress, initializes a root-private staged runtime, then installs the
release control APK and provisions its actual local trust before committing.
First installation starts no core, changes no routes and never enables VPN.

Existing private runtimes, unknown same-package APKs/modules and reserved-object
conflicts prevent first installation. Only a
verifiable interrupted same-release transaction can resume, retaining its key
and committed configuration. Subsequent normal boots use saved desired state;
enabled cold boots have a window before network protection is ready.

Teardown performs a controlled stop, removes only proven-owned objects and
retains a root-private backup. Unavailable PM/network administration can defer
cleanup. See [recovery](install.en.md#uninstalling-and-retained-data).

## Native application trust

The default package is `com.codex.u30vpn`. Provision the root-side trust record
from the installed application's actual schema, package name, UID, APK path
and SHA-256. Do not reuse another device's UID or a previous APK's digest.
The runtime trust record is not distributed in this repository.

The standalone Android build defaults to an unsigned APK. Release builds use a
dedicated identity outside the checkout. Installer trust uses actual UID/path
only after the installed APK matches the fixed release digest. Unknown APKs
and same-package signer conflicts are refused. The installer never uninstalls
an existing app to bypass a signer conflict.

## Gateway adaptation

Read the fixed assumptions in [gateway/reference/README.en.md](../gateway/reference/README.en.md):
they include `br0`, the LAN subnet, SFP control, TUN name, route table, rule
priorities and protected directories. The two digest parameters in the supervisor
template must be rendered from the actual reviewed core and final configuration.
The reference directory remains source/templates rather than the release startup
entry. `release/module/` provides that entry and `web-manager/internal/installer/`
handles first-install recovery. Copying the reference template alone does not run
the installer. Ports still need reviewed baseline, digest and recovery adaptations.

## Verification scope

Automated tests cover envelope cryptography, input limits, transaction recovery,
node operations and screen geometry using mocked controllers and synthetic nodes.
Everyday use has been tested successfully on the author's own U30 Pro. See
[Testing](testing.en.md) for the full test scope and device-validation record.
