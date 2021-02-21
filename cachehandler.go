// Package cachehandler provides net/http middleware that caches HTTP responses.
// Inspired by go-chi/stampede. https://github.com/go-chi/stampede
// The HTTP response is stored in the cache and calls to handlers with the same key are merged.
package cachehandler

import (
	"bytes"
	"net/http"
	"sync"
	"time"

	"github.com/felixge/httpsnoop"
	expirablecache "github.com/go-pkgz/expirable-cache"
	"golang.org/x/sync/singleflight"
)

// BasicKeyFunc returns a KeyFunc that uses http method and url as key.
func BasicKeyFunc() KeyFunc {
	return func(w http.ResponseWriter, r *http.Request) (string, bool) {
		return r.Method + r.URL.String(), true
	}
}

// KeyFunc is type that returns key for cache.
// If the key func returns false, the Middleware does not call next handler chains.
type KeyFunc func(w http.ResponseWriter, r *http.Request) (string, bool)

type response struct {
	header     http.Header
	statusCode int
	body       []byte
}

// Middleware describes cachehandler
type Middleware struct {
	ttl   time.Duration
	keyFn KeyFunc
	cache expirablecache.Cache
	pool  sync.Pool
	group singleflight.Group
}

// CacheStats describes cache statistics.
type CacheStats struct {
	Hits, Misses   int // cache effectiveness
	Added, Evicted int // number of added and evicted records
}

func (m *Middleware) Stats() CacheStats {
	stats := m.cache.Stat()
	return CacheStats{
		Hits:    stats.Hits,
		Misses:  stats.Misses,
		Added:   stats.Added,
		Evicted: stats.Evicted,
	}
}

func (m *Middleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, ok := m.keyFn(w, r)
		if !ok {
			return
		}
		if v, ok := m.cache.Get(key); ok {
			resp, ok := v.(*response)
			if !ok {
				goto MISS
			}
			header := w.Header()
			for k, list := range resp.header {
				for _, h := range list {
					header.Set(k, h)
				}
			}
			w.WriteHeader(resp.statusCode)
			w.Write(resp.body)
			return
		}

	MISS:
		var (
			header      http.Header
			wroteHeader bool
			status      int
		)
		buf := m.pool.Get().(*bytes.Buffer)
		defer m.pool.Put(buf)

		m.group.Do(key, func() (interface{}, error) {
			next.ServeHTTP(httpsnoop.Wrap(w, httpsnoop.Hooks{
				WriteHeader: func(whf httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
					return func(code int) {
						whf(code)
						if !wroteHeader {
							status = code
							wroteHeader = true
						}
					}
				},
				Write: func(wf httpsnoop.WriteFunc) httpsnoop.WriteFunc {
					return func(b []byte) (int, error) {
						n, err := wf(b)
						buf.Write(b)
						return n, err
					}
				},
				Header: func(hf httpsnoop.HeaderFunc) httpsnoop.HeaderFunc {
					return func() http.Header {
						h := hf()
						header = h
						return h
					}
				},
			}), r)
			return nil, nil
		})

		if status == 0 {
			status = http.StatusOK
		}

		resp := &response{
			header:     header,
			statusCode: status,
			body:       buf.Bytes(),
		}
		m.cache.Set(key, resp, m.ttl)
	})
}

// NewMiddleware returns net/http middleware that caches http responses.
func NewMiddleware(max int, ttl time.Duration, keyFn KeyFunc) *Middleware {
	cache, err := expirablecache.NewCache(expirablecache.MaxKeys(max), expirablecache.TTL(ttl))
	if err != nil {
		panic(err) // never happen
	}
	return &Middleware{
		ttl:   ttl,
		keyFn: keyFn,
		cache: cache,
		pool: sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
		group: singleflight.Group{},
	}
}
