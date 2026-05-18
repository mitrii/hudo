package notify_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"remote-sudo/internal/notify"
)

func TestSendGETWithPlaceholder(t *testing.T) {
	var gotURL string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.String()

		if r.Method != http.MethodGet {
			t.Errorf("method: got %s, want GET", r.Method)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	text := "hello world"
	webhookURL := srv.URL + "/notify?text={text}"

	if err := notify.Send(webhookURL, text); err != nil {
		t.Fatalf("Send: %v", err)
	}

	if !strings.Contains(gotURL, url.QueryEscape(text)) {
		t.Errorf("URL %q does not contain encoded text", gotURL)
	}
}

func TestSendPOSTWithoutPlaceholder(t *testing.T) {
	var (
		gotMethod string
		gotBody   string
		gotCT     string
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotCT = r.Header.Get("Content-Type")

		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	text := "my message"

	if err := notify.Send(srv.URL, text); err != nil {
		t.Fatalf("Send: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method: got %s, want POST", gotMethod)
	}

	if gotCT != "application/x-www-form-urlencoded" {
		t.Errorf("Content-Type: got %q", gotCT)
	}

	parsed, err := url.ParseQuery(gotBody)
	if err != nil {
		t.Fatalf("parse body: %v", err)
	}

	if parsed.Get("text") != text {
		t.Errorf("body text: got %q, want %q", parsed.Get("text"), text)
	}
}

func TestSendHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	err := notify.Send(srv.URL, "ping")
	if err == nil {
		t.Error("expected error for 5xx response")
	}

	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error %q does not mention status 500", err.Error())
	}
}

func TestSendConnectionRefused(t *testing.T) {
	err := notify.Send("http://127.0.0.1:1", "ping")
	if err == nil {
		t.Error("expected error for connection refused")
	}
}

func TestSendTextEncodedInURL(t *testing.T) {
	received := make(chan string, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		received <- r.URL.Query().Get("text")
	}))
	defer srv.Close()

	text := "rm -rf /tmp/test"
	webhookURL := srv.URL + "?text={text}"

	if err := notify.Send(webhookURL, text); err != nil {
		t.Fatalf("Send: %v", err)
	}

	if got := <-received; got != text {
		t.Errorf("decoded text: got %q, want %q", got, text)
	}
}
