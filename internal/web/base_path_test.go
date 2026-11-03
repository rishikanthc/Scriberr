package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithBasePath(t *testing.T) {
	var gotPath, gotPrefix string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotPrefix = r.URL.Path, r.Header.Get("X-Forwarded-Prefix")
	})
	h := WithBasePath("/scriberr", next)

	cases := []struct{ in, wantPath, wantPrefix string }{
		{"/scriberr/api/v1/x", "/api/v1/x", "/scriberr"},
		{"/scriberr/", "/", "/scriberr"},
		{"/health", "/health", ""},
		{"/scriberrx/a", "/scriberrx/a", ""},
	}
	for _, c := range cases {
		gotPath, gotPrefix = "", ""
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", c.in, nil))
		if gotPath != c.wantPath || gotPrefix != c.wantPrefix {
			t.Errorf("%s: got (%q, %q), want (%q, %q)", c.in, gotPath, gotPrefix, c.wantPath, c.wantPrefix)
		}
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/scriberr?a=1", nil))
	if rec.Code != http.StatusMovedPermanently || rec.Header().Get("Location") != "/scriberr/?a=1" {
		t.Errorf("bare prefix: got %d %q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestInjectBaseHref(t *testing.T) {
	got := string(injectBaseHref([]byte("<html><head><title>x</title></head>"), "/scriberr"))
	want := `<html><head><base href="/scriberr/"><title>x</title></head>`
	if got != want {
		t.Errorf("got %q", got)
	}
	if got := string(injectBaseHref([]byte("<head>"), "")); got != `<head><base href="/">` {
		t.Errorf("root: got %q", got)
	}
}
