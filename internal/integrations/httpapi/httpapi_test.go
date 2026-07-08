package httpapi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeBaseURL(t *testing.T) {
	cases := map[string]string{
		"https://x.example.com/":  "https://x.example.com",
		"https://x.example.com":   "https://x.example.com",
		"https://x.example.com//": "https://x.example.com/",
	}
	for in, want := range cases {
		if got := NormalizeBaseURL(in); got != want {
			t.Errorf("NormalizeBaseURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDoAppliesDecorator(t *testing.T) {
	var gotAuth, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := &Client{
		Provider: "test",
		HTTP:     srv.Client(),
		Decorate: func(req *http.Request) { req.SetBasicAuth("user", "pass") },
	}
	resp, err := c.Do(context.Background(), "GET", srv.URL, nil)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()
	if gotMethod != "GET" {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if gotAuth == "" || !strings.HasPrefix(gotAuth, "Basic ") {
		t.Errorf("decorator did not apply basic auth, got %q", gotAuth)
	}
}

func TestEnsureOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ok" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, "nope")
	}))
	defer srv.Close()

	c := &Client{Provider: "jira", HTTP: srv.Client()}

	okResp, err := c.Do(context.Background(), "GET", srv.URL+"/ok", nil)
	if err != nil {
		t.Fatalf("Do ok: %v", err)
	}
	defer okResp.Body.Close()
	if err := c.EnsureOK(okResp); err != nil {
		t.Errorf("EnsureOK(200) = %v, want nil", err)
	}

	badResp, err := c.Do(context.Background(), "GET", srv.URL+"/bad", nil)
	if err != nil {
		t.Fatalf("Do bad: %v", err)
	}
	defer badResp.Body.Close()
	err = c.EnsureOK(badResp)
	if err == nil {
		t.Fatal("EnsureOK(401) should return an error")
	}
	if !strings.Contains(err.Error(), "jira API error (status 401)") || !strings.Contains(err.Error(), "nope") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDoRead(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bad" {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, "boom")
			return
		}
		body, _ := io.ReadAll(r.Body)
		_, _ = w.Write(body) // echo
	}))
	defer srv.Close()

	c := &Client{Provider: "codecks", HTTP: srv.Client()}

	got, err := c.DoRead(context.Background(), "POST", srv.URL+"/", strings.NewReader("payload"))
	if err != nil {
		t.Fatalf("DoRead: %v", err)
	}
	if string(got) != "payload" {
		t.Errorf("DoRead body = %q, want %q", got, "payload")
	}

	_, err = c.DoRead(context.Background(), "POST", srv.URL+"/bad", nil)
	if err == nil {
		t.Fatal("DoRead should error on 500")
	}
	if !strings.Contains(err.Error(), "codecks API error (status 500)") || !strings.Contains(err.Error(), "boom") {
		t.Errorf("unexpected error: %v", err)
	}
}
