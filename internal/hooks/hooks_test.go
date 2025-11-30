package hooks

import (
	"context"
	"testing"
)

func TestNewHookManager(t *testing.T) {
	hm := NewHookManager(&Config{
		AutoCheckpointOnClear: true,
		AutoResumeOnStart:     true,
		CheckpointThreshold:   70,
		VerifyBeforeClear:     true,
	})
	if hm == nil {
		t.Fatal("NewHookManager returned nil")
	}
}

func TestExecuteHook_SessionStart(t *testing.T) {
	ctx := context.Background()
	hm := NewHookManager(&Config{AutoResumeOnStart: true})
	data := map[string]interface{}{"project_path": "/test/project"}
	err := hm.Execute(ctx, HookSessionStart, data)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestExecuteHook_BeforeClear(t *testing.T) {
	ctx := context.Background()
	hm := NewHookManager(&Config{AutoCheckpointOnClear: true})
	data := map[string]interface{}{"project_path": "/test/project"}
	err := hm.Execute(ctx, HookBeforeClear, data)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{"valid config", &Config{CheckpointThreshold: 70}, false},
		{"invalid threshold too low", &Config{CheckpointThreshold: 0}, true},
		{"invalid threshold too high", &Config{CheckpointThreshold: 100}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRegisterHandler(t *testing.T) {
	ctx := context.Background()
	hm := NewHookManager(&Config{})
	called := false
	handler := func(ctx context.Context, data map[string]interface{}) error {
		called = true
		return nil
	}
	hm.RegisterHandler(HookSessionStart, handler)
	err := hm.Execute(ctx, HookSessionStart, nil)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !called {
		t.Error("Handler was not called")
	}
}

func TestExecuteHook_NoHandler(t *testing.T) {
	ctx := context.Background()
	hm := NewHookManager(&Config{})
	err := hm.Execute(ctx, HookSessionStart, nil)
	if err != nil {
		t.Fatalf("Execute failed with no handler: %v", err)
	}
}
