package applemusic

type SearchResults struct {
	Albums  *ResourceList `json:"albums,omitempty"`
	Songs   *ResourceList `json:"songs,omitempty"`
	Artists *ResourceList `json:"artists,omitempty"`
}

type ResourceList struct {
	Data []Resource `json:"data"`
	Next string     `json:"next,omitempty"`
}

type Resource struct {
	ID            string        `json:"id"`
	Type          string        `json:"type"`
	Attributes    Attributes    `json:"attributes"`
	Relationships Relationships `json:"relationships"`
}

type Attributes struct {
	// shared stuff
	Name           string   `json:"name"`
	ArtistName     string   `json:"artistName,omitempty"`
	Artwork        *Artwork `json:"artwork,omitempty"`
	GenreNames     []string `json:"genreNames,omitempty"`
	ReleaseDate    string   `json:"releaseDate,omitempty"`
	ContentRating  string   `json:"contentRating,omitempty"` // "explicit" or "clean" or ""

	// album-specific
	TrackCount       int    `json:"trackCount,omitempty"`
	DurationInMillis int64  `json:"durationInMillis,omitempty"`
	RecordLabel      string `json:"recordLabel,omitempty"`
	IsCompilation    bool   `json:"isCompilation,omitempty"`

	// song-specific
	AlbumName    string `json:"albumName,omitempty"`
	TrackNumber  int    `json:"trackNumber,omitempty"`
	DiscNumber   int    `json:"discNumber,omitempty"`
	ISRC         string `json:"isrc,omitempty"`
	ComposerName string `json:"composerName,omitempty"`
	CurrentBPM   int    `json:"currentBpm,omitempty"`

	// playlist-specific
	Description *Description `json:"description,omitempty"`
	CanEdit     bool         `json:"canEdit,omitempty"`
	IsPublic    bool         `json:"isPublic,omitempty"`
	DateAdded   string       `json:"dateAdded,omitempty"`
	PlayParams  *PlayParams  `json:"playParams,omitempty"`
}

type Description struct {
	Standard string `json:"standard,omitempty"`
	Short    string `json:"short,omitempty"`
}

type PlayParams struct {
	ID          string `json:"id,omitempty"`
	Kind        string `json:"kind,omitempty"`
	IsLibrary   bool   `json:"isLibrary,omitempty"`
	CatalogID   string `json:"catalogId,omitempty"`
	PurchasedID string `json:"purchasedId,omitempty"`
}

type Artwork struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type Relationships struct {
	Albums    *ResourceList `json:"albums,omitempty"`
	Artists   *ResourceList `json:"artists,omitempty"`
	Tracks    *ResourceList `json:"tracks,omitempty"`
	Playlists *ResourceList `json:"playlists,omitempty"`
}
