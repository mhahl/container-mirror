package service

import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	docker "github.com/fsouza/go-dockerclient"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)


/**
 * containerIndexFile: Default file to load mirror configuration.
 * downedIndexFile: Index if downloaded files.
 */
const (
	containerIndexFile  = "containers.yaml"
	downloadedIndexFile = "containers-index.yaml"
)

var config Config;

/**
 * Config is the result of the parsed yaml for the mirror configuration.
 *
 * Cleanup:      Cleanup registry
 * Workers:      How many workers when mirroring
 * Repositories: List of repositories to mirror.
 * Target:       Target Repository
 */
type Config struct {
	Cleanup      bool         `yaml:"cleanup"`
	Workers      int          `yaml:"workers"`
	Repositories []Repository `yaml:"repositories,flow"`
	Target       TargetConfig `yaml:"target"`
}

/**
 * Target repository and optional prefix.
 */
type TargetConfig struct {
	Registry string `yaml:"registry"`
	Prefix   string `yaml:"prefix"`
}

/**
 * A single container repository.
 */
type Repository struct {
	PrivateRegistry string            `yaml:"private_registry"`
	Name            string            `yaml:"name"`
	MatchTags       []string          `yaml:"match_tag"`
	DropTags        []string          `yaml:"ignore_tag"`
	MaxTags         int               `yaml:"max_tags"`
//	MaxTagAge       *Duration         `yaml:"max_tag_age"`
	RemoteTagSource string            `yaml:"remote_tags_source"`
	RemoteTagConfig map[string]string `yaml:"remote_tags_config"`
	TargetPrefix    *string           `yaml:"target_prefix"`
	Host            string            `yaml:"host"`
}

/**
 * GetServiceInterface defines a Container service.
 */
type ContainerServiceInterface interface {
	Get() error
}

// GetService structure definition
type ContainerService struct {
	config       Config
	verbose      bool
	ignoreErrors bool
	logger       *log.Logger
}


// NewGetService return a new instace of GetService
func NewContainerService(config Config, verbose bool, ignoreErrors bool, logger *log.Logger) ContainerServiceInterface {
	return &ContainerService{
		config: config,
		verbose: verbose,
		ignoreErrors: ignoreErrors,
		logger: logger
	}
}

func (g *ContainerService) Get() error {
	fmt.Println("test")
	return nil
}