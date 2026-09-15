package security

import (
	"testing"
	"time"
)

func TestBasicConfig_IsValidUsingBcrypt(t *testing.T) {
	basicConfig := &BasicConfig{
		Username:                        "admin",
		PasswordBcryptHashBase64Encoded: "JDJhJDA4JDFoRnpPY1hnaFl1OC9ISlFsa21VS09wOGlPU1ZOTDlHZG1qeTFvb3dIckRBUnlHUmNIRWlT",
	}
	if !basicConfig.isValid() {
		t.Error("basicConfig should've been valid")
	}
}

func TestBasicConfig_IsValidWhenPasswordIsInvalidUsingBcrypt(t *testing.T) {
	basicConfig := &BasicConfig{
		Username:                        "admin",
		PasswordBcryptHashBase64Encoded: "",
	}
	if basicConfig.isValid() {
		t.Error("basicConfig shouldn't have been valid")
	}
}

func TestBasicConfig_ValidateAndSetDefaultsWithSessionTTL(t *testing.T) {
	scenarios := []struct {
		sessionTTL  time.Duration
		expectValid bool
		expectedTTL time.Duration
	}{
		{sessionTTL: 0, expectValid: true, expectedTTL: DefaultBasicSessionTTL},
		{sessionTTL: MinimumBasicSessionTTL, expectValid: true, expectedTTL: MinimumBasicSessionTTL},
		{sessionTTL: MaximumBasicSessionTTL, expectValid: true, expectedTTL: MaximumBasicSessionTTL},
		{sessionTTL: MinimumBasicSessionTTL - time.Second},
		{sessionTTL: MaximumBasicSessionTTL + time.Second},
		{sessionTTL: -time.Hour},
	}
	for _, scenario := range scenarios {
		basicConfig := &BasicConfig{Username: "admin", PasswordBcryptHashBase64Encoded: "hash", SessionTTL: scenario.sessionTTL}
		if valid := basicConfig.validateAndSetDefaults(); valid != scenario.expectValid {
			t.Errorf("session-ttl %s: expected valid=%v, got %v", scenario.sessionTTL, scenario.expectValid, valid)
		} else if valid && basicConfig.SessionTTL != scenario.expectedTTL {
			t.Errorf("session-ttl %s: expected %s, got %s", scenario.sessionTTL, scenario.expectedTTL, basicConfig.SessionTTL)
		}
	}
}
