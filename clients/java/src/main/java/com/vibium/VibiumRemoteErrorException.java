package com.vibium;

public final class VibiumRemoteErrorException extends VibiumException {
    public final String error;
    public final String remoteMessage;

    public VibiumRemoteErrorException(String error, String remoteMessage) {
        super(error + ": " + remoteMessage);
        this.error = error;
        this.remoteMessage = remoteMessage;
    }
}

