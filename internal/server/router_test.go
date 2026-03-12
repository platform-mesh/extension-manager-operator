package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/platform-mesh/extension-manager-operator/pkg/validation"
	"github.com/platform-mesh/golang-commons/logger"
)

func initLog() *logger.Logger {
	logConfig := logger.DefaultConfig()
	logConfig.Name = "router_test"
	logConfig.Level = "DEBUG"
	logConfig.NoJSON = false
	log, _ := logger.New(logConfig)
	return log
}

func TestCreateRouter(t *testing.T) {
	tests := []struct {
		name       string
		isLocal    bool
		method     string
		path       string
		expectCode int
		expectCORS bool
		reqBody    string
	}{
		{
			name:       "validate endpoint exists",
			isLocal:    false,
			method:     http.MethodPost,
			path:       "/validate",
			expectCode: http.StatusOK,
			reqBody: `{
				"contentType": "json",
				"contentConfiguration":"{\"luigiConfigFragment\": {\"data\": {\"nodeDefaults\": {\"entityType\": \"global\",\"isolateView\": true},\"nodes\": [{\"entityType\": \"global\",\"icon\": \"home\",\"label\": \"Overview\",\"pathSegment\": \"home\"}],\"texts\": [{\"locale\": \"de\",\"textDictionary\": {\"hello\": \"Hallo\"}}]}},\"name\": \"overview\"}"}"
			}`,
		},
		{
			name:       "wrong method not allowed",
			isLocal:    false,
			method:     http.MethodGet,
			path:       "/validate",
			expectCode: http.StatusMethodNotAllowed,
			reqBody: `{
				"contentType": "json",
				"contentConfiguration":"{\"luigiConfigFragment\": {\"data\": {\"nodeDefaults\": {\"entityType\": \"global\",\"isolateView\": true},\"nodes\": [{\"entityType\": \"global\",\"icon\": \"home\",\"label\": \"Overview\",\"pathSegment\": \"home\"}],\"texts\": [{\"locale\": \"de\",\"textDictionary\": {\"hello\": \"Hallo\"}}]}},\"name\": \"overview\"}"}"
			}`,
		},
		{
			name:       "local setup OK",
			isLocal:    true,
			method:     http.MethodPost,
			path:       "/validate",
			expectCode: http.StatusOK,
			reqBody: `{
				"contentType": "json",
				"contentConfiguration":"{\"luigiConfigFragment\": {\"data\": {\"nodeDefaults\": {\"entityType\": \"global\",\"isolateView\": true},\"nodes\": [{\"entityType\": \"global\",\"icon\": \"home\",\"label\": \"Overview\",\"pathSegment\": \"home\"}],\"texts\": [{\"locale\": \"de\",\"textDictionary\": {\"hello\": \"Hallo\"}}]}},\"name\": \"overview\"}"}"
			}`,
		},
		{
			name:       "invalid request body",
			isLocal:    true,
			method:     http.MethodPost,
			path:       "/validate",
			expectCode: http.StatusInternalServerError,
			reqBody: `{
				"contentType": "json",
				"contentConfiguration":"{\"luigiConfigFragment\": {\"dat
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := initLog()

			validator := validation.NewContentConfiguration()

			router := CreateRouter(tt.isLocal, log, validator, nil)
			assert.NotNil(t, router)

			req := httptest.NewRequest(tt.method, tt.path, bytes.NewBufferString(tt.reqBody))
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectCode, rr.Code)

		})
	}
}

func TestCreateRouter_WithEntityTypeRegistry(t *testing.T) {
	registry := validation.NewEntityTypeRegistry()
	registry.Bulkload([]validation.ContentConfiguration{
		{
			LuigiConfigFragment: validation.LuigiConfigFragment{
				Data: validation.LuigiConfigData{
					Nodes: []validation.Node{
						{EntityType: "global", DefineEntity: &validation.DefineEntity{Id: "project"}},
					},
				},
			},
		},
	})

	tests := []struct {
		name             string
		reqBody          string
		expectCode       int
		expectValidation bool
	}{
		{
			name: "valid entityType with registry",
			reqBody: `{
				"contentType": "json",
				"contentConfiguration": "{\"name\": \"test\", \"luigiConfigFragment\": {\"data\": {\"nodes\": [{\"entityType\": \"global\", \"pathSegment\": \"home\"}]}}}"
			}`,
			expectCode: http.StatusOK,
		},
		{
			name: "unknown entityType with registry",
			reqBody: `{
				"contentType": "json",
				"contentConfiguration": "{\"name\": \"test\", \"luigiConfigFragment\": {\"data\": {\"nodes\": [{\"entityType\": \"nonexistent\", \"pathSegment\": \"home\"}]}}}"
			}`,
			expectCode:       http.StatusOK,
			expectValidation: true,
		},
		{
			name: "known entityType project with registry",
			reqBody: `{
				"contentType": "json",
				"contentConfiguration": "{\"name\": \"test\", \"luigiConfigFragment\": {\"data\": {\"nodes\": [{\"entityType\": \"project\", \"pathSegment\": \"home\"}]}}}"
			}`,
			expectCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := initLog()
			validator := validation.NewContentConfiguration()

			router := CreateRouter(false, log, validator, registry)
			assert.NotNil(t, router)

			req := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewBufferString(tt.reqBody))
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectCode, rr.Code)

			if tt.expectValidation {
				var resp Response
				err := json.NewDecoder(rr.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.NotEmpty(t, resp.ValidationErrors)
			}
		})
	}
}
