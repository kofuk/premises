package dockerstack

import (
	"context"
	"errors"

	docker "github.com/docker/docker/client"
	"github.com/kofuk/premises/ostack-fake/dockerstack/wrapper"
	"github.com/kofuk/premises/ostack-fake/entity"
)

func GetServerDetails(ctx context.Context, docker *docker.Client) (*entity.ServerDetailsResp, error) {
	containers, err := wrapper.GetManagedContainers(ctx, docker)
	if err != nil {
		return nil, err
	}

	result := make([]entity.ServerDetail, 0)
	for _, container := range containers {
		status := "ACTIVE"
		if container.State != "running" {
			status = "SHUTOFF"
		}

		serverDetail := entity.ServerDetail{
			ID:     container.Labels["org.kofuk.premises.id"],
			Name:   container.Labels["org.kofuk.premises.name"],
			Status: status,
			Addresses: map[string][]entity.ServerDetailAddress{
				"ext-127-0-0-1-xxx": {
					{
						Addr:    "::1",
						Version: 6,
					},
					{
						Addr:    "127.0.0.1",
						Version: 4,
					},
				},
			},
			Metadata: entity.ServerDetailMetadata{
				InstanceNameTag: container.Labels["org.kofuk.premises.instance_name_tag"],
			},
		}

		result = append(result, serverDetail)
	}

	return &entity.ServerDetailsResp{Servers: result}, nil
}

func GetServerDetail(ctx context.Context, docker *docker.Client, id string) (*entity.ServerDetailResp, error) {
	details, err := GetServerDetails(ctx, docker)
	if err != nil {
		return nil, err
	}

	for _, server := range details.Servers {
		if server.ID == id {
			return &entity.ServerDetailResp{
				Server: server,
			}, nil
		}
	}

	return nil, errors.New("Not found")
}

func GetImages(ctx context.Context, docker *docker.Client) (*entity.ImageResp, error) {
	images, err := wrapper.GetManagedImages(ctx, docker)
	if err != nil {
		return nil, err
	}

	result := make([]entity.Image, 0)
	for _, image := range images {
		result = append(result, entity.Image{
			ID:     image.Labels["org.kofuk.premises.id"],
			Name:   image.Labels["org.kofuk.premises.name"],
			Status: "active",
		})
	}

	return &entity.ImageResp{Images: result}, nil
}

func LaunchServer(ctx context.Context, docker *docker.Client, server entity.LaunchServerReq) (*entity.LaunchServerResp, error) {
	imageID := server.Server.ImageRef
	userData := server.Server.UserData
	nameTag := server.Server.MetaData.InstanceNameTag

	serverId, err := wrapper.LaunchContainer(ctx, docker, imageID, userData, nameTag)
	if err != nil {
		return nil, err
	}

	resp := entity.LaunchServerResp{}
	resp.Server.ID = serverId

	return &resp, nil
}

func StopServer(ctx context.Context, docker *docker.Client, serverId string) error {
	return wrapper.StopContainer(ctx, docker, serverId)
}

func CreateImage(ctx context.Context, docker *docker.Client, serverId, imageName string) error {
	return wrapper.CreateImage(ctx, docker, serverId, imageName)
}

func DeleteServer(ctx context.Context, docker *docker.Client, serverId string) error {
	return wrapper.DeleteServerAndImage(ctx, docker, serverId)
}
