package subsonic

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
)

const (
	apiVersion    = "1.16.1"
	serverType    = "aplsonic"
	serverVersion = "0.1.0"
)

// response is the inner subsonic-response object.
type response struct {
	XMLName      xml.Name `json:"-" xml:"subsonic-response"`
	XMLNS        string   `json:"-" xml:"xmlns,attr"`
	Status       string   `json:"status" xml:"status,attr"`
	Version      string   `json:"version" xml:"version,attr"`
	Type         string   `json:"type" xml:"type,attr"`
	ServerVer    string   `json:"serverVersion" xml:"serverVersion,attr"`
	OpenSubsonic bool     `json:"openSubsonic" xml:"openSubsonic,attr"`

	Error         *ErrorBody         `json:"error,omitempty" xml:"error,omitempty"`
	License       *LicenseBody       `json:"license,omitempty" xml:"license,omitempty"`
	User          *UserBody          `json:"user,omitempty" xml:"user,omitempty"`
	Users         *UsersBody         `json:"users,omitempty" xml:"users,omitempty"`
	AlbumList     *AlbumListBody     `json:"albumList,omitempty" xml:"albumList,omitempty"`
	AlbumList2    *AlbumList2Body    `json:"albumList2,omitempty" xml:"albumList2,omitempty"`
	RandomSongs   *RandomSongsBody   `json:"randomSongs,omitempty" xml:"randomSongs,omitempty"`
	SearchResult3 *SearchResult3Body `json:"searchResult3,omitempty" xml:"searchResult3,omitempty"`
	Album         *AlbumID3Body      `json:"album,omitempty" xml:"album,omitempty"`
	Artist        *ArtistID3Body     `json:"artist,omitempty" xml:"artist,omitempty"`
	Artists       *ArtistsBody       `json:"artists,omitempty" xml:"artists,omitempty"`
	Indexes       *IndexesBody       `json:"indexes,omitempty" xml:"indexes,omitempty"`
	Song          *ChildBody         `json:"song,omitempty" xml:"song,omitempty"`
}


// error, license, user stuff
type ErrorBody struct {
	Code    int    `json:"code" xml:"code,attr"`
	Message string `json:"message,omitempty" xml:"message,attr,omitempty"`
}

type LicenseBody struct {
	Valid          bool   `json:"valid" xml:"valid,attr"`
	Email          string `json:"email,omitempty" xml:"email,attr,omitempty"`
	LicenseExpires string `json:"licenseExpires,omitempty" xml:"licenseExpires,attr,omitempty"`
}

type UserBody struct {
	Username            string `json:"username" xml:"username,attr"`
	Email               string `json:"email,omitempty" xml:"email,attr,omitempty"`
	ScrobblingEnabled   bool   `json:"scrobblingEnabled" xml:"scrobblingEnabled,attr"`
	MaxBitRate          int    `json:"maxBitRate,omitempty" xml:"maxBitRate,attr,omitempty"`
	AdminRole           bool   `json:"adminRole" xml:"adminRole,attr"`
	SettingsRole        bool   `json:"settingsRole" xml:"settingsRole,attr"`
	DownloadRole        bool   `json:"downloadRole" xml:"downloadRole,attr"`
	UploadRole          bool   `json:"uploadRole" xml:"uploadRole,attr"`
	PlaylistRole        bool   `json:"playlistRole" xml:"playlistRole,attr"`
	CoverArtRole        bool   `json:"coverArtRole" xml:"coverArtRole,attr"`
	CommentRole         bool   `json:"commentRole" xml:"commentRole,attr"`
	PodcastRole         bool   `json:"podcastRole" xml:"podcastRole,attr"`
	StreamRole          bool   `json:"streamRole" xml:"streamRole,attr"`
	JukeboxRole         bool   `json:"jukeboxRole" xml:"jukeboxRole,attr"`
	ShareRole           bool   `json:"shareRole" xml:"shareRole,attr"`
	VideoConversionRole bool   `json:"videoConversionRole" xml:"videoConversionRole,attr"`
}

type UsersBody struct {
	User []UserBody `json:"user" xml:"user"`
}

// children of things

type ChildBody struct {
	ID          string `json:"id" xml:"id,attr"`
	IsDir       bool   `json:"isDir" xml:"isDir,attr"`
	Title       string `json:"title" xml:"title,attr"`
	Album       string `json:"album,omitempty" xml:"album,attr,omitempty"`
	AlbumID     string `json:"albumId,omitempty" xml:"albumId,attr,omitempty"`
	Artist      string `json:"artist,omitempty" xml:"artist,attr,omitempty"`
	ArtistID    string `json:"artistId,omitempty" xml:"artistId,attr,omitempty"`
	Track       int    `json:"track,omitempty" xml:"track,attr,omitempty"`
	DiscNumber  int    `json:"discNumber,omitempty" xml:"discNumber,attr,omitempty"`
	Year        int    `json:"year,omitempty" xml:"year,attr,omitempty"`
	Genre       string `json:"genre,omitempty" xml:"genre,attr,omitempty"`
	CoverArt    string `json:"coverArt,omitempty" xml:"coverArt,attr,omitempty"`
	Duration    int    `json:"duration,omitempty" xml:"duration,attr,omitempty"`
	BitRate     int    `json:"bitRate,omitempty" xml:"bitRate,attr,omitempty"`
	Size        int64  `json:"size,omitempty" xml:"size,attr,omitempty"`
	ContentType string `json:"contentType,omitempty" xml:"contentType,attr,omitempty"`
	Suffix      string `json:"suffix,omitempty" xml:"suffix,attr,omitempty"`
	Type        string `json:"type,omitempty" xml:"type,attr,omitempty"`
}

type AlbumID3Body struct {
	ID        string      `json:"id" xml:"id,attr"`
	Name      string      `json:"name" xml:"name,attr"`
	Artist    string      `json:"artist,omitempty" xml:"artist,attr,omitempty"`
	ArtistID  string      `json:"artistId,omitempty" xml:"artistId,attr,omitempty"`
	CoverArt  string      `json:"coverArt,omitempty" xml:"coverArt,attr,omitempty"`
	SongCount int         `json:"songCount" xml:"songCount,attr"`
	Duration  int         `json:"duration" xml:"duration,attr"`
	Year      int         `json:"year,omitempty" xml:"year,attr,omitempty"`
	Genre     string      `json:"genre,omitempty" xml:"genre,attr,omitempty"`
	Created   string      `json:"created" xml:"created,attr"`
	Song      []ChildBody `json:"song,omitempty" xml:"song,omitempty"`
}

type ArtistID3Body struct {
	ID         string         `json:"id" xml:"id,attr"`
	Name       string         `json:"name" xml:"name,attr"`
	CoverArt   string         `json:"coverArt,omitempty" xml:"coverArt,attr,omitempty"`
	AlbumCount int            `json:"albumCount" xml:"albumCount,attr"`
	Album      []AlbumID3Body `json:"album,omitempty" xml:"album,omitempty"`
}

type ArtistsBody struct {
	IgnoredArticles string        `json:"ignoredArticles" xml:"ignoredArticles,attr"`
	Index           []ArtistIndex `json:"index,omitempty" xml:"index,omitempty"`
}

type ArtistIndex struct {
	Name   string          `json:"name" xml:"name,attr"`
	Artist []ArtistID3Body `json:"artist,omitempty" xml:"artist,omitempty"`
}

type IndexesBody struct {
	LastModified    int64          `json:"lastModified" xml:"lastModified,attr"`
	IgnoredArticles string         `json:"ignoredArticles" xml:"ignoredArticles,attr"`
	Index           []IndexEntry   `json:"index,omitempty" xml:"index,omitempty"`
}

type IndexEntry struct {
	Name   string      `json:"name" xml:"name,attr"`
	Artist []ChildBody `json:"artist,omitempty" xml:"artist,omitempty"`
}

type AlbumListBody struct {
	Album []ChildBody `json:"album" xml:"album"`
}

type AlbumList2Body struct {
	Album []AlbumID3Body `json:"album" xml:"album"`
}

type RandomSongsBody struct {
	Song []ChildBody `json:"song" xml:"song"`
}

type SearchResult3Body struct {
	Artist []AlbumID3Body `json:"artist,omitempty" xml:"artist,omitempty"`
	Album  []AlbumID3Body `json:"album,omitempty" xml:"album,omitempty"`
	Song   []ChildBody    `json:"song,omitempty" xml:"song,omitempty"`
}

func baseResp(status string) *response {
	return &response{
		XMLNS:        "http://subsonic.org/restapi",
		Status:       status,
		Version:      apiVersion,
		Type:         serverType,
		ServerVer:    serverVersion,
		OpenSubsonic: true,
	}
}

func write(w http.ResponseWriter, r *http.Request, resp *response) {
	if r.URL.Query().Get("f") == "json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"subsonic-response": resp})
		return
	}
	w.Header().Set("Content-Type", "application/xml")
	w.Write([]byte(xml.Header))
	xml.NewEncoder(w).Encode(resp)
}

// ok response
func OK(w http.ResponseWriter, r *http.Request, set ...func(*response)) {
	resp := baseResp("ok")
	for _, fn := range set {
		fn(resp)
	}
	write(w, r, resp)
}

// uh oh response
func Fail(w http.ResponseWriter, r *http.Request, code int, msg string) {
	resp := baseResp("failed")
	fmt.Printf("failed %s due to %s (%d)\n", r.URL, msg, code)
	resp.Error = &ErrorBody{Code: code, Message: msg}
	write(w, r, resp)
}
