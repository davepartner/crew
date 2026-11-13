package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"io/fs"
	"net/http"
	"path"

	"github.com/davepartner/crew/internal/payauth"
)

const (
	sessionCookie = "crew_session"
	csrfCookie    = "crew_csrf"
)

func randomToken(bytes int) string {
	b := make([]byte, bytes)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString(b)
}

func isUnauthorized(err error) bool {
	return payauth.IsUnauthorized(err)
}

// spaHandler serves the built React app and falls back to index.html so deep
// links work without a server-side router.
type spaHandler struct {
	fsys fs.FS
}

func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name := path.Clean("/" + r.URL.Path)
	name = name[1:]

	if name == "" {
		name = "index.html"
	}

	if info, err := fs.Stat(h.fsys, name); err == nil && !info.IsDir() {
		http.ServeFileFS(w, r, h.fsys, name)
		return
	}

	data, err := fs.ReadFile(h.fsys, "index.html")
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(placeholderPage))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

const placeholderPage = `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Crew</title>
<style>body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;background:#f6f5f2;color:#16181d;display:grid;place-items:center;min-height:100vh;margin:0}
main{max-width:520px;padding:32px;background:#fff;border-radius:16px;border:1px solid #e6e4df}
code{background:#f1efe9;padding:2px 6px;border-radius:6px}</style></head>
<body><main><h1>Crew is running</h1>
<p>The API is up, but the web app has not been built yet.</p>
<p>Run <code>pnpm install &amp;&amp; pnpm build</code> in <code>web/</code>, then copy <code>web/dist</code> to <code>internal/webui/dist</code> (or run <code>make web</code>).</p>
</main></body></html>`
