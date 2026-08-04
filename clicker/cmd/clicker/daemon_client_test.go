package main

import (
	"reflect"
	"testing"
)

func TestRequestedLaunchOptionsOnlyIncludesExplicitValues(t *testing.T) {
	oldHeadless, oldEngine, oldChannel := headless, engineName, firefoxChannel
	oldHeadlessSet, oldEngineSet, oldChannelSet := headlessSet, engineSet, channelSet
	t.Cleanup(func() {
		headless, engineName, firefoxChannel = oldHeadless, oldEngine, oldChannel
		headlessSet, engineSet, channelSet = oldHeadlessSet, oldEngineSet, oldChannelSet
	})

	headless, engineName, firefoxChannel = true, "firefox", "beta"
	headlessSet, engineSet, channelSet = false, true, true
	want := map[string]interface{}{"engine": "firefox", "channel": "beta"}
	if got := requestedLaunchOptions(); !reflect.DeepEqual(got, want) {
		t.Fatalf("requestedLaunchOptions() = %#v, want %#v", got, want)
	}
}

func TestRequestedFirefoxDefaultsToReleaseChannel(t *testing.T) {
	oldEngine, oldChannel := engineName, firefoxChannel
	oldEngineSet, oldChannelSet := engineSet, channelSet
	t.Cleanup(func() {
		engineName, firefoxChannel = oldEngine, oldChannel
		engineSet, channelSet = oldEngineSet, oldChannelSet
	})
	t.Setenv("VIBIUM_FIREFOX_CHANNEL", "")

	engineName, firefoxChannel = "firefox", ""
	engineSet, channelSet = true, false
	want := map[string]interface{}{"engine": "firefox", "channel": "release"}
	if got := requestedLaunchOptions(); !reflect.DeepEqual(got, want) {
		t.Fatalf("requestedLaunchOptions() = %#v, want %#v", got, want)
	}
}
