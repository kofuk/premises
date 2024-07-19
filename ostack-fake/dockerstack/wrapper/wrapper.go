package wrapper

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/strslice"
	docker "github.com/docker/docker/client"
	"github.com/google/uuid"
)

func GetManagedContainers(ctx context.Context, docker *docker.Client) ([]types.Container, error) {
	return docker.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("label", "org.kofuk.premises.managed")),
	})
}

func GetManagedImages(ctx context.Context, docker *docker.Client) ([]image.Summary, error) {
	return docker.ImageList(ctx, image.ListOptions{
		Filters: filters.NewArgs(filters.Arg("label", "org.kofuk.premises.managed")),
	})
}

func FindDockerImageByOstackImageID(ctx context.Context, docker *docker.Client, imageId string) (string, error) {
	image, err := docker.ImageList(ctx, image.ListOptions{
		Filters: filters.NewArgs(
			filters.Arg("label", "org.kofuk.premises.managed"),
			filters.Arg("label", fmt.Sprintf("org.kofuk.premises.id=%s", imageId)),
		),
	})
	if err != nil {
		return "", err
	}
	if len(image) == 0 {
		return "", fmt.Errorf("unknown image: %s", imageId)
	}
	return image[0].ID, nil
}

func getHostNameserver() (string, error) {
	f, err := os.Open("/etc/resolv.conf")
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, " ")
		if len(fields) != 2 {
			continue
		}

		if fields[0] == "nameserver" {
			return fields[1], nil
		}
	}

	return "", errors.New("nameserver setting not found")
}

func createContainer(ctx context.Context, docker *docker.Client, imageID, nameTag string) (*container.CreateResponse, string, error) {
	image, err := FindDockerImageByOstackImageID(ctx, docker, imageID)
	if err != nil {
		return nil, "", err
	}

	serverId := uuid.New().String()

	containerConfig := container.Config{
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

	// Forwarding host's nameserver setting is required because
	// the container needs to resolve Dev Container's service container.
	ns, err := getHostNameserver()
	if err != nil {
		return nil, "", err
	}

	hostConfig := container.HostConfig{
		Binds:       binds,
		CapAdd:      strslice.StrSlice{"MKNOD"},
		NetworkMode: "host",
		Privileged:  true,
		DNS:         []string{ns},
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

	if err := docker.CopyToContainer(ctx, containerId, "/", &buf, container.CopyToContainerOptions{}); err != nil {
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

	if err := docker.ContainerStart(ctx, createResp.ID, container.StartOptions{}); err != nil {
		return "", err
	}

	return serverId, nil
}

func getContainerIdByServerId(ctx context.Context, docker *docker.Client, serverId string) (string, error) {
	containers, err := docker.ContainerList(ctx, container.ListOptions{
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
		return "", fmt.Errorf("unknown container: %s", serverId)
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

	if _, err := docker.ContainerCommit(ctx, containerId, container.CommitOptions{
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
	imageInspect, _, err := docker.ImageInspectWithRaw(ctx, imageId)
	if err != nil {
		return err
	}
	if len(imageInspect.RepoTags) > 0 {
		imageId = imageInspect.RepoTags[0]
	}

	if _, err := docker.ImageRemove(ctx, imageId, image.RemoveOptions{}); err != nil {
		return fmt.Errorf("error removing image: %s: %w", imageId, err)
	}
	return nil
}

func DeleteServerAndImage(ctx context.Context, docker *docker.Client, serverId string) error {
	containerId, err := getContainerIdByServerId(ctx, docker, serverId)
	if err != nil {
		return err
	}

	containerInfo, err := docker.ContainerInspect(ctx, containerId)
	if err != nil {
		return err
	}

	if err := docker.ContainerRemove(ctx, containerId, container.RemoveOptions{}); err != nil {
		return err
	}

	if err := removeOrUntagImage(ctx, docker, containerInfo.Image); err != nil {
		return err
	}

	tagName := fmt.Sprintf("premises.kofuk.org/dev-temp:%s", serverId)
	if err := removeOrUntagImage(ctx, docker, tagName); err != nil {
		return err
	}

	return nil
}
