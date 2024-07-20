package s3wrap

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4Signer "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type Client struct {
	s3 *s3.Client
}

type noAcceptEncodingSigner struct {
	signer s3.HTTPSignerV4
}

func (signer *noAcceptEncodingSigner) SignHTTP(ctx context.Context, credentials aws.Credentials, r *http.Request, payloadHash string, service string, region string, signingTime time.Time, optFns ...func(*v4Signer.SignerOptions)) error {
	acceptEncoding := r.Header.Get("Accept-Encoding")
	r.Header.Del("Accept-Encoding")
	err := signer.signer.SignHTTP(ctx, credentials, r, payloadHash, service, region, signingTime, optFns...)
	if acceptEncoding != "" {
		r.Header.Set("Accept-Encoding", acceptEncoding)
	}
	return err
}

func New(ctx context.Context, forcePathStyle bool) (*Client, error) {
	config, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	s3Client := s3.NewFromConfig(config, func(options *s3.Options) {
		options.UsePathStyle = forcePathStyle
		defSigner := v4Signer.NewSigner(func(so *v4Signer.SignerOptions) {
			so.Logger = options.Logger
			so.LogSigning = options.ClientLogMode.IsSigning()
			so.DisableURIPathEscaping = true
		})
		options.HTTPSignerV4 = &noAcceptEncodingSigner{signer: defSigner}
	})

	return &Client{s3: s3Client}, nil
}

type ObjectMetaData struct {
	Key       string
	Timestamp time.Time
}

type ListObjectsOption func(*s3.ListObjectsV2Input)

func WithPrefix(prefix string) ListObjectsOption {
	return func(o *s3.ListObjectsV2Input) {
		o.Prefix = &prefix
	}
}

func (client *Client) ListObjects(ctx context.Context, bucket string, opts ...ListObjectsOption) ([]ObjectMetaData, error) {
	params := &s3.ListObjectsV2Input{
		Bucket: &bucket,
	}
	for _, opt := range opts {
		opt(params)
	}

	var result []ObjectMetaData
	var continuationToken *string
	first := true
	for first || continuationToken != nil {
		first = false
		params.ContinuationToken = continuationToken

		resp, err := client.s3.ListObjectsV2(ctx, params)
		if err != nil {
			return nil, err
		}

		continuationToken = resp.NextContinuationToken
		for _, obj := range resp.Contents {
			if strings.HasSuffix(*obj.Key, "/") {
				// Quirk: GCS's XML API returns directries as a object. We'll filter them out.
				continue
			}

			result = append(result, ObjectMetaData{
				Key:       *obj.Key,
				Timestamp: *obj.LastModified,
			})
		}
	}

	return result, nil
}

func (client *Client) DeleteObjects(ctx context.Context, bucket string, keys []string) error {
	var objectIds []types.ObjectIdentifier
	for _, key := range keys {
		objectIds = append(objectIds, types.ObjectIdentifier{
			Key: &key,
		})
	}
	if _, err := client.s3.DeleteObjects(context.Background(), &s3.DeleteObjectsInput{
		Bucket: &bucket,
		Delete: &types.Delete{
			Objects: objectIds,
		},
	}); err != nil {
		return err
	}

	return nil
}

type Object struct {
	Size int64
	Body io.ReadCloser
}

func (client *Client) GetObject(ctx context.Context, bucket, key string) (*Object, error) {
	resp, err := client.s3.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		return nil, err
	}

	return &Object{
		Size: *resp.ContentLength,
		Body: resp.Body,
	}, nil
}

func (client *Client) PutObject(ctx context.Context, bucket, key string, body io.ReadSeeker, size int64) error {
	if _, err := client.s3.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: &bucket,
		Key:    &key,
		Body:   body,
	}); err != nil {
		return err
	}

	return nil
}

func (client *Client) GetPresignedGetURL(ctx context.Context, bucket, key string, expires time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(client.s3)
	req, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expires
	})
	if err != nil {
		return "", err
	}

	return req.URL, nil
}

func (client *Client) GetPresignedPutURL(ctx context.Context, bucket, key string, expires time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(client.s3)
	req, err := presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expires
	})
	if err != nil {
		return "", err
	}

	return req.URL, nil
}
