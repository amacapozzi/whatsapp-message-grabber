package wasabi

import (
	"bytes"
	"fmt"
	"mime"
	"path/filepath"
	"time"

	"msg-grabber/internal/config"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type WasabiRepository interface {
	UploadFile(fileBytes []byte, filename string, contentType string) (string, error)
}

type WasabiDataInteraction struct {
	s3     *s3.S3
	bucket string
}

func (w *WasabiDataInteraction) UploadFile(fileBytes []byte, filename string, contentType string) (string, error) {
	if w.s3 == nil {
		return "", fmt.Errorf("s3 client is nil")
	}
	if w.bucket == "" {
		return "", fmt.Errorf("bucket name is empty")
	}

	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(filename))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	}

	key := fmt.Sprintf("uploads/%d_%s", time.Now().Unix(), filename)

	_, err := w.s3.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(w.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(fileBytes),
		ContentType: aws.String(contentType),
		ACL:         aws.String("private"),
	})
	if err != nil {
		return "", fmt.Errorf("put object: %w", err)
	}

	url := fmt.Sprintf("https://%s.s3.wasabisys.com/%s", w.bucket, key)
	return url, nil
}

func CreateSession() *WasabiDataInteraction {
	sess, err := session.NewSession(&aws.Config{
		Credentials:      credentials.NewStaticCredentials(config.API_CONFIG.WasabiAccessKeyID, config.API_CONFIG.WasabiSecretAccessKey, ""),
		Endpoint:         aws.String(config.API_CONFIG.WasabiEndpoint),
		Region:           aws.String("us-east-1"),
		S3ForcePathStyle: aws.Bool(true),
	})
	if err != nil {
		fmt.Println("❌ failed to create wasabi session:", err)
		return nil
	}

	svc := s3.New(sess)
	return &WasabiDataInteraction{
		s3:     svc,
		bucket: config.API_CONFIG.WasabiBucket,
	}
}

func NewWasabiRepository() *WasabiDataInteraction {
	return CreateSession()
}
