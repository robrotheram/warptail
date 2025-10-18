package botprotect

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"text/template"
	"time"
	"warptail/pkg/utils"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"
	"github.com/prometheus/client_golang/prometheus"
)

//go:embed "challenge.tmpl.html"
var challengeHtml []byte

var (
	botChallengeFailedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "warptail_bot_challenge_failed_total",
			Help: "Total number of failed bot challenge verifications",
		},
		[]string{"reason"},
	)
	botChallengePageRenderCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "warptail_bot_challenge_page_rendered_total",
			Help: "Total number of times the bot challenge page was rendered (likely bots that never verify)",
		},
		[]string{},
	)
	botChallengeSuccessCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "warptail_bot_challenge_success_total",
			Help: "Total number of successful bot challenge verifications",
		},
	)
)

func init() {
	prometheus.MustRegister(botChallengeFailedCounter)
	prometheus.MustRegister(botChallengePageRenderCounter)
	prometheus.MustRegister(botChallengeSuccessCounter)
}

type BotChallenge struct {
	challengeStore map[string]Verify
	mutex          sync.Mutex
	challengeTTL   time.Duration
	sessionStore   *sessions.CookieStore
	tmpl           *template.Template
}

func NewBotChallenge(mux *chi.Mux, config utils.AuthenticationConfig) *BotChallenge {
	bc := &BotChallenge{
		challengeStore: make(map[string]Verify),
		mutex:          sync.Mutex{},
		challengeTTL:   5 * time.Minute,
		tmpl:           template.Must(template.New("challenge").Parse(string(challengeHtml))),
	}

	bc.sessionStore = sessions.NewCookieStore([]byte(config.SessionSecret))
	bc.sessionStore.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   0,
		HttpOnly: true,
		Secure:   false,
	}
	mux.HandleFunc("/bot/verify", bc.handleVerify)
	go bc.cleanExpiredChallenges()
	return bc
}

func (bc *BotChallenge) cleanExpiredChallenges() {
	for {
		time.Sleep(1 * time.Minute)
		bc.mutex.Lock()
		now := time.Now()
		for id, ch := range bc.challengeStore {
			if ch.ExpiresAt.Before(now) {
				delete(bc.challengeStore, id)
			}
		}
		bc.mutex.Unlock()
	}
}

func (bc *BotChallenge) fingerprint(r *http.Request) string {
	// Collect values that are often unique per client
	ip := r.RemoteAddr
	ua := r.Header.Get("User-Agent")
	accept := r.Header.Get("Accept")
	lang := r.Header.Get("Accept-Language")
	enc := r.Header.Get("Accept-Encoding")
	conn := r.Header.Get("Connection")
	forwarded := r.Header.Get("X-Forwarded-For")

	// Concatenate values
	raw := strings.Join([]string{
		ip, ua, accept, lang, enc, conn, forwarded,
	}, "|")

	// Hash for privacy and fixed length
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}

func (bc *BotChallenge) valid(r *http.Request) bool {
	session, _ := bc.sessionStore.Get(r, "warptail-bot-session")
	challengeId, ok := session.Values["challengeId"].(string)
	if !ok {
		return false
	}
	bc.mutex.Lock()
	challenge, ok := bc.challengeStore[challengeId]
	bc.mutex.Unlock()
	if !ok || time.Now().After(challenge.ExpiresAt) {
		fmt.Println("Challenge expired or not found")
		return false
	}
	return true
}

type Verify struct {
	Nonce      string `json:"nonce"`
	Hash       string `json:"hash"`
	It         int    `json:"it"`
	Difficulty int    `json:"difficulty"`
	ExpiresAt  time.Time
}

func (v *Verify) VerifyHash() bool {
	hash := sha512Hash(fmt.Sprintf("%s%v", v.Nonce, v.It))
	if hash != v.Hash {
		return false
	}
	return hash[0:v.Difficulty] == strings.Repeat("0", v.Difficulty)
}

func sha512Hash(input string) string {
	hash := sha512.Sum512([]byte(input))
	return hex.EncodeToString(hash[:])
}

func gernerateNonce() string {
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return ""
	}
	return hex.EncodeToString(randomBytes)
}

func (bc *BotChallenge) renderTemplate(w http.ResponseWriter, r *http.Request) {
	data := Verify{
		Nonce:      gernerateNonce(),
		Difficulty: 4,
	}

	bc.mutex.Lock()
	bc.challengeStore[bc.fingerprint(r)] = data
	bc.mutex.Unlock()

	err := bc.tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}

func (bc *BotChallenge) Middleware(w http.ResponseWriter, r *http.Request, handler func(w http.ResponseWriter, r *http.Request)) {
	if r.Method == "POST" && strings.Contains(r.URL.Path, "/bot/verify") {
		bc.handleVerify(w, r)
		return
	}
	if !bc.valid(r) {
		// Prometheus: count challenge page renders (likely bots without WebCrypto)
		botChallengePageRenderCounter.WithLabelValues().Inc()
		bc.renderTemplate(w, r)
		return
	}
	handler(w, r)
}

func (bc *BotChallenge) handleVerify(w http.ResponseWriter, r *http.Request) {
	var verify Verify
	err := json.NewDecoder(r.Body).Decode(&verify)
	if err != nil {
		http.Error(w, "Failed to decode JSON", http.StatusBadRequest)
		// Prometheus: count decode errors as failed challenges
		botChallengeFailedCounter.WithLabelValues("decode_error").Inc()
		return
	}
	if !verify.VerifyHash() {
		http.Error(w, "Hash mismatch", http.StatusBadRequest)
		// Prometheus: count hash mismatches as failed challenges
		botChallengeFailedCounter.WithLabelValues("hash_mismatch").Inc()
		return
	}
	// Prometheus: count successful challenges
	botChallengeSuccessCounter.Inc()
	bc.mutex.Lock()
	challengeId := bc.fingerprint(r)
	verify.ExpiresAt = time.Now().Add(bc.challengeTTL)
	bc.challengeStore[challengeId] = verify
	bc.mutex.Unlock()

	session, _ := bc.sessionStore.Get(r, "warptail-bot-session")
	session.Values["challengeId"] = challengeId
	session.Save(r, w)

	utils.WriteData(w, map[string]bool{"verified": true})
}
