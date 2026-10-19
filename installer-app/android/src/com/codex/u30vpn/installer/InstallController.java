package com.codex.u30vpn.installer;

import android.content.Context;
import android.content.SharedPreferences;
import android.os.Handler;
import android.os.Looper;
import android.provider.Settings;
import java.io.File;
import java.io.IOException;
import java.io.InputStream;
import java.lang.ref.WeakReference;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

/** One install per app process. UI destruction never cancels a root transaction. */
final class InstallController {
    interface Observer { void changed(); }
    private static InstallController instance;
    private final Context context;
    private final SharedPreferences prefs;
    private final Handler main = new Handler(Looper.getMainLooper());
    private final ExecutorService worker = Executors.newSingleThreadExecutor();
    private final InstallerCore.Session session = new InstallerCore.Session();
    private WeakReference<Observer> observer = new WeakReference<>(null);
    private volatile String summary = "需要已有 Magisk。首装联网下载核心。";

    static synchronized InstallController get(Context context) {
        if (instance == null) instance = new InstallController(context.getApplicationContext());
        return instance;
    }
    private InstallController(Context context) {
        this.context = context;
        prefs = context.getSharedPreferences("install-outcome", Context.MODE_PRIVATE);
        if (prefs.getBoolean("pending", false)) {
            session.finish(InstallerCore.Phase.UNKNOWN);
            summary = "上次安装结果未知。请在 Magisk 核实模块，勿重复安装。";
        } else if (prefs.getBoolean("staged", false)) {
            int before = prefs.getInt("boot", -1), now = bootCount();
            boolean rebooted = before >= 0 && now >= 0 && before != now;
            session.finish(rebooted ? InstallerCore.Phase.REVIEW : InstallerCore.Phase.STAGED);
            summary = rebooted ? "设备已重启。请核实 Magisk 模块并打开控制。" : "已暂存，请重启。此时不代表控制服务已就绪。";
        }
    }
    private int bootCount() {
        try { return Settings.Global.getInt(context.getContentResolver(), Settings.Global.BOOT_COUNT, -1); }
        catch (RuntimeException unavailable) { return -1; }
    }
    void observe(Observer value) { observer = new WeakReference<>(value); }
    void detach(Observer value) { if (observer.get() == value) observer.clear(); }
    InstallerCore.Phase phase() { return session.phase(); }
    String summary() { return summary; }
    private void update(InstallerCore.Phase phase, String summary) {
        this.summary = summary; session.finish(phase);
        main.post(() -> { Observer value = observer.get(); if (value != null) value.changed(); });
    }
    private ProcessRunner.Result root(String fixedCommand, long timeout) throws IOException {
        return ProcessRunner.run(new String[] {"su", "-c", fixedCommand}, timeout);
    }
    void install() {
        if (!session.beginInstall()) return;
        update(InstallerCore.Phase.EXTRACTING, "正在校验内置安装包…");
        worker.execute(this::installInBackground);
    }
    private void installInBackground() {
        boolean invokedInstall = false;
        try {
            File zip;
            try (InputStream asset = context.getAssets().open("u30pro-module.zip")) {
                zip = InstallerCore.verifyAndCopy(asset, context.getFilesDir(), BuildConfig.MODULE_ZIP_SHA256, BuildConfig.MODULE_ZIP_BYTES);
            } catch (IOException invalid) {
                update(InstallerCore.Phase.ERROR, "内置安装包校验失败。请重新取得官方安装器。"); return;
            }
            update(InstallerCore.Phase.AUTHORIZING, "请在 Magisk 提示中授权 root…");
            ProcessRunner.Result uid = root("id -u", 60000);
            if (!InstallerCore.rootGranted(uid.exit, uid.stdout, uid.uncertain, uid.overflow)) {
                update(InstallerCore.Phase.ERROR, "未取得 root 授权。请检查 Magisk 授权设置。"); return;
            }
            ProcessRunner.Result version = root("magisk -V", 30000);
            if (version.uncertain || version.overflow || version.exit != 0) {
                update(InstallerCore.Phase.ERROR, "Magisk 不可用或安装不完整。"); return;
            }
            if (!InstallerCore.supportedVersion(version.stdout)) {
                update(InstallerCore.Phase.ERROR, "需要兼容的 Magisk 28.1 或更高版本。"); return;
            }
            ProcessRunner.Result help = root("magisk -h 2>&1", 30000);
            if (help.uncertain || help.overflow || !InstallerCore.supportsInstall(help.stdout)) {
                update(InstallerCore.Phase.ERROR, "此 Magisk 不支持模块安装命令。"); return;
            }
            // Magisk replaces its staging directory before sourcing customize.sh.
            // Refuse an existing module before handing the ZIP to that framework.
            ProcessRunner.Result preflight = root(InstallerCore.preflightCommand(), 30000);
            if (preflight.uncertain || preflight.overflow || preflight.exit != 0) {
                update(InstallerCore.Phase.ERROR, "发现已有或待安装的同名模块，或无法核实。本版仅支持全新安装。"); return;
            }
            // Persist before root mutation. Process death after this point is an unknown result.
            if (!prefs.edit().putBoolean("pending", true).putBoolean("staged", false).commit()) {
                update(InstallerCore.Phase.ERROR, "无法保存安装状态，已停止。"); return;
            }
            invokedInstall = true;
            update(InstallerCore.Phase.INSTALLING, "正在暂存模块并下载核心。最多等待 10 分钟。");
            ProcessRunner.Result result = root(InstallerCore.installCommand(zip.getAbsolutePath()), 600000);
            InstallerCore.Phase outcome = InstallerCore.classifyInstall(result.exit, result.stdout, result.uncertain, result.overflow);
            if (outcome == InstallerCore.Phase.STAGED) {
                boolean saved = prefs.edit().putBoolean("pending", false).putBoolean("staged", true).putInt("boot", bootCount()).commit();
                update(saved ? outcome : InstallerCore.Phase.UNKNOWN, saved ? "已暂存，请重启。此时不代表控制服务已就绪。" : "模块已返回暂存标记，但状态保存失败。请在 Magisk 核实。");
            } else if (outcome == InstallerCore.Phase.UNKNOWN) {
                update(outcome, "安装结果未知，可能仍在执行。请在 Magisk 核实；不能确认已回退。");
            } else {
                if (prefs.edit().putBoolean("pending", false).commit()) update(outcome, "模块安装未成功。请在 Magisk 核实模块状态。");
                else update(InstallerCore.Phase.UNKNOWN, "安装状态保存失败。请在 Magisk 核实模块。");
            }
        } catch (IOException error) {
            update(invokedInstall ? InstallerCore.Phase.UNKNOWN : InstallerCore.Phase.ERROR,
                    invokedInstall ? "安装结果未知。请在 Magisk 核实模块，勿重复安装。" : "无法启动 root 命令。请确认已有 Magisk 与 su。");
        } catch (RuntimeException error) {
            update(invokedInstall ? InstallerCore.Phase.UNKNOWN : InstallerCore.Phase.ERROR,
                    invokedInstall ? "安装结果未知。请在 Magisk 核实模块。" : "安装准备失败，尚未调用模块安装。");
        }
    }
    void reboot() {
        if (!session.beginReboot()) return;
        update(InstallerCore.Phase.REBOOTING, "已请求重启，请稍候…");
        worker.execute(() -> {
            try {
                ProcessRunner.Result result = root("reboot", 30000);
                if (result.uncertain || result.exit != 0) update(InstallerCore.Phase.STAGED, "重启请求未确认。可使用设备系统菜单重启。");
            } catch (IOException error) {
                update(InstallerCore.Phase.STAGED, "无法发出重启请求。请使用设备系统菜单重启。");
            }
        });
    }
}
