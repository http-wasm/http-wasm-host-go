package handler

import (
	"testing"
)

// TestFeatures_ZeroIsInvalid reminds maintainers that a bitset cannot use zero as a flag!
// This is why we start iota with 1.
func TestFeatures_ZeroIsInvalid(t *testing.T) {
	f := Features(0)
	f = f.WithEnabled(0)

	if f.IsEnabled(0) {
		t.Errorf("expected zero to not be enabled")
	}
}

// TestFeatures tests the bitset works as expected
func TestFeatures(t *testing.T) {
	tests := []struct {
		name    string
		feature Features
	}{
		{
			name:    "one is the smallest flag",
			feature: 1,
		},
		{
			name:    "31 is the largest feature flag", // because uint32
			feature: 1 << 31,
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			f := Features(0)

			// Defaults to false
			if f.IsEnabled(tc.feature) {
				t.Errorf("expected %v to not be enabled", tc.feature)
			}

			// Set true makes it true
			f = f.WithEnabled(tc.feature)
			if !f.IsEnabled(tc.feature) {
				t.Errorf("expected %v to be enabled", tc.feature)
			}
		})
	}
}

func TestFeatures_String(t *testing.T) {
	tests := []struct {
		name     string
		feature  Features
		expected string
	}{
		{name: "none", feature: 0, expected: ""},
		{name: "buffer_request", feature: FeatureBufferRequest, expected: "buffer_request"},
		{name: "buffer_response", feature: FeatureBufferResponse, expected: "buffer_response"},
		{name: "trailers", feature: FeatureTrailers, expected: "trailers"},
		{name: "all", feature: FeatureBufferRequest | FeatureBufferResponse | FeatureTrailers, expected: "buffer_request|buffer_response|trailers"},
		{name: "undefined", feature: 1 << 31, expected: ""},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			if want, have := tc.expected, tc.feature.String(); want != have {
				t.Errorf("unexpected string, want: %q, have: %q", want, have)
			}
		})
	}
}
