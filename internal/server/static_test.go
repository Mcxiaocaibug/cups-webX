package server

import (
	"io/fs"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestNewEmbeddedServerPrefersDistAssets(t *testing.T) {
	root := fstest.MapFS{
		"dist/index.html":      &fstest.MapFile{Data: []byte("dist-index")},
		"dist/assets/app.js":   &fstest.MapFile{Data: []byte("console.log('dist')")},
		"fallback/index.html":  &fstest.MapFile{Data: []byte("fallback-index")},
		"fallback/unused.html": &fstest.MapFile{Data: []byte("unused")},
	}

	handler := NewEmbeddedServer(fs.FS(root))
	req := httptest.NewRequest("GET", "/assets/app.js", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "console.log('dist')") {
		t.Fatalf("expected dist asset body, got %q", rr.Body.String())
	}
}

func TestNewEmbeddedServerFallsBackToIndexForSPARoutes(t *testing.T) {
	root := fstest.MapFS{
		"fallback/index.html": &fstest.MapFile{Data: []byte("fallback-index")},
	}

	handler := NewEmbeddedServer(fs.FS(root))
	req := httptest.NewRequest("GET", "/print", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if rr.Body.String() != "fallback-index" {
		t.Fatalf("expected fallback index body, got %q", rr.Body.String())
	}
}
