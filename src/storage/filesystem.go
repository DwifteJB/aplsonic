package storage

import (
	"io"
	"os"
	"path/filepath"

	"github.com/DwifteJB/aplsonic/src/config"
)

type fsBackend struct {
	root    string
	songDir string
	artDir  string
}

func newFSBackend() (*fsBackend, error) {
	root := config.AppConfig.Storage.Path
	if root == "" {
		root = "./data"
	}

	b := &fsBackend{
		root:    root,
		songDir: filepath.Join(root, "songs"),
		artDir:  filepath.Join(root, "art"),
	}

	if err := os.MkdirAll(b.songDir, 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(b.artDir, 0o755); err != nil {
		return nil, err
	}

	println("storage: filesystem ready at " + root)
	return b, nil
}

func (b *fsBackend) songPath(id string) string {
	return filepath.Join(b.songDir, id+".m4a")
}

func (b *fsBackend) HasSong(id string) bool {
	_, err := os.Stat(b.songPath(id))
	return err == nil
}

func (b *fsBackend) PutSong(id, localPath, contentType string) (int64, error) {
	src, err := os.Open(localPath)
	if err != nil {
		return 0, err
	}
	defer src.Close()

	dst, err := os.Create(b.songPath(id))
	if err != nil {
		return 0, err
	}
	defer dst.Close()

	n, err := io.Copy(dst, src)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (b *fsBackend) GetSong(id string) (io.ReadSeekCloser, ObjectInfo, error) {
	f, err := os.Open(b.songPath(id))
	if err != nil {
		return nil, ObjectInfo{}, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, ObjectInfo{}, err
	}
	return f, ObjectInfo{Size: info.Size(), LastModified: info.ModTime()}, nil
}

func (b *fsBackend) GetArt(key string) ([]byte, bool) {
	data, err := os.ReadFile(filepath.Join(b.artDir, filepath.Base(key)))
	if err != nil {
		return nil, false
	}
	return data, true
}

func (b *fsBackend) PutArt(key string, data []byte) error {
	return os.WriteFile(filepath.Join(b.artDir, filepath.Base(key)), data, 0o644)
}
