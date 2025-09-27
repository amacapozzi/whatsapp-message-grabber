package wasabi

import (
	"bytes"
	"fmt"
	"msg-grabber/internal/config"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type WasabiDataInteraction struct {
	s3     *s3.S3
	bucket string
}

func (w *WasabiDataInteraction) UploadFile(fileName string, fileBytes []byte, mime string) (string, error) {
	_, err := w.s3.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(w.bucket),
		Key:         aws.String(fileName),
		Body:        bytes.NewReader(fileBytes),
		ContentType: aws.String(mime),
		ACL:         aws.String("public-read"),
	})
	if err != nil {
		return "", err
	}

	// Construir URL pública
	url := fmt.Sprintf("https://s3.us-east-1.wasabisys.com/%s/%s", w.bucket, fileName)
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
		panic(err)
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
