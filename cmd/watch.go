// Copyright © 2016 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"github.com/spf13/cobra"
	"path/filepath"
	"os"
	"github.com/codegangsta/envy/lib"
	"github.com/arekkas/gimlet/lib"
	"github.com/pborman/uuid"
	"strings"
)

// watchCmd represents the watch command
var watchCmd = &cobra.Command{
	Use:   "watch <command>",
	Short: "Watch directories for changes and build and run go code when changes are registered",
	Long: `Because go run uses subprocesses, there is no easy way to kill an app that was started by go run. This
is why gimlet builds the program first, and then executes the binary.

Examples:

- gimlet --immediate ` + "`--this-will-be-passed-down some-argument`" + `
- gimlet --path ./src
`,
	Run: MainAction,
}

func init() {
	RootCmd.AddCommand(watchCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// watchCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	//watchCmd.Flags().StringSliceP("watch", "w", []string{"./"}, "Help message for toggle")
	watchCmd.Flags().StringP("listen", "l", "", "Listening address for the proxy server.")
	watchCmd.Flags().IntP("port", "p", 3000, "Port for the proxy server.")
	watchCmd.Flags().StringP("app-port", "a", "3001", "Port for the Go web server.")
	watchCmd.Flags().BoolP("immediate", "i", false, "Run the server immediately after it's built instead of on first http request.")
	watchCmd.Flags().StringSliceP("exclude", "e", []string{".git", "vendor"}, "Relative directories to exclude.")
	watchCmd.Flags().String("path", ".", "Path to watch files from.")
	watchCmd.Flags().Int("interval", 200, "Interval for polling in ms. Lower values require more CPU time.")
	watchCmd.Flags().Bool("kill-on-error", false, "Set to true to kill gimlet if an error occurrs during build or run.")
}

func MainAction(cmd *cobra.Command, args []string) {
	laddr, _ := cmd.Flags().GetString("listen")
	port, _ := cmd.Flags().GetInt("port")
	appPort, _ := cmd.Flags().GetString("app-port")
	immediate, _ = cmd.Flags().GetBool("immediate")
	path, _ := cmd.Flags().GetString("path")
	interval, _ := cmd.Flags().GetInt("interval")
	exclude, _ := cmd.Flags().GetStringSlice("exclude")
	killOnError, _ := cmd.Flags().GetBool("kill-on-error")
	id := uuid.New()
	// Bootstrap the environment
	envy.Bootstrap()

	if len(args) == 1 {
		args = []string{strings.Replace(args[0], "`", "", -1)}
	} else if len(args) > 1 {
		logger.Fatal("You can only provide zero or one arguments.")
		return
	}

	// Set the PORT env
	os.Setenv("PORT", appPort)

	var err error
	builder := gin.NewBuilder(path, id, false, os.TempDir())
	runner := gin.NewRunner(filepath.Join(os.TempDir(), builder.Binary()), args...)
	runner.SetWriter(os.Stdout)
	proxy := gin.NewProxy(builder, runner)

	config := &gin.Config{
		Laddr:   laddr,
		Port:    port,
		ProxyTo: "http://localhost:" + appPort,
	}

	err = proxy.Run(config)
	if err != nil {
		logger.Fatal(err)
	}

	if laddr != "" {
		logger.Printf("listening at %s:%d\n", laddr, port)
	} else {
		logger.Printf("listening on port %d\n", port)
	}

	shutdown(runner)

	// build right now
	build(interval, builder, runner, logger, killOnError)

	// scan for changes
	scanChanges(interval, path, exclude, func(path string) {
		runner.Kill()
		build(interval, builder, runner, logger, killOnError)
	})
}
