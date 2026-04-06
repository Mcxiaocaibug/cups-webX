package server

import (
	"bytes"
	"io"
	"io/fs"
	"net/http"
	"path"
	"time"
)

// NewEmbeddedServer returns an http.Handler that serves embedded frontend files
// and falls back to index.html for SPA routes.
func NewEmbeddedServer(root fs.FS) http.Handler {
	sub := firstAvailableSubFS(root, "dist", "fallback", ".")
	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := path.Clean(r.URL.Path)
		if p == "/" {
			p = "index.html"
		} else {
			p = p[1:]
		}

		if _, err := fs.Stat(sub, p); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		f, err := sub.Open("index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()

		data, err := io.ReadAll(f)
		if err != nil {
			http.Error(w, "failed to read embedded index.html", http.StatusInternalServerError)
			return
		}

		modTime := time.Now()
		if fi, err := f.Stat(); err == nil {
			modTime = fi.ModTime()
		}

		http.ServeContent(w, r, "index.html", modTime, bytes.NewReader(data))
	})
}

func firstAvailableSubFS(root fs.FS, candidates ...string) fs.FS {
	for _, candidate := range candidates {
		if candidate == "." {
			if _, err := fs.Stat(root, "index.html"); err == nil {
				return root
			}
			continue
		}
		sub, err := fs.Sub(root, candidate)
		if err != nil {
			continue
		}
		if _, err := fs.Stat(sub, "index.html"); err == nil {
			return sub
		}
	}
	return root
}
