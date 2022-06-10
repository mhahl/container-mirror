package service;

import (
	"os"
	"fmt"
	"strconv"
	//"strings"
	//"sync"
	//"context"
	"bytes"
	"encoding/json"
	"net/http"
	"time"
	"sort"
	"strings"
	log "github.com/sirupsen/logrus"
	//"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

const defaultSleepDuration time.Duration = 60 * time.Second

const (
	dockerHub = "hub.docker.com"
	quay      = "quay.io"
	gcr       = "gcr.io"
	k8s       = "k8s.gcr.io"
)

var (
	PTransport = &http.Transport{Proxy: http.ProxyFromEnvironment}
	httpClient = &http.Client{Timeout: 10 * time.Second, Transport: PTransport}
)

// DockerTagsResponse is Docker Registry v2 compatible struct
type DockerTagsResponse struct {
	Count    int             `json:"count"`
	Next     *string         `json:"next"`
	Previous *string         `json:"previous"`
	Results  []RepositoryTag `json:"results"`
}

// QuayTagsResponse is Quay API v1 compatible struct
type QuayTagsResponse struct {
	HasAdditional bool            `json:"has_additional"`
	Page          int             `json:"page"`
	Tags          []RepositoryTag `json:"tags"`
}

// GCRTagsResponse is GCR API v2 compatible struct
type GCRTagsResponse struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

/**
 * RepositoryTag is Docker, Quay, GCR API compatible struct, holding the individual
 * tags for the requested repository.
 */
type RepositoryTag struct {
	Name         string    `json:"name"`
	LastUpdated  time.Time `json:"last_updated"`
	LastModified time.Time `json:"last_modified"`
}


type Mirror struct {
	mirrorClient *client.Client   // docker client used to pull, tag and push images
	log          *log.Entry      // logrus logger with the relevant custom fields
	repo         Repository      // repository the mirror
	remoteTags   []RepositoryTag // list of remote repository tags (post filtering)
}

/**
 * Setup a repository for mirroring.
 */
func (m *Mirror) setup(repo Repository) (err error) {
	m.log = log.WithField("full_repo", repo.Name)
	m.repo = repo

	if strings.Contains(repo.Name, ":") {
		chunk := strings.SplitN(repo.Name, ":", 2)
		m.repo.Name = chunk[0]
		m.repo.MatchTags = []string{chunk[1]}
	}

	return nil

}

/**
 * Return an array of remote tags.
 */
func (m *Mirror) getRemoteTags() ([]RepositoryTag, error) {
	
// Get tags information from Docker Hub, Quay, GCR or k8s.gcr.io.
	var url string
	fullRepoName := m.repo.Name
	token := ""

	switch m.repo.Host {
	case dockerHub:
		if !strings.Contains(fullRepoName, "/") {
			fullRepoName = "library/" + m.repo.Name
		}

		if os.Getenv("DOCKERHUB_USER") != "" && os.Getenv("DOCKERHUB_PASSWORD") != "" {
			m.log.Info("Getting tags using docker hub credentials from environment")

			message, err := json.Marshal(map[string]string{
				"username": os.Getenv("DOCKERHUB_USER"),
				"password": os.Getenv("DOCKERHUB_PASSWORD"),
			})

			if err != nil {
				return nil, err
			}

			resp, err := http.Post("https://hub.docker.com/v2/users/login/", "application/json", bytes.NewBuffer(message))
			if err != nil {
				return nil, err
			}

			var result map[string]interface{}

			json.NewDecoder(resp.Body).Decode(&result)
			token = result["token"].(string)
		}

		url = fmt.Sprintf("https://registry.hub.docker.com/v2/repositories/%s/tags/?page_size=2048", fullRepoName)
	case quay:
		url = fmt.Sprintf("https://quay.io/api/v1/repository/%s/tag", fullRepoName)
	case gcr:
		url = fmt.Sprintf("https://gcr.io/v2/%s/tags/list", fullRepoName)
	case k8s:
		url = fmt.Sprintf("https://k8s.gcr.io/v2/%s/tags/list", fullRepoName)
	}

	var allTags []RepositoryTag

	fmt.Println(url)
	fmt.Println(allTags)
	fmt.Println(token)


search:
	for {
		var (
			err     error
			res     *http.Response
			req     *http.Request
			retries int = 5
		)

		for retries > 0 {
			req, err = http.NewRequest("GET", url, nil)
			if err != nil {
				return nil, err
			}

			if token != "" {
				req.Header.Set("Authorization", fmt.Sprintf("JWT %s", token))
			}

			res, err = httpClient.Do(req)

			if err != nil {
				m.log.Warningf(err.Error())
				m.log.Warningf("Failed to get %s, retrying", url)
				retries--
			} else if res.StatusCode == 429 {
				sleepTime := getSleepTime(res.Header.Get("X-RateLimit-Reset"), time.Now())
				m.log.Infof("Rate limited on %s, sleeping for %s", url, sleepTime)
				time.Sleep(sleepTime)
				retries--
			} else if res.StatusCode < 200 || res.StatusCode >= 300 {
				m.log.Warningf("Get %s failed with %d, retrying", url, res.StatusCode)
				retries--
			} else {
				break
			}

		}

		if err != nil {
			return nil, err
		}
		defer res.Body.Close()

		dc := json.NewDecoder(res.Body)

		switch m.repo.Host {
		case dockerHub:
			var tags DockerTagsResponse
			if err = dc.Decode(&tags); err != nil {
				return nil, err
			}

			allTags = append(allTags, tags.Results...)
			if tags.Next == nil {
				break search
			}

			url = *tags.Next
		case quay:
			var tags QuayTagsResponse
			if err := dc.Decode(&tags); err != nil {
				return nil, err
			}
			allTags = append(allTags, tags.Tags...)
			break search
		case gcr:
			var tags GCRTagsResponse
			if err := dc.Decode(&tags); err != nil {
				return nil, err
			}
			for _, tag := range tags.Tags {
				allTags = append(allTags, RepositoryTag{
					Name: tag,
				})
			}
			break search
		case k8s:
			var tags GCRTagsResponse
			if err := dc.Decode(&tags); err != nil {
				return nil, err
			}
			for _, tag := range tags.Tags {
				allTags = append(allTags, RepositoryTag{
					Name: tag,
				})
			}
			break search
		}
	}

	// sort the tags by updated/modified time if applicable, newest first
	switch m.repo.Host {
	case dockerHub:
		sort.Slice(allTags, func(i, j int) bool {
			return allTags[i].LastUpdated.After(allTags[j].LastUpdated)
		})
	case quay:
		sort.Slice(allTags, func(i, j int) bool {
			return allTags[i].LastModified.After(allTags[j].LastModified)
		})
	}

	fmt.Println(allTags)
	return allTags, nil

}

func getSleepTime(rateLimitReset string, now time.Time) time.Duration {
	rateLimitResetInt, err := strconv.ParseInt(rateLimitReset, 10, 64)

	if err != nil {
		return defaultSleepDuration
	}

	sleepTime := time.Unix(rateLimitResetInt, 0)
	calculatedSleepTime := sleepTime.Sub(now)

	if calculatedSleepTime < (0 * time.Second) {
		return 0 * time.Second
	}

	return calculatedSleepTime
}