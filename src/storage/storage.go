package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/DwifteJB/aplsonic/src/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var (
	client *minio.Client
	bucket string
)

// when not configured
var ErrNotConfigured = errors.New("storage not configured")

func Init() error {
	s := config.AppConfig.Storage
	if s.Endpoint == "" {
		println("storage: no endpoint configured, audio streaming disabled")
		return nil
	}

	c, err := minio.New(s.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(s.AccessKey, s.SecretKey, ""),
		Secure: s.UseSSL,
		Region: s.Region,
	})
	if err != nil {
		return fmt.Errorf("storage: %w", err)
	}

	ctx := context.Background()
	exists, err := c.BucketExists(ctx, s.Bucket)
	if err != nil {
		return fmt.Errorf("storage: bucket check: %w", err)
	}
	if !exists {
		if err := c.MakeBucket(ctx, s.Bucket, minio.MakeBucketOptions{Region: s.Region}); err != nil {
			return fmt.Errorf("storage: make bucket: %w", err)
		}
		fmt.Printf("storage: created bucket %q\n", s.Bucket)
	}

	client = c
	bucket = s.Bucket
	return nil
}

func Ready() bool {
	return client != nil
}

func ObjectKey(songID string) string {
	return "songs/" + songID + ".m4a"
}

func Has(songID string) bool {
	if client == nil {
		return false
	}
	_, err := client.StatObject(context.Background(), bucket, ObjectKey(songID), minio.StatObjectOptions{})
	return err == nil
}

func Put(songID, localPath, contentType string) (int64, error) {
	if client == nil {
		return 0, ErrNotConfigured
	}
	info, err := client.FPutObject(context.Background(), bucket, ObjectKey(songID), localPath, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return 0, err
	}
	return info.Size, nil
}

func Get(songID string) (*minio.Object, minio.ObjectInfo, error) {
	if client == nil {
		return nil, minio.ObjectInfo{}, ErrNotConfigured
	}
	obj, err := client.GetObject(context.Background(), bucket, ObjectKey(songID), minio.GetObjectOptions{})
	if err != nil {
		return nil, minio.ObjectInfo{}, err
	}
	info, err := obj.Stat()
	if err != nil {
		obj.Close()
		return nil, minio.ObjectInfo{}, err
	}
	return obj, info, nil
}
