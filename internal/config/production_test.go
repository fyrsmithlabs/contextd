package config

import (
	"os"
	"testing"
)

func TestProductionConfig_Defaults(t *testing.T) {
	os.Unsetenv("CONTEXTD_PRODUCTION_MODE")
	os.Unsetenv("CONTEXTD_LOCAL_MODE")

	cfg := Load()

	if cfg.Production.Enabled {
		t.Error("Production.Enabled = true, want false (disabled by default)")
	}
}

func TestProductionConfig_EnabledViaEnv(t *testing.T) {
	defer os.Unsetenv("CONTEXTD_PRODUCTION_MODE")
	os.Setenv("CONTEXTD_PRODUCTION_MODE", "1")

	cfg := Load()

	if !cfg.Production.Enabled {
		t.Error("Production.Enabled = false, want true when CONTEXTD_PRODUCTION_MODE=1")
	}
}
