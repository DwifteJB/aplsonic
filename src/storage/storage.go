package storage

import (
	"errors"
	"io"
	"time"

	"github.com/DwifteJB/aplsonic/src/config"
)

var ErrNotConfigured = errors.New("storage not configured")

type ObjectInfo struct {
	Size         int64
	LastModified time.Time
}

type Backend interface {
	HasSong(id string) bool
	PutSong(id, localPath, contentType string) (int64, error)
	GetSong(id string) (io.ReadSeekCloser, ObjectInfo, error)

	GetArt(key string) ([]byte, bool)
	PutArt(key string, data []byte) error
}

var backend Backend

func Init() error {
	mode := config.AppConfig.Storage.Mode
	if mode == "" {
		mode = "filesystem"
	}

	switch mode {
	case "filesystem":
		b, err := newFSBackend()
		if err != nil {
			return err
		}
		backend = b
	case "s3":
		b, err := newS3Backend()
		if err != nil {
			return err
		}
		if b == nil {
			println("storage: s3 mode but no endpoint configured, audio streaming disabled")
			return nil
		}
		backend = b
	default:
		return errors.New("storage: unknown mode " + mode)
	}

	return nil
}

func Ready() bool {
	return backend != nil
}

func SongKey(id string) string {
	return "songs/" + id + ".m4a"
}

func ObjectKey(id string) string {
	return SongKey(id)
}

func Has(id string) bool {
	if backend == nil {
		return false
	}
	return backend.HasSong(id)
}

func Put(id, localPath, contentType string) (int64, error) {
	if backend == nil {
		return 0, ErrNotConfigured
	}
	return backend.PutSong(id, localPath, contentType)
}

func Get(id string) (io.ReadSeekCloser, ObjectInfo, error) {
	if backend == nil {
		return nil, ObjectInfo{}, ErrNotConfigured
	}
	return backend.GetSong(id)
}

func GetArt(key string) ([]byte, bool) {
	if backend == nil {
		return nil, false
	}
	return backend.GetArt(key)
}

func PutArt(key string, data []byte) error {
	if backend == nil {
		return ErrNotConfigured
	}
	return backend.PutArt(key, data)
}
