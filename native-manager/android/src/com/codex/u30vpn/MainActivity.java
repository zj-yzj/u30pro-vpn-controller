package com.codex.u30vpn;

import android.app.Activity;
import android.content.Intent;
import android.content.res.ColorStateList;
import android.graphics.Color;
import android.graphics.Typeface;
import android.graphics.drawable.GradientDrawable;
import android.net.Uri;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;
import android.provider.Settings;
import android.text.TextUtils;
import android.util.TypedValue;
import android.view.Gravity;
import android.view.MotionEvent;
import android.view.View;
import android.view.Window;
import android.view.WindowManager;
import android.widget.Button;
import android.widget.FrameLayout;
import android.widget.LinearLayout;
import android.widget.ScrollView;
import android.widget.TextView;
import android.widget.Toast;
import java.security.SecureRandom;
import java.util.Base64;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import org.json.JSONObject;

/** Native widgets for the device's 240 x 240 screen; all work runs off the UI thread. */
public final class MainActivity extends Activity {
    private static final int NAVY = Color.rgb(23, 48, 75);
    private static final int BLUE = Color.rgb(22, 100, 215);
    private static final int TEAL = Color.rgb(17, 133, 107);
    private static final int GREY = Color.rgb(95, 111, 128);
    private static final int BACKGROUND = Color.rgb(241, 245, 249);
    private final Handler handler = new Handler(Looper.getMainLooper());
    private final ExecutorService worker = Executors.newSingleThreadExecutor();
    private final BridgeClient bridge = new BridgeClient();
    private final SecureRandom random = new SecureRandom();
    private volatile boolean visible;
    private volatile int generation;
    private boolean inFlight;
    private boolean pending;
    private String pendingAction = "";
    private String pendingNodeId = "";
    private boolean fresh;
    private Model model;
    private String page = "home";
    private String notice = "正在读取设备状态…";
    private LinearLayout root;
    private FrameLayout screen;
    private ScrollView nodeScroll;
    private int nodePosition;
    private String expandedNodeId = "";
    private boolean nodeTouchActive;
    private boolean nodeRenderPending;
    private final Runnable poll = this::refresh;

    @Override public void onCreate(Bundle savedState) {
        super.onCreate(savedState);
        requestWindowFeature(Window.FEATURE_NO_TITLE);
        getWindow().setFlags(WindowManager.LayoutParams.FLAG_FULLSCREEN, WindowManager.LayoutParams.FLAG_FULLSCREEN);
        getWindow().setStatusBarColor(BACKGROUND);
        getWindow().setNavigationBarColor(BACKGROUND);
        if (savedState != null) {
            page = savedState.getString("page", "home");
            expandedNodeId = savedState.getString("expandedNode", "");
        }
        render();
        fullscreen();
    }

    private void fullscreen() {
        getWindow().getDecorView().setSystemUiVisibility(View.SYSTEM_UI_FLAG_FULLSCREEN
                | View.SYSTEM_UI_FLAG_HIDE_NAVIGATION | View.SYSTEM_UI_FLAG_IMMERSIVE_STICKY
                | View.SYSTEM_UI_FLAG_LAYOUT_FULLSCREEN | View.SYSTEM_UI_FLAG_LAYOUT_HIDE_NAVIGATION
                | View.SYSTEM_UI_FLAG_LAYOUT_STABLE);
    }

    @Override public void onWindowFocusChanged(boolean focused) {
        super.onWindowFocusChanged(focused);
        if (focused) fullscreen();
    }

    @Override protected void onResume() {
        super.onResume();
        visible = true;
        generation++;
        fresh = false;
        EntryService.setActivityVisible(true);
        EntryService.sync(this);
        refresh();
    }

    @Override protected void onPause() {
        visible = false;
        generation++;
        handler.removeCallbacks(poll);
        bridge.cancel();
        inFlight = false;
        pending = false;
        pendingAction = "";
        pendingNodeId = "";
        nodeTouchActive = false;
        nodeRenderPending = false;
        EntryService.setActivityVisible(false);
        super.onPause();
    }

    @Override protected void onDestroy() {
        handler.removeCallbacksAndMessages(null);
        bridge.cancel();
        worker.shutdownNow();
        super.onDestroy();
    }

    @Override protected void onSaveInstanceState(Bundle bundle) {
        bundle.putString("page", page);
        bundle.putString("expandedNode", expandedNodeId);
        super.onSaveInstanceState(bundle);
    }

    @Override public void onBackPressed() {
        if (!"home".equals(page)) navigate("home"); else finish();
    }

    // Physical screen units, scaled from the 240px circular framebuffer. Do not
    // let logical density or system font scaling push controls outside the circle.
    private int dp(float value) {
        int extent = Math.min(getResources().getDisplayMetrics().widthPixels,
                getResources().getDisplayMetrics().heightPixels);
        return Math.round(value * extent / 240f);
    }

    private void font(TextView view, int size) { view.setTextSize(TypedValue.COMPLEX_UNIT_PX, dp(size)); }

    private boolean busy() { return pending || (model != null && model.busy()); }

    private boolean probing() {
        return (pending && "probe-nodes".equals(pendingAction)) || (model != null && model.probing());
    }

    private boolean probingNode(String id) {
        return (pending && "probe-node".equals(pendingAction) && pendingNodeId.equals(id))
                || (model != null && model.probingNode(id));
    }

    private boolean probingOneNode() {
        return (pending && "probe-node".equals(pendingAction))
                || (model != null && "probe-node".equals(model.jobKind) && "running".equals(model.jobStatus));
    }

    private void refresh() {
        if (!visible || inFlight) return;
        handler.removeCallbacks(poll);
        final int epoch = generation;
        inFlight = true;
        worker.execute(() -> {
            if (!visible || epoch != generation) return;
            try {
                JSONObject json = bridge.status();
                Model next = json.optBoolean("ok") ? new Model(json) : null;
                handler.post(() -> {
                    if (!visible || epoch != generation) return;
                    inFlight = false;
                    pending = false;
                    pendingAction = "";
                    pendingNodeId = "";
                    fresh = next != null;
                    if (next != null) {
                        model = next;
                        notice = next.jobMessage.isEmpty() ? next.message : next.jobMessage;
                    } else notice = Model.error(json);
                    render();
                    schedulePoll();
                });
            } catch (Exception ignored) {
                handler.post(() -> {
                    if (!visible || epoch != generation) return;
                    inFlight = false;
                    pending = false;
                    pendingAction = "";
                    pendingNodeId = "";
                    fresh = false;
                    notice = "本机服务未连接，请刷新";
                    render();
                    schedulePoll();
                });
            }
        });
    }

    private void schedulePoll() {
        handler.removeCallbacks(poll);
        if (visible) handler.postDelayed(poll, busy() ? 1500 : 5000);
    }

    private void send(String action, Boolean enabled, String nodeId) {
        if (!visible || !fresh || model == null || busy() || model.publicKey.isEmpty()) return;
        // A periodic read must not swallow a visible button click. Invalidate its
        // callback and close only that read; the serial executor submits the
        // user's action next. Pending mutations still fail the busy gate above.
        if (inFlight) {
            generation++;
            bridge.cancel();
            inFlight = false;
        }
        handler.removeCallbacks(poll);
        final int epoch = generation;
        final Model snapshot = model;
        pending = true;
        pendingAction = action;
        pendingNodeId = nodeId == null ? "" : nodeId;
        inFlight = true;
        notice = "正在提交…";
        render();
        worker.execute(() -> {
            if (!visible || epoch != generation) return;
            try {
                byte[] nonce = new byte[16];
                random.nextBytes(nonce);
                JSONObject plain = new JSONObject().put("v", 1).put("issuedAt", snapshot.requestTime())
                        .put("nonce", Base64.getUrlEncoder().withoutPadding().encodeToString(nonce)).put("action", action);
                if (enabled != null) plain.put("enabled", enabled.booleanValue());
                if (nodeId != null) plain.put("id", nodeId);
                if ("update-subscription".equals(action)) plain.put("url", "");
                JSONObject response = bridge.request(CryptoEnvelope.seal(snapshot.publicKey, plain.toString()));
                handler.post(() -> {
                    if (!visible || epoch != generation) return;
                    inFlight = false;
                    pending = response.optBoolean("ok");
                    if (!pending) { pendingAction = ""; pendingNodeId = ""; }
                    notice = pending ? "操作已提交，正在处理…" : Model.error(response);
                    render();
                    if (!pending) Toast.makeText(this, notice, Toast.LENGTH_SHORT).show();
                    handler.postDelayed(poll, 350);
                });
            } catch (Exception ignored) {
                handler.post(() -> {
                    if (!visible || epoch != generation) return;
                    inFlight = false;
                    pending = false;
                    pendingAction = "";
                    pendingNodeId = "";
                    fresh = false;
                    notice = "提交未完成，请刷新状态";
                    render();
                    schedulePoll();
                });
            }
        });
    }

    private void navigate(String next) {
        if (nodeScroll != null && nodeScroll.isLaidOut()) nodePosition = nodeScroll.getScrollY();
        page = next;
        render();
    }

    private void probeNodes() {
        if (!fresh || model == null || busy()) return;
        if (!model.enabled || !model.running) {
            notice = "请先开启 VPN 再测速";
            Toast.makeText(this, notice, Toast.LENGTH_SHORT).show();
            render();
            return;
        }
        send("probe-nodes", null, null);
    }

    private void probeNode(String id) {
        if (!id.matches("n-[a-f0-9]{16,64}") || !fresh || model == null || busy()) return;
        if (!model.enabled || !model.running) {
            notice = "请先开启 VPN 再测速";
            Toast.makeText(this, notice, Toast.LENGTH_SHORT).show();
            render();
            return;
        }
        send("probe-node", null, id);
    }

    private String nodeHint() {
        if (probing()) return "正在测速，请稍候";
        if (probingOneNode()) return "单节点测速中";
        if (busy()) return "正在处理…";
        if (!fresh || model == null) return notice;
        if (!model.enabled || !model.running) return "先开启 VPN 再测速";
        if (("probe-nodes".equals(model.jobKind) || "probe-node".equals(model.jobKind))
                && !model.jobMessage.isEmpty()) return model.jobMessage;
        return "左滑节点可单独测速";
    }

    private void render() {
        if (nodeTouchActive && "nodes".equals(page)) { nodeRenderPending = true; return; }
        if (nodeScroll != null && nodeScroll.isLaidOut()) nodePosition = nodeScroll.getScrollY();
        nodeScroll = null;
        screen = new FrameLayout(this);
        screen.setBackgroundColor(BACKGROUND);
        root = new LinearLayout(this);
        root.setOrientation(LinearLayout.VERTICAL);
        FrameLayout.LayoutParams safe = new FrameLayout.LayoutParams(dp(164), dp(164), Gravity.CENTER);
        screen.addView(root, safe);
        setContentView(screen);
        title();
        if ("nodes".equals(page)) nodesPage();
        else if ("subscription".equals(page)) subscriptionPage();
        else if ("entry".equals(page)) entryPage();
        else if ("more".equals(page)) morePage();
        else homePage();
        fullscreen();
    }

    private void title() {
        String title = "nodes".equals(page) ? "节点" : "subscription".equals(page) ? "订阅"
                : "entry".equals(page) ? "快捷入口" : "more".equals(page) ? "更多" : "VPN 控制";
        TextView text = text(title, 14, NAVY);
        text.setTypeface(null, Typeface.BOLD);
        FrameLayout.LayoutParams params = new FrameLayout.LayoutParams(dp(100), dp(18), Gravity.TOP | Gravity.CENTER_HORIZONTAL);
        params.topMargin = dp(18);
        screen.addView(text, params);
    }

    private void homePage() {
        LinearLayout body = new LinearLayout(this);
        body.setOrientation(LinearLayout.VERTICAL);
        body.setGravity(Gravity.CENTER);
        root.addView(body, new LinearLayout.LayoutParams(-1, 0, 1));
        boolean enabled = model != null && model.enabled;
        String powerLabel = pending ? "正在处理…" : (enabled ? "关闭 VPN" : "开启 VPN");
        Button power = button(powerLabel, enabled ? TEAL : BLUE, true,
                () -> send("set-enabled", !model.enabled, null));
        font(power, 16);
        power.setEnabled(fresh && !busy() && !inFlight);
        body.addView(power, new LinearLayout.LayoutParams(-1, dp(44)));
        TextView state = text(fresh && model != null ? model.actualState() : "正在确认运行状态", 13,
                fresh && model != null && model.running ? TEAL : GREY);
        body.addView(state, new LinearLayout.LayoutParams(-1, dp(20)));
        String exit = "当前出口：尚未确定";
        if (fresh && model != null) exit = model.running ? "当前出口：" + model.name(model.activeId)
                : model.enabled ? "当前出口：等待启动" : "当前出口：原厂联网";
        TextView current = text(exit, 12, NAVY);
        current.setMaxLines(2);
        current.setEllipsize(TextUtils.TruncateAt.END);
        body.addView(current, new LinearLayout.LayoutParams(-1, dp(40)));
        LinearLayout nav = row();
        addWeighted(nav, button("节点", BLUE, false, () -> navigate("nodes")), 44);
        addWeighted(nav, button("更多", BLUE, false, () -> navigate("more")), 44);
        body.addView(nav);
        footer(notice);
    }

    private void morePage() {
        root.setGravity(Gravity.CENTER_VERTICAL);
        LinearLayout first = row();
        addWeighted(first, button("订阅", BLUE, false, () -> navigate("subscription")), 48);
        addWeighted(first, button("快捷入口", BLUE, false, () -> navigate("entry")), 48);
        root.addView(first);
        LinearLayout second = row();
        second.addView(button("退出", NAVY, false, this::finish), new LinearLayout.LayoutParams(-1, dp(48)));
        LinearLayout.LayoutParams rowParams = new LinearLayout.LayoutParams(-1, dp(48));
        rowParams.topMargin = dp(4);
        root.addView(second, rowParams);
        backButton("返回主页", "home");
        footer(notice);
    }

    private void nodesPage() {
        nodeScroll = new NodeScrollView();
        nodeScroll.setFillViewport(false);
        LinearLayout list = new LinearLayout(this);
        list.setOrientation(LinearLayout.VERTICAL);
        nodeScroll.addView(list);
        root.addView(nodeScroll, new LinearLayout.LayoutParams(-1, 0, 1));
        nodeButton(list, "Auto", "自动选择", "按上次测速优选");
        if (model != null) for (Model.Node node : model.nodes) nodeButton(list, node.id, node.name, node.latency());
        if (model == null) list.addView(text("读取状态后显示节点", 13, GREY));
        final ScrollView scroll = nodeScroll;
        final int position = nodePosition;
        scroll.post(() -> scroll.scrollTo(0, position));
        LinearLayout actions = row();
        addWeighted(actions, button("返回主页", NAVY, false, () -> navigate("home")), 40);
        Button refresh = button(probing() ? "测速中…" : "刷新", BLUE, false, this::probeNodes);
        refresh.setEnabled(fresh && model != null && model.enabled && model.running && !busy());
        refresh.setAlpha(refresh.isEnabled() ? 1f : 0.45f);
        addWeighted(actions, refresh, 40);
        LinearLayout.LayoutParams actionsParams = new LinearLayout.LayoutParams(-1, dp(40));
        actionsParams.topMargin = dp(4);
        root.addView(actions, actionsParams);
        footer(nodeHint());
    }

    private void nodeButton(LinearLayout list, String id, String name, String detail) {
        boolean selected = model != null && id.equals(model.selectedId);
        String state = probingNode(id) ? "测速中…" : detail;
        Button button = button(name + "\n" + (selected ? "已选择 · " : "") + state,
                selected ? TEAL : NAVY, false, () -> {
                    if (fresh && model != null && !busy() && !id.equals(model.selectedId)) send("select", null, id);
                });
        button.setGravity(Gravity.CENTER_VERTICAL | Gravity.START);
        font(button, 12);
        button.setMaxLines(3);
        button.setEllipsize(TextUtils.TruncateAt.END);
        button.setPadding(dp(6), dp(3), dp(6), dp(3));
        if (!"Auto".equals(id) && id.equals(expandedNodeId)) {
            CharSequence title = TextUtils.ellipsize(name, button.getPaint(), dp(97), TextUtils.TruncateAt.END);
            button.setText(title + "\n" + (selected ? "已选择 · " : "") + state);
        }
        LinearLayout.LayoutParams params = new LinearLayout.LayoutParams(-1, dp(60));
        params.bottomMargin = dp(3);
        if ("Auto".equals(id)) {
            button.setEnabled(fresh && !busy() && !selected);
            list.addView(button, params);
            return;
        }
        // Keep the selected row touchable: swiping it must still expose its test
        // action. Selection is gated inside the click listener, never by disabling
        // the whole view and accidentally disabling its gesture handling.
        LinearLayout row = row();
        row.addView(button, new LinearLayout.LayoutParams(0, dp(60), 1));
        button.setOnTouchListener(new NodeSwipeTouch(id));
        button.setAlpha(fresh && !busy() ? 1f : 0.75f);
        button.setContentDescription(name + "，" + (selected ? "已选择，" : "") + state + "，左滑显示测速");
        if (id.equals(expandedNodeId)) {
            Button test = button(probingNode(id) ? "测速中…" : "测速", BLUE, true, () -> probeNode(id));
            font(test, 11);
            test.setEnabled(fresh && model != null && model.enabled && model.running && !busy());
            test.setAlpha(test.isEnabled() ? 1f : 0.45f);
            LinearLayout.LayoutParams action = new LinearLayout.LayoutParams(dp(52), dp(60));
            action.leftMargin = dp(3);
            row.addView(test, action);
        }
        list.addView(row, params);
    }

    private final class NodeScrollView extends ScrollView {
        NodeScrollView() { super(MainActivity.this); }

        @Override public boolean dispatchTouchEvent(MotionEvent event) {
            if (event.getActionMasked() == MotionEvent.ACTION_DOWN) nodeTouchActive = true;
            try {
                return super.dispatchTouchEvent(event);
            } finally {
                if (event.getActionMasked() == MotionEvent.ACTION_UP || event.getActionMasked() == MotionEvent.ACTION_CANCEL) {
                    nodeTouchActive = false;
                    if (nodeRenderPending) {
                        nodeRenderPending = false;
                        if (visible && "nodes".equals(page)) render();
                    }
                }
            }
        }
    }

    private final class NodeSwipeTouch implements View.OnTouchListener {
        private final String id;
        private final SwipeGesture gesture = new SwipeGesture(dp(8), dp(20));

        NodeSwipeTouch(String id) { this.id = id; }

        @Override public boolean onTouch(View view, MotionEvent event) {
            switch (event.getActionMasked()) {
                case MotionEvent.ACTION_DOWN:
                    gesture.down(event.getRawX(), event.getRawY());
                    view.getParent().requestDisallowInterceptTouchEvent(true);
                    return true;
                case MotionEvent.ACTION_MOVE:
                    gesture.move(event.getRawX(), event.getRawY());
                    if (gesture.vertical()) view.getParent().requestDisallowInterceptTouchEvent(false);
                    return true;
                case MotionEvent.ACTION_UP:
                    SwipeGesture.Action action = gesture.up(event.getRawX(), event.getRawY());
                    view.getParent().requestDisallowInterceptTouchEvent(false);
                    if (action == SwipeGesture.Action.REVEAL) { expandedNodeId = id; render(); }
                    else if (action == SwipeGesture.Action.CLOSE && id.equals(expandedNodeId)) { expandedNodeId = ""; render(); }
                    else if (action == SwipeGesture.Action.TAP) view.performClick();
                    return true;
                case MotionEvent.ACTION_CANCEL:
                    gesture.cancel();
                    view.getParent().requestDisallowInterceptTouchEvent(false);
                    return true;
                case MotionEvent.ACTION_POINTER_DOWN:
                    gesture.cancel();
                    view.getParent().requestDisallowInterceptTouchEvent(false);
                    return true;
                default:
                    return true;
            }
        }
    }

    private void subscriptionPage() {
        LinearLayout body = new LinearLayout(this);
        body.setOrientation(LinearLayout.VERTICAL);
        body.setGravity(Gravity.CENTER_VERTICAL);
        root.addView(body, new LinearLayout.LayoutParams(-1, 0, 1));
        TextView saved = text(model != null && model.subscriptionConfigured ? "已保存订阅" : "尚未保存订阅", 14, NAVY);
        saved.setTypeface(null, Typeface.BOLD);
        body.addView(saved, new LinearLayout.LayoutParams(-1, dp(18)));
        String updated = model == null || model.updatedAt.isEmpty() ? "尚无更新时间" : "上次：" + model.updatedAt.replace('T', ' ');
        TextView timestamp = text(updated, 11, GREY);
        timestamp.setMaxLines(2);
        body.addView(timestamp, new LinearLayout.LayoutParams(-1, dp(24)));
        Button update = button(busy() ? "正在处理…" : "更新已保存订阅", BLUE, true,
                () -> send("update-subscription", null, null));
        update.setEnabled(fresh && model != null && model.subscriptionConfigured && !busy() && !inFlight);
        body.addView(update, new LinearLayout.LayoutParams(-1, dp(40)));
        String result = notice.isEmpty() ? "更换订阅链接请使用网页管理" : notice;
        TextView feedback = text(result, 12, GREY);
        feedback.setMaxLines(2);
        feedback.setEllipsize(TextUtils.TruncateAt.END);
        body.addView(feedback, new LinearLayout.LayoutParams(-1, dp(36)));
        backButton("返回更多", "more");
        footer("换链接请用网页");
    }

    private void entryPage() {
        LinearLayout body = new LinearLayout(this);
        body.setOrientation(LinearLayout.VERTICAL);
        body.setGravity(Gravity.CENTER);
        root.addView(body, new LinearLayout.LayoutParams(-1, 0, 1));
        boolean permitted = Settings.canDrawOverlays(this);
        boolean enabled = EntryService.enabled(this);
        TextView help = text("首页显示可拖动的 VPN 入口；\n其他页面与锁屏时隐藏。", 12, NAVY);
        help.setMaxLines(3);
        body.addView(help, new LinearLayout.LayoutParams(-1, dp(44)));
        Button entry = button(!permitted ? "授权快捷入口" : enabled ? "关闭快捷入口" : "开启快捷入口", BLUE, true, () -> {
            if (!Settings.canDrawOverlays(this)) {
                EntryService.setEnabled(this, true);
                try {
                    startActivity(new Intent(Settings.ACTION_MANAGE_OVERLAY_PERMISSION, Uri.parse("package:" + getPackageName())));
                } catch (Exception ignored) { Toast.makeText(this, "系统授权页面不可用", Toast.LENGTH_LONG).show(); }
            } else {
                EntryService.setEnabled(this, !EntryService.enabled(this));
                render();
            }
        });
        body.addView(entry, new LinearLayout.LayoutParams(-1, dp(40)));
        TextView state = text(!permitted ? "需要你在系统设置中允许显示" : enabled ? "快捷入口已启用" : "快捷入口已关闭", 12, GREY);
        state.setMaxLines(2);
        body.addView(state, new LinearLayout.LayoutParams(-1, dp(32)));
        backButton("返回更多", "more");
    }

    private LinearLayout row() {
        LinearLayout row = new LinearLayout(this);
        row.setOrientation(LinearLayout.HORIZONTAL);
        row.setGravity(Gravity.CENTER_VERTICAL);
        return row;
    }

    private TextView text(String value, int size, int color) {
        TextView text = new TextView(this);
        text.setText(value);
        font(text, size);
        text.setTextColor(color);
        text.setGravity(Gravity.CENTER);
        text.setIncludeFontPadding(false);
        return text;
    }

    private Button button(String label, int color, boolean filled, Runnable action) {
        Button button = new Button(this);
        button.setText(label);
        font(button, 12);
        button.setAllCaps(false);
        button.setTextColor(filled ? Color.WHITE : color);
        button.setMinHeight(0);
        button.setMinWidth(0);
        button.setMinimumHeight(0);
        button.setMinimumWidth(0);
        button.setPadding(dp(3), 0, dp(3), 0);
        GradientDrawable background = new GradientDrawable();
        background.setCornerRadius(dp(10));
        background.setColor(filled ? color : Color.WHITE);
        background.setStroke(dp(1), filled ? color : Color.rgb(216, 225, 235));
        button.setBackground(background);
        button.setBackgroundTintList((ColorStateList) null);
        button.setOnClickListener(view -> action.run());
        return button;
    }

    private void addWeighted(LinearLayout row, View child, int height) {
        LinearLayout.LayoutParams params = new LinearLayout.LayoutParams(0, dp(height), 1);
        params.leftMargin = dp(1);
        params.rightMargin = dp(1);
        row.addView(child, params);
    }

    private void backButton(String label, String destination) {
        LinearLayout.LayoutParams params = new LinearLayout.LayoutParams(-1, dp(40));
        params.topMargin = dp(4);
        root.addView(button(label, NAVY, false, () -> navigate(destination)), params);
    }

    private void footer(String value) {
        TextView footer = text(value, 10, GREY);
        footer.setSingleLine();
        footer.setEllipsize(TextUtils.TruncateAt.END);
        FrameLayout.LayoutParams params = new FrameLayout.LayoutParams(dp(104), dp(16), Gravity.BOTTOM | Gravity.CENTER_HORIZONTAL);
        params.bottomMargin = dp(18);
        screen.addView(footer, params);
    }
}
