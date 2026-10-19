#!/system/bin/sh
# Magisk sources this file after extracting the module.
[ "$BOOTMODE" = true ] || abort "Install from the running Magisk application only."
[ "$ARCH" = arm64 ] || abort "This release requires an arm64 U30 Pro."
[ "$API" = 35 ] || abort "This release requires the supported Android 15 firmware."
ui_print "U30 Pro VPN: checking this device and preparing the module."
ui_print "The official core download requires an Internet connection."
set_perm_recursive "$MODPATH" 0 0 0700 0600
set_perm "$MODPATH/customize.sh" 0 0 0700
set_perm "$MODPATH/u30setup" 0 0 0700
set_perm "$MODPATH/service.sh" 0 0 0700
set_perm "$MODPATH/action.sh" 0 0 0700
set_perm "$MODPATH/uninstall.sh" 0 0 0700
set_perm "$MODPATH/payload/webctl" 0 0 0700
set_perm "$MODPATH/payload/webbridge" 0 0 0700
set_perm "$MODPATH/payload/control.sh" 0 0 0700
set_perm "$MODPATH/payload/vpn-service.sh" 0 0 0700
"$MODPATH/u30setup" prepare --module "$MODPATH" || abort "Preparation failed. Read the fixed error category above; the live gateway was not installed."
ui_print "U30PRO_MODULE_STAGED"
ui_print "Module staged. Reboot to initialize; VPN will initially remain OFF."
