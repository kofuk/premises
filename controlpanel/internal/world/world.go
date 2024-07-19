package world

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/kofuk/premises/internal/entity/web"
	"github.com/kofuk/premises/internal/s3wrap"
)

type WorldService struct {
	s3     *s3wrap.Client
	bucket string
}

func New(ctx context.Context, bucket string, forcePathStyle bool) (*WorldService, error) {
	client, err := s3wrap.New(ctx, forcePathStyle)
	if err != nil {
		return nil, err
	}

	return &WorldService{
		s3:     client,
		bucket: bucket,
	}, nil
}

func extractWorldInfoFromKey(key string) (string, string, error) {
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

func (ws *WorldService) GetWorlds(ctx context.Context) ([]web.World, error) {
	worlds := make(map[string]web.World)
	objects, err := ws.s3.ListObjects(ctx, ws.bucket)
	if err != nil {
		return nil, err
	}

	for _, obj := range objects {
		world, gen, err := extractWorldInfoFromKey(obj.Key)
		if err != nil {
			return nil, err
		}
		worlds[world] = web.World{
			WorldName: world,
			Generations: append(worlds[world].Generations, web.WorldGeneration{
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

	result := make([]web.World, 0)
	for _, k := range keys {
		result = append(result, worlds[k])
	}

	return result, nil
}
