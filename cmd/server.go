/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"time"

	platformmeshcontext "github.com/platform-mesh/golang-commons/context"
	"github.com/platform-mesh/golang-commons/traces"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/platform-mesh/extension-manager-operator/api/v1alpha1"
	"github.com/platform-mesh/extension-manager-operator/internal/server"
	"github.com/platform-mesh/extension-manager-operator/pkg/validation"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "server with configuration validation endpoint",
	Run:   RunServer,
}

func RunServer(_ *cobra.Command, _ []string) { // coverage-ignore
	ctrl.SetLogger(log.ComponentLogger("srv").Logr())

	ctx, cancelMain, shutdown := platformmeshcontext.StartContext(log, operatorCfg, defaultCfg.ShutdownTimeout)
	defer shutdown()

	var err error
	var providerShutdown func(ctx context.Context) error
	if defaultCfg.Tracing.Enabled {
		providerShutdown, err = traces.InitProvider(ctx, defaultCfg.Tracing.Collector)
		if err != nil {
			log.Fatal().Err(err).Msg("unable to start gRPC-Sidecar TracerProvider")
		}
	} else {
		providerShutdown, err = traces.InitLocalProvider(ctx, defaultCfg.Tracing.Collector, false)
		if err != nil {
			log.Fatal().Err(err).Msg("unable to start local TracerProvider")
		}
	}

	defer func() {
		if err := providerShutdown(ctx); err != nil {
			log.Fatal().Err(err).Msg("failed to shutdown TracerProvider")
		}
	}()

	// Create Prometheus metrics handler
	metricsHandler := promhttp.Handler()

	// Initialize entity type registry from cluster if enabled
	var registry *validation.EntityTypeRegistry
	if serverCfg.EntityTypeValidationEnabled {
		registry = initServerEntityTypeRegistry(ctx)
	}

	// Register Prometheus metrics endpoint
	rt := server.CreateRouter(defaultCfg.IsLocal, log, validation.NewContentConfiguration(), registry)
	rt.Handle("/metrics", metricsHandler)

	srv := &http.Server{
		Addr:         ":" + serverCfg.ServerPort,
		Handler:      rt,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("Server failed")
			cancelMain(err)
		}
	}()
	log.Info().Msg("Server started")

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = srv.Shutdown(shutdownCtx)
	if err != nil {
		log.Error().Err(err).Msg("Graceful shutdown failed")
	}
	log.Info().Msg("Server stopped")
}

func initServerEntityTypeRegistry(ctx context.Context) *validation.EntityTypeRegistry { // coverage-ignore
	registry := validation.NewEntityTypeRegistry()

	kubeconfigPath := os.Getenv("KUBECONFIG")
	var restCfg *rest.Config
	var err error
	if kubeconfigPath != "" {
		restCfg, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			log.Warn().Err(err).Msg("failed to load kubeconfig, entity type validation will only recognize 'global'")
			return registry
		}
	} else {
		restCfg, err = rest.InClusterConfig()
		if err != nil {
			log.Warn().Err(err).Msg("not running in cluster, entity type validation will only recognize 'global'")
			return registry
		}
	}

	k8sClient, err := client.New(restCfg, client.Options{Scheme: scheme})
	if err != nil {
		log.Warn().Err(err).Msg("failed to create k8s client, entity type validation will only recognize 'global'")
		return registry
	}

	var ccList v1alpha1.ContentConfigurationList
	if err := k8sClient.List(ctx, &ccList); err != nil {
		log.Warn().Err(err).Msg("failed to list ContentConfigurations, entity type validation will only recognize 'global'")
		return registry
	}

	var configs []validation.ContentConfiguration
	for _, cc := range ccList.Items {
		if cc.Status.ConfigurationResult == "" {
			continue
		}
		var parsed validation.ContentConfiguration
		if err := json.Unmarshal([]byte(cc.Status.ConfigurationResult), &parsed); err != nil {
			log.Warn().Err(err).Str("name", cc.Name).Msg("failed to parse ConfigurationResult for entity type registry")
			continue
		}
		configs = append(configs, parsed)
	}

	registry.Bulkload(configs)
	log.Info().Int("entityTypes", len(registry.KnownTypes())).Msg("initialized server entity type registry")
	return registry
}
