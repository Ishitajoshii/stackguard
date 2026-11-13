package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/Ishitajoshii/stackguard/internal/model"
	"github.com/Ishitajoshii/stackguard/internal/secrets"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

// HTTP forwards POST/GET/DELETE and session/version headers to one fixed
// upstream. ReverseProxy streams SSE immediately and does not follow redirects.
func HTTP(upstream string, o *Observer) (http.Handler, error) {
	u, e := url.Parse(upstream)
	if e != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" {
		return nil, errors.New("upstream must be an HTTP(S) URL without credentials or query")
	}
	if u.Scheme == "http" && !Loopback(u.Hostname()) {
		return nil, errors.New("plain HTTP upstream must be loopback")
	}
	p := httputil.NewSingleHostReverseProxy(u)
	p.FlushInterval = -1
	p.ErrorHandler = func(w http.ResponseWriter, r *http.Request, e error) {
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
	}
	p.ModifyResponse = func(r *http.Response) error {
		if r.Request.Method != "POST" {
			return nil
		}
		v, _ := r.Request.Context().Value(httpMetaKey{}).(httpMeta)
		if v.session == "" {
			return nil
		}
		r.Body = &observedBody{ReadCloser: r.Body, ctx: r.Request.Context(), observer: o, session: v.session, sse: strings.Contains(r.Header.Get("Content-Type"), "text/event-stream")}
		return nil
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			v, e := url.Parse(origin)
			if e != nil || !strings.EqualFold(v.Host, r.Host) {
				http.Error(w, "origin rejected", 403)
				return
			}
		}
		if r.Method != "GET" && r.Method != "POST" && r.Method != "DELETE" {
			w.Header().Set("Allow", "GET, POST, DELETE")
			http.Error(w, "method not allowed", 405)
			return
		}
		if r.Method == "POST" {
			b, e := io.ReadAll(io.LimitReader(r.Body, (1<<20)+1))
			r.Body.Close()
			if e != nil {
				http.Error(w, "request read failed", 400)
				return
			}
			if len(b) > 1<<20 {
				http.Error(w, "MCP frame exceeds 1 MiB", 413)
				return
			}
			session := secrets.Fingerprint("http:" + r.Header.Get("Mcp-Session-Id"))
			var uses []model.CredentialUse
			for k, values := range r.Header {
				for _, v := range values {
					for _, m := range secrets.Detect(v, k) {
						uses = append(uses, model.CredentialUse{ID: m.ID, Attribution: "direct"})
					}
				}
			}
			if e = o.Observe(r.Context(), session, b, false, uses); e != nil {
				http.Error(w, "observation unavailable or invalid request", 503)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(b))
			r = r.WithContext(context.WithValue(r.Context(), httpMetaKey{}, httpMeta{session: session}))
		}
		p.ServeHTTP(w, r)
	}), nil
}

type httpMetaKey struct{}
type httpMeta struct{ session string }
type observedBody struct {
	io.ReadCloser
	ctx       context.Context
	observer  *Observer
	session   string
	sse       bool
	buf, data []byte
	failed    bool
}

func (b *observedBody) Read(p []byte) (int, error) {
	n, e := b.ReadCloser.Read(p)
	if n > 0 {
		if err := b.consume(p[:n]); err != nil {
			return n, err
		}
	}
	if e == io.EOF && !b.sse && len(b.buf) > 0 {
		if json.Valid(b.buf) {
			if err := b.observer.Observe(b.ctx, b.session, b.buf, true, nil); err != nil {
				return n, err
			}
		}
	}
	return n, e
}
func (b *observedBody) consume(p []byte) error {
	if !b.sse {
		if len(b.buf)+len(p) > 1<<20 {
			return errors.New("response observation exceeds 1 MiB")
		}
		b.buf = append(b.buf, p...)
		return nil
	}
	for _, c := range p {
		if len(b.buf) >= 1<<20 {
			return errors.New("SSE line exceeds 1 MiB")
		}
		b.buf = append(b.buf, c)
		if c != '\n' {
			continue
		}
		line := strings.TrimSuffix(strings.TrimSuffix(string(b.buf), "\n"), "\r")
		b.buf = b.buf[:0]
		if line == "" {
			if len(b.data) > 0 {
				if err := b.observer.Observe(b.ctx, b.session, bytes.TrimSpace(b.data), true, nil); err != nil {
					return err
				}
				b.data = nil
			}
		} else if strings.HasPrefix(line, "data:") {
			if len(b.data)+len(line) > 1<<20 {
				return errors.New("SSE event exceeds 1 MiB")
			}
			b.data = append(b.data, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " ")...)
			b.data = append(b.data, '\n')
		}
	}
	return nil
}
func Loopback(host string) bool {
	return strings.EqualFold(host, "localhost") || net.ParseIP(host) != nil && net.ParseIP(host).IsLoopback()
}
func Serve(ctx context.Context, addr string, h http.Handler) error {
	host, _, e := net.SplitHostPort(addr)
	if e != nil || !Loopback(host) {
		return errors.New("listen address must be loopback host:port")
	}
	ln, e := net.Listen("tcp", addr)
	if e != nil {
		return errors.New("listen failed")
	}
	srv := &http.Server{Handler: h, ReadHeaderTimeout: 5 * time.Second, MaxHeaderBytes: 32 << 10}
	done := make(chan error, 1)
	go func() { done <- srv.Serve(ln) }()
	select {
	case <-ctx.Done():
		srv.Close()
		<-done
		return ctx.Err()
	case e := <-done:
		if errors.Is(e, http.ErrServerClosed) {
			return nil
		}
		return e
	}
}
