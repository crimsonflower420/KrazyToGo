// Command krazytogo is a hyper-minimal pure-Go file server.
// Same static binary for gokrazy, Docker/Podman (Krazy Kontainer), and Kubernetes.
package main

import (
	"errors"
	"flag"
	"fmt"
	"html"
	"io"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Defaults for local / container builds. Overridden in main_gokrazy.go.
var (
	defaultRoot   = "./data"
	defaultAddr   = ":8080"
	defaultBrowse = true
)

var errPathEscape = errors.New("path escapes root")

func main() {
	cfg := resolveConfig(os.Args[1:], os.Environ())
	if err := os.MkdirAll(cfg.root, 0o755); err != nil {
		log.Fatalf("create root %q: %v", cfg.root, err)
	}
	absRoot, err := filepath.Abs(cfg.root)
	if err != nil {
		log.Fatalf("resolve root %q: %v", cfg.root, err)
	}
	cfg.root = absRoot

	srv := &http.Server{
		Addr:              cfg.addr,
		Handler:           newHandler(cfg.root, cfg.browse),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("krazytogo listening on %s root=%s browse=%v", cfg.addr, cfg.root, cfg.browse)
	log.Fatal(srv.ListenAndServe())
}

type config struct {
	root   string
	addr   string
	browse bool
}

// resolveConfig applies defaults → env (KRAZY_*) → flags. Flags win.
// args should be os.Args[1:]; env is os.Environ()-style "KEY=value" pairs.
func resolveConfig(args []string, env []string) config {
	cfg := config{
		root:   defaultRoot,
		addr:   defaultAddr,
		browse: defaultBrowse,
	}
	envMap := envToMap(env)
	if v, ok := envMap["KRAZY_ROOT"]; ok && v != "" {
		cfg.root = v
	}
	if v, ok := envMap["KRAZY_ADDR"]; ok && v != "" {
		cfg.addr = v
	}
	if v, ok := envMap["KRAZY_BROWSE"]; ok && v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.browse = b
		}
	}

	fs := flag.NewFlagSet("krazytogo", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("root", cfg.root, "document root directory (env KRAZY_ROOT)")
	addr := fs.String("addr", cfg.addr, "listen address (env KRAZY_ADDR)")
	browse := fs.Bool("browse", cfg.browse, "allow directory listing (env KRAZY_BROWSE)")
	_ = fs.Parse(args)
	cfg.root = *root
	cfg.addr = *addr
	cfg.browse = *browse
	return cfg
}

func envToMap(env []string) map[string]string {
	m := make(map[string]string, len(env))
	for _, e := range env {
		k, v, ok := strings.Cut(e, "=")
		if ok {
			m[k] = v
		}
	}
	return m
}

// newHandler routes without http.ServeMux so ".." is not redirected before sanitization.
func newHandler(root string, browse bool) http.Handler {
	files := makeFileHandler(root, browse)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			handleHealthz(w, r)
			return
		}
		files(w, r)
	})
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = io.WriteString(w, "ok")
	}
}

func makeFileHandler(root string, browse bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		reqPath := r.URL.Path
		if reqPath == "" {
			reqPath = "/"
		}
		full, err := safeJoin(root, reqPath)
		if err != nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		info, err := os.Stat(full)
		if err != nil {
			if os.IsNotExist(err) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		if info.IsDir() {
			if !strings.HasSuffix(r.URL.Path, "/") {
				http.Redirect(w, r, path.Clean(r.URL.Path)+"/", http.StatusMovedPermanently)
				return
			}
			index := filepath.Join(full, "index.html")
			if fi, err := os.Stat(index); err == nil && !fi.IsDir() {
				serveFile(w, r, index, fi)
				return
			}
			if !browse {
				http.NotFound(w, r)
				return
			}
			if err := serveDirListing(w, r, full, r.URL.Path); err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
			return
		}

		serveFile(w, r, full, info)
	}
}

// safeJoin cleans reqPath and joins it under root.
// Rejects ".." segments and any result that escapes root.
func safeJoin(root, reqPath string) (string, error) {
	if root == "" {
		return "", errPathEscape
	}
	if strings.ContainsRune(reqPath, 0) {
		return "", errPathEscape
	}

	// Normalize to slash path; reject backslash as segment trick.
	p := strings.ReplaceAll(reqPath, `\`, "/")
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}

	// Reject ".." and empty segments that survive unclean paths before Clean.
	for _, seg := range strings.Split(p, "/") {
		if seg == ".." {
			return "", errPathEscape
		}
	}

	cleaned := path.Clean(p)
	if cleaned != "/" && !strings.HasPrefix(cleaned, "/") {
		return "", errPathEscape
	}
	rel := strings.TrimPrefix(cleaned, "/")
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return "", errPathEscape
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	absRoot = filepath.Clean(absRoot)

	full := absRoot
	if rel != "" {
		full = filepath.Join(absRoot, filepath.FromSlash(rel))
	}
	full = filepath.Clean(full)

	sep := string(os.PathSeparator)
	if full != absRoot && !strings.HasPrefix(full, absRoot+sep) {
		return "", errPathEscape
	}
	return full, nil
}

func serveFile(w http.ResponseWriter, r *http.Request, full string, info fs.FileInfo) {
	f, err := os.Open(full)
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	if ctype := mime.TypeByExtension(filepath.Ext(full)); ctype != "" {
		w.Header().Set("Content-Type", ctype)
	}
	http.ServeContent(w, r, info.Name(), info.ModTime(), f)
}

func serveDirListing(w http.ResponseWriter, r *http.Request, dir, urlPath string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html><head><meta charset=\"utf-8\"><title>")
	b.WriteString(html.EscapeString(urlPath))
	b.WriteString("</title></head><body>\n<h1>")
	b.WriteString(html.EscapeString(urlPath))
	b.WriteString("</h1>\n<ul>\n")
	if urlPath != "/" {
		b.WriteString(`<li><a href="../">../</a></li>` + "\n")
	}
	for _, e := range entries {
		name := e.Name()
		href := path.Join(urlPath, name)
		display := name
		if e.IsDir() {
			href += "/"
			display += "/"
		}
		fmt.Fprintf(&b, `<li><a href="%s">%s</a></li>`+"\n",
			html.EscapeString(href), html.EscapeString(display))
	}
	b.WriteString("</ul>\n</body></html>\n")
	if r.Method == http.MethodHead {
		w.Header().Set("Content-Length", strconv.Itoa(b.Len()))
		w.WriteHeader(http.StatusOK)
		return nil
	}
	_, err = io.WriteString(w, b.String())
	return err
}
