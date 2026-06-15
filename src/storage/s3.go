package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/DwifteJB/aplsonic/src/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type s3Backend struct {
	client *minio.Client
	bucket string
}

func newS3Backend() (*s3Backend, error) {
	s := config.AppConfig.Storage
	if s.Endpoint == "" {
		return nil, nil
	}

	c, err := minio.New(s.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(s.AccessKey, s.SecretKey, ""),
		Secure: s.UseSSL,
		Region: s.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("storage: %w", err)
	}

	ctx := context.Background()
	exists, err := c.BucketExists(ctx, s.Bucket)
	if err != nil {
		return nil, fmt.Errorf("storage: bucket check: %w", err)
	}
	if !exists {
		if err := c.MakeBucket(ctx, s.Bucket, minio.MakeBucketOptions{Region: s.Region}); err != nil {
			return nil, fmt.Errorf("storage: make bucket: %w", err)
		}
		fmt.Printf("storage: created bucket %q\n", s.Bucket)
	}

	return &s3Backend{client: c, bucket: s.Bucket}, nil
}

func (b *s3Backend) HasSong(id string) bool {
	_, err := b.client.StatObject(context.Background(), b.bucket, SongKey(id), minio.StatObjectOptions{})
	return err == nil
}

func (b *s3Backend) PutSong(id, localPath, contentType string) (int64, error) {
	info, err := b.client.FPutObject(context.Background(), b.bucket, SongKey(id), localPath, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return 0, err
	}
	return info.Size, nil
}

func (b *s3Backend) GetSong(id string) (io.ReadSeekCloser, ObjectInfo, error) {
	obj, err := b.client.GetObject(context.Background(), b.bucket, SongKey(id), minio.GetObjectOptions{})
	if err != nil {
		return nil, ObjectInfo{}, err
	}
	info, err := obj.Stat()
	if err != nil {
		obj.Close()
		return nil, ObjectInfo{}, err
	}
	return obj, ObjectInfo{Size: info.Size, LastModified: info.LastModified}, nil
}

func (b *s3Backend) GetArt(key string) ([]byte, bool) {
	obj, err := b.client.GetObject(context.Background(), b.bucket, "art/"+key, minio.GetObjectOptions{})
	if err != nil {
		return nil, false
	}
	defer obj.Close()
	data, err := io.ReadAll(obj)
	if err != nil || len(data) == 0 {
		return nil, false
	}
	return data, true
}

func (b *s3Backend) PutArt(key string, data []byte) error {
	_, err := b.client.PutObject(context.Background(), b.bucket, "art/"+key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{})
	return err
}
