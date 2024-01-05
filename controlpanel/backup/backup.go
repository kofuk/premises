package backup

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4Signer "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go/logging"
	entity "github.com/kofuk/premises/common/entity/web"
	log "github.com/sirupsen/logrus"
)

type BackupProvider struct {
	s3Client *s3.Client
	bucket   string
}

type noAcceptEncodingSigner struct {
	signer s3.HTTPSignerV4
}

func newNoAcceptEncodingSigner(signer s3.HTTPSignerV4) *noAcceptEncodingSigner {
	return &noAcceptEncodingSigner{
		signer: signer,
	}
}

func (self *noAcceptEncodingSigner) SignHTTP(ctx context.Context, credentials aws.Credentials, r *http.Request, payloadHash string, service string, region string, signingTime time.Time, optFns ...func(*v4Signer.SignerOptions)) error {
	acceptEncoding := r.Header.Get("Accept-Encoding")
	r.Header.Del("Accept-Encoding")
	err := self.signer.SignHTTP(ctx, credentials, r, payloadHash, service, region, signingTime, optFns...)
	if acceptEncoding != "" {
		r.Header.Set("Accept-Encoding", acceptEncoding)
	}
	return err
}

func New(awsAccessKey, awsSecretKey, s3Endpoint, bucket string) *BackupProvider {
	config := aws.Config{
		Credentials:  credentials.NewStaticCredentialsProvider(awsAccessKey, awsSecretKey, ""),
		BaseEndpoint: &s3Endpoint,
		Logger: logging.LoggerFunc(func(classification logging.Classification, format string, v ...interface{}) {
			log.WithField("source", "aws-sdk").Info(v...)
		}),
		ClientLogMode: aws.LogRequestWithBody | aws.LogResponseWithBody,
	}

	s3Client := s3.NewFromConfig(config, func(options *s3.Options) {
		options.UsePathStyle = true
		defSigner := v4Signer.NewSigner(func(so *v4Signer.SignerOptions) {
			so.Logger = options.Logger
			so.LogSigning = options.ClientLogMode.IsSigning()
			so.DisableURIPathEscaping = true
		})
		options.HTTPSignerV4 = newNoAcceptEncodingSigner(defSigner)
	})

	return &BackupProvider{
		s3Client: s3Client,
		bucket:   bucket,
	}
}

func extractBackupInfoFromKey(key string) (string, string, error) {
	splitIndex := strings.IndexRune(key, '/')
	if splitIndex < 0 {
		return "", "", fmt.Errorf("Invalid backup key: %s", key)
	}
	world := string(key[0:splitIndex])
	name := string(key[splitIndex+1:])
	if strings.HasSuffix(name, ".tar.zst") {
		name = name[:len(name)-8]
	} else if strings.HasSuffix(name, ".tar.xz") {
		name = name[:len(name)-7]
	} else if strings.HasSuffix(name, ".zip") {
		name = name[:len(name)-4]
	}
	return world, name, nil
}

type objectMetaData struct {
	key       string
	timestamp time.Time
}

func (self *BackupProvider) fetchAllObjects(ctx context.Context) ([]objectMetaData, error) {
	var result []objectMetaData
	var continuationToken *string
	first := true
	for first || continuationToken != nil {
		first = false
		resp, err := self.s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket: &self.bucket,
		})
		if err != nil {
			return nil, err
		}
		continuationToken = resp.NextContinuationToken
		for _, obj := range resp.Contents {
			result = append(result, objectMetaData{
				key:       *obj.Key,
				timestamp: *obj.LastModified,
			})
		}
	}

	return result, nil
}

func (self *BackupProvider) GetWorlds(ctx context.Context) ([]entity.WorldBackup, error) {
	worlds := make(map[string]entity.WorldBackup)
	objects, err := self.fetchAllObjects(ctx)
	if err != nil {
		return nil, err
	}

	for _, obj := range objects {
		world, gen, err := extractBackupInfoFromKey(obj.key)
		if err != nil {
			return nil, err
		}
		worlds[world] = entity.WorldBackup{
			WorldName: world,
			Generations: append(worlds[world].Generations, entity.BackupGeneration{
				Gen:       gen,
				ID:        obj.key,
				Timestamp: int(obj.timestamp.UnixMilli()),
			}),
		}
	}

	for _, world := range worlds {
		sort.Slice(world.Generations, func(i, j int) bool {
			return world.Generations[i].Timestamp > world.Generations[j].Timestamp
		})
	}

	var keys []string
	for k := range worlds {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	result := make([]entity.WorldBackup, 0)
	for _, k := range keys {
		result = append(result, worlds[k])
	}

	return result, nil
}
