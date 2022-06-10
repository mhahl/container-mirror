package service

import (
	"fmt"
	"io/ioutil"
	//"os"
	"runtime"
	//"strconv"
	"strings"
	"sync"
	"time"
	"context"
	log "github.com/sirupsen/logrus"


	"github.com/cenkalti/backoff/v4"
	//"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"gopkg.in/yaml.v3"
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
//	RemoteTagSource string            `yaml:"remote_tags_source"`
//	RemoteTagConfig map[string]string `yaml:"remote_tags_config"`
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
	config       *ContainerConfig
	dockerClient *client.Client
	prefix       string
	verbose      bool
	ignoreErrors bool
	logger 		*log.Logger
}


func validateConfig(containerConfig *ContainerConfig) () {
	if containerConfig.Target.Registry == "" {
		log.Fatalf("Missing `target.registry` in configuration file")
	}

	if containerConfig.Workers == 0 {
		log.Info("Setting workers to z")
		containerConfig.Workers = runtime.NumCPU()
	}

}

func NewContainerService(configFile string, prefix string, verbose bool, ignoreErrors bool, logger *log.Logger) ContainerServiceInterface {

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

	log.Infof("Connected to Docker daemon: %s @ %s", info.Name, info.ServerVersion)

	/**
	 * Configure backoff settings 
	 * for the container pull.
	 */
	backoffSettings := backoff.NewExponentialBackOff()
	backoffSettings.InitialInterval = 1 * time.Second
	backoffSettings.MaxElapsedTime = 10 * time.Second

	/* Retun the new Object */
	return &ContainerService{
		config: &containerConfig,
		dockerClient: dockerClient,
		prefix: prefix,
		verbose: verbose,
		ignoreErrors: ignoreErrors,
		logger: logger,
	}
}

func (s *ContainerService) Get() error {

	workerCh := make(chan Repository, 5)
	var wg sync.WaitGroup

	/* Start background workers */
	for i := 0; i < s.config.Workers; i++ {
		go worker(&wg, workerCh, s.dockerClient, s.config)
	}

	// add jobs for the workers
	for _, dockerRepo := range s.config.Repositories {

		/**
		 * Check if the repo matches the `--prefix` command line flag so
		 * the user can sync only repos which match this prefix.
		 */
		if s.prefix != "" && !strings.HasPrefix(dockerRepo.Name, s.prefix) {
			continue
		}

		wg.Add(1)
		workerCh <- dockerRepo
	}

	// wait for all workers to complete
	wg.Wait()
	log.Info("Done")

	fmt.Println(s.config)
	fmt.Println("test")
	return nil
}

func worker(wg *sync.WaitGroup, workerCh chan Repository, mc *client.Client, config *ContainerConfig) {
	fmt.Println("Starting worker")

	for {
		select {
		case repo := <-workerCh:

			/**
			 * Only support mirroring from thefollowing list.
			 */
			if repo.Host != "" && repo.Host != dockerHub && repo.Host != quay && repo.Host != gcr && repo.Host != k8s {
				log.Errorf("Could not pull images from host: %s. We support %s, %s, %s, and %s", repo.Host, dockerHub, quay, gcr, k8s)
				wg.Done()
				continue
			}

			/**
			 * If there is no host default to docker.
			 */
			if repo.Host == "" {
				repo.Host = dockerHub
			}

			mirrorClient := Mirror {
				config: config,
				mirrorClient: mc,
			}

			if err := mirrorClient.setup(repo); err != nil {
				log.Errorf("Failed to setup mirror for repository %s: %s", repo.Name, err)
				wg.Done()
				continue
			}

			//mirrorClient.work()
			mirrorClient.work()
			wg.Done()
		}
		
	}

		wg.Done()

}