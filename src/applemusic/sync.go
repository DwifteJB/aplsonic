package applemusic

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
	"gorm.io/gorm/clause"

	"github.com/DwifteJB/aplsonic/src/db"
	"github.com/DwifteJB/aplsonic/src/db/schema"
)

// syncing playlists so apple music playlists are reflected in the local db
const playlistSyncTTL = 15 * time.Minute

var (
	playlistSyncGroup singleflight.Group
	playlistSyncMu    sync.Mutex
	playlistLastSync  = map[string]time.Time{}
)

// only if config.SyncOnSearch = true
func SyncSearchResults(results *SearchResults) {
	if results.Albums != nil {
		syncAlbums(results.Albums.Data)
	}
	if results.Songs != nil {
		syncSongs(results.Songs.Data)
	}
	if results.Artists != nil {
		syncArtists(results.Artists.Data)
	}
}

func SyncAlbum(r *Resource) {
	syncAlbums([]Resource{*r})
	if r.Relationships.Tracks != nil {
		for i := range r.Relationships.Tracks.Data {
			r.Relationships.Tracks.Data[i].Relationships.Albums = &ResourceList{
				Data: []Resource{{ID: r.ID}},
			}
		}
		syncSongs(r.Relationships.Tracks.Data)
	}
}

func SyncArtist(r *Resource) {
	syncArtists([]Resource{*r})
	if r.Relationships.Albums != nil {
		syncAlbums(r.Relationships.Albums.Data)
	}
}

func syncArtists(resources []Resource) {
	for _, r := range resources {
		artist := schema.Artist{
			ID:   r.ID,
			Name: r.Attributes.Name,
		}
		if r.Attributes.Artwork != nil {
			artist.CoverArt = FormatArtworkURL(r.Attributes.Artwork.URL)
		}
		db.DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(&artist)
	}
}

func syncAlbums(resources []Resource) {
	for _, r := range resources {
		album := schema.Album{
			ID:            r.ID,
			Name:          r.Attributes.Name,
			Artist:        r.Attributes.ArtistName,
			DisplayArtist: r.Attributes.ArtistName,
			SongCount:     r.Attributes.TrackCount,
			Duration:      int(r.Attributes.DurationInMillis / 1000),
			IsCompilation: r.Attributes.IsCompilation,
		}

		if len(r.Attributes.GenreNames) > 0 {
			album.Genre = r.Attributes.GenreNames[0]
		}
		if r.Attributes.Artwork != nil {
			album.CoverArt = FormatArtworkURL(r.Attributes.Artwork.URL)
		}
		if r.Attributes.ReleaseDate != "" {
			album.Year = parseYear(r.Attributes.ReleaseDate)
		}
		if r.Attributes.RecordLabel != "" {
			album.RecordLabels = r.Attributes.RecordLabel
		}

		// get a stable artist ID from the name
		if r.Attributes.ArtistName != "" {
			album.ArtistID = artistIDFromName(r.Attributes.ArtistName)
		}

		// create a stub artist to link to (albums come before artists in results)
		if album.ArtistID != "" {
			artist := schema.Artist{ID: album.ArtistID, Name: r.Attributes.ArtistName}
			if album.CoverArt != "" {
				artist.CoverArt = album.CoverArt
			}
			db.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&artist)
		}

		db.DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(&album)
	}
}

func syncSongs(resources []Resource) {
	for _, r := range resources {
		song := schema.Song{
			ID:              r.ID,
			Title:           r.Attributes.Name,
			Album:           r.Attributes.AlbumName,
			Artist:          r.Attributes.ArtistName,
			DisplayArtist:   r.Attributes.ArtistName,
			Track:           r.Attributes.TrackNumber,
			DiscNumber:      r.Attributes.DiscNumber,
			Duration:        int(r.Attributes.DurationInMillis / 1000),
			ISRC:            r.Attributes.ISRC,
			DisplayComposer: r.Attributes.ComposerName,
			ExplicitStatus:  r.Attributes.ContentRating,
			BPM:             r.Attributes.CurrentBPM,
		}

		if len(r.Attributes.GenreNames) > 0 {
			song.Genre = r.Attributes.GenreNames[0]
		}
		if r.Attributes.Artwork != nil {
			song.CoverArt = FormatArtworkURL(r.Attributes.Artwork.URL)
		}
		if r.Attributes.ReleaseDate != "" {
			song.Year = parseYear(r.Attributes.ReleaseDate)
		}
		if r.Attributes.ArtistName != "" {
			song.ArtistID = artistIDFromName(r.Attributes.ArtistName)
		}

		// link to relationship
		if r.Relationships.Albums != nil && len(r.Relationships.Albums.Data) > 0 {
			song.AlbumID = r.Relationships.Albums.Data[0].ID
		}

		db.DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(&song)
	}
}

func SyncAllPlaylists() {
	var users []schema.User
	db.DB.Find(&users)
	for i := range users {
		u := users[i]
		if !HasMediaUserToken(u.AppleCookies) {
			continue
		}
		if err := SyncUserPlaylists(&u); err != nil {
			fmt.Printf("startup playlist sync failed for %s: %v\n", u.Username, err)
			continue
		}
		var count int64
		db.DB.Model(&schema.Playlist{}).Where("owner = ?", u.Username).Count(&count)
		fmt.Printf("startup playlist sync: %s has %d playlists\n", u.Username, count)
	}
}

// self explanatory, checks timer
func SyncUserPlaylists(user *schema.User) error {
	var count int64
	db.DB.Model(&schema.Playlist{}).Where("owner = ?", user.Username).Count(&count)

	if count > 0 {
		playlistSyncMu.Lock()
		last := playlistLastSync[user.Username]
		playlistSyncMu.Unlock()
		if !last.IsZero() && time.Since(last) < playlistSyncTTL {
			return nil
		}
	}

	_, err, _ := playlistSyncGroup.Do(user.Username, func() (any, error) {
		return nil, syncUserPlaylists(user)
	})
	return err
}

// fetches from apple music updates / creates to local db
func syncUserPlaylists(user *schema.User) error {
	client, err := NewClientFromCookies(user.AppleCookies)
	if err != nil {
		return err
	}

	playlists, err := client.GetLibraryPlaylists()
	if err != nil {
		return err
	}

	for _, p := range playlists {
		tracks, err := client.GetLibraryPlaylistTracks(p.ID)
		if err != nil {
			continue
		}

		var songIDs []string
		var trackResources []Resource
		for _, t := range tracks {
			catID := ""
			if t.Attributes.PlayParams != nil {
				catID = t.Attributes.PlayParams.CatalogID
				if catID == "" {
					catID = t.Attributes.PlayParams.PurchasedID
				}
			}
			if catID == "" {
				continue
			}
			t.ID = catID
			songIDs = append(songIDs, catID)
			trackResources = append(trackResources, t)
		}

		syncSongs(trackResources)

		pl := schema.Playlist{
			ID:      p.ID,
			AppleID: p.ID,
			Source:  "apple",
			Owner:   user.Username,
			Name:    p.Attributes.Name,
			Public:  p.Attributes.IsPublic,
		}
		if p.Attributes.Description != nil {
			pl.Comment = p.Attributes.Description.Standard
		}
		if p.Attributes.Artwork != nil {
			pl.CoverArt = FormatArtworkURL(p.Attributes.Artwork.URL)
		}
		pl.Fingerprint = playlistFingerprint(pl.Name, songIDs)
		now := time.Now()
		pl.SyncedAt = &now

		var existing schema.Playlist
		unchanged := db.DB.First(&existing, "id = ?", p.ID).Error == nil && existing.Fingerprint == pl.Fingerprint

		db.DB.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"apple_id", "source", "owner", "name", "public", "comment", "cover_art", "fingerprint", "synced_at",
			}),
		}).Create(&pl)

		if unchanged {
			continue
		}

		db.DB.Where("playlist_id = ?", p.ID).Delete(&schema.PlaylistEntry{})
		for i, sid := range songIDs {
			db.DB.Create(&schema.PlaylistEntry{PlaylistID: p.ID, SongID: sid, Position: i})
		}
	}

	playlistSyncMu.Lock()
	playlistLastSync[user.Username] = time.Now()
	playlistSyncMu.Unlock()
	return nil
}

func playlistFingerprint(name string, songIDs []string) string {
	h := sha256.New()
	h.Write([]byte(name))
	h.Write([]byte{0})
	for _, id := range songIDs {
		h.Write([]byte(id))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// derive 16 bit artist ID
func artistIDFromName(name string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(name))))
	return hex.EncodeToString(sum[:8])
}

func FormatArtworkURL(rawURL string) string {
	u := strings.Replace(rawURL, "{w}", "500", 1)
	u = strings.Replace(u, "{h}", "500", 1)
	return u
}

func parseYear(date string) int {
	if len(date) >= 4 {
		y, _ := strconv.Atoi(date[:4])
		return y
	}
	return 0
}

func UpdateAlbumSongCounts() {
	db.DB.Exec(`
		UPDATE albums a
		SET song_count = (SELECT COUNT(*) FROM songs s WHERE s.album_id = a.id AND s.deleted_at IS NULL)
		WHERE a.deleted_at IS NULL
	`)
}
