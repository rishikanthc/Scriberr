package config

import "testing"

func TestNormalizeBasePath(t *testing.T) {
	for in, want := range map[string]string{"": "", "/": "", "scriberr": "/scriberr", "/scriberr/": "/scriberr", " /a/b/ ": "/a/b"} {
		if got := NormalizeBasePath(in); got != want {
			t.Errorf("NormalizeBasePath(%q) = %q, want %q", in, got, want)
		}
	}
}
