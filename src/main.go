package main

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
	"github.com/google/go-containerregistry/pkg/crane"
	"net/http"
	"strings"
	"sync"
	"time"
)

type ContainerInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Image     string    `json:"image"`
	LatestSHA string    `json:"latest_sha"`
	Updatable bool      `json:"updatable"`
	BuildAt   time.Time `json:"build_at"`
	Status    string    `json:"status"`
	ImageInfo ImageInfo `json:"image_info"`
}

type ImageInfo struct {
	ID       string   `json:"id"`
	RepoTags []string `json:"repo_tags"`
	Digests  []string `json:"digests"`
	Created  string   `json:"created"`
}

func main() {

	containers := getContainers()
	containers = getContainerSHAs(containers)

	for _, c := range containers {
		fmt.Printf("ID: %s, Name: %s, Image: %s, BuildAt: %s, LatestSHA: %s, Updatable: %t, ImageInfo: %s \n",
			c.ID, c.Name, c.Image, c.BuildAt, c.LatestSHA, c.Updatable, c.ImageInfo)
	}

	router := gin.Default()
	router.GET("/containers", getContainersRoute)

	err := router.Run(":8000")
	if err != nil {
		return
	}

}

func getContainerSHAs(containers []ContainerInfo) []ContainerInfo {
	var wg sync.WaitGroup
	result := make([]ContainerInfo, len(containers))

	for i, c := range containers {
		wg.Add(1)
		go func(index int, container ContainerInfo, res []ContainerInfo) {
			defer wg.Done()
			res[index] = getContainerSHA(container)
		}(i, c, result)
	}

	// Wait for all goroutines to finish
	wg.Wait()

	return result
}

func getContainerSHA(c ContainerInfo) ContainerInfo {
	if len(c.ImageInfo.Digests) != 0 {
		var repoTag = c.ImageInfo.RepoTags[0]

		split := strings.Split(repoTag, ":")
		repo := split[0]
		tag := split[1]

		c.LatestSHA = getLatestSHAbyAPI(repo, tag)
		if c.LatestSHA == "" {
			fmt.Println("Crane fallback!")
			c.LatestSHA = getLatestSHAbyCrane(repoTag)
		}

		for _, digest := range c.ImageInfo.Digests {
			c.Updatable = !strings.Contains(digest, c.LatestSHA)
		}
	}

	return c
}

func getContainers() []ContainerInfo {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	returnedContainers, err := cli.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		panic(err)
	}

	var containers []ContainerInfo
	for _, rc := range returnedContainers {
		imageInfo, err := getImageInfo(ctx, cli, rc.ImageID)
		if err != nil {
			panic(err)
		}

		container := ContainerInfo{
			ID:        rc.ID,
			Name:      rc.Names[0],
			Image:     rc.Image,
			BuildAt:   time.Unix(rc.Created, 0),
			ImageInfo: imageInfo,
			LatestSHA: "",
			Updatable: false,
		}

		containers = append(containers, container)
	}

	return containers
}

func getContainersRoute(c *gin.Context) {
	containers := getContainers()
	containers = getContainerSHAs(containers)
	c.IndentedJSON(http.StatusOK, containers)
}

func getImageInfo(ctx context.Context, cli *client.Client, imageID string) (ImageInfo, error) {
	image, _, err := cli.ImageInspectWithRaw(ctx, imageID)
	if err != nil {
		return ImageInfo{}, err
	}

	repoTags := make([]string, len(image.RepoTags))
	for i, tag := range image.RepoTags {
		parts := strings.Split(tag, ":")
		if len(parts) > 1 {
			repoTags[i] = parts[0] + ":" + parts[1]
		}
	}

	//parsedTime, err := time.Parse("2006-01-02T15:04:05.999999999Z", image.Created)
	if err != nil {
		panic(err)
	}

	// parts := strings.Split(repoTags[0], ":")
	// fmt.Println(getDigestInfo(repoTags[0]))

	return ImageInfo{
		ID:       image.ID,
		RepoTags: repoTags,
		Digests:  image.RepoDigests,
		// TODO: Convert to time.Time
		Created: image.Created,
	}, nil
}

func getLatestSHAbyCrane(image string) string {
	digest, err := crane.Digest(image)
	if err != nil {
		panic(err)
	}

	return digest
}
