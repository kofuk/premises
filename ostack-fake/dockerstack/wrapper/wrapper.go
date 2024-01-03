package wrapper

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/strslice"
	docker "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
)

func GetManagedContainers(ctx context.Context, docker *docker.Client) ([]types.Container, error) {
	return docker.ContainerList(ctx, types.ContainerListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("label", "org.kofuk.premises.managed")),
	})
}

func GetManagedImages(ctx context.Context, docker *docker.Client) ([]types.ImageSummary, error) {
	return docker.ImageList(ctx, types.ImageListOptions{
		Filters: filters.NewArgs(filters.Arg("label", "org.kofuk.premises.managed")),
	})
}

func FindDockerImageByOstackImageID(ctx context.Context, docker *docker.Client, imageId string) (string, error) {
	image, err := docker.ImageList(ctx, types.ImageListOptions{
		Filters: filters.NewArgs(
			filters.Arg("label", "org.kofuk.premises.managed"),
			filters.Arg("label", fmt.Sprintf("org.kofuk.premises.id=%s", imageId)),
		),
	})
	if err != nil {
		return "", err
	}
	if len(image) == 0 {
		return "", fmt.Errorf("Unknown image: %s", imageId)
	}
	return image[0].ID, nil
}

func createContainer(ctx context.Context, docker *docker.Client, imageID, nameTag string) (*container.CreateResponse, string, error) {
	image, err := FindDockerImageByOstackImageID(ctx, docker, imageID)
	if err != nil {
		return nil, "", err
	}

	serverId := uuid.New().String()

	containerConfig := container.Config{
		ExposedPorts: nat.PortSet{
			"8521/tcp":  struct{}{},
			"25565/tcp": struct{}{},
		},
		Image: image,
		Labels: map[string]string{
			"org.kofuk.premises.managed":           "true",
			"org.kofuk.premises.id":                serverId,
			"org.kofuk.premises.name":              "",
			"org.kofuk.premises.instance_name_tag": nameTag,
		},
	}

	var binds []string
	if _, err := os.Stat(filepath.Join(os.TempDir(), "premises")); err == nil {
		binds = append(binds, fmt.Sprintf("%s:/premises-dev", filepath.Join(os.TempDir(), "premises")))
	}
	if runtime.GOOS == "linux" {
		os.MkdirAll("/tmp/premises-data", 0755)
		binds = append(binds, "/tmp/premises-data:/opt/premises")
	}

	hostConfig := container.HostConfig{
		Binds: binds,
		PortBindings: nat.PortMap{
			"8521/tcp": []nat.PortBinding{
				{
					HostPort: "8521",
				},
			},
			"25565/tcp": []nat.PortBinding{
				{
					HostPort: "25565",
				},
			},
		},
		CapAdd: strslice.StrSlice{"MKNOD"},
		ExtraHosts: []string{
			"host.docker.internal:host-gateway",
		},
		Privileged: true,
	}

	resp, err := docker.ContainerCreate(ctx, &containerConfig, &hostConfig, nil, nil, "")
	if err != nil {
		return nil, "", err
	}
	return &resp, serverId, nil
}

func copyUserDataToContainer(ctx context.Context, docker *docker.Client, containerId string, userData []byte) error {
	buf := bytes.Buffer{}
	tarWriter := tar.NewWriter(&buf)
	tarWriter.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     "userdata",
		Size:     int64(len(userData)),
		Mode:     0644,
		Uid:      1000,
		Gid:      1000,
		Format:   tar.FormatGNU,
	})
	tarWriter.Write(userData)
	tarWriter.Close()

	if err := docker.CopyToContainer(ctx, containerId, "/", &buf, types.CopyToContainerOptions{}); err != nil {
		return err
	}

	return nil
}

func LaunchContainer(ctx context.Context, docker *docker.Client, imageID, userData, nameTag string) (string, error) {
	createResp, serverId, err := createContainer(ctx, docker, imageID, nameTag)
	if err != nil {
		return "", err
	}

	if err := copyUserDataToContainer(ctx, docker, createResp.ID, []byte(userData)); err != nil {
		return "", err
	}

	if err := docker.ContainerStart(ctx, createResp.ID, types.ContainerStartOptions{}); err != nil {
		return "", err
	}

	return serverId, nil
}

func getContainerIdByServerId(ctx context.Context, docker *docker.Client, serverId string) (string, error) {
	containers, err := docker.ContainerList(ctx, types.ContainerListOptions{
		All: true,
		Filters: filters.NewArgs(
			filters.Arg("label", "org.kofuk.premises.managed"),
			filters.Arg("label", fmt.Sprintf("org.kofuk.premises.id=%s", serverId)),
		),
	})
	if err != nil {
		return "", err
	}
	if len(containers) == 0 {
		return "", fmt.Errorf("Unknown container: %s", serverId)
	}
	return containers[0].ID, nil
}

func StopContainer(ctx context.Context, docker *docker.Client, serverId string) error {
	containerId, err := getContainerIdByServerId(ctx, docker, serverId)
	if err != nil {
		return err
	}
	if err := docker.ContainerStop(ctx, containerId, container.StopOptions{
		Signal: "SIGINT",
	}); err != nil {
		return err
	}
	return nil
}

func createBuildContextForRebuild(baseImageRef string) io.Reader {
	dockerfile := []byte(fmt.Sprintf("FROM %s\n", baseImageRef))
	buf := bytes.Buffer{}
	tarWriter := tar.NewWriter(&buf)
	tarWriter.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     "Dockerfile",
		Size:     int64(len(dockerfile)),
		Mode:     0644,
		Uid:      1000,
		Gid:      1000,
		Format:   tar.FormatGNU,
	})
	tarWriter.Write(dockerfile)
	tarWriter.Close()

	return &buf
}

func CreateImage(ctx context.Context, docker *docker.Client, serverId, imageName string) error {
	containerId, err := getContainerIdByServerId(ctx, docker, serverId)
	if err != nil {
		return err
	}

	tagName := fmt.Sprintf("premises.kofuk.org/dev-temp:%s", serverId)

	if _, err := docker.ContainerCommit(ctx, containerId, types.ContainerCommitOptions{
		Reference: tagName,
	}); err != nil {
		return err
	}

	buildContext := createBuildContextForRebuild(tagName)

	imageId := uuid.New().String()

	buildResp, err := docker.ImageBuild(ctx, buildContext, types.ImageBuildOptions{
		Tags:           []string{fmt.Sprintf("premises.kofuk.org/dev-runner:%s", serverId)},
		SuppressOutput: true,
		Remove:         true,
		Dockerfile:     "Dockerfile",
		Labels: map[string]string{
			"org.kofuk.premises.managed": "true",
			"org.kofuk.premises.id":      imageId,
			"org.kofuk.premises.name":    imageName,
		},
	})
	if err != nil {
		return err
	}
	io.ReadAll(buildResp.Body)

	return nil
}

func removeOrUntagImage(ctx context.Context, docker *docker.Client, imageId string) error {
	image, _, err := docker.ImageInspectWithRaw(ctx, imageId)
	if err != nil {
		return err
	}
	if len(image.RepoTags) > 0 {
		imageId = image.RepoTags[0]
	}

	if _, err := docker.ImageRemove(ctx, imageId, types.ImageRemoveOptions{}); err != nil {
		return fmt.Errorf("Error removing image: %s: %w", imageId, err)
	}
	return nil
}

func DeleteServerAndImage(ctx context.Context, docker *docker.Client, serverId string) error {
	containerId, err := getContainerIdByServerId(ctx, docker, serverId)
	if err != nil {
		return err
	}

	container, err := docker.ContainerInspect(ctx, containerId)
	if err != nil {
		return err
	}

	if err := docker.ContainerRemove(ctx, containerId, types.ContainerRemoveOptions{}); err != nil {
		return err
	}

	if err := removeOrUntagImage(ctx, docker, container.Image); err != nil {
		return err
	}

	tagName := fmt.Sprintf("premises.kofuk.org/dev-temp:%s", serverId)
	if err := removeOrUntagImage(ctx, docker, tagName); err != nil {
		return err
	}

	return nil
}
