package utils

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
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

// UploadFile uploads data from a reader into an S3 bucket object.
func (h *S3Helper) UploadFile(ctx context.Context,
	bucketName string,
	objectKey string,
	body io.Reader) (*s3.PutObjectOutput, error) {
	uploadedVideoDetails, err := h.S3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
		Body:   body,
	})
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "EntityTooLarge" {
			log.Printf("S3 upload failed: object exceeds service limit (file should be validated at handler layer, max 500MB)\n")
		} else {
			log.Printf("Couldn't upload object %v to %v. Here's why: %v\n",
				objectKey, bucketName, err)
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

func (h *S3Helper) ObjectExists(ctx context.Context, bucketName string, objectKey string) (bool, error) {
	_, err := h.S3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "NotFound" {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (h *S3Helper) PresignGetObject(ctx context.Context, bucketName, objectKey string, ttl time.Duration) (string, error) {
    presignClient := s3.NewPresignClient(h.S3Client)
    req, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
        Bucket: aws.String(bucketName),
        Key:    aws.String(objectKey),
    }, s3.WithPresignExpires(ttl))
    if err != nil {
        return "", fmt.Errorf("failed to presign object: %w", err)
    }
    return req.URL, nil
}

// GetObject fetches the raw bytes of an S3 object and returns them.
// The caller is responsible for closing the returned ReadCloser.
func (h *S3Helper) GetObject(ctx context.Context, bucketName, objectKey string) (io.ReadCloser, error) {
	result, err := h.S3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object %s: %w", objectKey, err)
	}
	return result.Body, nil
}

func ExtractS3Key(s3Url, bucketName, region string) string {
	prefix := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/", bucketName, region)
	return strings.TrimPrefix(s3Url, prefix)
}