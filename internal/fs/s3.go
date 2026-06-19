package fs

//
// This file implements the Store interface by wrapping an S3 client.
//
// At the time of writing, this implementation is meant to work specifically
// with a Garage FS container. Ostensibly, Garage implements the S3 API, which
// means that we can use the AWS SDK's S3 client (in fact, Garage doesn't
// implement its own SDK as of now, it just expects you to use existing ones).
// However, Garage uses the old style of bucket-naming where buckets are
// specified as a path parameter (i.e., ".../bucket-name/item-key") instead of
// the more recent convention of specifying buckets as a subdomain. AWS's S3
// client does have an option to use this older convention, but it isn't 
// actually supported; setting the option does not do anything. This leads to 
// the odd pattern you'll see in the code below where we specify a bucket name 
// in the input to an S3 call, but then *also* include the bucket in the item 
// key so that Garage actually understands which bucket we're talking about.
// I really don't understand why Garage doesn't have it's own SDK and why AWS
// has an incomplete one, but such is life.
//
// Links:
// - Garage -- https://garagehq.deuxfleurs.fr/documentation/quick-start/
// - AWS S3 Client -- pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/s3
//

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/draw"
	"log/slog"
	"net/url"
	"os"
	"time"
	"ubco-team15/omr/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	transport "github.com/aws/smithy-go/endpoints"
)

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
	client  *s3.Client
	bucket  string
	timeout time.Duration // The per-request timeout.
}

func (s *s3Store) ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), s.timeout)
}

func NewS3Store(bucket string) (Store, error) {

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
		client:  client,
		bucket:  bucket,
		timeout: time.Second * 5,
	}, nil
}

func (s *s3Store) ImgExists(key string) bool {
	ctx, cancel := s.ctx()
	defer cancel()

	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.bucket + "/" + key),
	})
	return err == nil
}

func (s *s3Store) GetImg(key string) (image.Image, error) {
	ctx, cancel := s.ctx()
	defer cancel()

	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.bucket + "/" + key),
	})
	if err != nil {
		return nil, err
	}
	defer out.Body.Close()

	return DecodeImg(out.Body)
}

func (s *s3Store) PutImg(key string, img image.Image) error {
	ctx, cancel := s.ctx()
	defer cancel()

	var buf bytes.Buffer
	if err := EncodeImg(&buf, img); err != nil {
		return err
	}

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(s.bucket + "/" + key),
		Body:        bytes.NewReader(buf.Bytes()),
		ContentType: aws.String("image/png"),
		ACL:         types.ObjectCannedACLPrivate,
	})
	return err
}

func (s *s3Store) DeleteImg(key string) error {
	ctx, cancel := s.ctx()
	defer cancel()

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.bucket + "/" + key),
	})
	return err
}

func (s *s3Store) ImgSnippet(
	key string,
	x, y, width, height int,
) (image.Image, error) {
	img, err := s.GetImg(key)
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()

	if x < bounds.Min.X ||
		y < bounds.Min.Y ||
		x+width > bounds.Max.X ||
		y+height > bounds.Max.Y {
		return nil, errors.New("requested region outside image bounds")
	}

	rect := image.Rect(0, 0, width, height)
	cropped := image.NewRGBA(rect)
	draw.Draw(
		cropped,
		rect,
		img,
		image.Point{X: x, Y: y},
		draw.Src,
	)
	return cropped, nil
}
