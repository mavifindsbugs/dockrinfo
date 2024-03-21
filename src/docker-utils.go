package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type TokenResponse struct {
	Token string `json:"token"`
}

var urlAuth = "https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s:pull"
var urlRepo = "https://registry-1.docker.io/v2/%s/manifests/%s"

func getLatestSHAbyAPI(repo string, tag string) string {
	if !strings.Contains(repo, "/") {
		repo = "library/" + repo
	}

	res, err := http.Get(fmt.Sprintf(urlAuth, repo))
	if err != nil {
		panic(err)
	}

	if res.StatusCode != http.StatusOK {
		panic(fmt.Errorf("unexpected status code: %d", res.StatusCode))
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	var tokenResponse TokenResponse
	err = json.Unmarshal(body, &tokenResponse)
	if err != nil {
		panic(err)
	}

	url := fmt.Sprintf(urlRepo, repo, tag)
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		panic(err)
	}

	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.list.v2+json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenResponse.Token))

	res, err = http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	dockerContentDigest := res.Header.Get("docker-content-digest")

	return dockerContentDigest
}
