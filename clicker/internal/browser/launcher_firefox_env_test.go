package browser

import (
	"slices"
	"testing"
)

// Firefox 159 requires MOZ_REMOTE_ALLOW_SYSTEM_ACCESS=1 for privileged BiDi
// operations like setContent; the launcher must set it on every platform
// (#673). The parent environment is preserved alongside it.
func TestFirefoxLaunchEnvSetsSystemAccess(t *testing.T) {
	t.Setenv("VIBIUM_FF_ENV_PROBE", "keep-me")
	env := firefoxLaunchEnv("")

	if !slices.Contains(env, "MOZ_REMOTE_ALLOW_SYSTEM_ACCESS=1") {
		t.Errorf("launch env missing MOZ_REMOTE_ALLOW_SYSTEM_ACCESS=1: %v", env)
	}
	if !slices.Contains(env, "VIBIUM_FF_ENV_PROBE=keep-me") {
		t.Error("launch env dropped the inherited parent environment")
	}
}

// On Linux the private XDG config dir is also set, without losing the system
// access variable.
func TestFirefoxLaunchEnvKeepsXDGOverride(t *testing.T) {
	env := firefoxLaunchEnv("/tmp/vibium-xdg")

	if !slices.Contains(env, "XDG_CONFIG_HOME=/tmp/vibium-xdg") {
		t.Errorf("launch env missing the XDG override: %v", env)
	}
	if !slices.Contains(env, "MOZ_REMOTE_ALLOW_SYSTEM_ACCESS=1") {
		t.Error("the XDG override must not drop MOZ_REMOTE_ALLOW_SYSTEM_ACCESS")
	}
}
