package subsonic

import (
	"net/http"
	"time"

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

func holdUntilDownloaded(w http.ResponseWriter, r *http.Request, user *schema.User, song *schema.Song) error {
	interval := config.AppConfig.StreamKeepalive
	if interval <= 0 {
		return download.EnsureSong(user, song)
	}

	done := make(chan error, 1)
	go func() { done <- download.EnsureSong(user, song) }()

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case err := <-done:
			return err
		case <-ticker.C:
			w.WriteHeader(http.StatusProcessing)
		case <-r.Context().Done():
			return r.Context().Err()
		}
	}
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

	if !attachment && config.AppConfig.DownloadPlaylist {
		go download.EnsureSongPlaylists(user, id)
	}

	// check if storage has song
	if !storage.Has(id) {
		if err := holdUntilDownloaded(w, r, user, &song); err != nil {
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
