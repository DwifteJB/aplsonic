package download

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/DwifteJB/aplsonic/src/config"
	"github.com/DwifteJB/aplsonic/src/db"
	"github.com/DwifteJB/aplsonic/src/db/schema"
	"github.com/DwifteJB/aplsonic/src/gamdl"
	"github.com/DwifteJB/aplsonic/src/storage"
	"golang.org/x/sync/singleflight"
)

var group singleflight.Group

// allowed extensions for gammmdl
var audioExts = map[string]bool{".m4a": true, ".aac": true, ".mp4": true, ".ec3": true, ".flac": true}

// ensures that the song is downloaded & stored in s3
func EnsureSong(user *schema.User, song *schema.Song) error {
	if !storage.Ready() {
		return fmt.Errorf("storage not configured")
	}
	if storage.Has(song.ID) {
		return nil
	}

	_, err, _ := group.Do(song.ID, func() (any, error) {
		if storage.Has(song.ID) {
			return nil, nil
		}
		return nil, downloadSong(user, song)
	})
	return err
}

func downloadSong(user *schema.User, song *schema.Song) error {
	if song.AlbumID == "" {
		return fmt.Errorf("song %s has no album id", song.ID)
	}

	tmp, err := os.MkdirTemp("", "aplsonic-dl-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	cookiePath := filepath.Join(tmp, "cookies.txt")
	if err := os.WriteFile(cookiePath, []byte(user.AppleCookies), 0600); err != nil {
		return err
	}

	g, err := gamdl.NewGamDL(cookiePath)
	if err != nil {
		return err
	}

	codec := config.AppConfig.Storage.DownloadCodec
	if codec == "" {
		codec = "aac-web"
	}

	url := fmt.Sprintf("https://music.apple.com/us/album/_/%s?i=%s", song.AlbumID, song.ID)
	outDir := filepath.Join(tmp, "out")
	cmd, err := g.Command(context.Background(),
		"--output-path", outDir,
		"--song-codec-priority", codec,
		url,
	)
	if err != nil {
		return err
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gamdl failed for %s: %w", song.ID, err)
	}

	file, err := findAudioFile(outDir)
	if err != nil {
		return err
	}

	contentType := "audio/mp4"
	size, err := storage.Put(song.ID, file, contentType)
	if err != nil {
		return err
	}

	// persist the "downloaded" marker on the song row
	db.DB.Model(&schema.Song{}).Where("id = ?", song.ID).Updates(map[string]any{
		"path":         storage.ObjectKey(song.ID),
		"size":         size,
		"suffix":       "m4a",
		"content_type": contentType,
	})

	fmt.Printf("download: stored %s (%s) %d bytes\n", song.ID, song.Title, size)
	return nil
}

func findAudioFile(dir string) (string, error) {
	var best string
	var bestMod int64
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !audioExts[strings.ToLower(filepath.Ext(path))] {
			return nil
		}
		if mod := info.ModTime().UnixNano(); mod >= bestMod {
			bestMod = mod
			best = path
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if best == "" {
		return "", fmt.Errorf("no audio file produced in %s", dir)
	}
	return best, nil
}

// ensures the album's songs are downloaded & stored in s3
func EnsureAlbum(user *schema.User, albumID string) {
	if !storage.Ready() {
		return
	}

	var songs []schema.Song
	db.DB.Where("album_id = ?", albumID).Find(&songs)

	const workers = 3
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	for i := range songs {
		s := songs[i]
		if storage.Has(s.ID) {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			if err := EnsureSong(user, &s); err != nil {
				fmt.Printf("download: album %s song %s failed: %v\n", albumID, s.ID, err)
			}
		}()
	}
	wg.Wait()
}
