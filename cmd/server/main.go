// Copyright 2023 LiveKit, Inc.
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

package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/livekit/livekit-server/pkg/config"
	"github.com/livekit/livekit-server/pkg/logger"
	"github.com/livekit/livekit-server/pkg/service"
	"github.com/livekit/livekit-server/version"
)

func init() {
	// Seed random number generator
	rand.Seed(time.Now().UnixNano())
}

func main() {
	app := &cli.App{
		Name:    "livekit-server",
		Usage:   "LiveKit media server",
		Version: version.Version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Usage:   "path to LiveKit config file",
				EnvVars: []string{"LIVEKIT_CONFIG_FILE"},
			},
			&cli.StringFlag{
				Name:    "config-body",
				Usage:   "LiveKit config in YAML, read from stdin or as a string",
				EnvVars: []string{"LIVEKIT_CONFIG_BODY"},
			},
			&cli.StringFlag{
				Name:    "key-file",
				Usage:   "path to file that contains API keys/secrets",
				EnvVars: []string{"LIVEKIT_KEY_FILE"},
			},
			&cli.StringFlag{
				Name:    "keys",
				Usage:   "api keys (key: secret\nkey2: secret2)",
				EnvVars: []string{"LIVEKIT_KEYS"},
			},
			&cli.StringFlag{
				Name:    "node-ip",
				Usage:   "IP address of the current node, used to advertise to other nodes",
				EnvVars: []string{"NODE_IP"},
			},
			&cli.StringFlag{
				Name:    "redis",
				Usage:   "Redis URL (redis://[username:password@]host:port/db)",
				EnvVars: []string{"REDIS_URL"},
			},
			&cli.BoolFlag{
				Name:  "dev",
				Usage: "run in development mode (insecure, single node)",
			},
			// Added for local testing: allow overriding the bind address without a full config file
			&cli.StringFlag{
				Name:    "bind",
				Usage:   "address to bind the server to (e.g. 0.0.0.0)",
				EnvVars: []string{"LIVEKIT_BIND_ADDRESS"},
				Value:   "0.0.0.0",
			},
		},
		Action: startServer,
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func startServer(c *cli.Context) error {
	// Load configuration
	conf, err := config.NewConfig(c.String("config"), c.String("config-body"), c)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize logger
	if err := logger.InitFromConfig(&conf.Logging, "livekit"); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	logger.Infow("starting LiveKit server", "version", version.Version)

	// Create and start the server
	server, err := service.InitializeServer(conf)
	if err != nil {
		return fmt.Errorf("failed to initialize server: %w", err)
	}

	// Handle OS signals for graceful shutdown
	// Also listen for SIGTERM so the server shuts down cleanly in containerized environments
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Infow("received signal, shutting down", "signal", sig)
		server.Stop(true)
	}()

	return server.Start()
}
