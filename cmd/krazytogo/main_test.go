package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSafeJoin(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "ok", "nested"), 0o755); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name    string
		reqPath string
		wantErr bool
		wantRel string // relative to root when ok; "." means root itself
	}{
		{name: "root", reqPath: "/", wantErr: false, wantRel: "."},
		{name: "simple", reqPath: "/ok/nested", wantErr: false, wantRel: "ok/nested"},
		{name: "dotdot", reqPath: "/../etc/passwd", wantErr: true},
		{name: "encoded-ish clean", reqPath: "/ok/../../etc/passwd", wantErr: true},
		{name: "nested traversal", reqPath: "/ok/nested/../../../etc", wantErr: true},
		{name: "dot segments ok", reqPath: "/ok/./nested", wantErr: false, wantRel: "ok/nested"},
		{name: "empty", reqPath: "", wantErr: false, wantRel: "."},
		{name: "no leading slash", reqPath: "ok/nested", wantErr: false, wantRel: "ok/nested"},
		{name: "double slash", reqPath: "//ok//nested", wantErr: false, wantRel: "ok/nested"},
		{name: "null byte", reqPath: "/ok/\x00/evil", wantErr: true},
		{name: "backslash mix", reqPath: `/ok\..\secret`, wantErr: true},
		{name: "relative dotdot", reqPath: "../secret", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := safeJoin(root, tc.reqPath)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got path %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			want := root
			if tc.wantRel != "." {
				want = filepath.Join(root, filepath.FromSlash(tc.wantRel))
			}
			if got != want {
				t.Fatalf("got %q want %q", got, want)
			}
		})
	}
}

func TestSafeJoinRejectsEscapeFromRoot(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "root")
	outside := filepath.Join(base, "secret.txt")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outside, []byte("nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := safeJoin(root, "../secret.txt")
	if err == nil {
		t.Fatal("expected escape rejection")
	}
}

func TestHealthz(t *testing.T) {
	h := newHandler(t.TempDir(), true)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
	if body := rr.Body.String(); body != "ok" {
		t.Fatalf("body %q", body)
	}
}

func TestBrowseOnOff(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "subdir", "a.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("browse on", func(t *testing.T) {
		h := newHandler(root, true)
		req := httptest.NewRequest(http.MethodGet, "/subdir/", nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status %d body %q", rr.Code, rr.Body.String())
		}
		if !strings.Contains(rr.Body.String(), "a.txt") {
			t.Fatalf("listing missing a.txt: %s", rr.Body.String())
		}
	})

	t.Run("browse off", func(t *testing.T) {
		h := newHandler(root, false)
		req := httptest.NewRequest(http.MethodGet, "/subdir/", nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("want 404, got %d", rr.Code)
		}
	})

	t.Run("file still served when browse off", func(t *testing.T) {
		h := newHandler(root, false)
		req := httptest.NewRequest(http.MethodGet, "/subdir/a.txt", nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status %d", rr.Code)
		}
		if rr.Body.String() != "hi" {
			t.Fatalf("body %q", rr.Body.String())
		}
	})
}

func TestMIMEAndRange(t *testing.T) {
	root := t.TempDir()
	payload := []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	path := filepath.Join(root, "sample.txt")
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "data.json"), []byte(`{"a":1}`), 0o644); err != nil {
		t.Fatal(err)
	}

	h := newHandler(root, true)

	t.Run("mime text", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/sample.txt", nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status %d", rr.Code)
		}
		ct := rr.Header().Get("Content-Type")
		if !strings.HasPrefix(ct, "text/plain") {
			t.Fatalf("Content-Type %q", ct)
		}
	})

	t.Run("mime json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/data.json", nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status %d", rr.Code)
		}
		ct := rr.Header().Get("Content-Type")
		if !strings.Contains(ct, "json") {
			t.Fatalf("Content-Type %q", ct)
		}
	})

	t.Run("range", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/sample.txt", nil)
		req.Header.Set("Range", "bytes=0-4")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusPartialContent {
			t.Fatalf("status %d", rr.Code)
		}
		if body := rr.Body.String(); body != "ABCDE" {
			t.Fatalf("body %q", body)
		}
		cr := rr.Header().Get("Content-Range")
		if !strings.HasPrefix(cr, "bytes 0-4/") {
			t.Fatalf("Content-Range %q", cr)
		}
	})
}

func TestPathTraversalHTTP(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "www")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "secret.txt"), []byte("TOPSECRET"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "public.txt"), []byte("public"), 0o644); err != nil {
		t.Fatal(err)
	}

	h := newHandler(root, true)
	attacks := []string{
		"/../secret.txt",
		"/../../secret.txt",
		"/./../secret.txt",
		"/ok/../../secret.txt",
	}
	for _, p := range attacks {
		t.Run(p, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, p, nil)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			body, _ := io.ReadAll(rr.Body)
			if strings.Contains(string(body), "TOPSECRET") {
				t.Fatalf("leaked secret via %s status=%d body=%q", p, rr.Code, body)
			}
			if rr.Code != http.StatusForbidden {
				t.Fatalf("want 403, got %d body=%q", rr.Code, body)
			}
		})
	}

	// Sanity: legitimate file still works.
	req := httptest.NewRequest(http.MethodGet, "/public.txt", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK || rr.Body.String() != "public" {
		t.Fatalf("public.txt: status=%d body=%q", rr.Code, rr.Body.String())
	}
}

func TestResolveConfigPrecedence(t *testing.T) {
	origRoot, origAddr, origBrowse := defaultRoot, defaultAddr, defaultBrowse
	t.Cleanup(func() {
		defaultRoot, defaultAddr, defaultBrowse = origRoot, origAddr, origBrowse
	})
	defaultRoot = "./data"
	defaultAddr = ":8080"
	defaultBrowse = true

	cases := []struct {
		name       string
		args       []string
		env        []string
		wantRoot   string
		wantAddr   string
		wantBrowse bool
	}{
		{
			name:       "defaults",
			args:       nil,
			env:        nil,
			wantRoot:   "./data",
			wantAddr:   ":8080",
			wantBrowse: true,
		},
		{
			name:       "env only",
			args:       nil,
			env:        []string{"KRAZY_ROOT=/data", "KRAZY_ADDR=:9090", "KRAZY_BROWSE=false"},
			wantRoot:   "/data",
			wantAddr:   ":9090",
			wantBrowse: false,
		},
		{
			name:       "flags override env",
			args:       []string{"-root", "/flag-root", "-addr", ":7070", "-browse=true"},
			env:        []string{"KRAZY_ROOT=/data", "KRAZY_ADDR=:9090", "KRAZY_BROWSE=false"},
			wantRoot:   "/flag-root",
			wantAddr:   ":7070",
			wantBrowse: true,
		},
		{
			name:       "partial env",
			args:       nil,
			env:        []string{"KRAZY_ADDR=:1111"},
			wantRoot:   "./data",
			wantAddr:   ":1111",
			wantBrowse: true,
		},
		{
			name:       "browse env true string",
			args:       nil,
			env:        []string{"KRAZY_BROWSE=1"},
			wantRoot:   "./data",
			wantAddr:   ":8080",
			wantBrowse: true,
		},
		{
			name:       "flag browse false",
			args:       []string{"-browse=false"},
			env:        nil,
			wantRoot:   "./data",
			wantAddr:   ":8080",
			wantBrowse: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := resolveConfig(tc.args, tc.env)
			if cfg.root != tc.wantRoot {
				t.Errorf("root got %q want %q", cfg.root, tc.wantRoot)
			}
			if cfg.addr != tc.wantAddr {
				t.Errorf("addr got %q want %q", cfg.addr, tc.wantAddr)
			}
			if cfg.browse != tc.wantBrowse {
				t.Errorf("browse got %v want %v", cfg.browse, tc.wantBrowse)
			}
		})
	}
}

func TestIndexHTMLPreferred(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("<html>idx</html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	h := newHandler(root, true)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "idx") {
		t.Fatalf("body %q", rr.Body.String())
	}
}
