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
