package subsonic

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/DwifteJB/aplsonic/src/applemusic"
	"github.com/DwifteJB/aplsonic/src/config"
	"github.com/DwifteJB/aplsonic/src/db"
	"github.com/DwifteJB/aplsonic/src/db/schema"
)

const (
	maxSimilarArtists = 20
	maxTopSongs       = 100
)

type artistHint struct {
	Name    string
	AppleID string
}

var (
	artistHints     = newTTLCache[artistHint](24*time.Hour, 8192)
	artistInfoCache = newTTLCache[*applemusic.Resource](time.Hour, 256)
	topSongsCache   = newTTLCache[[]applemusic.Resource](time.Hour, 256)
	artistSyncGroup singleflight.Group
	artistInfoGroup singleflight.Group
	topSongsGroup   singleflight.Group
)

func artistRef(name string) string {
	if name == "" {
		return ""
	}
	id := applemusic.ArtistIDFromName(name)
	if _, ok := artistHints.get(id); !ok {
		artistHints.set(id, artistHint{Name: name})
	}
	return id
}

func rememberAppleArtist(r applemusic.Resource) {
	hint := artistHint{Name: r.Attributes.Name, AppleID: r.ID}
	artistHints.set(r.ID, hint)
	artistHints.set(applemusic.ArtistIDFromName(hint.Name), hint)
}

func findArtist(id string) schema.Artist {
	var artist schema.Artist
	if db.DB.First(&artist, "id = ?", id).Error != nil {
		var song schema.Song
		db.DB.Select("artist").Limit(1).Find(&song, "artist_id = ?", id)
		artist = schema.Artist{ID: id, Name: song.Artist}
	}
	if hint, ok := artistHints.get(id); ok {
		if artist.Name == "" {
			artist.Name = hint.Name
		}
		if artist.AppleID == "" {
			artist.AppleID = hint.AppleID
		}
	}
	return artist
}

func syncArtist(user *schema.User, artist *schema.Artist) error {
	v, err, _ := artistSyncGroup.Do(artist.ID, func() (any, error) {
		client, err := applemusic.NewClientFromCookies(user.AppleCookies)
		if err != nil {
			return nil, err
		}
		synced := *artist
		if err := applemusic.SyncArtist(client, &synced); err != nil {
			return nil, err
		}
		return synced, nil
	})
	if err != nil {
		return err
	}
	*artist = v.(schema.Artist)
	return nil
}

func artistAppleID(client *applemusic.Client, artist schema.Artist) (string, error) {
	if artist.AppleID != "" {
		return artist.AppleID, nil
	}
	if artist.ID != applemusic.ArtistIDFromName(artist.Name) {
		return artist.ID, nil
	}
	found, err := client.FindArtistID(artist.Name)
	if err != nil {
		return "", err
	}
	if found != "" {
		artistHints.set(artist.ID, artistHint{Name: artist.Name, AppleID: found})
	}
	return found, nil
}

func artistInfo(client *applemusic.Client, artist schema.Artist) (*applemusic.Resource, error) {
	appleID, err := artistAppleID(client, artist)
	if err != nil || appleID == "" {
		return nil, err
	}
	if info, ok := artistInfoCache.get(appleID); ok {
		return info, nil
	}
	v, err, _ := artistInfoGroup.Do(appleID, func() (any, error) {
		info, err := client.GetArtistInfo(appleID, maxSimilarArtists)
		if err != nil {
			return nil, err
		}
		artistInfoCache.set(appleID, info)
		return info, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*applemusic.Resource), nil
}

func artistTopSongs(client *applemusic.Client, artist schema.Artist, count int) ([]applemusic.Resource, error) {
	appleID, err := artistAppleID(client, artist)
	if err != nil || appleID == "" {
		return nil, err
	}
	key := appleID + ":" + strconv.Itoa(count)
	if songs, ok := topSongsCache.get(key); ok {
		return songs, nil
	}
	v, err, _ := topSongsGroup.Do(key, func() (any, error) {
		songs, err := client.GetArtistTopSongs(appleID, count)
		if err != nil {
			return nil, err
		}
		topSongsCache.set(key, songs)
		return songs, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]applemusic.Resource), nil
}

func GetArtistInfo2(w http.ResponseWriter, r *http.Request) {
	user, code, msg := Authenticate(r)
	if code != 0 {
		Fail(w, r, code, msg)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		Fail(w, r, 10, "Required parameter 'id' is missing.")
		return
	}
	count := min(intParam(r, "count", maxSimilarArtists), maxSimilarArtists)

	artist := findArtist(id)
	image := artist.CoverArt
	var body ArtistInfo2Body

	client, err := applemusic.NewClientFromCookies(user.AppleCookies)
	var info *applemusic.Resource
	if err == nil {
		info, err = artistInfo(client, artist)
	}
	if err != nil {
		fmt.Printf("getArtistInfo2: artist %s: %v\n", id, err)
	}
	if info != nil {
		body.Biography = info.Attributes.ArtistBio
		if info.Attributes.Artwork != nil {
			image = applemusic.FormatArtworkURL(info.Attributes.Artwork.URL)
		}
		stars := loadStars(user.Username)
		for _, res := range firstResources(info.Views["similar-artists"], count) {
			rememberAppleArtist(res)
			similar := appleArtistToID3(res)
			similar.Starred = stars["artist:"+similar.ID]
			body.SimilarArtist = append(body.SimilarArtist, similar)
		}
	}

	if image != "" {
		body.SmallImageURL = replaceSize(image, "300")
		body.MediumImageURL = replaceSize(image, "600")
		body.LargeImageURL = replaceSize(image, "1200")
	}

	OK(w, r, func(resp *response) {
		resp.ArtistInfo2 = &body
	})
}

func GetTopSongs(w http.ResponseWriter, r *http.Request) {
	user, code, msg := Authenticate(r)
	if code != 0 {
		Fail(w, r, code, msg)
		return
	}

	name := r.URL.Query().Get("artist")
	if name == "" {
		Fail(w, r, 10, "Required parameter 'artist' is missing.")
		return
	}
	count := min(intParam(r, "count", 50), maxTopSongs)

	var songs []applemusic.Resource
	if count > 0 {
		client, err := applemusic.NewClientFromCookies(user.AppleCookies)
		if err != nil {
			Fail(w, r, 0, "Apple Music client error: "+err.Error())
			return
		}
		artist := findArtist(applemusic.ArtistIDFromName(name))
		if artist.Name == "" {
			artist.Name = name
		}
		songs, err = artistTopSongs(client, artist, count)
		if err != nil {
			Fail(w, r, 0, "Apple Music top songs failed: "+err.Error())
			return
		}
	}

	var bodies []ChildBody
	if config.AppConfig.SyncOnSearch && len(songs) > 0 {
		applemusic.SyncSongs(songs)

		var rows []schema.Song
		db.DB.Where("id IN ?", resourceIDs(songs)).Find(&rows)
		byID := make(map[string]schema.Song, len(rows))
		for _, s := range rows {
			byID[s.ID] = s
		}
		for _, res := range songs {
			if s, ok := byID[res.ID]; ok {
				bodies = append(bodies, songToChild(s))
			}
		}
	} else {
		for _, res := range songs {
			bodies = append(bodies, appleSongToChild(res))
		}
	}

	loadStars(user.Username).markChildren(bodies)

	OK(w, r, func(resp *response) {
		resp.TopSongs = &TopSongsBody{Song: bodies}
	})
}
