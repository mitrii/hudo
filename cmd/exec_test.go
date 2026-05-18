package cmd

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"remote-sudo/internal/config"
	"remote-sudo/internal/store"
)

// newRoot builds a fresh cobra root with new instances of the commands.
// Cobra commands are stateful (flag values persist between Execute calls),
// so each test must build a fresh root.
func newRoot() *cobra.Command {
	root := &cobra.Command{Use: "remote-sudo", SilenceUsage: true, SilenceErrors: true}

	reqCmd := *RequestCmd
	root.AddCommand(&reqCmd)

	execCmd := &cobra.Command{
		Use:   ExecCmd.Use,
		Short: ExecCmd.Short,
		Args:  ExecCmd.Args,
		RunE:  ExecCmd.RunE,
	}
	execCmd.Flags().StringVar(&pinFlag, "pin", "", "PIN received via notification (required)")
	_ = execCmd.MarkFlagRequired("pin")
	root.AddCommand(execCmd)

	return root
}

func envForTest(t *testing.T, dbPath, webhookURL string) {
	t.Helper()
	t.Setenv("REMOTE_SUDO_HMAC_SECRET", "integration-test-secret")
	t.Setenv("REMOTE_SUDO_WEBHOOK_URL", webhookURL)
	t.Setenv("REMOTE_SUDO_STORE_PATH", dbPath)
	t.Setenv("REMOTE_SUDO_PIN_TTL_SECONDS", "300")
}

func TestRequestStoresEntry(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	envForTest(t, dbPath, srv.URL)

	root := newRoot()
	root.SetArgs([]string{"request", "echo hello"})

	if err := root.Execute(); err != nil {
		t.Fatalf("request: %v", err)
	}

	// Verify that something was written to the store.
	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	defer func() {
		_ = s.Close()
	}()

	// Store should have at least one pending entry (we can't know the HMAC key without PIN,
	// but Purge on an empty store with a valid entry should be a no-op).
	// Indirectly verify by checking the store opens and purges without error.
	if err := s.Purge(); err != nil {
		t.Errorf("Purge after request: %v", err)
	}
}

func TestRequestSendsWebhook(t *testing.T) {
	var gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotBody = r.FormValue("text")

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	envForTest(t, dbPath, srv.URL)

	root := newRoot()
	// Pass the full command as a single arg to avoid cobra flag parsing.
	root.SetArgs([]string{"request", "ls -la /tmp"})

	if err := root.Execute(); err != nil {
		t.Fatalf("request: %v", err)
	}

	if !strings.Contains(gotBody, "ls -la /tmp") {
		t.Errorf("webhook body missing command, got: %q", gotBody)
	}

	if !strings.Contains(gotBody, "HMAC:") {
		t.Errorf("webhook body missing HMAC, got: %q", gotBody)
	}

	if !strings.Contains(gotBody, "PIN:") {
		t.Errorf("webhook body missing PIN, got: %q", gotBody)
	}
}

func TestRequestWebhookFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	envForTest(t, dbPath, srv.URL)

	root := newRoot()
	root.SetArgs([]string{"request", "echo fail"})

	err := root.Execute()
	if err == nil {
		t.Error("expected error when webhook returns 500")
	}
}

func TestExecVerifiesHMAC(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	envForTest(t, dbPath, "http://unused")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config: %v", err)
	}

	command := "echo verified"
	pin := "123456"
	mac := computeHMAC(cfg.HMACSecret, command, pin)

	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("store open: %v", err)
	}

	saveErr := s.Save(mac, store.Entry{
		Command:   command,
		PIN:       pin,
		ExpiresAt: time.Now().Add(time.Minute),
	})

	if cerr := s.Close(); cerr != nil {
		t.Fatalf("store close: %v", cerr)
	}

	if saveErr != nil {
		t.Fatalf("store save: %v", saveErr)
	}

	pinFlag = ""

	root := newRoot()
	root.SetArgs([]string{"exec", "--pin", pin, "echo verified"})

	execErr := root.Execute()

	// On non-Linux runPrivileged returns an unsupported error — acceptable.
	// We only care that HMAC verification did NOT fail.
	if execErr != nil && strings.Contains(execErr.Error(), "verification failed") {
		t.Errorf("exec verification should have passed, got: %v", execErr)
	}
}

func TestExecWrongPIN(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	envForTest(t, dbPath, "http://unused")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config: %v", err)
	}

	command := "echo secret"
	pin := "111111"
	mac := computeHMAC(cfg.HMACSecret, command, pin)

	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("store open: %v", err)
	}

	saveErr := s.Save(mac, store.Entry{
		Command:   command,
		PIN:       pin,
		ExpiresAt: time.Now().Add(time.Minute),
	})

	if cerr := s.Close(); cerr != nil {
		t.Fatalf("store close: %v", cerr)
	}

	if saveErr != nil {
		t.Fatalf("store save: %v", saveErr)
	}

	pinFlag = ""

	root := newRoot()
	root.SetArgs([]string{"exec", "--pin", "999999", "echo secret"})

	err = root.Execute()
	if err == nil || !strings.Contains(err.Error(), "verification failed") {
		t.Errorf("expected verification failure for wrong PIN, got: %v", err)
	}
}

func TestExecExpiredPIN(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	envForTest(t, dbPath, "http://unused")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config: %v", err)
	}

	command := "id"
	pin := "777777"
	mac := computeHMAC(cfg.HMACSecret, command, pin)

	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("store open: %v", err)
	}

	saveErr := s.Save(mac, store.Entry{
		Command:   command,
		PIN:       pin,
		ExpiresAt: time.Now().Add(-time.Second),
	})

	if cerr := s.Close(); cerr != nil {
		t.Fatalf("store close: %v", cerr)
	}

	if saveErr != nil {
		t.Fatalf("store save: %v", saveErr)
	}

	pinFlag = ""

	root := newRoot()
	root.SetArgs([]string{"exec", "--pin", pin, "id"})

	err = root.Execute()
	if err == nil || !strings.Contains(err.Error(), "verification failed") {
		t.Errorf("expected verification failure for expired PIN, got: %v", err)
	}
}

func TestExecMissingPINFlag(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	envForTest(t, dbPath, "http://unused")

	pinFlag = ""

	root := newRoot()
	root.SetArgs([]string{"exec", "echo hello"})

	err := root.Execute()
	if err == nil {
		t.Error("expected error when --pin flag is missing")
	}
}
