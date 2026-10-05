package applemusic

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"
	"gorm.io/gorm/clause"

	"github.com/DwifteJB/aplsonic/src/db"
	"github.com/DwifteJB/aplsonic/src/db/schema"
)

// lyrics need a media-user-token (active subscription)
// syllable-lyrics gives word timing, lyrics is line timed only. ttmlLocalizations adds
// translations + transliterations (l[script] picks the romanization) to the ttml head
// returns "" with no error when apple has no lyrics for the song
func (c *Client) GetLyricsTTML(songID string) (string, error) {
	params := url.Values{
		"l[lyrics]": {apiLanguage},
		"extend":    {"ttmlLocalizations"},
		"l[script]": {"en-Latn"},
	}

	var lastErr error
	for _, kind := range []string{"syllable-lyrics", "lyrics"} {
		path := fmt.Sprintf("/v1/catalog/%s/songs/%s/%s", c.Storefront, songID, kind)
		data, err := browserDo("GET", path, params, nil, c.MediaUserToken)
		if err != nil {
			// 404 just means apple has no lyrics of this kind for the song
			if !strings.Contains(err.Error(), "): 404 ") {
				lastErr = err
			}
			continue
		}

		var wrapper struct {
			Data []struct {
				Attributes struct {
					TTML              string `json:"ttml"`
					TTMLLocalizations string `json:"ttmlLocalizations"`
				} `json:"attributes"`
			} `json:"data"`
		}
		if err := json.Unmarshal(data, &wrapper); err != nil {
			lastErr = fmt.Errorf("parsing %s response: %w", kind, err)
			continue
		}
		if len(wrapper.Data) == 0 {
			continue
		}

		// localizations is the same ttml plus translations/transliterations in the head
		attrs := wrapper.Data[0].Attributes
		if attrs.TTMLLocalizations != "" {
			return attrs.TTMLLocalizations, nil
		}
		if attrs.TTML != "" {
			return attrs.TTML, nil
		}
	}

	if lastErr != nil {
		return "", lastErr
	}
	return "", nil
}

const (
	lyricsMissingTTL = 24 * time.Hour
	lyricsTTL        = 30 * 24 * time.Hour
)

var lyricsGroup singleflight.Group

// cached lyrics for a song, nil when apple has none
func SongLyrics(user *schema.User, songID string) (*Lyrics, error) {
	var cached schema.SongLyrics
	db.DB.Where("song_id = ?", songID).Limit(1).Find(&cached)
	hasCache := cached.SongID != ""
	if hasCache {
		age := time.Since(cached.FetchedAt)
		if cached.Missing && age < lyricsMissingTTL {
			return nil, nil
		}
		if !cached.Missing && age < lyricsTTL {
			return ParseTTML(cached.TTML)
		}
	}

	ttml, err, _ := lyricsGroup.Do(songID, func() (any, error) {
		client, err := NewClientFromCookies(user.AppleCookies)
		if err != nil {
			return "", err
		}
		ttml, err := client.GetLyricsTTML(songID)
		if err != nil {
			return "", err
		}
		db.DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(&schema.SongLyrics{
			SongID:    songID,
			TTML:      ttml,
			Missing:   ttml == "",
			FetchedAt: time.Now(),
		})
		return ttml, nil
	})
	if err != nil {
		// stale is better than nothing
		if hasCache && !cached.Missing {
			return ParseTTML(cached.TTML)
		}
		return nil, err
	}
	if ttml.(string) == "" {
		return nil, nil
	}
	return ParseTTML(ttml.(string))
}
