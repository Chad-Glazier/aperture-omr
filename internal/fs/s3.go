package fs

import (
	"context"
	// "ubco-team15/omr/config"

	// "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

//
// This file implements the Store interface by wrapping an S3 client.
//

type s3Store struct {
	client *s3.Client
	bucket string
	ctx    context.Context
}

func NewS3Store(ctx context.Context) Store {
	return nil
}
