package types

import "testing"

func TestParseProvider(t *testing.T) {
	tests := []struct {
		input   string
		want    Provider
		wantErr bool
	}{
		{"openrouter", ProviderOpenRouter, false},
		{"OPENROUTER", ProviderOpenRouter, false},
		{" openai ", ProviderOpenAI, false},
		{"Anthropic", ProviderAnthropic, false},
		{"google", ProviderGoogle, false},
		{"unknown_provider", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseProvider(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseProvider(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("ParseProvider(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestProvider_IsValid(t *testing.T) {
	for _, p := range AllProviders {
		if !p.IsValid() {
			t.Errorf("expected %q to be valid", p)
		}
	}

	if Provider("invalid").IsValid() {
		t.Errorf("expected invalid provider to be false")
	}
}

