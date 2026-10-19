package com.codex.u30vpn;

import android.net.LocalSocket;
import android.net.LocalSocketAddress;
import java.io.BufferedInputStream;
import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.nio.ByteBuffer;
import java.nio.charset.CodingErrorAction;
import java.nio.charset.StandardCharsets;
import org.json.JSONException;
import org.json.JSONObject;

/** Fixed local RPC transport: no HTTP listener, shell, credentials or node parameters. */
final class BridgeClient {
    private static final String SOCKET = "codex_u30vpn_v1";
    private static final int MAX_REQUEST = 32768;
    private static final int MAX_RESPONSE = 2 * 1024 * 1024;
    private LocalSocket active;

    JSONObject status() throws IOException, JSONException {
        return exchange(new JSONObject().put("v", 1).put("op", "status"));
    }

    JSONObject request(String envelope) throws IOException, JSONException {
        if (envelope.length() == 0 || envelope.length() > 24576 || !envelope.matches("[A-Za-z0-9_-]+")) {
            throw new IOException("invalid request");
        }
        return exchange(new JSONObject().put("v", 1).put("op", "request").put("envelope", envelope));
    }

    private JSONObject exchange(JSONObject request) throws IOException, JSONException {
        byte[] bytes = (request.toString() + "\n").getBytes(StandardCharsets.UTF_8);
        if (bytes.length > MAX_REQUEST) throw new IOException("request too large");
        LocalSocket socket = new LocalSocket();
        synchronized (this) { active = socket; }
        try {
            socket.connect(new LocalSocketAddress(SOCKET, LocalSocketAddress.Namespace.ABSTRACT));
            socket.setSoTimeout(12000);
            if (socket.getPeerCredentials().getUid() != 0) throw new IOException("untrusted peer");
            socket.getOutputStream().write(bytes);
            socket.getOutputStream().flush();
            BufferedInputStream input = new BufferedInputStream(socket.getInputStream(), 8192);
            ByteArrayOutputStream output = new ByteArrayOutputStream(16384);
            boolean complete = false;
            int value;
            while ((value = input.read()) != -1) {
                if (value == '\n') { complete = true; break; }
                if (output.size() >= MAX_RESPONSE) throw new IOException("response too large");
                output.write(value);
            }
            if (!complete || output.size() == 0) throw new IOException("incomplete response");
            String json = StandardCharsets.UTF_8.newDecoder().onMalformedInput(CodingErrorAction.REPORT)
                    .onUnmappableCharacter(CodingErrorAction.REPORT).decode(ByteBuffer.wrap(output.toByteArray())).toString();
            return new JSONObject(json);
        } finally {
            synchronized (this) { if (active == socket) active = null; }
            socket.close();
        }
    }

    synchronized void cancel() {
        if (active == null) return;
        try { active.close(); } catch (IOException ignored) { }
        active = null;
    }
}
