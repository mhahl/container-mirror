/*
Copyright © 2022 Mark Hahl <mark@hahl.id.au>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/
package cmd

import (
	"os"
	"github.com/spf13/cobra"
	log "github.com/sirupsen/logrus"
	"github.com/mhahl/container-mirror/service"
)


var (
	/* Verbose defines if the command is being run with verbose mode */
	Verbose bool

	/* IgnoreErrors ignores errors when mirroring */
	IgnoreErrors bool

	/* Path to the config file */
	configFile string

	/* Only sync repos with prefix */
	prefix string

	logger       *log.Logger
	logLevel     string
)


// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "container-mirror",
	Short: "",
	Long: `Mirror containers from index file into a local registry.`,
	Run: runCmd,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

/**
 * Create the service and start the mirror process.
 */
func runCmd(cmd *cobra.Command, args []string) {
		containerService := service.NewContainerService(configFile, prefix, true, true, logger)
		containerService.Get()
}

func init() {
	logger = log.New()
	rootCmd.Flags().StringVar(&configFile, "config", "config.yaml", "Set configuration file")
	rootCmd.Flags().StringVar(&prefix, "prefix", "", "Only sync repos which match `prefix`")
}


