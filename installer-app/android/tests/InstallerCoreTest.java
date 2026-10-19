package com.codex.u30vpn.installer;

import java.io.ByteArrayInputStream;
import java.io.File;
import java.lang.reflect.InvocationTargetException;
import java.lang.reflect.Method;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.util.Arrays;

/** Host tests exercise package integrity, command boundaries and state transitions. */
public final class InstallerCoreTest {
    private static Class<?> core;
    private static int checks;

    public static void main(String[] args) throws Exception {
        try { core = Class.forName("com.codex.u30vpn.installer.InstallerCore"); }
        catch (ClassNotFoundException missing) { throw new AssertionError("InstallerCore package-integrity implementation is missing"); }
        integrity();
        commands();
        rootAuthorization();
        results();
        transitions();
        System.out.println("InstallerCoreTest: " + checks + " checks passed");
    }

    private static Object call(String name, Class<?>[] types, Object... values) throws Exception {
        Method method = core.getDeclaredMethod(name, types);
        method.setAccessible(true);
        try { return method.invoke(null, values); }
        catch (InvocationTargetException wrapped) {
            Throwable cause = wrapped.getCause();
            if (cause instanceof Exception) throw (Exception) cause;
            throw wrapped;
        }
    }
    private static void check(boolean condition, String message) {
        checks++;
        if (!condition) throw new AssertionError(message);
    }
    private static File copy(byte[] data, File dir, String hash, long bytes) throws Exception {
        return (File) call("verifyAndCopy", new Class<?>[] {java.io.InputStream.class, File.class, String.class, long.class},
                new ByteArrayInputStream(data), dir, hash, bytes);
    }
    private static void rejects(byte[] bytes, File dir, String hash, long count) throws Exception {
        try { copy(bytes, dir, hash, count); throw new AssertionError("Bad asset was accepted"); }
        catch (java.io.IOException expected) { checks++; }
    }
    private static void integrity() throws Exception {
        File dir = Files.createTempDirectory("u30-installer-test-").toFile();
        byte[] abc = "abc".getBytes(StandardCharsets.UTF_8);
        String sha = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad";
        try {
            File asset = copy(abc, dir, sha, 3);
            check(Arrays.equals(abc, Files.readAllBytes(asset.toPath())), "Verified bytes were not published");
            rejects("abd".getBytes(StandardCharsets.UTF_8), dir, sha, 3);
            rejects("ab".getBytes(StandardCharsets.UTF_8), dir, sha, 3);
            rejects("abcd".getBytes(StandardCharsets.UTF_8), dir, sha, 3);
            check(Arrays.equals(abc, Files.readAllBytes(asset.toPath())), "A failed copy replaced the prior asset");
            check(dir.list().length == 1, "Failed extraction left temporary files");
        } finally {
            for (File file : dir.listFiles()) Files.delete(file.toPath());
            Files.delete(dir.toPath());
        }
    }
    private static String text(String name, String input) throws Exception {
        return (String) call(name, new Class<?>[] {String.class}, input);
    }
    private static boolean flag(String name, String input) throws Exception {
        return (Boolean) call(name, new Class<?>[] {String.class}, input);
    }
    private static void commands() throws Exception {
        String preflight = (String) call("preflightCommand", new Class<?>[] {});
        check(preflight.contains("/data/adb/modules/codex_u30pro_vpn") && preflight.contains("/data/adb/modules_update/codex_u30pro_vpn"), "Both module namespaces must be checked before Magisk overwrites staging");
        check(!preflight.contains("rm ") && !preflight.contains("mkdir "), "Preflight must be read only");
        check("'/data/user/0/a b/module.zip'".equals(text("shellQuote", "/data/user/0/a b/module.zip")), "Spaces escaped incorrectly");
        check("'a'\\''; reboot; #'".equals(text("shellQuote", "a'; reboot; #")), "A quote can escape the shell argument");
        check("magisk --install-module '/data/user/0/a/module.zip'".equals(text("installCommand", "/data/user/0/a/module.zip")), "Installer command changed or path became shell code");
        try { text("installCommand", "../module.zip"); throw new AssertionError("Relative module path accepted"); }
        catch (IllegalArgumentException expected) { checks++; }
        try { text("shellQuote", "bad\u0000path"); throw new AssertionError("NUL shell argument accepted"); }
        catch (IllegalArgumentException expected) { checks++; }
        check(flag("supportedVersion", "28100\n"), "Minimum supported Magisk rejected");
        check(!flag("supportedVersion", "28099"), "Old Magisk accepted");
        check(!flag("supportedVersion", "28100; reboot"), "Non-numeric version accepted");
        check(flag("supportsInstall", "Options:\n   --install-module ZIP      install a module zip file\n"), "Official CLI help rejected");
        check(!flag("supportsInstall", "not --install-module ZIP"), "Unrelated help text accepted");
    }
    private static void rootAuthorization() throws Exception {
        try { core.getDeclaredMethod("rootGranted", int.class, String.class, boolean.class, boolean.class); }
        catch (NoSuchMethodException missing) { throw new AssertionError("Root authorization policy is missing"); }
        Class<?>[] types = {int.class, String.class, boolean.class, boolean.class};
        check((Boolean) call("rootGranted", types, 0, "0\n", false, false), "Authorized root rejected");
        check(!(Boolean) call("rootGranted", types, 1, "0\n", false, false), "Denied su accepted due to output");
        check(!(Boolean) call("rootGranted", types, 0, "2000\n", false, false), "Non-root shell accepted");
        check(!(Boolean) call("rootGranted", types, 0, "0\n", true, false), "Uncertain authorization accepted");
        check(!(Boolean) call("rootGranted", types, 0, "0\n", false, true), "Truncated authorization accepted");
    }
    private static String result(int exit, String output, boolean timeout, boolean overflow) throws Exception {
        return call("classifyInstall", new Class<?>[] {int.class, String.class, boolean.class, boolean.class}, exit, output, timeout, overflow).toString();
    }
    private static void results() throws Exception {
        check("STAGED".equals(result(0, "- prepare\nU30PRO_MODULE_STAGED\n- Done\n", false, false)), "Verified stage marker was ignored");
        check("ERROR".equals(result(1, "U30PRO_MODULE_STAGED\n", false, false)), "Nonzero install exit was accepted");
        check("UNKNOWN".equals(result(0, "- Done\n", false, false)), "Missing project marker was called successful");
        check("UNKNOWN".equals(result(0, "prefix U30PRO_MODULE_STAGED\n", false, false)), "Non-line marker was accepted");
        check("UNKNOWN".equals(result(0, "U30PRO_MODULE_STAGED\n", true, false)), "Timeout was called staged or rolled back");
        check("UNKNOWN".equals(result(0, "U30PRO_MODULE_STAGED\n", false, true)), "Truncated log was treated as complete evidence");
    }
    private static void transitions() throws Exception {
        Class<?> type = Class.forName("com.codex.u30vpn.installer.InstallerCore$Session");
        Object session = type.getDeclaredConstructor().newInstance();
        Method begin = type.getDeclaredMethod("beginInstall");
        Method finish = type.getDeclaredMethod("finish", Class.forName("com.codex.u30vpn.installer.InstallerCore$Phase"));
        Method reboot = type.getDeclaredMethod("beginReboot");
        Class<?> phases = Class.forName("com.codex.u30vpn.installer.InstallerCore$Phase");
        Object staged = phases.getField("STAGED").get(null);
        Object unknown = phases.getField("UNKNOWN").get(null);
        check(!(Boolean) reboot.invoke(session), "Reboot permitted before stage");
        check((Boolean) begin.invoke(session), "First install was not admitted");
        check(!(Boolean) begin.invoke(session), "Concurrent install was admitted");
        finish.invoke(session, staged);
        check(!(Boolean) begin.invoke(session), "Staged install was repeated");
        check((Boolean) reboot.invoke(session), "Explicit reboot rejected after stage");
        check(!(Boolean) reboot.invoke(session), "Duplicate reboot was admitted");
        finish.invoke(session, unknown);
        check(!(Boolean) begin.invoke(session), "Unknown root outcome admitted another install");
        check(!(Boolean) reboot.invoke(session), "Unknown outcome enabled reboot");
    }
}
