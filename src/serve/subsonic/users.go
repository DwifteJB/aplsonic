package subsonic

import (
	"net/http"

	"github.com/DwifteJB/aplsonic/src/db"
	"github.com/DwifteJB/aplsonic/src/db/schema"
)

func userBody(u *schema.User) UserBody {
	return UserBody{
		Username:            u.Username,
		Email:               u.Email,
		ScrobblingEnabled:   u.ScrobblingEnabled,
		MaxBitRate:          u.MaxBitRate,
		AdminRole:           u.AdminRole,
		SettingsRole:        u.SettingsRole,
		DownloadRole:        u.DownloadRole,
		UploadRole:          u.UploadRole,
		PlaylistRole:        u.PlaylistRole,
		CoverArtRole:        u.CoverArtRole,
		CommentRole:         u.CommentRole,
		PodcastRole:         u.PodcastRole,
		StreamRole:          u.StreamRole,
		JukeboxRole:         u.JukeboxRole,
		ShareRole:           u.ShareRole,
		VideoConversionRole: u.VideoConversionRole,
	}
}

func GetAvatar(w http.ResponseWriter, r *http.Request) {
	// TODO: send some image bytes (maybe use funny cat pics lol)
}

// handles getting user info
func GetUser(w http.ResponseWriter, r *http.Request) {
	authed, code, msg := Authenticate(r)
	if code != 0 {
		Fail(w, r, code, msg)
		return
	}

	target := r.URL.Query().Get("username")
	if target == "" {
		Fail(w, r, 10, "Required parameter 'username' is missing.")
		return
	}

	if !authed.AdminRole && target != authed.Username {
		Fail(w, r, 50, "User is not authorized for the given operation.")
		return
	}

	var user schema.User
	if res := db.DB.Where("username = ?", target).First(&user); res.Error != nil {
		Fail(w, r, 70, "User not found.")
		return
	}

	OK(w, r, func(resp *response) {
		body := userBody(&user)
		resp.User = &body
	})
}

func GetUsers(w http.ResponseWriter, r *http.Request) {
	authed, code, msg := Authenticate(r)
	if code != 0 {
		Fail(w, r, code, msg)
		return
	}

	if !authed.AdminRole {
		Fail(w, r, 50, "User is not authorized for the given operation.")
		return
	}

	var users []schema.User
	if res := db.DB.Find(&users); res.Error != nil {
		Fail(w, r, 0, "Database error.")
		return
	}

	bodies := make([]UserBody, len(users))
	for i, u := range users {
		bodies[i] = userBody(&u)
	}

	OK(w, r, func(resp *response) {
		resp.Users = &UsersBody{User: bodies}
	})
}
