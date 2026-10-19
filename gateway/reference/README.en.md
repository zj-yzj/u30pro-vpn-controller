# Fixed-layout Android gateway reference

[简体中文](README.md) · [Project home](../../README.en.md)

This directory contains shell source for the fixed-layout Android gateway:
a control entry, Mihomo supervisor and Magisk startup template. Integration
requires root, a Mihomo executable, private configuration and the matching
core/configuration digests.

## Files

| File | Purpose |
| --- | --- |
| `control.sh` | Fixed-path start, stop, status and disable entry with supervisor identity checks, explicit stop markers and conservative cleanup. |
| `vpn-service.sh.in` | File-provider supervisor with process/boot identity, digest checks, dedicated routing/firewall objects, SFP control, failure blocking and explicit-stop restoration. Only core/config digests become parameters. |
| `service.sh.in` | Magisk startup template with root/module-disable checks, optional native-bridge ownership/permissions, bounded network readiness waiting and `webctl boot`. |

For first installation, use the ZIP or APK in
[Installation and recovery](../../docs/install.en.md). The release module entry
is in `release/module/`, with installation recovery in
`web-manager/internal/installer/`. This directory is for development and porting;
do not copy and execute it directly on a device. Existing private deployments
need a separately designed migration procedure.

## Required digests

Render `vpn-service.sh.in` before it can be used as a runtime script:

| Parameter | Value |
| --- | --- |
| `@@MIHOMO_SHA256@@` | SHA-256 of the independently obtained and verified executable that will actually be installed; exactly 64 lowercase hexadecimal characters. |
| `@@CONFIG_SHA256@@` | SHA-256 of the final fixed `config.yaml` bytes; the same format. |

Each parameter appears exactly once. Calculate the digests locally, substitute
them literally, reject invalid values and any remaining `@@...@@` marker, check
syntax, review the diff and install through a recoverable transaction with
private backups. Do not replace checks with unconditional success. Core or
fixed-configuration changes require renewed validation and a matching digest
update in the same recoverable transaction.

`service.sh.in` is an integration template. Release modules handle installation
recovery through `release/module/service.sh`, which calls `u30setup boot` before
normal saved-state startup. Ports also need to provide this recovery stage.

## Explicit device assumptions

- Android root shell and Magisk module startup. Runtime root is
  `/data/adb/u30pro-vpn`; module root is `/data/adb/modules/codex_u30pro_vpn`.
- LAN bridge `br0`, subnet `192.168.0.0/24`, gateway `192.168.0.1` and already
  enabled IPv4 forwarding.
- SFP switch `/proc/net/sfp/enable`, with normal restoration expected to be `1`.
  Other hardware, firmware or baseline values require a fresh review.
- TUN `U30VPN`, route table `20131`, rule priorities `9490` and `9491`, and the
  dedicated firewall chains named in source. Establish ownership and absence
  of conflicts before integration.
- Compatible `ip`, `iptables`, `ip6tables`, `stat`, `sha256sum`, `nohup`,
  `setsid`, `timeout` and other commands used in source.
- Root-private runtime, cache, log and web-management directories; configuration,
  provider, state and keys. First installation creates empty nodes/subscription
  and VPN off; the supervisor is used only after user configuration and enable.
- Optional `native-ui/webbridge`, whose executable and parent directory must
  satisfy the source's root ownership/permission checks. Release installation
  provisions trust from the fixed APK digest and actual local UID/path. Other
  integrations must manage their own signer/trust.

The scripts use these fixed values. Review and adapt each assumption when
porting to other firmware, hotspot models or OpenWrt.

## Protection and measurements

The supervisor's `healthy()` checks process identity, TUN ifindex, SFP and
dedicated routes. It does not run node latency checks. Failure handling retains
blocking and controlled restart; only explicit stop permits cleaning up owned
network objects. It does not globally clean up unknown locks or objects.

With [the reference configuration](../../web-manager/device/config.yaml), the
provider has no automatic health check and `Auto` is an inner select group set
by the manager from manual measurement results. Do not introduce periodic
URLTest/provider checks while claiming the existing manual-only semantics.

See [Testing](../../docs/testing.en.md) for test coverage and device-validation
records. Enabled cold boots have a window before network protection is ready;
[Installation and recovery](../../docs/install.en.md) describes startup, deferred
cleanup and retained private backups.
