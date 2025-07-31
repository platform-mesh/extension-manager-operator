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

package controllerruntime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	//+kubebuilder:scaffold:imports
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	platformmeshconfig "github.com/platform-mesh/golang-commons/config"
	platformmeshcontext "github.com/platform-mesh/golang-commons/context"
	"github.com/platform-mesh/golang-commons/logger"

	"github.com/platform-mesh/extension-manager-operator/api/v1alpha1"
	"github.com/platform-mesh/extension-manager-operator/internal/config"
)

const (
	defaultTestTimeout  = 10 * time.Second
	defaultTickInterval = 250 * time.Millisecond
	defaultNamespace    = "default"
)

type ContentConfigurationControllerTestSuite struct {
	suite.Suite

	kubernetesClient  client.Client
	kubernetesManager ctrl.Manager
	testEnv           *envtest.Environment
	logger            *logger.Logger

	cancel context.CancelFunc
}

func (suite *ContentConfigurationControllerTestSuite) SetupSuite() {
	logConfig := logger.DefaultConfig()
	logConfig.NoJSON = true
	logConfig.Name = "ContentConfigurationControllerTestSuite"
	log, err := logger.New(logConfig)
	suite.logger = log
	suite.Nil(err)
	// Disable color logging as vs-code does not support color logging in the test output
	log = logger.NewFromZerolog(log.Output(&zerolog.ConsoleWriter{Out: os.Stdout, NoColor: true}))

	testContext, _, _ := platformmeshcontext.StartContext(log, nil, 1*time.Second)

	testContext = logger.SetLoggerInContext(testContext, log.ComponentLogger("TestSuite"))

	suite.testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
		BinaryAssetsDirectory: filepath.Join("..", "..", "..", "bin", "k8s", fmt.Sprintf("1.30.0-%s-%s", runtime.GOOS, runtime.GOARCH)),
	}

	cfg, err := suite.testEnv.Start()
	suite.Nil(err)

	utilruntime.Must(v1alpha1.AddToScheme(scheme.Scheme))
	utilruntime.Must(v1.AddToScheme(scheme.Scheme))

	// +kubebuilder:scaffold:scheme

	suite.kubernetesClient, err = client.New(cfg, client.Options{
		Scheme: scheme.Scheme,
	})
	suite.Nil(err)
	ctrl.SetLogger(log.Logr())
	suite.kubernetesManager, err = ctrl.NewManager(cfg, ctrl.Options{
		Scheme:      scheme.Scheme,
		BaseContext: func() context.Context { return testContext },
	})
	suite.Nil(err)

	defaultConfig := &platformmeshconfig.CommonServiceConfig{}
	appCfg := config.OperatorConfig{}
	appCfg.Subroutines.ContentConfiguration.Enabled = true

	contentConfigurationReconciler := NewContentConfigurationReconcilerCR(log, suite.kubernetesManager, appCfg)
	err = contentConfigurationReconciler.SetupWithManager(suite.kubernetesManager, defaultConfig, log)
	suite.Nil(err)

	go suite.startController()
}

func (suite *ContentConfigurationControllerTestSuite) startController() {
	var controllerContext context.Context
	controllerContext, suite.cancel = context.WithCancel(context.Background())
	err := suite.kubernetesManager.Start(controllerContext)
	suite.Nil(err)
}

func (suite *ContentConfigurationControllerTestSuite) TearDownSuite() {
	suite.cancel()
	err := suite.testEnv.Stop()
	suite.Nil(err)
}
