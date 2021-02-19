package cachehandler_test

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/johejo/cachehandler"
)

func Test(t *testing.T) {
	mux := http.NewServeMux()
	m := cachehandler.NewMiddleware(1000, 1*time.Hour, cachehandler.BasicKeyFunc())

	var called int
	mux.Handle("/", m.Wrap(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Add("X-Test", "test-value")
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte("abcdefg"))
		called++
	})))

	ts := httptest.NewServer(mux)
	defer ts.Close()

	testRequest := func(t *testing.T, mux http.Handler, method, url string, reader io.Reader) {
		t.Helper()
		req, err := http.NewRequest(method, ts.URL+url, reader)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		gotBody := string(body)
		const wantBody = "abcdefg"
		if wantBody != gotBody {
			t.Errorf("body should be %s but got %s", wantBody, gotBody)
		}

		gotHeader := resp.Header.Get("X-Test")
		const wantHeader = "test-value"
		if gotHeader != wantHeader {
			t.Errorf("header should be %s but got %s", wantHeader, gotHeader)
		}

		if resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("status should be %d but got %d", http.StatusInternalServerError, resp.StatusCode)
		}
	}
	for i := 0; i < 5; i++ {
		t.Run("GET /foo", func(t *testing.T) {
			testRequest(t, mux, http.MethodGet, "/foo", nil)
			if called != 1 {
				t.Errorf("called should be 1 but called %d times", called)
			}
		})
	}

	t.Run("GET /foo/bar", func(t *testing.T) {
		testRequest(t, mux, http.MethodGet, "/foo/var", nil)
		if called != 2 {
			t.Errorf("called should be 2 but called %d times", called)
		}
	})

	t.Run("DELETE /foo/bar", func(t *testing.T) {
		testRequest(t, mux, http.MethodDelete, "/foo/var", nil)
		if called != 3 {
			t.Errorf("called should be 3 but called %d times", called)
		}
	})

	t.Run("POST /foo/bar", func(t *testing.T) {
		testRequest(t, mux, http.MethodPost, "/foo/var", strings.NewReader("test-body"))
		if called != 4 {
			t.Errorf("called should be 4 but called %d times", called)
		}
	})
}

func TestGroup(t *testing.T) {
	mux := http.NewServeMux()
	m := cachehandler.NewMiddleware(1000, 1*time.Hour, cachehandler.BasicKeyFunc())

	var called int
	mux.Handle("/", m.Wrap(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Second)
		called++
	})))

	ts := httptest.NewServer(mux)
	defer ts.Close()

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get(ts.URL + "/foo")
			if err != nil {
				panic(err)
			}
			defer resp.Body.Close()
		}()
	}
	wg.Wait()

	if called != 1 {
		t.Errorf("should be called once but called %d-times", called)
	}
}
