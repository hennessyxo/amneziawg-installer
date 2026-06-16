// Package server implements the AmneziaWG web panel: a session-authenticated,
// htmx-driven dashboard for viewing live client traffic and managing clients.
package server

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/skip2/go-qrcode"

	"github.com/hennessyxo/amneziawg-installer/internal/auth"
	"github.com/hennessyxo/amneziawg-installer/internal/awg"
	"github.com/hennessyxo/amneziawg-installer/internal/awgctl"
	"github.com/hennessyxo/amneziawg-installer/internal/format"
	"github.com/hennessyxo/amneziawg-installer/internal/web"
)

const cookieName = "awgsess"

// Server holds the panel's dependencies and HTTP routes.
type Server struct {
	ctrl     awgctl.Controller
	sessions *auth.Store
	pwHash   string
	iface    string
	secure   bool // set the Secure flag on cookies (true behind HTTPS)
	tmpl     *template.Template
	mux      *http.ServeMux
	rates    *rateTracker
}

// New builds a Server. It takes the Controller interface (so tests can inject a
// fake) and returns the concrete struct.
func New(ctrl awgctl.Controller, sessions *auth.Store, pwHash, iface string, secure bool) (*Server, error) {
	tmpl, err := template.ParseFS(web.Templates, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parsing templates: %w", err)
	}
	s := &Server{
		ctrl:     ctrl,
		sessions: sessions,
		pwHash:   pwHash,
		iface:    iface,
		secure:   secure,
		tmpl:     tmpl,
		rates:    newRateTracker(),
	}
	s.routes()
	return s, nil
}

// Handler returns the HTTP handler for the panel.
func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) routes() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /login", s.loginPage)
	mux.HandleFunc("POST /login", s.doLogin)
	mux.HandleFunc("POST /logout", s.doLogout)
	mux.HandleFunc("GET /{$}", s.requireAuth(s.dashboard))
	mux.HandleFunc("GET /partials/clients", s.requireAuth(s.clientsPartial))
	mux.HandleFunc("POST /clients", s.requireAuth(s.addClient))
	mux.HandleFunc("POST /clients/{name}/revoke", s.requireAuth(s.revokeClient))
	mux.HandleFunc("GET /clients/{name}/qr.png", s.requireAuth(s.qrPNG))
	mux.HandleFunc("GET /clients/{name}/config", s.requireAuth(s.downloadConfig))

	staticFS, _ := fs.Sub(web.Static, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	s.mux = mux
}

// --- auth plumbing ---------------------------------------------------------

func (s *Server) session(r *http.Request) (auth.Session, bool) {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return auth.Session{}, false
	}
	return s.sessions.Valid(c.Value)
}

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := s.session(r); !ok {
			// htmx requests can't follow a 303; tell htmx to redirect instead.
			if r.Header.Get("HX-Request") == "true" {
				w.Header().Set("HX-Redirect", "/login")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

// checkCSRF validates the form's csrf token against the session.
func (s *Server) checkCSRF(r *http.Request) bool {
	sess, ok := s.session(r)
	if !ok {
		return false
	}
	return r.FormValue("csrf") != "" && r.FormValue("csrf") == sess.CSRF
}

// --- handlers --------------------------------------------------------------

func (s *Server) loginPage(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.session(r); ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	s.render(w, "login", map[string]any{})
}

func (s *Server) doLogin(w http.ResponseWriter, r *http.Request) {
	pw := r.FormValue("password")
	if !auth.CheckPassword(s.pwHash, pw) {
		w.WriteHeader(http.StatusUnauthorized)
		s.render(w, "login", map[string]any{"Error": "Неверный пароль"})
		return
	}
	token, _ := s.sessions.Create()
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.secure,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) doLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(cookieName); err == nil {
		s.sessions.Delete(c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	sess, _ := s.session(r)
	s.render(w, "dashboard", map[string]any{"Iface": s.iface, "CSRF": sess.CSRF})
}

func (s *Server) clientsPartial(w http.ResponseWriter, r *http.Request) {
	sess, _ := s.session(r)
	data, err := s.buildClientsData(sess.CSRF)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, `<p class="err">Не удалось получить данные: %s</p>`, template.HTMLEscapeString(err.Error()))
		return
	}
	s.render(w, "clients", data)
}

func (s *Server) addClient(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRF(r) {
		http.Error(w, "bad csrf token", http.StatusForbidden)
		return
	}
	name, ok := awgctl.SanitizeName(r.FormValue("name"))
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `<div class="created"><div class="created-head">Некорректное имя клиента</div></div>`)
		return
	}
	client, err := s.ctrl.AddClient(name)
	if err != nil {
		w.WriteHeader(http.StatusConflict)
		fmt.Fprintf(w, `<div class="created"><div class="created-head">Ошибка: %s</div></div>`, template.HTMLEscapeString(err.Error()))
		return
	}
	s.render(w, "created", map[string]any{"Name": client.Name, "Config": client.Config})
}

func (s *Server) revokeClient(w http.ResponseWriter, r *http.Request) {
	if !s.checkCSRF(r) {
		http.Error(w, "bad csrf token", http.StatusForbidden)
		return
	}
	name := r.PathValue("name")
	if err := s.ctrl.RevokeClient(name); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `<p class="err">%s</p>`, template.HTMLEscapeString(err.Error()))
		return
	}
	// Return the refreshed table so the row disappears immediately.
	sess, _ := s.session(r)
	data, err := s.buildClientsData(sess.CSRF)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	s.render(w, "clients", data)
}

func (s *Server) qrPNG(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.ctrl.ClientConfig(r.PathValue("name"))
	if err != nil {
		http.Error(w, "config unavailable", http.StatusNotFound)
		return
	}
	png, err := qrcode.Encode(cfg, qrcode.Medium, 256)
	if err != nil {
		http.Error(w, "qr error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(png)
}

func (s *Server) downloadConfig(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	cfg, err := s.ctrl.ClientConfig(name)
	if err != nil {
		http.Error(w, "config unavailable", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-client-%s.conf"`, s.iface, name))
	fmt.Fprint(w, cfg)
}

// --- view assembly ---------------------------------------------------------

type peerView struct {
	Name, Endpoint               string
	RateRx, RateTx, RxStr, TxStr string
	HandshakeAgo                 string
	Online, HasConfig            bool
}

type clientsData struct {
	Peers                     []peerView
	Online, Total             int
	TotalRx, TotalTx, TimeStr string
	CSRF                      string
}

func (s *Server) buildClientsData(csrf string) (clientsData, error) {
	snap, err := s.ctrl.Snapshot()
	if err != nil {
		return clientsData{}, err
	}
	s.rates.update(snap)
	now := snap.Time

	peers := make([]awg.Peer, len(snap.Peers))
	copy(peers, snap.Peers)
	sort.SliceStable(peers, func(i, j int) bool {
		oi, oj := peers[i].Online(now), peers[j].Online(now)
		if oi != oj {
			return oi
		}
		return displayName(peers[i]) < displayName(peers[j])
	})

	views := make([]peerView, 0, len(peers))
	for _, p := range peers {
		rx, tx := s.rates.rate(p.PublicKey)
		endpoint := p.Endpoint
		if endpoint == "" {
			endpoint = "—"
		}
		views = append(views, peerView{
			Name:         displayName(p),
			Endpoint:     endpoint,
			RateRx:       format.HumanRate(rx),
			RateTx:       format.HumanRate(tx),
			RxStr:        format.HumanBytes(p.RxBytes),
			TxStr:        format.HumanBytes(p.TxBytes),
			HandshakeAgo: format.Ago(p.LatestHandshake, now),
			Online:       p.Online(now),
			HasConfig:    true,
		})
	}

	return clientsData{
		Peers:   views,
		Online:  snap.OnlineCount(),
		Total:   len(snap.Peers),
		TotalRx: format.HumanBytes(snap.TotalRx()),
		TotalTx: format.HumanBytes(snap.TotalTx()),
		TimeStr: now.Format("15:04:05"),
		CSRF:    csrf,
	}, nil
}

func (s *Server) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func displayName(p awg.Peer) string {
	if p.Name != "" {
		return p.Name
	}
	if len(p.PublicKey) > 8 {
		return p.PublicKey[:8]
	}
	return p.PublicKey
}

// --- rate tracker ----------------------------------------------------------

type rateTracker struct {
	mu       sync.Mutex
	prev     map[string]awg.Peer
	prevTime time.Time
	rx, tx   map[string]float64
}

func newRateTracker() *rateTracker {
	return &rateTracker{rx: map[string]float64{}, tx: map[string]float64{}}
}

func (rt *rateTracker) update(s awg.Snapshot) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.prev != nil {
		dt := s.Time.Sub(rt.prevTime).Seconds()
		if dt > 0 {
			for _, p := range s.Peers {
				old, ok := rt.prev[p.PublicKey]
				if !ok {
					continue
				}
				rt.rx[p.PublicKey] = deltaRate(p.RxBytes, old.RxBytes, dt)
				rt.tx[p.PublicKey] = deltaRate(p.TxBytes, old.TxBytes, dt)
			}
		}
	}
	rt.prev = make(map[string]awg.Peer, len(s.Peers))
	for _, p := range s.Peers {
		rt.prev[p.PublicKey] = p
	}
	rt.prevTime = s.Time
}

func (rt *rateTracker) rate(pub string) (rx, tx float64) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.rx[pub], rt.tx[pub]
}

func deltaRate(cur, old uint64, dt float64) float64 {
	if cur < old || dt <= 0 {
		return 0
	}
	return float64(cur-old) / dt
}
