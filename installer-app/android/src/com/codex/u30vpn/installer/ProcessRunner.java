package com.codex.u30vpn.installer;

import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.nio.charset.StandardCharsets;
import java.util.concurrent.TimeUnit;

/** Captures only bounded stdout; stderr is drained and is never sent to the UI. */
public final class ProcessRunner {
    private ProcessRunner() { }
    private static final int MAX_OUTPUT = 65536;
    public static final class Result {
        public final int exit;
        public final String stdout;
        public final boolean uncertain;
        public final boolean overflow;
        private Result(int exit, String stdout, boolean uncertain, boolean overflow) {
            this.exit = exit; this.stdout = stdout; this.uncertain = uncertain; this.overflow = overflow;
        }
    }
    private static final class Capture implements Runnable {
        private final InputStream stream;
        private final boolean retain;
        private final ByteArrayOutputStream bytes = new ByteArrayOutputStream();
        private volatile boolean overflow;
        private volatile boolean failed;
        private Capture(InputStream stream, boolean retain) { this.stream = stream; this.retain = retain; }
        @Override public void run() {
            try {
                byte[] block = new byte[4096]; int count;
                while ((count = stream.read(block)) != -1) {
                    if (retain) synchronized (bytes) {
                        int keep = Math.min(count, MAX_OUTPUT - bytes.size());
                        bytes.write(block, 0, keep);
                        if (keep < count) overflow = true;
                    }
                }
            } catch (IOException error) { failed = true; }
        }
        private String value() { synchronized (bytes) { return new String(bytes.toByteArray(), StandardCharsets.UTF_8); } }
    }
    public static Result run(String[] argv, long timeoutMillis) throws IOException {
        if (timeoutMillis <= 0 || timeoutMillis > 600000) throw new IllegalArgumentException("Invalid subprocess deadline");
        Process process = new ProcessBuilder(argv).start();
        process.getOutputStream().close();
        Capture output = new Capture(process.getInputStream(), true);
        Capture error = new Capture(process.getErrorStream(), false);
        Thread outReader = new Thread(output, "u30-install-stdout");
        Thread errReader = new Thread(error, "u30-install-stderr");
        outReader.setDaemon(true); errReader.setDaemon(true);
        outReader.start(); errReader.start();
        boolean uncertain = false;
        int exit = -1;
        try {
            if (process.waitFor(timeoutMillis, TimeUnit.MILLISECONDS)) exit = process.exitValue();
            else { uncertain = true; process.destroyForcibly(); }
            outReader.join(500); errReader.join(500);
        } catch (InterruptedException interrupted) {
            uncertain = true; process.destroyForcibly(); Thread.currentThread().interrupt();
        } finally {
            try { process.getInputStream().close(); } catch (IOException ignored) { }
            try { process.getErrorStream().close(); } catch (IOException ignored) { }
        }
        uncertain |= outReader.isAlive() || errReader.isAlive() || output.failed;
        return new Result(exit, output.value(), uncertain, output.overflow);
    }
}
