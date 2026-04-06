package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveStoredPrintPathPrefersConvertedPDF(t *testing.T) {
	baseDir := t.TempDir()
	storedRel := "20260405/test.docx"
	originalAbs := filepath.Join(baseDir, filepath.FromSlash(storedRel))
	if err := os.MkdirAll(filepath.Dir(originalAbs), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(originalAbs, []byte("original"), 0644); err != nil {
		t.Fatalf("WriteFile original: %v", err)
	}

	convertedRel := convertedRelPath(storedRel)
	convertedAbs := filepath.Join(baseDir, filepath.FromSlash(convertedRel))
	if err := os.WriteFile(convertedAbs, []byte("%PDF-1.7"), 0644); err != nil {
		t.Fatalf("WriteFile converted: %v", err)
	}

	gotRel, gotAbs, err := resolveStoredPrintPath(storedRel, baseDir)
	if err != nil {
		t.Fatalf("resolveStoredPrintPath: %v", err)
	}
	if gotRel != convertedRel || gotAbs != convertedAbs {
		t.Fatalf("got (%q, %q), want (%q, %q)", gotRel, gotAbs, convertedRel, convertedAbs)
	}
}

func TestResolveStoredPrintPathFallsBackToOriginal(t *testing.T) {
	baseDir := t.TempDir()
	storedRel := "20260405/test.pdf"
	originalAbs := filepath.Join(baseDir, filepath.FromSlash(storedRel))
	if err := os.MkdirAll(filepath.Dir(originalAbs), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(originalAbs, []byte("%PDF-1.7"), 0644); err != nil {
		t.Fatalf("WriteFile original: %v", err)
	}

	gotRel, gotAbs, err := resolveStoredPrintPath(storedRel, baseDir)
	if err != nil {
		t.Fatalf("resolveStoredPrintPath: %v", err)
	}
	if gotRel != storedRel || gotAbs != originalAbs {
		t.Fatalf("got (%q, %q), want (%q, %q)", gotRel, gotAbs, storedRel, originalAbs)
	}
}
