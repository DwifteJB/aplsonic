package subsonic

import (
	"net/http"

	"github.com/DwifteJB/aplsonic/src/config"
	"github.com/DwifteJB/aplsonic/src/db"
	"github.com/DwifteJB/aplsonic/src/db/schema"
	"github.com/DwifteJB/aplsonic/src/download"
	"github.com/DwifteJB/aplsonic/src/storage"
)

// stream serve audio
func Stream(w http.ResponseWriter, r *http.Request) {
	serveAudio(w, r, false)
}

// serve as an attachment 
func Download(w http.ResponseWriter, r *http.Request) {
	serveAudio(w, r, true)
}

func serveAudio(w http.ResponseWriter, r *http.Request, attachment bool) {
	user, code, msg := Authenticate(r)
	if code != 0 {
		Fail(w, r, code, msg)
		return
	}
	if attachment && !user.DownloadRole {
		Fail(w, r, 50, "User is not allowed to download.")
		return
	}
	if !attachment && !user.StreamRole {
		Fail(w, r, 50, "User is not allowed to stream.")
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		Fail(w, r, 10, "Required parameter 'id' is missing.")
		return
	}

	var song schema.Song
	if res := db.DB.First(&song, "id = ?", id); res.Error != nil {
		Fail(w, r, 70, "Song not found.")
		return
	}

	// if playAlbum, then download rest of the album in the background
	if !attachment && config.AppConfig.Download == "playAlbum" && song.AlbumID != "" {
		go download.EnsureAlbum(user, song.AlbumID)
	}

	// check if storage has song
	if !storage.Has(id) {
		if err := download.EnsureSong(user, &song); err != nil {
			Fail(w, r, 0, "could not download song: "+err.Error())
			return
		}
	}

	obj, info, err := storage.Get(id)
	if err != nil {
		Fail(w, r, 0, "could not read song: "+err.Error())
		return
	}
	defer obj.Close()

	contentType := song.ContentType
	if contentType == "" {
		contentType = "audio/mp4"
	}
	w.Header().Set("Content-Type", contentType)
	if attachment {
		w.Header().Set("Content-Disposition", `attachment; filename="`+id+`.m4a"`)
	}

	// ServeContent handles range/206, content-length, accept-ranges and 416
	http.ServeContent(w, r, id+".m4a", info.LastModified, obj)
}
