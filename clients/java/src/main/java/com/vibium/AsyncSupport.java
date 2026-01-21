package com.vibium;

import java.util.concurrent.Executor;
import java.util.concurrent.Executors;

final class AsyncSupport {
    private static final Executor EXECUTOR = Executors.newCachedThreadPool(r -> {
        Thread t = new Thread(r, "vibium-async");
        t.setDaemon(true);
        return t;
    });

    private AsyncSupport() {}

    static Executor executor() {
        return EXECUTOR;
    }
}

