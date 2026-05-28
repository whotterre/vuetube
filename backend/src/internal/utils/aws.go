package utils

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/whotterre/vuetube/src/internal/config"
)

type S3Helper struct {
	S3Client *s3.Client
}

func NewS3Helper(ctx context.Context) *S3Helper {
	cfg, err := config.LoadAWSConfig(ctx)
	if err != nil {

	}
	client := s3.NewFromConfig(cfg)
	return &S3Helper{S3Client: client}
}

// UploadFile reads from a file and puts the data into an object in an S3 bucket.
func (h *S3Helper) UploadFile(ctx context.Context,
	bucketName string,
	objectKey string,
	fileName string) (*s3.PutObjectOutput, error) {
	file, err := os.Open(fileName)
	if err != nil {
		log.Printf("Couldn't open file %v to upload. Here's why: %v\n", fileName, err)
		return nil, err
	}
	defer file.Close()

	uploadedVideoDetails, err := h.S3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
		Body:   file,
	})
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "EntityTooLarge" {
			log.Printf("S3 upload failed: object exceeds service limit (file should be validated at handler layer, max 500MB)\n")
		} else {
			log.Printf("Couldn't upload file %v to %v:%v. Here's why: %v\n",
				fileName, bucketName, objectKey, err)
		}
		return nil, err
	}

	waitErr := s3.NewObjectExistsWaiter(h.S3Client).Wait(
		ctx, &s3.HeadObjectInput{Bucket: aws.String(bucketName), Key: aws.String(objectKey)}, time.Minute)
	if waitErr != nil {
		log.Printf("Failed attempt to wait for object %s to exist: %v\n", objectKey, waitErr)
	}

	return uploadedVideoDetails, nil
}
