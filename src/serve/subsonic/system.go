package subsonic

import "net/http"

// handles ping
func Ping(w http.ResponseWriter, r *http.Request) {
	if _, code, msg := Authenticate(r); code != 0 {
		Fail(w, r, code, msg)
		return
	}
	OK(w, r)
}

// get license handles GET/POST /rest/getLicense
// since we are using open subsonic, this is ALWAYS perm!!!
func GetLicense(w http.ResponseWriter, r *http.Request) {
	OK(w, r, func(resp *response) {
		resp.License = &LicenseBody{
			Valid:          true,
			Email:          "me@rmfosho.me",
			LicenseExpires: "2099-01-01T00:00:00.000Z",
		}
	})
}
