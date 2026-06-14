package admin

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/DwifteJB/aplsonic/src/db"
	"github.com/DwifteJB/aplsonic/src/db/schema"
)

const (
	sessionCookie = "aplsonic_admin"
	sessionMaxAge = 7 * 24 * time.Hour
)

// creates admin acc
func EnsureAdmin() error {
	var cfg schema.AdminConfig
	if err := db.DB.First(&cfg).Error; err == nil {
		return nil // already bootstrapped
	}

	password := randString(9) // ~14 char base32-ish password
	salt := randHex(16)
	secret := randHex(32)

	cfg = schema.AdminConfig{
		PasswordHash:  hashPassword(password, salt),
		Salt:          salt,
		SessionSecret: secret,
	}
	if err := db.DB.Create(&cfg).Error; err != nil {
		return err
	}

	fmt.Println("\n========================================================")
	fmt.Println("  Admin account has been created! Save this password!!! IT WILL BE LOST IF NOT!!!")
	fmt.Printf("    password: %s\n", password)
	fmt.Println("  change this if you want within the settings :)")
	fmt.Println("========================================================")
	return nil
}

// simple sha256 hash, switch to bcrypt?
func hashPassword(password, salt string) string {
	sum := sha256.Sum256([]byte(password + salt))
	return hex.EncodeToString(sum[:])
}

func loadAdmin() (*schema.AdminConfig, error) {
	var cfg schema.AdminConfig
	if err := db.DB.First(&cfg).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}

// session cookie format: "<expiryUnix>.<hex(hmac(secret, expiryUnix))>"
func signSession(secret string, exp int64) string {
	payload := strconv.FormatInt(exp, 10)
	return payload + "." + hmacHex(secret, payload)
}

func validSession(secret, value string) bool {
	parts := strings.SplitN(value, ".", 2)
	if len(parts) != 2 {
		return false
	}
	if !hmac.Equal([]byte(parts[1]), []byte(hmacHex(secret, parts[0]))) {
		return false
	}
	exp, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return false
	}
	return time.Now().Unix() < exp
}

func hmacHex(secret, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func setSession(w http.ResponseWriter, r *http.Request, secret string) {
	exp := time.Now().Add(sessionMaxAge).Unix()
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    signSession(secret, exp),
		Path:     "/admin",
		HttpOnly: true,
		Secure:   isHTTPS(r),
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(exp, 0),
	})
}

func clearSession(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/admin",
		HttpOnly: true,
		Secure:   isHTTPS(r),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func isAuthed(r *http.Request) bool {
	c, err := r.Cookie(sessionCookie)
	if err != nil || c.Value == "" {
		return false
	}
	cfg, err := loadAdmin()
	if err != nil {
		return false
	}
	return validSession(cfg.SessionSecret, c.Value)
}

func isHTTPS(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

// login
func handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	cfg, err := loadAdmin()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "admin not configured"})
		return
	}
	if !hmac.Equal([]byte(hashPassword(body.Password, cfg.Salt)), []byte(cfg.PasswordHash)) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "wrong password"})
		return
	}
	setSession(w, r, cfg.SessionSecret)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	clearSession(w, r)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func handleMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"authenticated": isAuthed(r)})
}

func handleChangeAdminPassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Current string `json:"current"`
		New     string `json:"new"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if len(body.New) < 6 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "new password must be at least 6 characters"})
		return
	}
	cfg, err := loadAdmin()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "admin not configured"})
		return
	}
	if !hmac.Equal([]byte(hashPassword(body.Current, cfg.Salt)), []byte(cfg.PasswordHash)) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "current password is wrong"})
		return
	}
	cfg.Salt = randHex(16)
	cfg.PasswordHash = hashPassword(body.New, cfg.Salt)
	if err := db.DB.Save(cfg).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not save"})
		return
	}
	setSession(w, r, cfg.SessionSecret)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func randHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func randString(n int) string {
	const alphabet = "abcdefghjkmnpqrstuvwxyz23456789"
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	for i := range b {
		b[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(b)
}
