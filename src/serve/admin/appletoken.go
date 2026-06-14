package admin

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/DwifteJB/aplsonic/src/applemusic"
	"github.com/DwifteJB/aplsonic/src/db"
)

// reads apple tokens
func handleReplenish(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromURL(w, r)
	if !ok {
		return
	}
	var body struct {
		Cookies string `json:"cookies"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	cookies := body.Cookies
	if !applemusic.HasMediaUserToken(cookies) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no media-user-token found in the uploaded cookies.txt"})
		return
	}
	var expiresAt *time.Time
	if exp, found := applemusic.MediaTokenExpiry(cookies); found {
		expiresAt = &exp
	}

	user.AppleCookies = cookies
	user.AppleTokenExpiresAt = expiresAt
	now := time.Now()
	user.AppleTokenLastCheckedAt = &now

	// validates
	status, validationErr := validate(cookies)
	user.AppleTokenStatus = status
	if validationErr != "" {
		user.AppleTokenLastError = validationErr
	} else {
		user.AppleTokenLastError = ""
	}

	if err := db.DB.Save(user).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not save"})
		return
	}
	writeJSON(w, http.StatusOK, toDTO(*user))
}

// handleRecheck re-validates a user's stored credentials on demand.
func handleRecheck(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromURL(w, r)
	if !ok {
		return
	}
	status, validationErr := validate(user.AppleCookies)
	now := time.Now()
	user.AppleTokenStatus = status
	user.AppleTokenLastCheckedAt = &now
	user.AppleTokenLastError = validationErr
	db.DB.Save(user)
	writeJSON(w, http.StatusOK, toDTO(*user))
}

// validate reports a jar's health. The per-user media-user-token can't be
// live-checked (catalog calls go through the shared browser, which isn't logged in
// as the user), so this checks token presence + the cookie's own expiry date.
func validate(cookies string) (status, errMsg string) {
	if !applemusic.HasMediaUserToken(cookies) {
		return "missing", "no media-user-token in cookies"
	}
	if exp, ok := applemusic.MediaTokenExpiry(cookies); ok && exp.Before(time.Now()) {
		return "expired", "media-user-token cookie expired"
	}
	return "ok", ""
}
