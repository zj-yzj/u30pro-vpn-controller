package com.codex.u30vpn.installer;

import java.io.File;
import java.lang.reflect.InvocationTargetException;
import java.lang.reflect.Method;

public final class ProcessRunnerTest {
    public static void main(String[] args) throws Exception {
        if (args.length > 0) {
            if (args[0].equals("flood")) { for (int n = 0; n < 70000; n++) System.out.print('x'); }
            if (args[0].equals("timeout")) Thread.sleep(5000);
            return;
        }
        Class<?> runner;
        try { runner = Class.forName("com.codex.u30vpn.installer.ProcessRunner"); }
        catch (ClassNotFoundException missing) { throw new AssertionError("Bounded subprocess implementation is missing"); }
        Method run = runner.getDeclaredMethod("run", String[].class, long.class);
        String java = new File(System.getProperty("java.home"), "bin/java" + (File.separatorChar == '\\' ? ".exe" : "")).getPath();
        Object flood = run.invoke(null, new String[] {java, "-cp", System.getProperty("java.class.path"), ProcessRunnerTest.class.getName(), "flood"}, 10000L);
        if (!flood.getClass().getField("overflow").getBoolean(flood)) throw new AssertionError("Output overflow was hidden");
        if (((String) flood.getClass().getField("stdout").get(flood)).length() != 65536) throw new AssertionError("Output is not bounded to 64 KiB");
        long start = System.nanoTime();
        Object timeout = run.invoke(null, new String[] {java, "-cp", System.getProperty("java.class.path"), ProcessRunnerTest.class.getName(), "timeout"}, 150L);
        if (!timeout.getClass().getField("uncertain").getBoolean(timeout)) throw new AssertionError("Timed-out root result was treated as known");
        if ((System.nanoTime() - start) / 1000000 > 2500) throw new AssertionError("Deadline did not bound subprocess wait");
        System.out.println("ProcessRunnerTest: output limit and deadline passed");
    }
}
