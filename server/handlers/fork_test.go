package handlers

import (
	"testing"

	"main/models"
)

func TestCanWriteForkDirectly(t *testing.T) {
	cases := []struct {
		role      string
		isTrusted bool
		want      bool
	}{
		{"dnr", false, true},
		{"dnr", true, true},
		{"admin", false, true},
		{"agent", false, false},
		{"agent", true, false}, // trust does not bypass verification
		{"listing", false, false},
	}
	for _, tc := range cases {
		u := &models.User{Role: tc.role, IsTrusted: tc.isTrusted}
		if got := canWriteForkDirectly(u); got != tc.want {
			t.Errorf("role=%q isTrusted=%v: got %v want %v", tc.role, tc.isTrusted, got, tc.want)
		}
	}
	if canWriteForkDirectly(nil) {
		t.Fatal("nil user should not write fork data directly")
	}
}
