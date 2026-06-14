package subsonic

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// uses the "id" param as the art URL, which is how the Subsonic clients expect it.
// warning: this *COULD* BREAK some clients, to be tested
func GetCoverArt(w http.ResponseWriter, r *http.Request) {
	if _, code, msg := Authenticate(r); code != 0 {
		Fail(w, r, code, msg)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	// if a size param, replace for apple CDN URLS (ones that are like 000x000.jpg)
	if size := r.URL.Query().Get("size"); size != "" {
		id = replaceSize(id, size)
	}

	data, contentType, err := fetchOrCacheArt(id)
	if err != nil {
		http.Error(w, "could not fetch cover art: "+err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Write(data)
}

// fetchOrCacheArt returns the image bytes for artURL, hitting the cache first.
// TODO: use diff cache than file? maybe put some in memory?
func fetchOrCacheArt(artURL string) ([]byte, string, error) {
	artDir := "./data/art"

	cachePath := artCachePath(artDir, artURL)

	// try read, see if cache has it
	if data, err := os.ReadFile(cachePath); err == nil {
		return data, contentTypeForPath(cachePath), nil
	}

	// nope? get from CDN :(
	resp, err := http.Get(artURL)
	if err != nil {
		return nil, "", fmt.Errorf("fetching %s: %w", artURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("CDN returned %d for %s", resp.StatusCode, artURL)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = contentTypeForPath(artURL)
	}

	// check if paths exist
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return nil, "", fmt.Errorf("creating art cache dir: %w", err)
	}
	_ = os.WriteFile(cachePath, data, 0644)

	return data, ct, nil
}


// gets proper path for caching an art URL by hashing the URL and preserving the extension.
func artCachePath(artDir, artURL string) string {
	sum := sha256.Sum256([]byte(artURL))
	name := hex.EncodeToString(sum[:]) + extFromURL(artURL)
	return filepath.Join(artDir, name)
}

// gets the extension (ignores all params and such)
func extFromURL(u string) string {
	path := u
	if i := strings.Index(path, "?"); i != -1 {
		path = path[:i]
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		ext = ".jpg"
	}
	return ext
}

// for content type header
func contentTypeForPath(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	default:
		return "image/jpeg"
	}
}

// tries to replace the size token in an Apple CDN URL with the requested size, if possible.
func replaceSize(artURL, size string) string {
	sizeToken := fmt.Sprintf("%sx%sbb", size, size)
	if i := strings.LastIndex(artURL, "/"); i != -1 {
		segment := artURL[i+1:]
		if strings.Contains(segment, "bb.") {
			ext := filepath.Ext(segment)
			return artURL[:i+1] + sizeToken + ext
		}
	}
	return artURL
}
