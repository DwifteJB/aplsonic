package subsonic

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/DwifteJB/aplsonic/src/applemusic"
	"github.com/DwifteJB/aplsonic/src/config"
	"github.com/DwifteJB/aplsonic/src/db"
	"github.com/DwifteJB/aplsonic/src/db/schema"
	"github.com/DwifteJB/aplsonic/src/download"
)

func GetPlaylists(w http.ResponseWriter, r *http.Request) {
	user, code, msg := Authenticate(r)
	if code != 0 {
		Fail(w, r, code, msg)
		return
	}
	if !user.PlaylistRole {
		Fail(w, r, 50, "User is not allowed to use playlists.")
		return
	}

	if applemusic.HasMediaUserToken(user.AppleCookies) {
		if err := applemusic.SyncUserPlaylists(user); err != nil {
			fmt.Printf("playlist sync failed for %s: %v\n", user.Username, err)
		}
	}

	var playlists []schema.Playlist
	db.DB.Where("owner = ?", user.Username).Order("name ASC").Find(&playlists)

	bodies := make([]PlaylistBody, len(playlists))
	for i, p := range playlists {
		_, count, duration := loadPlaylistSongs(p.ID)
		bodies[i] = playlistToBody(p, count, duration)
	}

	OK(w, r, func(resp *response) {
		resp.Playlists = &PlaylistsBody{Playlist: bodies}
	})
}

func GetPlaylist(w http.ResponseWriter, r *http.Request) {
	user, code, msg := Authenticate(r)
	if code != 0 {
		Fail(w, r, code, msg)
		return
	}
	if !user.PlaylistRole {
		Fail(w, r, 50, "User is not allowed to use playlists.")
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		Fail(w, r, 10, "Required parameter 'id' is missing.")
		return
	}

	var playlist schema.Playlist
	if db.DB.First(&playlist, "id = ? AND owner = ?", id, user.Username).Error != nil {
		Fail(w, r, 70, "Playlist not found.")
		return
	}

	songs, count, duration := loadPlaylistSongs(id)

	if config.AppConfig.Download == "playAlbum" {
		go download.EnsurePlaylist(user, id)
	}

	body := PlaylistWithSongsBody{PlaylistBody: playlistToBody(playlist, count, duration)}
	body.Entry = make([]ChildBody, len(songs))
	for i, s := range songs {
		body.Entry[i] = songToChild(s)
	}

	OK(w, r, func(resp *response) {
		resp.Playlist = &body
	})
}

func CreatePlaylist(w http.ResponseWriter, r *http.Request) {
	user, code, msg := Authenticate(r)
	if code != 0 {
		Fail(w, r, code, msg)
		return
	}
	if !user.PlaylistRole {
		Fail(w, r, 50, "User is not allowed to use playlists.")
		return
	}

	q := r.URL.Query()
	if existingID := q.Get("playlistId"); existingID != "" {
		updatePlaylistSongs(w, r, user, existingID, q.Get("name"), q["songId"])
		return
	}

	name := q.Get("name")
	if name == "" {
		Fail(w, r, 10, "Required parameter 'name' is missing.")
		return
	}
	songIDs := q["songId"]

	id := localPlaylistID(user.Username, name)
	now := time.Now()
	playlist := schema.Playlist{
		ID:        id,
		Source:    "local",
		Owner:     user.Username,
		Name:      name,
		Public:    false,
		CreatedAt: now,
		UpdatedAt: now,
	}
	db.DB.Create(&playlist)
	for i, sid := range songIDs {
		db.DB.Create(&schema.PlaylistEntry{PlaylistID: id, SongID: sid, Position: i})
	}

	if applemusic.HasMediaUserToken(user.AppleCookies) {
		if client, err := applemusic.NewClientFromCookies(user.AppleCookies); err == nil {
			if appleID, err := client.CreateLibraryPlaylist(name, "", songIDs); err == nil {
				db.DB.Model(&schema.Playlist{}).Where("id = ?", id).Update("apple_id", appleID)
			} else {
				fmt.Printf("create playlist on apple failed: %v\n", err)
			}
		}
	}

	songs, count, duration := loadPlaylistSongs(id)
	db.DB.First(&playlist, "id = ?", id)
	body := PlaylistWithSongsBody{PlaylistBody: playlistToBody(playlist, count, duration)}
	body.Entry = make([]ChildBody, len(songs))
	for i, s := range songs {
		body.Entry[i] = songToChild(s)
	}

	OK(w, r, func(resp *response) {
		resp.Playlist = &body
	})
}

func UpdatePlaylist(w http.ResponseWriter, r *http.Request) {
	user, code, msg := Authenticate(r)
	if code != 0 {
		Fail(w, r, code, msg)
		return
	}
	if !user.PlaylistRole {
		Fail(w, r, 50, "User is not allowed to use playlists.")
		return
	}

	q := r.URL.Query()
	id := q.Get("playlistId")
	if id == "" {
		Fail(w, r, 10, "Required parameter 'playlistId' is missing.")
		return
	}

	var playlist schema.Playlist
	if db.DB.First(&playlist, "id = ? AND owner = ?", id, user.Username).Error != nil {
		Fail(w, r, 70, "Playlist not found.")
		return
	}

	updates := map[string]any{}
	if name := q.Get("name"); name != "" {
		updates["name"] = name
	}
	if comment := q.Get("comment"); comment != "" {
		updates["comment"] = comment
	}
	if pub := q.Get("public"); pub != "" {
		updates["public"] = pub == "true"
	}
	if len(updates) > 0 {
		db.DB.Model(&schema.Playlist{}).Where("id = ?", id).Updates(updates)
	}

	toAdd := q["songIdToAdd"]
	toRemove := q["songIndexToRemove"]

	if len(toAdd) > 0 || len(toRemove) > 0 {
		songIDs, _, _ := playlistSongIDs(id)

		removeSet := map[int]bool{}
		for _, idx := range toRemove {
			if n, err := strconv.Atoi(idx); err == nil {
				removeSet[n] = true
			}
		}

		var kept []string
		for i, sid := range songIDs {
			if !removeSet[i] {
				kept = append(kept, sid)
			}
		}
		kept = append(kept, toAdd...)

		db.DB.Where("playlist_id = ?", id).Delete(&schema.PlaylistEntry{})
		for i, sid := range kept {
			db.DB.Create(&schema.PlaylistEntry{PlaylistID: id, SongID: sid, Position: i})
		}

		if len(toAdd) > 0 && playlist.AppleID != "" && applemusic.HasMediaUserToken(user.AppleCookies) {
			if client, err := applemusic.NewClientFromCookies(user.AppleCookies); err == nil {
				if err := client.AddTracksToLibraryPlaylist(playlist.AppleID, toAdd); err != nil {
					fmt.Printf("add tracks on apple failed: %v\n", err)
				}
			}
		}
	}

	db.DB.Model(&schema.Playlist{}).Where("id = ?", id).Update("updated_at", time.Now())
	OK(w, r)
}

func DeletePlaylist(w http.ResponseWriter, r *http.Request) {
	user, code, msg := Authenticate(r)
	if code != 0 {
		Fail(w, r, code, msg)
		return
	}
	if !user.PlaylistRole {
		Fail(w, r, 50, "User is not allowed to use playlists.")
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		Fail(w, r, 10, "Required parameter 'id' is missing.")
		return
	}

	var playlist schema.Playlist
	if db.DB.First(&playlist, "id = ? AND owner = ?", id, user.Username).Error != nil {
		Fail(w, r, 70, "Playlist not found.")
		return
	}

	db.DB.Where("playlist_id = ?", id).Delete(&schema.PlaylistEntry{})
	db.DB.Delete(&schema.Playlist{}, "id = ?", id)
	OK(w, r)
}

func updatePlaylistSongs(w http.ResponseWriter, r *http.Request, user *schema.User, id, name string, songIDs []string) {
	var playlist schema.Playlist
	if db.DB.First(&playlist, "id = ? AND owner = ?", id, user.Username).Error != nil {
		Fail(w, r, 70, "Playlist not found.")
		return
	}

	if name != "" {
		db.DB.Model(&schema.Playlist{}).Where("id = ?", id).Update("name", name)
	}

	existing, _, _ := playlistSongIDs(id)
	existingSet := map[string]bool{}
	for _, sid := range existing {
		existingSet[sid] = true
	}
	var added []string
	for _, sid := range songIDs {
		if !existingSet[sid] {
			added = append(added, sid)
		}
	}

	db.DB.Where("playlist_id = ?", id).Delete(&schema.PlaylistEntry{})
	for i, sid := range songIDs {
		db.DB.Create(&schema.PlaylistEntry{PlaylistID: id, SongID: sid, Position: i})
	}

	if len(added) > 0 && playlist.AppleID != "" && applemusic.HasMediaUserToken(user.AppleCookies) {
		if client, err := applemusic.NewClientFromCookies(user.AppleCookies); err == nil {
			if err := client.AddTracksToLibraryPlaylist(playlist.AppleID, added); err != nil {
				fmt.Printf("add tracks on apple failed: %v\n", err)
			}
		}
	}

	OK(w, r)
}

func playlistSongIDs(playlistID string) ([]string, int, int) {
	var entries []schema.PlaylistEntry
	db.DB.Where("playlist_id = ?", playlistID).Order("position ASC").Find(&entries)
	ids := make([]string, len(entries))
	for i, e := range entries {
		ids[i] = e.SongID
	}
	return ids, len(ids), 0
}

func loadPlaylistSongs(playlistID string) ([]schema.Song, int, int) {
	var entries []schema.PlaylistEntry
	db.DB.Where("playlist_id = ?", playlistID).Order("position ASC").Find(&entries)

	songs := make([]schema.Song, 0, len(entries))
	duration := 0
	for _, e := range entries {
		var song schema.Song
		if db.DB.First(&song, "id = ?", e.SongID).Error != nil {
			continue
		}
		songs = append(songs, song)
		duration += song.Duration
	}
	return songs, len(songs), duration
}

func playlistToBody(p schema.Playlist, count, duration int) PlaylistBody {
	return PlaylistBody{
		ID:        p.ID,
		Name:      p.Name,
		Comment:   p.Comment,
		Owner:     p.Owner,
		Public:    p.Public,
		SongCount: count,
		Duration:  duration,
		Created:   p.CreatedAt.Format(time.RFC3339),
		Changed:   p.UpdatedAt.Format(time.RFC3339),
		CoverArt:  p.CoverArt,
	}
}

func localPlaylistID(owner, name string) string {
	sum := sha256.Sum256([]byte(owner + "\x00" + name + "\x00" + strconv.FormatInt(time.Now().UnixNano(), 10)))
	return "local." + hex.EncodeToString(sum[:8])
}
