package qnamefmt

import "testing"

func TestClark(t *testing.T) {
	tests := []struct {
		space, local, want string
	}{
		{"", "a", "a"},
		{"u", "a", "{u}a"},
		{"", "", ""},
		{"u", "", "{u}"},
	}
	for _, tt := range tests {
		if got := Clark(tt.space, tt.local); got != tt.want {
			t.Errorf("Clark(%q, %q) = %q, want %q", tt.space, tt.local, got, tt.want)
		}
	}
}

func TestDisplay(t *testing.T) {
	tests := []struct {
		space, local, want string
	}{
		{XMLSchemaNS, "schema", "xs:schema"},
		{XMLSchemaInstanceNS, "type", "xsi:type"},
		{"", "a", "a"},
		{"u", "a", "{u}a"},
		{"", "", ""},
		{XMLSchemaNS, "", "xs:"},
	}
	for _, tt := range tests {
		if got := Display(tt.space, tt.local); got != tt.want {
			t.Errorf("Display(%q, %q) = %q, want %q", tt.space, tt.local, got, tt.want)
		}
	}
}
