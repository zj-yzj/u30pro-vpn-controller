#!/system/bin/sh
set -u
[ "$(id -u)" = 0 ] || { echo "Root required"; exit 1; }
U30_LIBRARY=1
. /data/adb/u30pro-vpn/vpn-service.sh
unset U30_LIBRARY
case "${1:-status}" in
    start)
        if supervisor_matches; then cat "$STATE"; exit 0; fi
        [ ! -e "$BASE/stop" ] || { objects_absent && [ "$(cat "$SFP")" = 1 ] || { echo CHECK_REQUIRED; exit 1; }; rm -f "$BASE/stop"; }
        [ ! -e "$BASE/hold-restart" ] || { echo "Restart hold is set; preserving it"; exit 1; }
        umask 077
        : >> "$BASE/log/supervisor.log"; chmod 600 "$BASE/log/supervisor.log" || exit 1
        /system/bin/nohup /system/bin/setsid /system/bin/sh "$BASE/vpn-service.sh" >> "$BASE/log/supervisor.log" 2>&1 < /dev/null &
        task_wait=0
        while [ "$task_wait" -lt 10 ]; do
            if supervisor_matches; then cat "$STATE" 2>/dev/null || echo STARTING; exit 0; fi
            sleep 1; task_wait=$((task_wait + 1))
        done
        echo CHECK_REQUIRED; exit 1
        ;;
    stop|disable)
        if [ "$1" = disable ]; then : > "$MODULE/disable"; chmod 600 "$MODULE/disable"; fi
        if ! supervisor_matches; then
            if [ ! -e "$LOCK" ] && objects_absent && [ "$(cat "$SFP")" = 1 ]; then echo STOPPED; exit 0; fi
            echo "CHECK_REQUIRED - no verified supervisor; no global cleanup attempted"; exit 1
        fi
        : > "$STOP"; chmod 600 "$STOP"
        task_wait=0
        while [ "$task_wait" -lt 40 ]; do
            if ! supervisor_matches; then
                [ "$(cat "$STATE")" = STOPPED ] && objects_absent && [ "$(cat "$SFP")" = 1 ] && { echo STOPPED; exit 0; }
                [ "$(cat "$STATE")" != CHECK_REQUIRED ] || { echo CHECK_REQUIRED; exit 1; }
            fi
            sleep 1; task_wait=$((task_wait + 1))
        done
        echo "CHECK_REQUIRED - supervisor stop timeout; guards preserved"; exit 1
        ;;
    status)
        if supervisor_matches; then cat "$STATE"; else echo "SUPERVISOR_NOT_RUNNING"; [ ! -r "$STATE" ] || cat "$STATE"; fi
        ;;
    *) echo "Usage: control.sh start|stop|status|disable"; exit 1 ;;
esac
