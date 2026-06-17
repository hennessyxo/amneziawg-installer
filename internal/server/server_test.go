package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/hennessyxo/amneziawg-installer/internal/auth"
	"github.com/hennessyxo/amneziawg-installer/internal/awg"
	"github.com/hennessyxo/amneziawg-installer/internal/awgctl"
	"github.com/hennessyxo/amneziawg-installer/internal/lifecycle"
)

// postForm is a small helper for authenticated form POSTs.
func postForm(t *testing.T, s *Server, path string, form url.Values, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("POST", path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	return rr
}

// fakeCtrl is an in-memory Controller for handler tests.
type fakeCtrl struct {
	snap      awg.Snapshot
	added     []string
	revoked   []string
	disabled  []string
	enabled   []string
	updated   []string
	renamed   []string
	configs   map[string]string
	addErr    error
	revokeErr error
}

func (f *fakeCtrl) Snapshot() (awg.Snapshot, error) { return f.snap, nil }
func (f *fakeCtrl) ClientConfig(n string) (string, error) {
	if c, ok := f.configs[n]; ok {
		return c, nil
	}
	return "", fmt.Errorf("no config for %s", n)
}
func (f *fakeCtrl) AddClient(n string, _ awgctl.AddOptions) (awgctl.Client, error) {
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
func (f *fakeCtrl) DisableClient(n string) error { f.disabled = append(f.disabled, n); return nil }
func (f *fakeCtrl) EnableClient(n string) error  { f.enabled = append(f.enabled, n); return nil }
func (f *fakeCtrl) UpdateClient(n string, _ awgctl.UpdateOptions) error {
	f.updated = append(f.updated, n)
	return nil
}
func (f *fakeCtrl) RenameClient(o, n string) error {
	f.renamed = append(f.renamed, o+"->"+n)
	return nil
}

const testPassword = "s3cret"

func newTestServer(t *testing.T, ctrl awgctl.Controller) *Server {
	t.Helper()
	hash, err := auth.HashPassword(testPassword)
	if err != nil {
		t.Fatal(err)
	}
	s, err := New(ctrl, auth.NewStore(time.Hour), nil, hash, "awg0", false)
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

func TestEditForm_Renders(t *testing.T) {
	s := newTestServer(t, &fakeCtrl{snap: sampleSnapshot()})
	cookie := login(t, s)
	req := httptest.NewRequest("GET", "/clients/phone/edit", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `name="speed_mbit"`) || !strings.Contains(body, `value="phone"`) {
		t.Errorf("edit form missing fields/prefill:\n%s", body)
	}
}

func TestUpdateClient_RenameAndLimits(t *testing.T) {
	f := &fakeCtrl{snap: sampleSnapshot()}
	s := newTestServer(t, f)
	cookie := login(t, s)
	csrf := csrfToken(t, s, cookie)

	form := url.Values{
		"csrf": {csrf}, "name": {"newphone"},
		"speed_mbit": {"10"}, "quota_gb": {"5"}, "expires_days": {"3"},
	}
	rr := postForm(t, s, "/clients/phone/update", form, cookie)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if len(f.renamed) != 1 || f.renamed[0] != "phone->newphone" {
		t.Errorf("rename calls = %v, want [phone->newphone]", f.renamed)
	}
	if len(f.updated) != 1 || f.updated[0] != "newphone" {
		t.Errorf("update calls = %v, want [newphone]", f.updated)
	}
}

func TestLogin_LockoutAfterFailures(t *testing.T) {
	s := newTestServer(t, &fakeCtrl{})
	bad := url.Values{"password": {"nope"}}
	// 5 failures are allowed (each 401), the 6th attempt is locked (429).
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("POST", "/login", strings.NewReader(bad.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.RemoteAddr = "198.51.100.9:5000"
		rr := httptest.NewRecorder()
		s.Handler().ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: status = %d, want 401", i+1, rr.Code)
		}
	}
	req := httptest.NewRequest("POST", "/login", strings.NewReader(bad.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.RemoteAddr = "198.51.100.9:5000"
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("6th attempt: status = %d, want 429", rr.Code)
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

func TestLanguageSwitch(t *testing.T) {
	s := newTestServer(t, &fakeCtrl{snap: sampleSnapshot()})
	cookie := login(t, s)

	// Default (no lang cookie) is Russian.
	reqRu := httptest.NewRequest("GET", "/", nil)
	reqRu.AddCookie(cookie)
	rrRu := httptest.NewRecorder()
	s.Handler().ServeHTTP(rrRu, reqRu)
	if !strings.Contains(rrRu.Body.String(), "Клиенты") {
		t.Error("default dashboard should be Russian (Клиенты)")
	}

	// With lang=en cookie, the dashboard renders English.
	reqEn := httptest.NewRequest("GET", "/", nil)
	reqEn.AddCookie(cookie)
	reqEn.AddCookie(&http.Cookie{Name: "lang", Value: "en"})
	rrEn := httptest.NewRecorder()
	s.Handler().ServeHTTP(rrEn, reqEn)
	body := rrEn.Body.String()
	if !strings.Contains(body, "Clients") || !strings.Contains(body, "Sign out") {
		t.Errorf("en dashboard missing English strings:\n%s", body)
	}
}

func TestSetLang_SetsCookie(t *testing.T) {
	s := newTestServer(t, &fakeCtrl{})
	req := httptest.NewRequest("GET", "/lang/en", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", rr.Code)
	}
	var found bool
	for _, c := range rr.Result().Cookies() {
		if c.Name == "lang" && c.Value == "en" {
			found = true
		}
	}
	if !found {
		t.Error("lang cookie not set to en")
	}
}

func TestDisableEnableClient(t *testing.T) {
	f := &fakeCtrl{snap: sampleSnapshot()}
	s := newTestServer(t, f)
	cookie := login(t, s)
	csrf := csrfToken(t, s, cookie)

	if rr := postForm(t, s, "/clients/phone/disable", url.Values{"csrf": {csrf}}, cookie); rr.Code != http.StatusOK {
		t.Fatalf("disable status = %d, want 200", rr.Code)
	}
	if len(f.disabled) != 1 || f.disabled[0] != "phone" {
		t.Errorf("DisableClient calls = %v, want [phone]", f.disabled)
	}

	if rr := postForm(t, s, "/clients/phone/enable", url.Values{"csrf": {csrf}}, cookie); rr.Code != http.StatusOK {
		t.Fatalf("enable status = %d, want 200", rr.Code)
	}
	if len(f.enabled) != 1 || f.enabled[0] != "phone" {
		t.Errorf("EnableClient calls = %v, want [phone]", f.enabled)
	}
}

func TestEnforceOnce_DisablesExpiredAndOverQuota(t *testing.T) {
	store, err := lifecycle.Open(filepath.Join(t.TempDir(), "clients.json"))
	if err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-time.Hour)
	_ = store.Put(lifecycle.Record{Name: "expired", PubKey: "EX=", Octet: 2, ExpiresAt: &past})
	_ = store.Put(lifecycle.Record{Name: "heavy", PubKey: "HV=", Octet: 3, QuotaBytes: 100})

	// "heavy" transfers 160 bytes total → exceeds the 100-byte quota.
	f := &fakeCtrl{snap: awg.Snapshot{Time: time.Now(), Peers: []awg.Peer{
		{PublicKey: "HV=", RxBytes: 80, TxBytes: 80},
	}}}
	hash, _ := auth.HashPassword(testPassword)
	s, err := New(f, auth.NewStore(time.Hour), store, hash, "awg0", false)
	if err != nil {
		t.Fatal(err)
	}

	s.enforceOnce()

	// Expired and over-quota clients are both DISABLED, never deleted.
	if len(f.revoked) != 0 {
		t.Errorf("nothing should be deleted; revoked = %v", f.revoked)
	}
	if len(f.disabled) != 2 {
		t.Fatalf("both clients should be disabled; disabled = %v", f.disabled)
	}
	got := map[string]bool{f.disabled[0]: true, f.disabled[1]: true}
	if !got["expired"] || !got["heavy"] {
		t.Errorf("disabled = %v, want {expired, heavy}", f.disabled)
	}
}
