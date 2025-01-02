package config

import (
	"strings"
	"testing"
)

func TestCreateColorCallback(t *testing.T) {
	script := `
function get_color(v) {
  if (v < 100) {
    return "#FFFFFF";
  } else if (v < 250) {
    return "#FFA500";
  } else if (v < 500) {
    return "#FFFF00";
  } else if (v < 750) {
    return "#008000";
  } else {
    return "#FFC0CB";
  }
}
`
	cb, err := createColorCallback(script)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tests := []struct {
		value float64
		want  string
	}{
		{50, "#FFFFFF"},
		{200, "#FFA500"},
		{300, "#FFFF00"},
		{600, "#008000"},
		{900, "#FFC0CB"},
	}

	for _, tc := range tests {
		got, err := cb(tc.value)
		if err != nil {
			t.Errorf("got error for value %f: %v", tc.value, err)
		} else if got != tc.want {
			t.Errorf("for value %f: got %s, want %s", tc.value, got, tc.want)
		}
	}
}

func TestCreateColorCallbackErrors(t *testing.T) {
	tests := []struct {
		name    string
		script  string
		wantErr string
	}{
		{
			name:    "Empty script",
			script:  "",
			wantErr: "Empty script",
		},
		{
			name:    "Syntax error",
			script:  `function get_colr(v) { return "#FFF";`,
			wantErr: "Error running color script",
		},
		{
			name:    "Missing get_color function",
			script:  `function another_func() { return "#FFF"; }`,
			wantErr: "Error getting get_color function from script",
		},
		{
			name:    "Panic in script",
			script:  `function get_color(v) {panic()};`,
			wantErr: "Error running color script",
		},
		{
			name:    "Invalid color code returned",
			script:  `function get_color(v) { return "#FFFF"; }`,
			wantErr: "Invalid hex color code",
		},
		{
			name: "Color callback runtime error",
			script: `function get_color(v) {
               if (v < 0) throw "negative value not allowed";
               return "#FFFFFF";
            }`,
			wantErr: "negative value not allowed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cb, err := createColorCallback(tc.script)
			if err == nil {
				// Attempt a runtime error if callback creation succeeded
				_, runErr := cb(-1)
				if runErr == nil {
					t.Errorf("expected error, got none")
				} else if !strings.Contains(runErr.Error(), tc.wantErr) {
					t.Errorf("expected error to contain %q, got %q", tc.wantErr, runErr.Error())
				}
			} else if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("expected error to contain %q, got %q", tc.wantErr, err.Error())
			}
		})
	}
}
