package main

import (
	"strings"
	"testing"
)

func TestSetupBannerPlainHasMarkWordAndTagline(t *testing.T) {
	s := setupBanner(false)
	for _, need := range []string{
		"vibium",
		"setup",
		"Agents make it. We check it.",
		"██    █████████████████    ██",
		"██████████        ███████████",
		"████████████",
	} {
		if !strings.Contains(s, need) {
			t.Fatalf("banner missing %q:\n%s", need, s)
		}
	}
	if strings.Contains(s, "\x1b[") {
		t.Fatal("plain banner included ANSI")
	}
	t.Logf("\n%s", s)
}

func TestSetupBannerColorUsesAccent(t *testing.T) {
	s := setupBanner(true)
	if !strings.Contains(s, "\x1b[38;2;255;103;0m") {
		t.Fatal("colored banner missing #ff6700")
	}
	if !strings.Contains(s, "vibium") {
		t.Fatal("colored banner dropped wordmark")
	}
	if !strings.HasSuffix(s, ansiReset) && !strings.Contains(s, ansiReset) {
		t.Fatal("colored banner missing reset")
	}
}

func TestMaybePaintRespectsOff(t *testing.T) {
	if maybePaint(false, brandAccent, "x") != "x" {
		t.Fatal("paint leaked through when disabled")
	}
}
