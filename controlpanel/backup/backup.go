package backup

import (
	"context"
	"fmt"
	"sort"
	"strings"

	entity "github.com/kofuk/premises/internal/entity/web"
	"github.com/kofuk/premises/internal/s3wrap"
)

type BackupService struct {
	s3     *s3wrap.Client
	bucket string
}

func New(awsAccessKey, awsSecretKey, s3Endpoint, bucket string) *BackupService {
	if strings.HasPrefix(s3Endpoint, "http://s3.premises.local:") {
		// When S3 endpoint is localhost, it should be a development environment on Docker.
		// We implicitly rewrite the address so that we can access S3 host.
		s3Endpoint = strings.Replace(s3Endpoint, "http://s3.premises.local", "http://localhost", 1)
	}

	client := s3wrap.New(awsAccessKey, awsSecretKey, s3Endpoint)

	return &BackupService{
		s3:     client,
		bucket: bucket,
	}
}

func extractBackupInfoFromKey(key string) (string, string, error) {
	splitIndex := strings.IndexRune(key, '/')
	if splitIndex < 0 {
		return "", "", fmt.Errorf("invalid backup key: %s", key)
	}
	world := string(key[0:splitIndex])
	name := string(key[splitIndex+1:])
	if strings.HasSuffix(name, ".tar.zst") {
		name = strings.TrimSuffix(name, ".tar.zst")
	} else if strings.HasSuffix(name, ".tar.xz") {
		name = strings.TrimSuffix(name, ".tar.xz")
	} else {
		name = strings.TrimSuffix(name, ".zip")
	}
	return world, name, nil
}

func (backup *BackupService) GetWorlds(ctx context.Context) ([]entity.WorldBackup, error) {
	worlds := make(map[string]entity.WorldBackup)
	objects, err := backup.s3.ListObjects(ctx, backup.bucket)
	if err != nil {
		return nil, err
	}

	for _, obj := range objects {
		world, gen, err := extractBackupInfoFromKey(obj.Key)
		if err != nil {
			return nil, err
		}
		worlds[world] = entity.WorldBackup{
			WorldName: world,
			Generations: append(worlds[world].Generations, entity.BackupGeneration{
				Gen:       gen,
				ID:        obj.Key,
				Timestamp: int(obj.Timestamp.UnixMilli()),
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
