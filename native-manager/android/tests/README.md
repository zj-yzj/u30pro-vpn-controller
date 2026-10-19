# Native controller tests

Uses JDK 17 or later, the configured Android API 35 SDK, and no external runtime dependencies. From `native-manager/`, run:

```powershell
./build.ps1 -TestOnly
```

Checks fresh per-request encryption, RSA2048 / OAEP SHA256 and MGF1 SHA256 with the exact webctl label, AES256-GCM with the same authenticated label and a 12-byte IV, decryption roundtrip, modified tag rejection, and malformed public key rejection.

`CryptoEnvelopeTest --fixture` emits an ephemeral test-only PKCS8 private key and a Java-produced envelope as JSON. It is intended for verification against the existing Go `manager.Decrypt` implementation, using time `time.UnixMilli(1800000000000)`. Expected request action is `select`, ID `n-0123456789abcdef`, nonce `AAAAAAAAAAAAAAAAAAAAAA`. This fixture contains no device keys or actual configuration.

Native UI and service compilation uses the official API35 `android.jar`. Live validation must additionally exercise Android's own JCA provider, real Unix peer credentials, system overlay authorization, 240px layout, lifecycle, lock screen hiding, launcher entry and reboot behavior. Host tests do not prove those Android behaviors.

## Circular physical panel

The physical display is round, although `screencap` returns a square 240x240 framebuffer. The application places all interactive widgets in the centered 164x164 box (x38..202, y38..202). The farthest corner is 115.9655px from the panel center, within the 116px acceptance radius. Title and footer are centered in separate rectangles whose complete bounds also fit that radius.

Run `node ./android/tests/round-screen-geometry.mjs` from `native-manager/` to check these physical bounds and sample 1440 dragged shortcut directions including pixel rounding. This is a geometry check, not a layout or touch interaction test.

For visual acceptance, open `round-screen-preview.html`, select each real device screenshot, and inspect the circular view. Toggle the safe-box / inset-circle guides to see whether controls extend outside the intended bounds. Inspect home, more, subscription, entry settings, and the nodes list before and after scrolling. Also inspect the shortcut at its initial location and after dragging it toward the physical edges. This viewer reads local image files without uploading them. Finally confirm readability and touch positions on the physical screen; a square screenshot alone is insufficient.

## Single-node swipe gesture

`build.ps1` now runs `SwipeGestureTest` alongside the existing crypto test. This is a real pure-Java intent state machine shared with the Android touch adapter; it checks tap versus horizontal reveal/close, vertical and diagonal scrolling intent, canceled events, reversed short drags and reset. There are no Android view mocks or implementation-mirroring layout tests.

Live acceptance should left-swipe both an unselected and the currently selected node, wait through at least one foreground status refresh, then tap the separate 52px test button. Only that stable node ID must appear in the encrypted `probe-node` action; no selection change or batch probe should result. Right-swipe closes the row, left-swiping a second row closes the first, and Auto has no per-node test. Check vertical scrolling and normal tap selection as separate gestures. Single-node progress belongs only to its target row; the existing bottom Refresh button remains the batch action. With VPN stopped, test actions stay disabled and must not start VPN. A finger held on the list defers foreground redraw until the gesture finishes, preserving scroll position and the expanded row.
