package com.vibium;

public final class ElementInfo {
    public final String tag;
    public final String text;
    public final BoundingBox box;

    public ElementInfo(String tag, String text, BoundingBox box) {
        this.tag = tag;
        this.text = text;
        this.box = box;
    }
}

