package minio

import (
	"context"
	"fmt"
	"mime"
	"path/filepath"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/mirjalilova/voice_transcribe/config"
	"golang.org/x/exp/slog"
)

type MinIO struct {
	Client *minio.Client
	Cnf    *config.Config
}

var bucketName = "voice-transcribe"

func MinIOConnect(cnf *config.Config) (*MinIO, error) {

	minioClient, err := minio.New(cnf.MINIO_ENDPOINT, &minio.Options{
		Creds:  credentials.NewStaticV4(cnf.MINIO_ACCESS_KEY, cnf.MINIO_SECRET_KEY, ""),
		Secure: false,
	})
	if err != nil {
		slog.Error("Failed to connect to MinIO: %v", err)
		return nil, err
	}

	// Create the bucket if it doesn't exist
	err = minioClient.MakeBucket(context.Background(), cnf.MINIO_BUCKET_NAME, minio.MakeBucketOptions{})
	if err != nil {
		// Check if the bucket already exists
		exists, errBucketExists := minioClient.BucketExists(context.Background(), cnf.MINIO_BUCKET_NAME)
		if errBucketExists == nil && exists {
			slog.Warn("Bucket already exists: %s\n", cnf.MINIO_BUCKET_NAME)
		} else {
			slog.Error("Error while making bucket %s: %v\n", cnf.MINIO_BUCKET_NAME, err)
		}
	} else {
		slog.Info("Successfully created bucket: %s\n", cnf.MINIO_BUCKET_NAME)
	}

	policy := fmt.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": "*",
				"Action": ["s3:GetObject"],
				"Resource": ["arn:aws:s3:::%s/*"]
			}
		]
	}`, cnf.MINIO_BUCKET_NAME)

	err = minioClient.SetBucketPolicy(context.Background(), cnf.MINIO_BUCKET_NAME, policy)
	if err != nil {
		slog.Error("Error while setting bucket policy: %v", err)
		return nil, err
	}

	return &MinIO{
		Client: minioClient,
		Cnf:    cnf,
	}, nil
}

func (m *MinIO) Upload(cnf config.Config, fileName, filePath string) (string, error) {
	ext := filepath.Ext(fileName)
	contentType := mime.TypeByExtension(ext)

	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err := m.Client.FPutObject(context.Background(), cnf.MINIO_BUCKET_NAME, fileName, filePath, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		slog.Error("Error while uploading %s to bucket %s: %v\n", fileName, cnf.MINIO_BUCKET_NAME, err)
		return "", err
	}

	serverHost := "10.145.20.8"
	port := 9000
	minioURL := fmt.Sprintf("http://%s:%d/%s/%s", serverHost, port, cnf.MINIO_BUCKET_NAME, fileName)

	return minioURL, nil
}
