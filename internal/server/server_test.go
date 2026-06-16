package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/hennessyxo/amneziawg-installer/internal/auth"
	"github.com/hennessyxo/amneziawg-installer/internal/awg"
	"github.com/hennessyxo/amneziawg-installer/internal/awgctl"
)

// fakeCtrl is an in-memory Controller for handler tests.
type fakeCtrl struct {
	snap      awg.Snapshot
	added     []string
	revoked   []string
	configs   map[string]string
	addErr    error
	revokeErr error
}

func (f *fakeCtrl) Snapshot() (awg.Snapshot, error) { return f.snap, nil }
func (f *fakeCtrl) ListClients() ([]string, error)  { return nil, nil }
func (f *fakeCtrl) ClientConfig(n string) (string, error) {
	if c, ok := f.configs[n]; ok {
		return c, nil
	}
	return "", fmt.Errorf("no config for %s", n)
}
func (f *fakeCtrl) AddClient(n string) (awgctl.Client, error) {
	if f.addErr != nil {
		return awgctl.Client{}, f.addErr
	}
	f.added = append(f.added, n)
	return awgctl.Client{Name: n, Config: "CONFIG-" + n}, nil
}
func (f *fakeCtrl) RevokeClient(n string) error {
	if f.revokeErr != nil {
		return f.revokeErr
	}
	f.revoked = append(f.revoked, n)
	return nil
}

const testPassword = "s3cret"

func newTestServer(t *testing.T, ctrl awgctl.Controller) *Server {
	t.Helper()
	hash, err := auth.HashPassword(testPassword)
	if err != nil {
		t.Fatal(err)
	}
	s, err := New(ctrl, auth.NewStore(time.Hour), hash, "awg0", false)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func sampleSnapshot() awg.Snapshot {
	now := time.Now()
	return awg.Snapshot{
		Interface: "awg0", Time: now,
		Peers: []awg.Peer{
			{PublicKey: "PH=", Name: "phone", Endpoint: "203.0.113.5:4040", LatestHandshake: now.Add(-10 * time.Second), RxBytes: 1 << 20, TxBytes: 1 << 18},
			{PublicKey: "LT=", Name: "laptop", LatestHandshake: time.Time{}},
		},
	}
}

// login performs a login and returns the session cookie.
func login(t *testing.T, s *Server) *http.Cookie {
	t.Helper()
	form := url.Values{"password": {testPassword}}
	req := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("login status = %d, want 303", rr.Code)
	}
	for _, c := range rr.Result().Cookies() {
		if c.Name == cookieName {
			return c
		}
	}
	t.Fatal("no session cookie set on login")
	return nil
}

var csrfRe = regexp.MustCompile(`name="csrf" value="([^"]+)"`)

func csrfToken(t *testing.T, s *Server, cookie *http.Cookie) string {
	t.Helper()
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	m := csrfRe.FindStringSubmatch(rr.Body.String())
	if m == nil {
		t.Fatal("no csrf token found on dashboard")
	}
	return m[1]
}

func TestLogin_WrongPassword(t *testing.T) {
	s := newTestServer(t, &fakeCtrl{})
	form := url.Values{"password": {"nope"}}
	req := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

func TestDashboard_RequiresAuth(t *testing.T) {
	s := newTestServer(t, &fakeCtrl{})
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusSeeOther || rr.Header().Get("Location") != "/login" {
		t.Errorf("unauth dashboard: status=%d loc=%q, want 303 /login", rr.Code, rr.Header().Get("Location"))
	}
}

func TestClientsPartial_RendersPeers(t *testing.T) {
	s := newTestServer(t, &fakeCtrl{snap: sampleSnapshot()})
	cookie := login(t, s)
	req := httptest.NewRequest("GET", "/partials/clients", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{"phone", "laptop", "онлайн"} {
		if !strings.Contains(body, want) {
			t.Errorf("clients partial missing %q", want)
		}
	}
}

func TestAddClient_WithCSRF(t *testing.T) {
	f := &fakeCtrl{snap: sampleSnapshot()}
	s := newTestServer(t, f)
	cookie := login(t, s)
	csrf := csrfToken(t, s, cookie)

	form := url.Values{"name": {"laptop2"}, "csrf": {csrf}}
	req := httptest.NewRequest("POST", "/clients", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if len(f.added) != 1 || f.added[0] != "laptop2" {
		t.Errorf("AddClient calls = %v, want [laptop2]", f.added)
	}
	if !strings.Contains(rr.Body.String(), "CONFIG-laptop2") {
		t.Error("response missing generated config")
	}
}

func TestAddClient_BadCSRF(t *testing.T) {
	f := &fakeCtrl{snap: sampleSnapshot()}
	s := newTestServer(t, f)
	cookie := login(t, s)
	form := url.Values{"name": {"x"}, "csrf": {"wrong"}}
	req := httptest.NewRequest("POST", "/clients", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rr.Code)
	}
	if len(f.added) != 0 {
		t.Error("AddClient should not be called with bad CSRF")
	}
}

func TestRevokeClient(t *testing.T) {
	f := &fakeCtrl{snap: sampleSnapshot()}
	s := newTestServer(t, f)
	cookie := login(t, s)
	csrf := csrfToken(t, s, cookie)

	form := url.Values{"csrf": {csrf}}
	req := httptest.NewRequest("POST", "/clients/phone/revoke", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if len(f.revoked) != 1 || f.revoked[0] != "phone" {
		t.Errorf("RevokeClient calls = %v, want [phone]", f.revoked)
	}
}

func TestQRPNG(t *testing.T) {
	f := &fakeCtrl{snap: sampleSnapshot(), configs: map[string]string{"phone": "[Interface]\nPrivateKey=x\n"}}
	s := newTestServer(t, f)
	cookie := login(t, s)

	req := httptest.NewRequest("GET", "/clients/phone/qr.png", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("content-type = %q, want image/png", ct)
	}

	// Missing config → 404.
	req2 := httptest.NewRequest("GET", "/clients/ghost/qr.png", nil)
	req2.AddCookie(cookie)
	rr2 := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusNotFound {
		t.Errorf("missing config status = %d, want 404", rr2.Code)
	}
}
