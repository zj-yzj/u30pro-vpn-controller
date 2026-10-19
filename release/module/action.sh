#!/system/bin/sh
/system/bin/am start --user 0 -n com.codex.u30vpn/.MainActivity || {
  echo "Control application unavailable. Reboot and check module status."
  exit 1
}
