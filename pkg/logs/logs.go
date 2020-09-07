// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package logs

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/DataDog/datadog-agent/cmd/agent/common"
	coreConfig "github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/logs/metrics"

	"github.com/DataDog/datadog-agent/pkg/util/log"

	"github.com/DataDog/datadog-agent/pkg/logs/client/http"
	"github.com/DataDog/datadog-agent/pkg/logs/config"
	"github.com/DataDog/datadog-agent/pkg/logs/scheduler"
	"github.com/DataDog/datadog-agent/pkg/logs/service"
	"github.com/DataDog/datadog-agent/pkg/logs/status"
)

const (
	// key used to display a warning message on the agent status
	invalidProcessingRules = "invalid_global_processing_rules"
	invalidEndpoints       = "invalid_endpoints"
)

var (
	// isRunning indicates whether logs-agent is running or not
	isRunning int32
	// logs-agent
	agent *Agent
)

// Start starts logs-agent
func Start() error {
	if IsAgentRunning() {
		return nil
	}

	// setup the sources and the services
	sources := config.NewLogSources()
	services := service.NewServices()

	// setup the config scheduler
	scheduler.CreateScheduler(sources, services)

	// setup the server config
	httpConnectivity := config.HTTPConnectivityFailure
	if endpoints, err := config.BuildHTTPEndpoints(); err == nil {
		httpConnectivity = http.CheckConnectivity(endpoints.Main)
	}
	endpoints, err := config.BuildEndpoints(httpConnectivity)
	if err != nil {
		message := fmt.Sprintf("Invalid endpoints: %v", err)
		status.AddGlobalError(invalidEndpoints, message)
		return errors.New(message)
	}
	status.CurrentTransport = status.TransportTCP
	if endpoints.UseHTTP {
		status.CurrentTransport = status.TransportHTTP
	}

	// setup the status
	status.Init(&isRunning, endpoints, sources, metrics.LogsExpvars)

	// setup global processing rules
	processingRules, err := config.GlobalProcessingRules()
	if err != nil {
		message := fmt.Sprintf("Invalid processing rules: %v", err)
		status.AddGlobalError(invalidProcessingRules, message)
		return errors.New(message)
	}

	// setup and start the agent
	agent = NewAgent(sources, services, processingRules, endpoints)
	log.Info("Starting logs-agent...")
	agent.Start()
	atomic.StoreInt32(&isRunning, 1)
	log.Info("logs-agent started")

	// add SNMP traps source forwarding SNMP traps as logs if enabled.
	if source := config.SNMPTrapsSource(); source != nil {
		log.Debug("Adding SNMPTraps source to the Logs Agent")
		sources.AddSource(source)
	}

	// adds the source collecting logs from all containers if enabled,
	// but ensure that it is enabled after the AutoConfig initialization
	if source := config.ContainerCollectAllSource(); source != nil {
		go func() {
			common.BlockUntilAutoConfigRanOnce(time.Millisecond * time.Duration(coreConfig.Datadog.GetInt("ac_load_timeout")))
			log.Debug("Adding ContainerCollectAll source to the Logs Agent")
			sources.AddSource(source)
		}()
	}

	return nil
}

// Stop stops properly the logs-agent to prevent data loss,
// it only returns when the whole pipeline is flushed.
func Stop() {
	log.Info("Stopping logs-agent")
	if IsAgentRunning() {
		if agent != nil {
			agent.Stop()
			agent = nil
		}
		if scheduler.GetScheduler() != nil {
			scheduler.GetScheduler().Stop()
		}
		status.Clear()
		atomic.StoreInt32(&isRunning, 0)
	}
	log.Info("logs-agent stopped")
}

// IsAgentRunning returns true if the logs-agent is running.
func IsAgentRunning() bool {
	return status.Get().IsRunning
}

// GetStatus returns logs-agent status
func GetStatus() status.Status {
	return status.Get()
}
