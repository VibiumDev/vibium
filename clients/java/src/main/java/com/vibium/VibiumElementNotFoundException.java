package com.vibium;

public final class VibiumElementNotFoundException extends VibiumException {
    public final String selector;

    public VibiumElementNotFoundException(String selector) {
        super("Element not found: " + selector);
        this.selector = selector;
    }
}

