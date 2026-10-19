# B23 backup before root and conditional recovery

[简体中文](backup-b23.md) · [B23 rooting record](root-b23.en.md) · [Full preparation guide](setup.en.md)

This page provides two-pass backups of five critical partitions and targeted recovery while the partition table remains intact, for the original **MU5358 / U30ProHW1.0 / `U30ProV1.0.0B23`**. The aim is to preserve complete original partition files from your own device before rooting. It does not cover the entire firmware, NV, user data or every unbootable-device scenario.

The commands were reviewed against the pinned `spd_dump` source commit **`23c5ab211ca1e58e2fb8c70108d2bccfd0110925`**. This version lacks the newer automatic GPT repair path. That does not newly verify download-mode entry or stock recovery on every B23 device. No device was connected, flashed or restored for this documentation.

## 1. Download and prepare before powering off

Download the [B23 backup support package](https://github.com/zj-yzj/u30pro-vpn-controller/releases/download/b23-backup-tools-20261019/U30Pro-B23-Backup-Tools-20261019.zip) ([separate Release](https://github.com/zj-yzj/u30pro-vpn-controller/releases/tag/b23-backup-tools-20261019)) and extract it completely. Its `tools/spd_dump.exe` is the pinned reader used here. `Test-U30ProBackup.ps1` and `expected-sizes.example.csv` are in the package root; the checker's repository source is under [support/b23-backup](../support/b23-backup/Test-U30ProBackup.ps1). Do not mix the reader with the same-named EXE in the historical root kit.

Prepare the [B23 kit's download-mode driver](root-b23.en.md#1-prepare-tools-and-recovery-material), a data-capable USB cable and enough local storage on the computer first. Download mode and Android ADB are separate channels: this page does not require existing root, Magisk, ordinary ADB or wireless ADB. **Do not run a one-click root BAT to enter backup mode**; those scripts continue into image writes.

Keep backups in a new directory outside the support package. These paths are examples. If a directory already exists, choose a new name instead of overwriting an earlier backup. Open PowerShell in the extracted support package's root and run:

```powershell
New-Item -ItemType Directory -Path 'C:\U30Pro-B23-Backup' -ErrorAction Stop
New-Item -ItemType Directory -Path 'C:\U30Pro-B23-Backup\pass1','C:\U30Pro-B23-Backup\pass2','C:\U30Pro-B23-Backup\readback' -ErrorAction Stop
Copy-Item -LiteralPath '.\expected-sizes.example.csv' -Destination 'C:\U30Pro-B23-Backup\expected-sizes.csv'
```

Record the device model, full firmware, hardware version and backup time. Files read after rooting describe the current state; they cannot be relabeled pre-root stock backups.

## 2. Enter only the recorded B23 download mode

Disconnect USB and power off the U30 Pro. From the support package's root, run:

```powershell
.\tools\spd_dump.exe --wait 300 --kickto 2 exec
```

Once it is waiting for a device, connect the powered-off U30 Pro. This uses the B23 original script's entry route with its later root writes removed. Wait for **`FDL2 >`** before continuing. On a timeout, wrong-mode error or unrecognized device, stop and check the connection and driver. Do not add another ROM's FDL, guess addresses, switch slots or proceed to test-point mode.

All subsequent commands without a program prefix belong at the reader's `FDL2 >` prompt, not in PowerShell. Inspect the partitions and sizes first:

```text
p
size_part init_boot_a
size_part init_boot_b
size_part trustos_a
size_part trustos_b
size_part misc
```

All five names must actually exist in the partition table, with normal A/B detection and current-slot information. Record the actual slot; do not assume slot B or use `slot` or `set_active` to manufacture the detection result. Stop here if there is a temporary `w_force` partition, a name/layout mismatch, a zero size or a read error.

`size_part` returns bytes. Record each result and check it against this device's layout. The `p` display and generated XML use MiB values, which cannot stand in for every partition's exact byte length. Replace every placeholder in `expected-sizes.csv` with the corresponding positive integer byte count returned by the device:

```csv
partition,bytes
init_boot_a,REPLACE_FROM_SIZE_PART
init_boot_b,REPLACE_FROM_SIZE_PART
trustos_a,REPLACE_FROM_SIZE_PART
trustos_b,REPLACE_FROM_SIZE_PART
misc,REPLACE_FROM_SIZE_PART
```

The unchanged template cannot be used for verification. Do not copy sizes from another device or use the length of a patched file such as `tos.bin` as the original partition size.

## 3. Read all five partitions twice

Every **`SIZE` below is a placeholder**. Replace it on each line with that partition's actual byte count from the CSV. Keep offset `0` and explicit slot suffixes; `misc` has no slot suffix. Do not replace a checked length with the automatic-size sentinel `0xffffffff`.

For the first pass, enter:

```text
path C:\U30Pro-B23-Backup\pass1
partition_list partition.xml
read_part init_boot_a 0 SIZE init_boot_a.bin
read_part init_boot_b 0 SIZE init_boot_b.bin
read_part trustos_a 0 SIZE trustos_a.bin
read_part trustos_b 0 SIZE trustos_b.bin
read_part misc 0 SIZE misc.bin
```

Confirm that each reported `target` and `read` equals the expected length, then read into a different empty directory:

```text
path C:\U30Pro-B23-Backup\pass2
partition_list partition.xml
read_part init_boot_a 0 SIZE init_boot_a.bin
read_part init_boot_b 0 SIZE init_boot_b.bin
read_part trustos_a 0 SIZE trustos_a.bin
read_part trustos_b 0 SIZE trustos_b.bin
read_part misc 0 SIZE misc.bin
```

`path` sets the output directory. It does not create it or redirect restore input files; without it, output goes to the current working directory. With `path` set, the reader uses each output filename's basename. Separate pass directories prevent the second read from overwriting the first. See the pinned [output-path handling](https://github.com/TomKing062/spreadtrum_flash/blob/23c5ab211ca1e58e2fb8c70108d2bccfd0110925/common.c#L715) and [read command](https://github.com/TomKing062/spreadtrum_flash/blob/23c5ab211ca1e58e2fb8c70108d2bccfd0110925/spd_dump.c#L890).

Startup may already have created `pgpt.bin`, `sprdpart.bin` or a timestamped partition XML in the support package's root. Retain those files and the session output separately. Setting `path` later does not move them into the backup directory. `pgpt.bin` is only the first 32 KiB of `user_partition`; it is not a complete GPT or disk backup, and the XML is not an automatic recovery script.

## 4. Verify offline before continuing with root

After both reads, run the following from the support package's root in another PowerShell window. It sends no commands to the device:

```powershell
powershell.exe -NoProfile -File .\Test-U30ProBackup.ps1 `
  -BackupDirectory 'C:\U30Pro-B23-Backup\pass1' `
  -SecondReadDirectory 'C:\U30Pro-B23-Backup\pass2' `
  -SizesFile 'C:\U30Pro-B23-Backup\expected-sizes.csv'
```

The checker validates all five partition files and their complete lengths in each pass, with matching SHA-256 values between reads. Each pass must also have a `partition.xml` containing the five names and no `w_force`, and the two tables' hashes must match. On success, it outputs JSON with `ok: true` and `scope: critical-partition-file-consistency-only`. Do not continue with root on an error. The reader's exit code or “Read Part Done” alone is insufficient: an interrupted read can leave a short file. See the [read loop](https://github.com/TomKing062/spreadtrum_flash/blob/23c5ab211ca1e58e2fb8c70108d2bccfd0110925/common.c#L867).

This uses Windows' built-in PowerShell; no separate PowerShell 7 installation is needed. If local policy blocks the script, verify and handle this script under that policy rather than permanently weakening the global execution policy.

A passing JSON result establishes only that the two file sets meet the offline checks. It cannot prove that they came from independent physical reads, hardware compatibility, stock contents, a full clone or a successful future restore. A/B detection remains a manual check during the session. Keep another copy of the original backups, size table and check result; do not upload backups containing device state. Once everything is retained, enter `reset` at `FDL2 >`, wait for normal startup, then decide whether to continue with the [B23 rooting record](root-b23.en.md).

## 5. Restore changed partitions only when the conditions hold

Targeted recovery requires pre-root backups from this device that passed the two-read checks, access to the same FDL2 environment, the same partition table/names/sizes, normal A/B detection without a temporary `w_force` name, and knowledge of the actual slot whose `init_boot` and `trustos` were changed. Unknown deployments, damaged tables, different firmware and an unreachable download mode are outside the template below.

After entering the same environment again, repeat `p`, the actual target's `size_part`, and the slot checks. **Restore only partitions actually changed**, rather than writing the other slot for completeness. Replace `PARTITION`, `SIZE` and the input path with the actual name, length and corresponding original file. This is a template and cannot be run unchanged:

```text
skip_confirm 0
w PARTITION C:\U30Pro-B23-Backup\pass1\PARTITION.bin
path C:\U30Pro-B23-Backup\readback
read_part PARTITION 0 SIZE PARTITION.bin
```

`skip_confirm 0` enables write confirmation. In the pinned version, normal `w` writes the named target directly when A/B is recognized and the exact suffixed name exists. Without that condition, ordinary writes can also enter `_bak` handling. See [name resolution](https://github.com/TomKing062/spreadtrum_flash/blob/23c5ab211ca1e58e2fb8c70108d2bccfd0110925/common.c#L1659) and [normal-write dispatch](https://github.com/TomKing062/spreadtrum_flash/blob/23c5ab211ca1e58e2fb8c70108d2bccfd0110925/common.c#L2178). Do not substitute unsuffixed `init_boot` or `trustos` for the actual target.

Read back immediately after each write. In another PowerShell window, compare the original and readback file's complete length and SHA-256:

```powershell
Get-Item -LiteralPath 'C:\U30Pro-B23-Backup\pass1\PARTITION.bin','C:\U30Pro-B23-Backup\readback\PARTITION.bin' | Select-Object Name,Length
Get-FileHash -Algorithm SHA256 -LiteralPath 'C:\U30Pro-B23-Backup\pass1\PARTITION.bin','C:\U30Pro-B23-Backup\readback\PARTITION.bin'
```

Both lengths must equal the recorded `SIZE`, and the hashes must match. Enter `reset` in the reader only after every actual target passes. After boot, check stock Home, Wi-Fi and connectivity, then the expected root state. If a write is rejected or readback differs, retain the result rather than escalating to a force write, repartition or slot switch. This page does not establish complete recovery for those failures.

In this procedure, `misc` and the partition table are retained for evidence and comparison, **not automatically written back**. Do not feed the entire backup directory to `write_parts`, or use `w_force`, `repartition`, NV writes or whole-disk writes. The old `w_force` implementation temporarily rebuilds the table and then restores it; it is not an ordinary single-partition write. See its [implementation](https://github.com/TomKing062/spreadtrum_flash/blob/23c5ab211ca1e58e2fb8c70108d2bccfd0110925/common.c#L1364).

## What this backup does not cover

`r all` skips `blackbox`, `cache` and `userdata`; `all_lite` also skips the other slot. This older version hardcodes `splloader` reads to 256 KiB. NV reads have special name/header handling, while bulk restoration infers slots and can change other objects. This page therefore covers only the limited five-partition backup and targeted recovery above. It is not a complete stock image or one-click unbrick procedure. The [pinned upstream guide](https://github.com/TomKing062/spreadtrum_flash/blob/23c5ab211ca1e58e2fb8c70108d2bccfd0110925/README.md) also lists the bulk-read exclusions.

On 2026-10-19, the [official ZTE U30 Pro support page](https://www.ztedevices.com/cn/supports/u30-pro/) listed no SD-card or software upgrade download. This research found no verifiable official B23 full recovery PAC. The Release therefore does not supply an unidentified "stock recovery" image or substitute B24 images for B23 originals.
