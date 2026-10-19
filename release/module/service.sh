#!/system/bin/sh
MODDIR=${0%/*}
"$MODDIR/u30setup" boot --module "$MODDIR"
