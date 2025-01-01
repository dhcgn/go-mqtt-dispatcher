package dispatcher

import (
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
