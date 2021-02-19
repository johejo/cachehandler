# cachehandler

Package cachehandler provides net/http middleware that caches HTTP responses.

Inspired by go-chi/stampede. https://github.com/go-chi/stampede

The HTTP response is stored in the LRU cache and calls to handlers with the same key are merged.

[![ci](https://github.com/johejo/cachehandler/workflows/ci/badge.svg?branch=main)](https://github.com/johejo/cachehandler/actions?query=workflow%3Aci)
[![Go Reference](https://pkg.go.dev/badge/github.com/johejo/cachehandler.svg)](https://pkg.go.dev/github.com/johejo/cachehandler)
[![codecov](https://codecov.io/gh/johejo/cachehandler/branch/main/graph/badge.svg)](https://codecov.io/gh/johejo/cachehandler)
[![Go Report Card](https://goreportcard.com/badge/github.com/johejo/cachehandler)](https://goreportcard.com/report/github.com/johejo/cachehandler)

## Example

```go
package cachehandler_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/johejo/cachehandler"
)

func Example() {
	mux := http.NewServeMux()
	m := cachehandler.NewMiddleware(1000, 1*time.Hour, cachehandler.BasicKeyFunc())
	var called int
	mux.Handle("/", m.Wrap(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		called++
	})))

	ts := httptest.NewServer(mux)
	defer ts.Close()

	resp1, err := http.Get(ts.URL + "/foo")
	if err != nil {
		panic(err)
	}
	defer resp1.Body.Close()

	resp2, err := http.Get(ts.URL + "/foo")
	if err != nil {
		panic(err)
	}
	defer resp2.Body.Close()

	fmt.Printf("called %d", called)

	// Output:
	// called 1
}
```

## License

MIT

## Author

Mitsuo Heijo
