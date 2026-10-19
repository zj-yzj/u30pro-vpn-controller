# Fixed-layout Android gateway reference

[简体中文](README.md) · [Project home](../../README.en.md)

This is reviewable shell source for the controller's gateway environment.
It contains no firmware, root tools, upstream binaries, node configurations,
device snapshots or historical installation transactions. Runtime protection
is retained. Completed per-device migration branches have been excluded, and
deployment-specific runtime digests are required template parameters.

## Files

| File | Purpose |
| --- | --- |
| `control.sh` | Fixed-path start, stop, status and disable entry. Retains supervisor identity checks, explicit stop markers and conservative cleanup. |
| `vpn-service.sh.in` | File-provider supervisor with process/boot identity, digest checks, dedicated routing/firewall objects, SFP control, failure blocking and explicit-stop restoration. Only core/config digests become parameters. |
| `service.sh.in` | Long-term Magisk startup reference. Retains root and module-disable checks, optional native-bridge ownership/permissions, bounded network readiness waiting and `webctl boot`. |

The original device's historical first-install, update, authenticated bootstrap,
diagnostic capture and migration-recovery scripts are excluded. They depended on
private baselines/transactions. The new `v1.3.0-alpha.1` fixed-profile installer
uses `release/module/` and `web-manager/internal/installer/`, without copying those
historical transactions. Downloaders should use [Installation and recovery](../../docs/install.en.md).
This reference directory itself is not an installation package or an old-deployment upgrader.

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

`service.sh.in` remains a template without its own installation recovery.
Release modules use a separate `release/module/service.sh` calling `u30setup boot`
to reconcile installation before normal saved-state startup. Ports must preserve
that recovery boundary rather than treat a template or an old device transaction
path as a complete installer.

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

These assumptions have not been generalized into device discovery. They do not
establish support for every U30 Pro firmware, other hotspot models or OpenWrt.

## Protection and measurements

The supervisor's `healthy()` checks process identity, TUN ifindex, SFP and
dedicated routes. It does not run node latency checks. Failure handling retains
blocking and controlled restart; only explicit stop permits cleaning up owned
network objects. It does not globally clean up unknown locks or objects.

With [the reference configuration](../../web-manager/device/config.yaml), the
provider has no automatic health check and `Auto` is an inner select group set
by the manager from manual measurement results. Do not introduce periodic
URLTest/provider checks while claiming the existing manual-only semantics.

Offline shell syntax and host tests do not verify physical link, SFP, DNS, TUN,
failure blocking or power-loss recovery. This alpha has no complete fresh-device
installation/reboot/network/teardown acceptance. Enabled cold boots also have a
startup window without continuous-protection guarantees from power-on. See
Installation/Release notes for scope, deferred cleanup and retained private backups.
