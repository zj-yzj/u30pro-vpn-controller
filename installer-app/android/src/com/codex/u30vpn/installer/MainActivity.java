package com.codex.u30vpn.installer;

import android.app.Activity;
import android.content.ActivityNotFoundException;
import android.content.Intent;
import android.graphics.Color;
import android.os.Bundle;
import android.util.TypedValue;
import android.view.Gravity;
import android.view.View;
import android.view.Window;
import android.view.WindowManager;
import android.widget.Button;
import android.widget.FrameLayout;
import android.widget.LinearLayout;
import android.widget.ScrollView;
import android.widget.TextView;
import android.widget.Toast;

/** All touch targets stay inside the central 164/240 square of the circular screen. */
public final class MainActivity extends Activity implements InstallController.Observer {
    private InstallController controller;
    private float scale;
    private Button open;
    @Override public void onCreate(Bundle savedState) {
        super.onCreate(savedState);
        requestWindowFeature(Window.FEATURE_NO_TITLE);
        getWindow().setFlags(WindowManager.LayoutParams.FLAG_FULLSCREEN, WindowManager.LayoutParams.FLAG_FULLSCREEN);
        getWindow().getDecorView().setSystemUiVisibility(View.SYSTEM_UI_FLAG_FULLSCREEN | View.SYSTEM_UI_FLAG_HIDE_NAVIGATION | View.SYSTEM_UI_FLAG_IMMERSIVE_STICKY);
        controller = InstallController.get(this);
        changed();
    }
    @Override protected void onResume() { super.onResume(); controller.observe(this); changed(); }
    @Override protected void onPause() { controller.detach(this); super.onPause(); }
    @Override protected void onDestroy() { controller.detach(this); super.onDestroy(); }
    private int px(float value) { return Math.max(1, Math.round(value * scale)); }
    private TextView text(String value, float size) {
        TextView view = new TextView(this);
        view.setText(value); view.setTextSize(TypedValue.COMPLEX_UNIT_PX, size * scale);
        view.setTextColor(Color.rgb(31, 48, 66)); view.setGravity(Gravity.CENTER);
        view.setPadding(0, px(2), 0, px(2));
        return view;
    }
    private Button button(String label, View.OnClickListener click) {
        Button value = new Button(this); value.setText(label);
        value.setTextSize(TypedValue.COMPLEX_UNIT_PX, 11 * scale);
        value.setMinHeight(0); value.setMinimumHeight(0); value.setMinWidth(0); value.setMinimumWidth(0);
        value.setPadding(px(3), 0, px(3), 0); value.setAllCaps(false);
        value.setOnClickListener(click);
        return value;
    }
    @Override public void changed() {
        if (isFinishing() || isDestroyed()) return;
        scale = Math.min(getResources().getDisplayMetrics().widthPixels, getResources().getDisplayMetrics().heightPixels) / 240f;
        FrameLayout frame = new FrameLayout(this); frame.setBackgroundColor(Color.rgb(241, 245, 249));
        ScrollView scroll = new ScrollView(this); scroll.setFillViewport(true); scroll.setClipToPadding(true);
        FrameLayout.LayoutParams safe = new FrameLayout.LayoutParams(px(164), px(164), Gravity.CENTER);
        frame.addView(scroll, safe);
        LinearLayout content = new LinearLayout(this); content.setOrientation(LinearLayout.VERTICAL);
        scroll.addView(content, new ScrollView.LayoutParams(-1, -2));
        content.addView(text("U30 Pro VPN 安装", 13));
        content.addView(text("v" + BuildConfig.MODULE_VERSION, 8));
        content.addView(text(controller.summary(), 10), new LinearLayout.LayoutParams(-1, -2));
        InstallerCore.Phase phase = controller.phase();
        Button install = button("安装模块", view -> controller.install());
        install.setEnabled(phase == InstallerCore.Phase.READY || phase == InstallerCore.Phase.ERROR);
        content.addView(install, new LinearLayout.LayoutParams(-1, px(27)));
        Button reboot = button("现在重启", view -> controller.reboot());
        reboot.setEnabled(phase == InstallerCore.Phase.STAGED);
        content.addView(reboot, new LinearLayout.LayoutParams(-1, px(27)));
        open = button("打开控制", view -> openControl());
        open.setEnabled(getPackageManager().getLaunchIntentForPackage("com.codex.u30vpn") != null);
        content.addView(open, new LinearLayout.LayoutParams(-1, px(27)));
        setContentView(frame);
    }
    private void openControl() {
        Intent launch = getPackageManager().getLaunchIntentForPackage("com.codex.u30vpn");
        if (launch == null) {
            open.setEnabled(false); Toast.makeText(this, "控制应用尚不可用，请先核实模块启动。", Toast.LENGTH_SHORT).show(); return;
        }
        try { startActivity(launch); }
        catch (ActivityNotFoundException | SecurityException unavailable) {
            Toast.makeText(this, "控制应用无法打开，请在 Magisk 核实模块。", Toast.LENGTH_SHORT).show();
        }
    }
}
