package com.codex.u30vpn;

import android.app.KeyguardManager;
import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.app.Service;
import android.content.BroadcastReceiver;
import android.content.Context;
import android.content.Intent;
import android.content.IntentFilter;
import android.content.SharedPreferences;
import android.content.pm.ServiceInfo;
import android.graphics.Color;
import android.graphics.PixelFormat;
import android.graphics.drawable.GradientDrawable;
import android.os.Build;
import android.os.Handler;
import android.os.IBinder;
import android.os.Looper;
import android.os.PowerManager;
import android.provider.Settings;
import android.view.Gravity;
import android.view.MotionEvent;
import android.view.View;
import android.view.WindowManager;
import android.widget.TextView;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import org.json.JSONObject;

/** Optional, explicitly authorized launcher shortcut. Never shown over the lock screen. */
public final class EntryService extends Service {
    private static final String PREFS = "device-entry";
    private static final String CHANNEL = "vpn-device-entry";
    private static volatile boolean activityVisible;
    private static volatile EntryService liveService;
    private final Handler handler = new Handler(Looper.getMainLooper());
    private final ExecutorService worker = Executors.newSingleThreadExecutor();
    private final BridgeClient bridge = new BridgeClient();
    private WindowManager windows;
    private TextView bubble;
    private WindowManager.LayoutParams layout;
    private boolean attached;
    private boolean polling;
    private volatile boolean destroyed;
    private int generation;
    private final Runnable poll = this::refresh;
    private final Runnable expireHome = this::hide;

    private static SharedPreferences preferences(Context context) {
        return context.getSharedPreferences(PREFS, Context.MODE_PRIVATE);
    }

    static boolean enabled(Context context) { return preferences(context).getBoolean("enabled", false); }

    static void setEnabled(Context context, boolean value) {
        preferences(context).edit().putBoolean("enabled", value).apply();
        sync(context);
    }

    static void sync(Context context) {
        Intent service = new Intent(context, EntryService.class);
        if (!enabled(context) || !Settings.canDrawOverlays(context)) {
            context.stopService(service);
            return;
        }
        try { context.startForegroundService(service); }
        catch (RuntimeException ignored) { /* Visible app can retry; never bypass platform restrictions. */ }
    }

    static void setActivityVisible(boolean value) {
        activityVisible = value;
        EntryService service = liveService;
        if (service != null) service.handler.post(() -> {
            if (value) service.hide();
            else {
                service.handler.removeCallbacks(service.poll);
                service.handler.post(service.poll);
            }
        });
    }

    private int dp(float value) { return Math.round(value * getResources().getDisplayMetrics().density); }

    private final BroadcastReceiver screen = new BroadcastReceiver() {
        @Override public void onReceive(Context context, Intent intent) {
            generation++;
            handler.removeCallbacks(poll);
            bridge.cancel();
            polling = false;
            hide();
            if (!Intent.ACTION_SCREEN_OFF.equals(intent.getAction())) handler.post(poll);
        }
    };

    @Override public void onCreate() {
        super.onCreate();
        liveService = this;
        windows = (WindowManager) getSystemService(WINDOW_SERVICE);
        NotificationManager notifications = (NotificationManager) getSystemService(NOTIFICATION_SERVICE);
        NotificationChannel channel = new NotificationChannel(CHANNEL, "VPN 本机快捷入口", NotificationManager.IMPORTANCE_LOW);
        channel.setDescription("仅在设备原厂首页显示 VPN 快捷入口");
        channel.setShowBadge(false);
        notifications.createNotificationChannel(channel);
        Intent open = new Intent(this, MainActivity.class).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK | Intent.FLAG_ACTIVITY_SINGLE_TOP);
        PendingIntent pending = PendingIntent.getActivity(this, 0, open, PendingIntent.FLAG_IMMUTABLE | PendingIntent.FLAG_UPDATE_CURRENT);
        Notification notification = new Notification.Builder(this, CHANNEL)
                .setSmallIcon(android.R.drawable.ic_lock_lock).setContentTitle("VPN 本机快捷入口")
                .setContentText("点击打开 VPN 控制，可在 App 中关闭入口")
                .setContentIntent(pending).setOngoing(true).setOnlyAlertOnce(true).build();
        if (Build.VERSION.SDK_INT >= 34) startForeground(31, notification, ServiceInfo.FOREGROUND_SERVICE_TYPE_SPECIAL_USE);
        else startForeground(31, notification);
        IntentFilter filter = new IntentFilter();
        filter.addAction(Intent.ACTION_SCREEN_OFF);
        filter.addAction(Intent.ACTION_SCREEN_ON);
        filter.addAction(Intent.ACTION_USER_PRESENT);
        filter.addAction(Intent.ACTION_CLOSE_SYSTEM_DIALOGS);
        if (Build.VERSION.SDK_INT >= 33) registerReceiver(screen, filter, Context.RECEIVER_NOT_EXPORTED);
        else registerReceiver(screen, filter);
        prepareBubble();
    }

    @Override public int onStartCommand(Intent intent, int flags, int startId) {
        if (!enabled(this) || !Settings.canDrawOverlays(this)) {
            stopSelf();
            return START_NOT_STICKY;
        }
        handler.removeCallbacks(poll);
        handler.post(poll);
        return START_STICKY;
    }

    private boolean mayShow() {
        if (destroyed || activityVisible || !enabled(this) || !Settings.canDrawOverlays(this)) return false;
        PowerManager power = (PowerManager) getSystemService(POWER_SERVICE);
        KeyguardManager keyguard = (KeyguardManager) getSystemService(KEYGUARD_SERVICE);
        return power != null && power.isInteractive() && keyguard != null
                && !keyguard.isKeyguardLocked() && !keyguard.isDeviceLocked();
    }

    private void refresh() {
        if (destroyed || polling) return;
        if (!mayShow()) {
            hide();
            PowerManager power = (PowerManager) getSystemService(POWER_SERVICE);
            if (power != null && power.isInteractive()) handler.postDelayed(poll, 3000);
            return;
        }
        final int epoch = generation;
        polling = true;
        worker.execute(() -> {
            boolean home = false;
            try {
                JSONObject state = bridge.status();
                home = state.optBoolean("ok") && state.optBoolean("deviceHomeVisible", false);
            } catch (Exception ignored) { }
            final boolean show = home;
            handler.post(() -> {
                if (destroyed || epoch != generation) return;
                polling = false;
                if (show && mayShow()) show(); else hide();
                handler.removeCallbacks(poll);
                PowerManager power = (PowerManager) getSystemService(POWER_SERVICE);
                if (power != null && power.isInteractive()) handler.postDelayed(poll, 3000);
            });
        });
    }

    private void prepareBubble() {
        bubble = new TextView(this);
        bubble.setText("VPN");
        bubble.setTextSize(11);
        bubble.setTextColor(Color.WHITE);
        bubble.setGravity(Gravity.CENTER);
        bubble.setContentDescription("打开 VPN 控制，拖动可调整位置");
        GradientDrawable background = new GradientDrawable();
        background.setColor(Color.rgb(22, 100, 215));
        background.setCornerRadius(dp(11));
        bubble.setBackground(background);
        bubble.setElevation(dp(4));
        layout = new WindowManager.LayoutParams(dp(38), dp(38), WindowManager.LayoutParams.TYPE_APPLICATION_OVERLAY,
                WindowManager.LayoutParams.FLAG_NOT_FOCUSABLE | WindowManager.LayoutParams.FLAG_NOT_TOUCH_MODAL
                        | WindowManager.LayoutParams.FLAG_LAYOUT_IN_SCREEN,
                PixelFormat.TRANSLUCENT);
        if (Build.VERSION.SDK_INT >= 30) layout.setFitInsetsTypes(0);
        layout.gravity = Gravity.TOP | Gravity.LEFT;
        SharedPreferences saved = preferences(this);
        int screenWidth = getResources().getDisplayMetrics().widthPixels;
        int screenHeight = getResources().getDisplayMetrics().heightPixels;
        double safeRadius = Math.max(0, Math.min(screenWidth, screenHeight) / 2.0
                - Math.hypot(layout.width / 2.0, layout.height / 2.0) - 4);
        layout.x = saved.getInt("x", (int) Math.round(screenWidth / 2.0 + safeRadius - layout.width / 2.0));
        layout.y = saved.getInt("y", (screenHeight - layout.height) / 2);
        clamp();
        bubble.setOnTouchListener(new View.OnTouchListener() {
            private float downX;
            private float downY;
            private int startX;
            private int startY;
            private boolean moved;
            @Override public boolean onTouch(View view, MotionEvent event) {
                switch (event.getActionMasked()) {
                    case MotionEvent.ACTION_DOWN:
                        downX = event.getRawX(); downY = event.getRawY();
                        startX = layout.x; startY = layout.y; moved = false;
                        return true;
                    case MotionEvent.ACTION_MOVE:
                        float dx = event.getRawX() - downX, dy = event.getRawY() - downY;
                        if (Math.abs(dx) + Math.abs(dy) > dp(6)) moved = true;
                        if (moved && attached) {
                            layout.x = startX + Math.round(dx); layout.y = startY + Math.round(dy); clamp();
                            try { windows.updateViewLayout(bubble, layout); } catch (RuntimeException ignored) { hide(); }
                        }
                        return true;
                    case MotionEvent.ACTION_UP:
                        if (moved) preferences(EntryService.this).edit().putInt("x", layout.x).putInt("y", layout.y).apply();
                        else {
                            hide();
                            try { startActivity(new Intent(EntryService.this, MainActivity.class)
                                    .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK | Intent.FLAG_ACTIVITY_SINGLE_TOP)); }
                            catch (RuntimeException ignored) { }
                        }
                        return true;
                    case MotionEvent.ACTION_CANCEL: return true;
                    default: return false;
                }
            }
        });
    }

    private void clamp() {
        int width = getResources().getDisplayMetrics().widthPixels;
        int height = getResources().getDisplayMetrics().heightPixels;
        double centerX = width / 2.0, centerY = height / 2.0;
        double halfWidth = layout.width / 2.0, halfHeight = layout.height / 2.0;
        double radius = Math.max(0, Math.min(width, height) / 2.0 - Math.hypot(halfWidth, halfHeight) - 5);
        double offsetX = layout.x + halfWidth - centerX;
        double offsetY = layout.y + halfHeight - centerY;
        double distance = Math.hypot(offsetX, offsetY);
        if (distance > radius && distance > 0) {
            offsetX *= radius / distance;
            offsetY *= radius / distance;
        }
        layout.x = (int) Math.round(centerX + offsetX - halfWidth);
        layout.y = (int) Math.round(centerY + offsetY - halfHeight);
    }

    private void show() {
        if (!mayShow()) { hide(); return; }
        handler.removeCallbacks(expireHome);
        handler.postDelayed(expireHome, 4000);
        if (attached) return;
        try { windows.addView(bubble, layout); attached = true; }
        catch (RuntimeException ignored) { attached = false; }
    }

    private void hide() {
        handler.removeCallbacks(expireHome);
        if (!attached) return;
        try { windows.removeViewImmediate(bubble); } catch (RuntimeException ignored) { }
        attached = false;
    }

    @Override public void onDestroy() {
        destroyed = true;
        generation++;
        if (liveService == this) liveService = null;
        handler.removeCallbacksAndMessages(null);
        hide();
        bridge.cancel();
        worker.shutdownNow();
        try { unregisterReceiver(screen); } catch (RuntimeException ignored) { }
        super.onDestroy();
    }

    @Override public IBinder onBind(Intent intent) { return null; }
}
