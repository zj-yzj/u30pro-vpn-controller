#!/system/bin/sh
MODDIR=${0%/*}
"$MODDIR/u30setup" uninstall --module "$MODDIR" || {
  echo "U30 Pro cleanup deferred. Private runtime is preserved; consult the recovery guide."
  exit 1
}
