package com.codex.u30vpn;

import java.util.ArrayList;
import java.util.List;
import org.json.JSONArray;
import org.json.JSONException;
import org.json.JSONObject;

final class Model {
    static final class Node {
        final String id;
        final String name;
        final int delay;
        final Boolean alive;
        Node(JSONObject json) {
            id = json.optString("id", "");
            name = text(json.optString("name", "节点"), 160);
            delay = json.isNull("delay") ? -1 : json.optInt("delay", -1);
            alive = json.isNull("alive") ? null : json.optBoolean("alive");
        }
        String latency() {
            if (Boolean.FALSE.equals(alive)) return "上次测速未通过";
            return delay > 0 ? delay + " ms" : "未测速";
        }
    }

    final boolean enabled;
    final boolean running;
    final String state;
    final String selectedId;
    final String activeId;
    final String publicKey;
    final String message;
    final long serverTime;
    final long receivedAt;
    final List<Node> nodes;
    final boolean subscriptionConfigured;
    final String updatedAt;
    final String jobKind;
    final String jobNodeId;
    final String jobStatus;
    final String jobMessage;

    Model(JSONObject json) throws JSONException {
        if (!json.optBoolean("ok") || !json.optBoolean("installed")) throw new JSONException("not installed");
        enabled = json.optBoolean("desiredEnabled");
        running = json.optBoolean("running");
        state = json.optString("state", "error");
        selectedId = json.optString("selectedId", "Auto");
        activeId = json.optString("activeId", "");
        publicKey = json.optString("publicKeyPem", "");
        message = text(json.optString("message", ""), 300);
        serverTime = json.optLong("serverTime", 0);
        receivedAt = android.os.SystemClock.elapsedRealtime();
        JSONArray source = json.optJSONArray("nodes");
        nodes = new ArrayList<>();
        if (source != null) {
            if (source.length() > 4096) throw new JSONException("too many nodes");
            for (int i = 0; i < source.length(); i++) {
                Node node = new Node(source.getJSONObject(i));
                if (!node.id.matches("n-[a-f0-9]{16,64}")) throw new JSONException("invalid node");
                nodes.add(node);
            }
        }
        JSONObject subscription = json.optJSONObject("subscription");
        subscriptionConfigured = subscription != null && subscription.optBoolean("configured");
        updatedAt = subscription == null ? "" : text(subscription.optString("updatedAt", ""), 64);
        JSONObject job = json.optJSONObject("job");
        jobKind = job == null ? "" : job.optString("kind", "");
        String target = job == null ? "" : job.optString("nodeId", "");
        jobNodeId = target.matches("n-[a-f0-9]{16,64}") ? target : "";
        jobStatus = job == null ? "" : job.optString("status", "");
        jobMessage = job == null ? "" : text(job.optString("message", ""), 300);
    }

    boolean busy() { return "busy".equals(state) || "running".equals(jobStatus); }

    boolean probing() { return "probe-nodes".equals(jobKind) && "running".equals(jobStatus); }

    boolean probingNode(String id) {
        return "probe-node".equals(jobKind) && "running".equals(jobStatus) && jobNodeId.equals(id);
    }

    long requestTime() {
        if (serverTime <= 0) return System.currentTimeMillis();
        return serverTime + Math.max(0, android.os.SystemClock.elapsedRealtime() - receivedAt);
    }

    String name(String id) {
        if ("Auto".equals(id)) return "自动选择";
        for (Node node : nodes) if (node.id.equals(id)) return node.name;
        return "尚未确定";
    }

    String actualState() {
        if (busy()) return "正在处理…";
        if (running) return enabled ? "VPN 正在运行" : "VPN 正在关闭…";
        if ("starting".equals(state)) return "VPN 正在启动…";
        if (enabled) return "VPN 未运行，请刷新";
        return "VPN 已关闭";
    }

    static String text(String raw, int maximum) {
        if (raw == null) return "";
        String safe = raw.replaceAll("[\\p{Cc}&&[^\\n\\t]]", "").replace('\n', ' ').replace('\t', ' ');
        return safe.length() <= maximum ? safe : safe.substring(0, maximum) + "…";
    }

    static String error(JSONObject json) {
        JSONObject error = json.optJSONObject("error");
        String code = error == null ? "" : error.optString("code", "");
        switch (code) {
            case "busy": return "正在处理上一项操作";
            case "invalid_request":
            case "replayed_request": return "请求已过期，请刷新重试";
            case "invalid_node": return "该节点已不存在，请刷新";
            case "download_failed": return "下载失败，已保留原配置";
            case "invalid_subscription": return "订阅无效，已保留原配置";
            case "apply_failed": return "应用失败，已恢复原配置";
            case "control_failed": return "开关未完成，请刷新状态";
            case "vpn_stopped": return "请先开启 VPN 再测速";
            case "probe_failed": return "测速未完成，请重试";
            case "unauthorized":
            case "untrusted_client": return "本机授权未通过，请重新安装";
            case "not_initialized": return "管理组件尚未初始化";
            default: return "操作未完成，请刷新后重试";
        }
    }
}
