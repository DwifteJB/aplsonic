package subsonic

import (
	"net/http"

	"github.com/DwifteJB/aplsonic/src/db"
	"github.com/DwifteJB/aplsonic/src/db/schema"
)

// getRandomSongs handles GET/POST /rest/getRandomSongs
func GetRandomSongs(w http.ResponseWriter, r *http.Request) {
	_, code, msg := Authenticate(r)
	if code != 0 {
		Fail(w, r, code, msg)
		return
	}

	size := intParam(r, "size", 10)
	if size > 500 {
		size = 500
	}

	tx := db.DB.Order("RAND()").Limit(size)

	if genre := r.URL.Query().Get("genre"); genre != "" {
		tx = tx.Where("genre = ?", genre)
	}

	from := intParam(r, "fromYear", 0)
	to := intParam(r, "toYear", 0)
	if from > 0 {
		tx = tx.Where("year >= ?", from)
	}
	if to > 0 {
		tx = tx.Where("year <= ?", to)
	}

	var songs []schema.Song
	if res := tx.Find(&songs); res.Error != nil {
		Fail(w, r, 0, res.Error.Error())
		return
	}

	children := make([]ChildBody, len(songs))
	for i, s := range songs {
		children[i] = songToChild(s)
	}

	OK(w, r, func(resp *response) {
		resp.RandomSongs = &RandomSongsBody{Song: children}
	})
}
