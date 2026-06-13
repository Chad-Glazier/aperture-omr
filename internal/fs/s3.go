package fs

import (
	"context"
	"image"
	"io"
	"log/slog"
	"net/url"
	"os"
	"ubco-team15/omr/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	// The "transport" aliasing of this package is redundant; it's already
	// named "transport." I just wanted to make it explicit because a 2
	// trillion dollar company can't follow basic naming practices for some
	// reason.
	transport "github.com/aws/smithy-go/endpoints"
)

//
// This file implements the Store interface by wrapping an S3 client.
//

type s3Credentials struct{}

func (s s3Credentials) Retrieve(ctx context.Context) (aws.Credentials, error) {
	return aws.Credentials{
		AccessKeyID:     config.GarageAccessKeyId,
		SecretAccessKey: config.GarageSecretAccessKey,
		CanExpire:       false,
	}, nil
}

type s3Endpoint struct{}

func (s s3Endpoint) ResolveEndpoint(
	ctx context.Context,
	params s3.EndpointParameters,
) (
	transport.Endpoint,
	error,
) {
	url, err := url.Parse(config.GarageEndpointUrl)
	if err != nil {
		slog.Error("couldn't parse URL", "url", config.GarageEndpointUrl)
		os.Exit(1)
	}
	return transport.Endpoint{URI: *url}, nil
}

type s3Store struct {
	client *s3.Client
	bucket string
}

func NewS3Store() (Store, error) {

	cfg, err := awsConfig.LoadDefaultConfig(
		context.TODO(),
		awsConfig.WithCredentialsProvider(s3Credentials{}),
		awsConfig.WithRegion(config.GarageRegion),
	)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(
		cfg,
		s3.WithEndpointResolverV2(s3Endpoint{}),
	)

	return &s3Store{
		client: client,
		bucket: config.GarageBucketName,
	}, nil
}

func (s *s3Store) ImgExists(key string) bool {
	panic("unimplemented call")
}

func (s *s3Store) GetImg(key string) (image.Image, error) {
	panic("unimplemented function")
}

func (s *s3Store) PutImg(key string, img image.Image) error {
	panic("unimplemented function")
}

func (s *s3Store) DeleteImg(key string) error {
	panic("unimplemented function")
}

func (s *s3Store) ImgSnippet(
	key string,
	x, y, width, height int,
) (image.Image, error) {
	panic("unimplemented function")
}

func (s *s3Store) ImgReader(key string) (io.ReadCloser, error) {
	panic("unimplemented function")
}

func (s *s3Store) ImgWriter(key string) (io.WriteCloser, error) {
	panic("unimplemented function")
}
