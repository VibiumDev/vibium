package main

import (
	"reflect"
	"testing"
)

func TestRequestedLaunchOptionsOnlyIncludesExplicitValues(t *testing.T) {
	oldHeadless, oldEngine, oldChannel := headless, engineName, engineChannel
	oldHeadlessSet, oldEngineSet, oldChannelSet := headlessSet, engineSet, channelSet
	t.Cleanup(func() {
		headless, engineName, engineChannel = oldHeadless, oldEngine, oldChannel
		headlessSet, engineSet, channelSet = oldHeadlessSet, oldEngineSet, oldChannelSet
	})

	tests := []struct {
		name                               string
		headless                           bool
		engine, channel                    string
		headlessSet, engineSet, channelSet bool
		want                               map[string]interface{}
	}{
		{
			name:      "explicit engine and channel",
			engine:    "firefox",
			channel:   "beta",
			engineSet: true, channelSet: true,
			want: map[string]interface{}{"engine": "firefox", "channel": "beta"},
		},
		{
			name:      "explicit engine without channel",
			engine:    "firefox",
			engineSet: true,
			want:      map[string]interface{}{"engine": "firefox"},
		},
		{
			name:        "explicit mode only",
			headless:    true,
			headlessSet: true,
			want:        map[string]interface{}{"headless": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headless = tt.headless
			engineName = tt.engine
			engineChannel = tt.channel
			headlessSet, engineSet, channelSet = tt.headlessSet, tt.engineSet, tt.channelSet
			if got := requestedLaunchOptions(); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("requestedLaunchOptions() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
