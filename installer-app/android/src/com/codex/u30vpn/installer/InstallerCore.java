package com.codex.u30vpn.installer;

import java.io.File;
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.nio.file.Files;
import java.nio.file.StandardCopyOption;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.util.UUID;

/** Platform-independent integrity and installation-result policy. */
public final class InstallerCore {
    private InstallerCore() { }
    public enum Phase { READY, EXTRACTING, AUTHORIZING, INSTALLING, STAGED, REBOOTING, REVIEW, UNKNOWN, ERROR }

    public static final class Session {
        private Phase phase = Phase.READY;
        public synchronized Phase phase() { return phase; }
        public synchronized boolean beginInstall() {
            if (phase != Phase.READY && phase != Phase.ERROR) return false;
            phase = Phase.EXTRACTING;
            return true;
        }
        public synchronized boolean beginReboot() {
            if (phase != Phase.STAGED) return false;
            phase = Phase.REBOOTING;
            return true;
        }
        public synchronized void finish(Phase next) { phase = next; }
    }

    public static String shellQuote(String argument) {
        if (argument == null || argument.indexOf('\0') >= 0) throw new IllegalArgumentException("Invalid shell argument");
        return "'" + argument.replace("'", "'\\''") + "'";
    }
    public static String installCommand(String path) {
        if (path == null || !path.startsWith("/")) throw new IllegalArgumentException("Absolute private path required");
        return "magisk --install-module " + shellQuote(path);
    }
    public static String preflightCommand() {
        return "for p in /data/adb/modules/codex_u30pro_vpn /data/adb/modules_update/codex_u30pro_vpn; do "
                + "if [ -e \"$p\" ] || [ -L \"$p\" ]; then exit 42; fi; done";
    }
    public static boolean supportedVersion(String output) {
        String value = output == null ? "" : output.trim();
        if (!value.matches("[0-9]{5,9}")) return false;
        try { return Integer.parseInt(value) >= 28100; }
        catch (NumberFormatException invalid) { return false; }
    }
    public static boolean supportsInstall(String output) {
        return output != null && java.util.regex.Pattern.compile("(?m)^\\s*--install-module\\s+ZIP(?:\\s|$)").matcher(output).find();
    }
    public static boolean rootGranted(int exit, String output, boolean uncertain, boolean overflow) {
        return !uncertain && !overflow && exit == 0 && output != null && output.trim().equals("0");
    }
    public static Phase classifyInstall(int exit, String output, boolean uncertain, boolean overflow) {
        if (uncertain || overflow) return Phase.UNKNOWN;
        if (exit != 0) return Phase.ERROR;
        if (output != null) {
            for (String line : output.split("\\r?\\n")) {
                if (line.equals("U30PRO_MODULE_STAGED")) return Phase.STAGED;
            }
        }
        return Phase.UNKNOWN;
    }

    /** Failed/truncated/oversized copies never replace a previously verified private ZIP. */
    public static File verifyAndCopy(InputStream source, File directory, String expectedHash, long expectedBytes) throws IOException {
        if (expectedBytes <= 0 || expectedHash == null || !expectedHash.matches("[0-9a-f]{64}")) throw new IOException("Invalid asset pins");
        File base = directory.getCanonicalFile();
        if (!base.isDirectory()) throw new IOException("Private directory unavailable");
        File target = new File(base, "u30pro-module.zip");
        if (Files.isSymbolicLink(target.toPath())) throw new IOException("Unsafe asset target");
        File temporary = new File(base, ".module-" + UUID.randomUUID().toString() + ".tmp");
        try {
            MessageDigest digest = MessageDigest.getInstance("SHA-256");
            long total = 0;
            try (FileOutputStream output = new FileOutputStream(temporary)) {
                byte[] block = new byte[8192];
                int count;
                while ((count = source.read(block)) != -1) {
                    if (count == 0) continue;
                    if (count > expectedBytes - total) throw new IOException("Asset size mismatch");
                    total += count;
                    digest.update(block, 0, count);
                    output.write(block, 0, count);
                }
                if (total != expectedBytes || !hex(digest.digest()).equals(expectedHash)) throw new IOException("Asset checksum mismatch");
                output.getFD().sync();
            }
            if (Files.getFileStore(base.toPath()).supportsFileAttributeView("posix")) {
                Files.setPosixFilePermissions(temporary.toPath(), java.nio.file.attribute.PosixFilePermissions.fromString("rw-------"));
            }
            Files.move(temporary.toPath(), target.toPath(), StandardCopyOption.ATOMIC_MOVE, StandardCopyOption.REPLACE_EXISTING);
            return target;
        } catch (NoSuchAlgorithmException impossible) {
            throw new IOException("SHA-256 unavailable", impossible);
        } finally {
            Files.deleteIfExists(temporary.toPath());
        }
    }
    private static String hex(byte[] bytes) {
        StringBuilder value = new StringBuilder(bytes.length * 2);
        for (byte b : bytes) value.append(Character.forDigit((b >>> 4) & 15, 16)).append(Character.forDigit(b & 15, 16));
        return value.toString();
    }
}
