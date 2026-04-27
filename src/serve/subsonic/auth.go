package subsonic

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/DwifteJB/aplsonic/src/db"
	"github.com/DwifteJB/aplsonic/src/db/schema"
)

// supports:
//   - Password auth: p=<password> or p=enc:<hex-encoded password>
//   - Token auth:    t=MD5(password+s) + s=<client-chosen salt>
func Authenticate(r *http.Request) (*schema.User, int, string) {
	q := r.URL.Query()
	username := q.Get("u")
	if username == "" {
		return nil, 10, "Required parameter 'u' is missing."
	}

	var user schema.User
	if res := db.DB.Where("username = ?", username).First(&user); res.Error != nil {
		return nil, 40, "Wrong username or password."
	}

	t, s := q.Get("t"), q.Get("s")
	p := q.Get("p")

	// token authentication (for clients that support it, and to avoid sending passwords over the wire)
	if t != "" && user.TokenPassword != "" {
		if s == "" {
			return nil, 10, "Required parameter 's' is missing."
		}
		sum := md5.Sum([]byte(user.TokenPassword + s))
		computed := hex.EncodeToString(sum[:])
		if computed != strings.ToLower(t) {
			return nil, 40, "Wrong username or password."
		}
		return &user, 0, ""
	}

	if p == "" {
		return nil, 10, "Required parameter 'p' or 't'+'s' is missing."
	}

	// password auth (encrypted, or plain)
	password := p
	if strings.HasPrefix(p, "enc:") {
		decoded, err := hex.DecodeString(p[4:])
		if err != nil {
			return nil, 40, "Wrong username or password."
		}
		password = string(decoded)
	}

	sum := sha256.Sum256([]byte(password + user.Salt))
	computed := hex.EncodeToString(sum[:])
	if computed != user.Password {
		return nil, 40, "Wrong username or password."
	}

	return &user, 0, ""
}
