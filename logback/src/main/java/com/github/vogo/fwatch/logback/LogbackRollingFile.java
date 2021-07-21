package com.github.vogo.fwatch.logback;

import ch.qos.logback.core.recovery.ResilientFileOutputStream;
import ch.qos.logback.core.rolling.helper.RenameUtil;

import java.io.File;
import java.nio.charset.StandardCharsets;

/**
 * @author wongoo
 */
public class LogbackRollingFile {

    public static void main(String[] args) throws Exception {
        String src = args[0];
        String target = args[1];

        System.out.println("rename " + src + " to " + target);
        RenameUtil renameUtil = new RenameUtil();
        renameUtil.rename(src, target);

        System.out.println("create new file " + src);
        File file = new File(src);
        ResilientFileOutputStream resilientFos = new ResilientFileOutputStream(file, true, 4096);
        resilientFos.write("logback rolling over".getBytes(StandardCharsets.UTF_8));

        resilientFos.close();

        System.out.println("over");
    }
}
