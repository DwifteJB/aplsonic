package admin

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/DwifteJB/aplsonic/src/applemusic"
	"github.com/DwifteJB/aplsonic/src/config"
	"github.com/DwifteJB/aplsonic/src/db"
	"github.com/DwifteJB/aplsonic/src/db/schema"
	"github.com/go-chi/chi/v5"
)

type userDTO struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`

	AdminRole    bool `json:"adminRole"`
	DownloadRole bool `json:"downloadRole"`

	HasAppleToken           bool       `json:"hasAppleToken"`
	AppleTokenStatus        string     `json:"appleTokenStatus"`
	AppleTokenExpiresAt     *time.Time `json:"appleTokenExpiresAt"`
	AppleTokenLastCheckedAt *time.Time `json:"appleTokenLastCheckedAt"`
	AppleTokenLastError     string     `json:"appleTokenLastError"`
}

func appleStatus(u schema.User) (status string, expiresAt *time.Time) {
	if !applemusic.HasMediaUserToken(u.AppleCookies) {
		return "missing", nil
	}
	expiresAt = u.AppleTokenExpiresAt
	if expiresAt == nil {
		if exp, ok := applemusic.MediaTokenExpiry(u.AppleCookies); ok {
			expiresAt = &exp
		}
	}
	now := time.Now()
	if (expiresAt != nil && expiresAt.Before(now)) || u.AppleTokenStatus == "expired" {
		return "expired", expiresAt
	}

	// to expire soon
	if warn := config.AppConfig.TokenWarnDays; warn > 0 && expiresAt != nil &&
		expiresAt.Before(now.Add(time.Duration(warn)*24*time.Hour)) {
		return "expiring", expiresAt
	}
	if u.AppleTokenStatus == "ok" {
		return "ok", expiresAt
	}
	return "unknown", expiresAt
}

func toDTO(u schema.User) userDTO {
	status, exp := appleStatus(u)
	return userDTO{
		ID:                      u.ID,
		Username:                u.Username,
		Email:                   u.Email,
		AdminRole:               u.AdminRole,
		DownloadRole:            u.DownloadRole,
		HasAppleToken:           applemusic.HasMediaUserToken(u.AppleCookies),
		AppleTokenStatus:        status,
		AppleTokenExpiresAt:     exp,
		AppleTokenLastCheckedAt: u.AppleTokenLastCheckedAt,
		AppleTokenLastError:     u.AppleTokenLastError,
	}
}

func handleListUsers(w http.ResponseWriter, r *http.Request) {
	var users []schema.User
	db.DB.Order("username ASC").Find(&users)
	out := make([]userDTO, len(users))
	for i, u := range users {
		out[i] = toDTO(u)
	}
	writeJSON(w, http.StatusOK, out)
}

func handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	body.Username = strings.TrimSpace(body.Username)
	if body.Username == "" || body.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username and password are required"})
		return
	}

	var existing int64
	db.DB.Model(&schema.User{}).Where("username = ?", body.Username).Count(&existing)
	if existing > 0 {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "username already exists"})
		return
	}

	salt := randHex(16)
	user := schema.User{
		Username:         body.Username,
		Email:            body.Email,
		Password:         hashPassword(body.Password, salt),
		Salt:             salt,
		TokenPassword:    body.Password,
		AppleTokenStatus: "missing",
	}
	if err := db.DB.Create(&user).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not create user"})
		return
	}
	writeJSON(w, http.StatusOK, toDTO(user))
}

func handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromURL(w, r)
	if !ok {
		return
	}
	var body struct {
		Password     *string `json:"password"`
		Email        *string `json:"email"`
		DownloadRole *bool   `json:"downloadRole"`
		AdminRole    *bool   `json:"adminRole"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if body.Password != nil {
		if *body.Password == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "password cannot be empty"})
			return
		}
		user.Salt = randHex(16)
		user.Password = hashPassword(*body.Password, user.Salt)
		user.TokenPassword = *body.Password
	}
	if body.Email != nil {
		user.Email = *body.Email
	}
	if body.DownloadRole != nil {
		user.DownloadRole = *body.DownloadRole
	}
	if body.AdminRole != nil {
		user.AdminRole = *body.AdminRole
	}
	if err := db.DB.Save(user).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not save"})
		return
	}
	writeJSON(w, http.StatusOK, toDTO(*user))
}

func handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromURL(w, r)
	if !ok {
		return
	}
	if err := db.DB.Delete(user).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not delete"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func userFromURL(w http.ResponseWriter, r *http.Request) (*schema.User, bool) {
	id := chi.URLParam(r, "id")
	var user schema.User
	if err := db.DB.First(&user, "id = ?", id).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return nil, false
	}
	return &user, true
}
