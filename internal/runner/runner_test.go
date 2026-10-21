package runner

import "testing"

func TestRunnerFailsClosedUntilPlatformValidation(t *testing.T) {
	profile := Describe()
	if profile.Status != "unverified" {
		t.Fatalf("status = %q", profile.Status)
	}
	if profile.OS == "" || profile.Reason == "" {
		t.Fatal("missing platform or concrete reason")
	}
	if profile.Fingerprint != "" {
		t.Fatal("unmeasured fingerprint must not be claimed")
	}
	if len(profile.Capabilities) != 6 {
		t.Fatalf("reported %d capabilities, want all six", len(profile.Capabilities))
	}
	for name, enabled := range profile.Capabilities {
		if enabled {
			t.Errorf("capability %q must remain disabled", name)
		}
	}
	if err := CheckReady(); err == nil {
		t.Fatal("unverified runner reported ready")
	}
}
