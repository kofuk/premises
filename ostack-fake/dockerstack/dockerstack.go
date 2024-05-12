package dockerstack

import (
	"context"
	"errors"
	"fmt"

	docker "github.com/docker/docker/client"
	"github.com/kofuk/premises/ostack-fake/dockerstack/wrapper"
	"github.com/kofuk/premises/ostack-fake/entity"
)

func ListServerDetails(ctx context.Context, docker *docker.Client) (*entity.ListServerDetailsResp, error) {
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
			Volumes: []entity.Volume{
				{
					ID: fmt.Sprintf("volume_%s", container.Labels["org.kofuk.premises.id"]),
				},
			},
		}

		result = append(result, serverDetail)
	}

	return &entity.ListServerDetailsResp{Servers: result}, nil
}

func GetServerDetail(ctx context.Context, docker *docker.Client, id string) (*entity.GetServerDetailResp, error) {
	details, err := ListServerDetails(ctx, docker)
	if err != nil {
		return nil, err
	}

	for _, server := range details.Servers {
		if server.ID == id {
			return &entity.GetServerDetailResp{
				Server: server,
			}, nil
		}
	}

	return nil, errors.New("not found")
}

func ListVolumes(ctx context.Context, docker *docker.Client) (*entity.ListVolumesResp, error) {
	images, err := wrapper.GetManagedImages(ctx, docker)
	if err != nil {
		return nil, err
	}

	result := make([]entity.Volume, 0)
	for _, image := range images {
		result = append(result, entity.Volume{
			ID:   image.Labels["org.kofuk.premises.id"],
			Name: image.Labels["org.kofuk.premises.name"],
		})
	}

	return &entity.ListVolumesResp{Volumes: result}, nil
}

func LaunchServer(ctx context.Context, docker *docker.Client, server entity.LaunchServerReq) (*entity.LaunchServerResp, error) {
	if len(server.Server.BlockDevices) == 0 {
		return nil, errors.New("block_device_mapping_v2 is missing")
	}

	imageID := server.Server.BlockDevices[0].UUID
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
