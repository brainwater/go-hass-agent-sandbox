// Copyright (c) 2023 Joshua Rich <joshua.rich@gmail.com>
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package agent

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	deviceConfig "github.com/joshuar/go-hass-agent/internal/config"
	"github.com/joshuar/go-hass-agent/internal/hass/api"

	"github.com/joshuar/go-hass-agent/internal/agent/config"
	"github.com/joshuar/go-hass-agent/internal/agent/ui"
	"github.com/joshuar/go-hass-agent/internal/device"
	"github.com/joshuar/go-hass-agent/internal/tracker"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

//go:generate sh -c "printf %s $(git tag | tail -1) > VERSION"
//go:embed VERSION
var version string

const (
	name = "go-hass-agent"
)

// Agent holds the data and structure representing an instance of the agent.
// This includes the data structure for the UI elements and tray and some
// strings such as app name and version.
type Agent struct {
	ui       AgentUI
	config   AgentConfig
	sensors  *tracker.SensorTracker
	done     chan struct{}
	name     string
	id       string
	version  string
	headless bool
}

//go:generate moq -out mock_AgentUI_test.go . AgentUI
type AgentUI interface {
	DisplayNotification(string, string)
	DisplayTrayIcon(context.Context, ui.Agent)
	DisplayRegistrationWindow(context.Context, ui.Agent, chan struct{})
	Run()
}

// AgentOptions holds options taken from the command-line that was used to
// invoke go-hass-agent that are relevant for agent functionality.
type AgentOptions struct {
	ID                 string
	Headless, Register bool
}

func newAgent(appID string, headless bool) *Agent {
	a := &Agent{
		id:       appID,
		version:  version,
		done:     make(chan struct{}),
		headless: headless,
	}
	a.ui = ui.NewFyneUI(a, headless)
	a.config = config.NewFyneConfig()
	return a
}

// Run is the "main loop" of the agent. It sets up the agent, loads the config
// then spawns a sensor tracker and the workers to gather sensor data and
// publish it to Home Assistant
func Run(options AgentOptions) {
	agent := newAgent(options.ID, options.Headless)
	defer close(agent.done)

	agent.setupLogging()
	registrationDone := make(chan struct{})
	configDone := make(chan struct{})

	go agent.registrationProcess(context.Background(), "", "", options.Register, options.Headless, registrationDone)

	var workerWg sync.WaitGroup
	var ctx context.Context
	var cancelFunc context.CancelFunc
	go func() {
		<-registrationDone
		var err error
		if err = UpgradeConfig(agent.config); err != nil {
			log.Warn().Err(err).Msg("Could not upgrade config.")
		}
		if err = ValidateConfig(agent.config); err != nil {
			log.Fatal().Err(err).Msg("Could not validate config.")
		}
		ctx, cancelFunc = agent.setupContext()
		close(configDone)
		if agent.sensors, err = tracker.NewSensorTracker(agent); err != nil {
			log.Fatal().Err(err).Msg("Could not start.")
		}

		sensorWorkers := deviceConfig.SensorWorkers()
		sensorWorkers = append(sensorWorkers, device.ExternalIPUpdater)

		workerWg.Add(1)
		go func() {
			defer workerWg.Done()
			device.StartWorkers(ctx, agent.sensors, sensorWorkers...)
		}()
		workerWg.Add(1)
		go func() {
			defer workerWg.Done()
			agent.runNotificationsWorker(ctx, options)
		}()
	}()

	go func() {
		<-configDone
		agent.handleSignals(cancelFunc)
		agent.handleShutdown(ctx)
		agent.ui.DisplayTrayIcon(ctx, agent)
	}()

	if !options.Headless {
		agent.ui.Run()
	}
	workerWg.Wait()
	<-ctx.Done()
}

// Register runs a registration flow. It either prompts the user for needed
// information or parses what is already provided. It will send a registration
// request to Home Assistant and handles the response. It will handle either a
// UI or non-UI registration flow.
func Register(options AgentOptions, server, token string) {
	agent := newAgent(options.ID, options.Headless)
	defer close(agent.done)

	agentCtx, cancelFunc := context.WithCancel(context.Background())

	// Don't proceed unless the agent is registered and forced is not set
	// if agent.IsRegistered() && !options.Register {
	// 	log.Warn().Msg("Agent is already registered and forced option not specified, not performing registration.")
	// 	cancelFunc()
	// 	return
	// }

	registrationDone := make(chan struct{})
	go agent.registrationProcess(agentCtx, server, token, options.Register, options.Headless, registrationDone)

	agent.handleSignals(cancelFunc)
	agent.handleShutdown(agentCtx)
	if !options.Headless {
		agent.ui.DisplayTrayIcon(agentCtx, agent)
		agent.ui.Run()
	}

	<-registrationDone
	log.Info().Msg("Device registered with Home Assistant.")
}

func ShowVersion(options AgentOptions) {
	agent := newAgent(options.ID, true)
	log.Info().Msgf("%s: %s", agent.name, agent.version)
}

func ShowInfo(options AgentOptions) {
	agent := newAgent(options.ID, true)
	var info strings.Builder
	var deviceName, deviceID string
	if err := agent.GetConfig(config.PrefDeviceName, &deviceName); err == nil && deviceName != "" {
		fmt.Fprintf(&info, "Device Name %s.", deviceName)
	}
	if err := agent.GetConfig(config.PrefDeviceID, &deviceID); err == nil && deviceID != "" {
		fmt.Fprintf(&info, "Device ID %s.", deviceID)
	}
	log.Info().Msg(info.String())
}

// setupLogging will attempt to create and then write logging to a file. If it
// cannot do this, logging will only be available on stdout
func (agent *Agent) setupLogging() {
	logFile, err := agent.config.StoragePath("go-hass-app.log")
	if err != nil {
		log.Error().Err(err).
			Msg("Unable to create a log file. Will only write logs to stdout.")
	} else {
		logWriter, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
		if err != nil {
			log.Error().Err(err).
				Msg("Unable to open log file for writing.")
		} else {
			consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout}
			multiWriter := zerolog.MultiLevelWriter(consoleWriter, logWriter)
			log.Logger = log.Output(multiWriter)
		}
	}
}

func (agent *Agent) setupContext() (context.Context, context.CancelFunc) {
	SharedConfig := &api.APIConfig{}
	if err := agent.config.Get(config.PrefAPIURL, &SharedConfig.APIURL); err != nil {
		log.Fatal().Err(err).Msg("Could not export apiURL.")
	}
	if err := agent.config.Get(config.PrefSecret, &SharedConfig.Secret); err != nil && SharedConfig.Secret != "NOTSET" {
		log.Fatal().Err(err).Msg("Could not export secret.")
	}
	ctx := api.NewContext(context.Background(), SharedConfig)
	return context.WithCancel(ctx)
}

// handleSignals will handle Ctrl-C of the agent
func (agent *Agent) handleSignals(cancelFunc context.CancelFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Debug().Msg("Ctrl-C pressed.")
		cancelFunc()
	}()
}

// handleShutdown will handle context cancellation of the agent
func (agent *Agent) handleShutdown(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Debug().Msg("Context canceled.")
				os.Exit(1)
			case <-agent.done:
				log.Debug().Msg("Agent done.")
				os.Exit(0)
			}
		}
	}()
}

// Agent satisfies ui.Agent, tracker.Agent and api.Agent interfaces

func (agent *Agent) IsHeadless() bool {
	return agent.headless
}

func (agent *Agent) AppName() string {
	return agent.name
}

func (agent *Agent) AppID() string {
	return agent.id
}

func (agent *Agent) AppVersion() string {
	return agent.version
}

func (agent *Agent) Stop() {
	close(agent.done)
}

func (agent *Agent) GetConfig(key string, value interface{}) error {
	return agent.config.Get(key, value)
}

func (agent *Agent) SetConfig(key string, value interface{}) error {
	return agent.config.Set(key, value)
}

func (agent *Agent) StoragePath(path string) (string, error) {
	return agent.config.StoragePath(path)
}

func (agent *Agent) SensorList() []string {
	return agent.sensors.SensorList()
}

func (agent *Agent) SensorValue(id string) (tracker.Sensor, error) {
	return agent.sensors.Get(id)
}
