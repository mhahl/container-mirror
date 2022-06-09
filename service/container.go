package service

import (
	"fmt"
	"io/ioutil"
	//"os"
	"runtime"
	//"strconv"
	//"strings"
	//"sync"
	"time"
	"context"
	log "github.com/sirupsen/logrus"


	"github.com/cenkalti/backoff/v4"
	//"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"gopkg.in/yaml.v3"
)

var (
	config ContainerConfig
	dockerClient client.Client
)

/**
 * Config is the result of the parsed yaml for the mirror configuration.
 *
 * Cleanup:      Cleanup registry
 * Workers:      How many workers when mirroring
 * Repositories: List of repositories to mirror.
 * Target:       Target Repository
 */
type ContainerConfig struct {
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
	config       ContainerConfig
	dockerClient *client.Client
	verbose      bool
	ignoreErrors bool
	logger 		*log.Logger
}


func validateConfig(containerConfig *ContainerConfig) {
	if containerConfig.Target.Registry == "" {
		log.Fatalf("Missing `target.registry` in configuration file")
	}

	if containerConfig.Workers == 0 {
		containerConfig.Workers = runtime.NumCPU()
	}

}

func NewContainerService(configFile string, verbose bool, ignoreErrors bool, logger *log.Logger) ContainerServiceInterface {

	/**
	* Read the configuration file.
	*/
	content, readError := ioutil.ReadFile(configFile)
	if (readError != nil) {
		log.Fatalf(fmt.Sprintf("Failed reading configuration: %s", readError))
	}

	/**
	* Parse the YAML and store into `containerConfig`
	*/
	var containerConfig ContainerConfig
	if parseError := yaml.Unmarshal(content, &containerConfig); parseError != nil {
		log.Fatalf(fmt.Sprintf("Failed parsing configuration: %s", parseError))
	}

	/**
	 * Check and set default values for configuration.
	 */
	validateConfig(&containerConfig)

	/**
	 * Create a new docker client.
	 */
	ctx := context.Background()
	dockerClient, dockerError := client.NewClientWithOpts(client.FromEnv)
	if dockerError != nil {
		log.Fatalf(fmt.Sprintf("Failed creating Docker client: %s", dockerError))
	}

	info, err := dockerClient.Info(ctx)
	if err != nil {
		log.Fatalf("Could not get Docker info: %s", err.Error())
	}


	log.Info(fmt.Sprintf("%s",info))

	/**
	 * Configure backoff settings 
	 * for the container pull.
	 */
	backoffSettings := backoff.NewExponentialBackOff()

	backoffSettings.InitialInterval = 1 * time.Second
	backoffSettings.MaxElapsedTime = 10 * time.Second

	/* Retun the new Object */
	return &ContainerService{
		config: containerConfig,
		dockerClient: dockerClient,
		verbose: verbose,
		ignoreErrors: ignoreErrors,
		logger: logger,
	}
}

func (g *ContainerService) Get() error {


	fmt.Println(config)
	fmt.Println("test")
	return nil
}