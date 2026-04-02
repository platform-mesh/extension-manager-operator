package subroutines

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	golangCommonErrors "github.com/platform-mesh/golang-commons/errors"
	"github.com/platform-mesh/golang-commons/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	apimachinery "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cachev1alpha1 "github.com/platform-mesh/extension-manager-operator/api/v1alpha1"
	"github.com/platform-mesh/extension-manager-operator/pkg/subroutines/mocks"
	commonTesting "github.com/platform-mesh/extension-manager-operator/pkg/util/testing"
	"github.com/platform-mesh/extension-manager-operator/pkg/validation"
	"github.com/platform-mesh/extension-manager-operator/pkg/validation/validation_test"
)

type ContentConfigurationSubroutineTestSuite struct {
	suite.Suite

	testObj *ContentConfigurationSubroutine

	// mocks
	clientMock *mocks.Client
}

func TestContentConfigurationSubroutineTestSuit(t *testing.T) {
	suite.Run(t, new(ContentConfigurationSubroutineTestSuite))
}

func (suite *ContentConfigurationSubroutineTestSuite) SetupTest() {
	// create new mock
	suite.clientMock = new(mocks.Client)

	// create new test object
	suite.testObj = NewContentConfigurationSubroutine(validation.NewContentConfiguration(), http.DefaultClient, nil, nil)
}

func (suite *ContentConfigurationSubroutineTestSuite) TestCreateAndUpdate_OK() {
	// Given
	contentConfiguration := &cachev1alpha1.ContentConfiguration{
		Spec: cachev1alpha1.ContentConfigurationSpec{
			InlineConfiguration: &cachev1alpha1.InlineConfiguration{
				Content:     validation_test.GetValidYAML(),
				ContentType: "yaml",
			},
		},
	}

	// When
	_, err := suite.testObj.Process(context.Background(), contentConfiguration)

	// Then
	suite.Require().Nil(err)

	equal, cmpErr := commonTesting.CompareJSON(
		validation_test.GetValidJSON(),
		contentConfiguration.Status.ConfigurationResult,
	)
	suite.Require().Nil(cmpErr)
	suite.Require().True(equal)

	// Now lets take the same object and update it
	// Given
	contentConfiguration.Spec.InlineConfiguration.Content = validation_test.GetValidYAMLFixtureButDifferentName()

	// When
	_, err2 := suite.testObj.Process(context.Background(), contentConfiguration)

	// Then
	suite.Require().Nil(err2)
	equal, cmpErr = commonTesting.CompareJSON(validation_test.GetValidJSONButDifferentName(), contentConfiguration.Status.ConfigurationResult)
	suite.Require().Nil(cmpErr)
	suite.Require().True(equal)
}

func (suite *ContentConfigurationSubroutineTestSuite) TestCreateAndUpdate_Error() {
	// Given
	contentConfiguration := &cachev1alpha1.ContentConfiguration{
		Spec: cachev1alpha1.ContentConfigurationSpec{
			InlineConfiguration: &cachev1alpha1.InlineConfiguration{
				Content:     validation_test.GetValidYAML(),
				ContentType: "yaml",
			},
		},
	}

	// When
	_, errCmp := suite.testObj.Process(context.Background(), contentConfiguration)

	// Then
	suite.Require().Nil(errCmp)

	// compare configuration and result YAMLs
	cmp, cmpErr := commonTesting.CompareJSON(validation_test.GetValidJSON(), contentConfiguration.Status.ConfigurationResult)
	suite.Require().Nil(cmpErr)
	suite.Require().True(cmp)

	// Given invalid configuration
	contentConfiguration.Spec.InlineConfiguration.Content = "invalid"

	// When
	_, errProcessInvalidConfig := suite.testObj.Process(context.Background(), contentConfiguration)
	time.Sleep(1 * time.Second)

	// Then
	suite.Require().Nil(errProcessInvalidConfig)
	// result shoundn't change
	equal, cmpErr := commonTesting.CompareJSON(validation_test.GetValidJSON(), contentConfiguration.Status.ConfigurationResult)
	suite.Require().Nil(cmpErr)
	suite.Require().True(equal)
}

func (suite *ContentConfigurationSubroutineTestSuite) TestGetName_OK() {
	// When
	result := suite.testObj.GetName()

	// Then
	suite.Equal(ContentConfigurationSubroutineName, result)
}

func (suite *ContentConfigurationSubroutineTestSuite) TestFinalize_OK() {
	// Given
	contentConfiguration := &cachev1alpha1.ContentConfiguration{}

	// When
	result, err := suite.testObj.Finalize(context.Background(), contentConfiguration)

	// Then
	suite.Assert().Zero(result.RequeueAfter)
	suite.Nil(err)
}

func (suite *ContentConfigurationSubroutineTestSuite) TestProcessingConfig() {
	remoteURL := "https://this-address-should-be-mocked-by-httpmock"

	tests := []struct {
		name                 string
		spec                 cachev1alpha1.ContentConfigurationSpec
		remoteURL            string
		statusCode           int
		expectedError        golangCommonErrors.OperatorError
		expectedConfigResult string
	}{
		{
			name: "InlineConfigYAML_OK",
			spec: cachev1alpha1.ContentConfigurationSpec{
				InlineConfiguration: &cachev1alpha1.InlineConfiguration{
					Content:     validation_test.GetValidYAML(),
					ContentType: "yaml",
				},
			},
			expectedConfigResult: validation_test.GetValidJSON(),
		},
		{
			name: "InlineConfigYAML_ValidationError",
			spec: cachev1alpha1.ContentConfigurationSpec{
				InlineConfiguration: &cachev1alpha1.InlineConfiguration{
					Content:     "I am not a valid yaml",
					ContentType: "yaml",
				},
			},
		},
		{
			name: "InlineConfigJSON_OK",
			spec: cachev1alpha1.ContentConfigurationSpec{
				InlineConfiguration: &cachev1alpha1.InlineConfiguration{
					Content:     validation_test.GetValidJSON(),
					ContentType: "json",
				},
			},
			expectedConfigResult: validation_test.GetValidJSON(),
		},
		{
			name: "InlineConfigJSON_ValidationError",
			spec: cachev1alpha1.ContentConfigurationSpec{
				InlineConfiguration: &cachev1alpha1.InlineConfiguration{
					Content:     "I am not a valid json",
					ContentType: "json",
				},
			},
		},
		{
			name: "RemoteConfig_OK",
			spec: cachev1alpha1.ContentConfigurationSpec{
				RemoteConfiguration: &cachev1alpha1.RemoteConfiguration{
					ContentType: "json",
					URL:         remoteURL,
				},
			},
			remoteURL:            remoteURL,
			statusCode:           http.StatusOK,
			expectedConfigResult: validation_test.GetValidJSON(),
		},
		{
			name: "RemoteConfig_http_error",
			spec: cachev1alpha1.ContentConfigurationSpec{
				RemoteConfiguration: &cachev1alpha1.RemoteConfiguration{
					URL: remoteURL,
				},
			},
			remoteURL:     remoteURL,
			statusCode:    http.StatusInternalServerError,
			expectedError: golangCommonErrors.NewOperatorError(errors.New("received non-200 status code: 500"), false, true),
		},
		{
			name:          "NoConfigProvider_Error",
			spec:          cachev1alpha1.ContentConfigurationSpec{},
			expectedError: golangCommonErrors.NewOperatorError(errors.New("no configuration provided"), false, true),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			if tt.remoteURL != "" {
				httpmock.Activate()
				defer httpmock.DeactivateAndReset()

				httpmock.RegisterResponder(
					"GET", tt.remoteURL, httpmock.NewStringResponder(tt.statusCode, validation_test.GetValidJSON()),
				)
			}

			// When
			contentConfiguration := cachev1alpha1.ContentConfiguration{
				Spec: tt.spec,
			}
			_, err := suite.testObj.Process(context.Background(), &contentConfiguration)

			// Then
			if tt.expectedError != nil {
				if err == nil {
					suite.Fail("expected error, but got nil")
				}
				suite.Require().Equal(tt.expectedError.Err().Error(), err.Err().Error())
			} else {
				suite.Nil(err)
			}

			if tt.expectedConfigResult == "" {
				assert.Equal(suite.T(), "", contentConfiguration.Status.ConfigurationResult)
			} else {
				cmp, cmpErr := commonTesting.CompareJSON(tt.expectedConfigResult, contentConfiguration.Status.ConfigurationResult)
				suite.Require().Nil(cmpErr)
				suite.Require().True(cmp)
			}
		})
	}
}

func (suite *ContentConfigurationSubroutineTestSuite) TestFinalizers_OK() {
	// Given
	contentConfiguration := &cachev1alpha1.ContentConfiguration{}

	// When
	result, err := suite.testObj.Finalize(context.Background(), contentConfiguration)

	// Then
	suite.Assert().Zero(result.RequeueAfter)
	suite.Nil(err)

	// When
	finalizers := suite.testObj.Finalizers(contentConfiguration)

	// Then
	suite.Equal([]string{}, finalizers)

}

func TestService_Do(t *testing.T) {
	log, err := logger.New(logger.DefaultConfig())
	require.NoError(t, err)
	tests := []struct {
		name           string
		url            string
		mockResponse   string
		mockStatusCode int
		mockError      error
		expectedBody   string
		expectError    bool
	}{
		{
			name:           "GET_request_OK",
			url:            "https://example.com/success",
			mockResponse:   `{"message": "success"}`,
			mockStatusCode: http.StatusOK,
			expectedBody:   `{"message": "success"}`,
			expectError:    false,
		},
		{
			name:           "status_code_500_Error",
			url:            "https://example.com/error",
			mockResponse:   `{"message": "error"}`,
			mockStatusCode: http.StatusInternalServerError,
			expectError:    true,
		},
		{
			name:           "status_code_404_Error",
			url:            "https://example.com/error",
			mockResponse:   `{"message": "error"}`,
			mockStatusCode: http.StatusNotFound,
			expectError:    true,
		},
		{
			name:        "network_Error",
			url:         "https://example.com/network-error",
			mockError:   errors.New("network error"),
			expectError: true,
		},
		{
			name:        "invalidurl",
			url:         "://invalid-url",
			mockError:   errors.New("network error"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			if tt.mockError != nil {
				httpmock.RegisterResponder(http.MethodGet, tt.url,
					httpmock.NewErrorResponder(tt.mockError))
			} else {
				httpmock.RegisterResponder(http.MethodGet, tt.url,
					httpmock.NewStringResponder(tt.mockStatusCode, tt.mockResponse))
			}

			r := NewContentConfigurationSubroutine(validation.NewContentConfiguration(), http.DefaultClient, nil, nil)

			body, err := r.getRemoteConfig(tt.url, log)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedBody, string(body))
			}
		})
	}
}

func (suite *ContentConfigurationSubroutineTestSuite) Test_IncompatibleSchemaUpdate() {
	// Given
	contentConfiguration := &cachev1alpha1.ContentConfiguration{
		Spec: cachev1alpha1.ContentConfigurationSpec{
			InlineConfiguration: &cachev1alpha1.InlineConfiguration{
				Content:     validation_test.GetValidYAML(),
				ContentType: "yaml",
			},
		},
		Status: cachev1alpha1.ContentConfigurationStatus{
			Conditions: []apimachinery.Condition{
				{
					Type:    "Ready",
					Status:  "True",
					Message: "The resource is ready",
					Reason:  "Complete",
				},
				{
					Message: "The subroutine is complete",
					Reason:  "Complete",
					Status:  "True",
					Type:    "ContentConfigurationSubroutine_Ready",
				},
			},
			ConfigurationResult: validation_test.GetValidJSON(),
		},
	}

	// Simulate incompatible schema update
	contentConfiguration.Spec.InlineConfiguration.Content = validation_test.GetValidIncompatibleYAML()

	// When
	_, err := suite.testObj.Process(context.Background(), contentConfiguration)
	// Then: should keep previously valid and currently invalid result
	suite.Require().Nil(err)

	cmp, cmpErr := commonTesting.CompareJSON(validation_test.GetValidJSON(), contentConfiguration.Status.ConfigurationResult)
	suite.Require().Nil(cmpErr)
	suite.Require().True(cmp)
	suite.Require().True(
		getCondition(contentConfiguration.Status.Conditions, ValidationConditionType).Status == apimachinery.ConditionFalse,
	)
	suite.Require().Equal(
		"ValidationFailed", getCondition(contentConfiguration.Status.Conditions, ValidationConditionType).Reason,
	)

	// make it valid and check if condition is removed
	contentConfiguration.Spec.InlineConfiguration.Content = validation_test.GetValidYAML()

	// When
	_, err = suite.testObj.Process(context.Background(), contentConfiguration)
	suite.Require().Nil(err)

	cmp, cmpErr = commonTesting.CompareJSON(validation_test.GetValidJSON(), contentConfiguration.Status.ConfigurationResult)
	suite.Require().NoError(cmpErr)
	suite.Require().True(cmp)

	suite.Require().Equal(
		"ValidationSucceeded", getCondition(contentConfiguration.Status.Conditions, ValidationConditionType).Reason,
	)
}

func getCondition(conditions []apimachinery.Condition, conditionType string) apimachinery.Condition { // nolint: unparam
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return condition
		}
	}
	return apimachinery.Condition{}
}

// validJSONWithEntityType returns a valid CC JSON that references the given entityType
// in both nodeDefaults and a single node.
func validJSONWithEntityType(entityType string) string {
	return `{
		"name": "test-cc",
		"luigiConfigFragment": {
			"data": {
				"nodeDefaults": {
					"entityType": "` + entityType + `"
				},
				"nodes": [
					{
						"entityType": "` + entityType + `",
						"pathSegment": "home",
						"label": "Home"
					}
				]
			}
		}
	}`
}

// validJSONDefiningEntityType returns a valid CC JSON that defines a new entity type
// via defineEntity under a "global" parent.
func validJSONDefiningEntityType(defineEntityId string) string {
	return `{
		"name": "definer-cc",
		"luigiConfigFragment": {
			"data": {
				"nodes": [
					{
						"entityType": "global",
						"pathSegment": "root",
						"label": "Root",
						"defineEntity": {
							"id": "` + defineEntityId + `",
							"contextKey": "` + defineEntityId + `Id"
						},
						"children": []
					}
				]
			}
		}
	}`
}

func newFakeReader(objects ...client.Object) client.Reader {
	scheme := runtime.NewScheme()
	err := cachev1alpha1.AddToScheme(scheme)
	if err != nil {
		panic(err)
	}
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
}

func TestProcess_EntityTypeValidation(t *testing.T) {
	tests := []struct {
		name                    string
		existingCCs             []client.Object
		inlineContent           string
		registry                *validation.EntityTypeRegistry
		k8sReader               client.Reader
		expectOperatorError     bool
		expectValidCondition    string
		expectValidReason       string
		expectConfigResult      bool
		expectRegistryContains  []string
	}{
		{
			name: "registry init succeeds and populates from existing CCs",
			existingCCs: []client.Object{
				&cachev1alpha1.ContentConfiguration{
					ObjectMeta: apimachinery.ObjectMeta{Name: "existing-cc"},
					Status: cachev1alpha1.ContentConfigurationStatus{
						ConfigurationResult: validJSONDefiningEntityType("project"),
					},
				},
			},
			inlineContent:          validJSONWithEntityType("project"),
			registry:               validation.NewEntityTypeRegistry(),
			expectValidCondition:   ConditionStatusTrue,
			expectValidReason:      ValidationConditionReasonSuccess,
			expectConfigResult:     true,
			expectRegistryContains: []string{"global", "project"},
		},
		{
			name:                   "registry init with empty CC list succeeds",
			existingCCs:            []client.Object{},
			inlineContent:          validJSONWithEntityType("global"),
			registry:               validation.NewEntityTypeRegistry(),
			expectValidCondition:   ConditionStatusTrue,
			expectValidReason:      ValidationConditionReasonSuccess,
			expectConfigResult:     true,
			expectRegistryContains: []string{"global"},
		},
		{
			name:                "registry init with nil reader returns error",
			inlineContent:       validJSONWithEntityType("global"),
			registry:            validation.NewEntityTypeRegistry(),
			k8sReader:           nil,
			expectOperatorError: true,
		},
		{
			name: "entity type validation failure sets Valid=False condition",
			existingCCs: []client.Object{},
			inlineContent:        validJSONWithEntityType("nonexistent-type"),
			registry:             validation.NewEntityTypeRegistry(),
			expectValidCondition: ConditionStatusFalse,
			expectValidReason:    ValidationConditionReasonFailed,
			expectConfigResult:   false,
		},
		{
			name: "entity type validation success updates registry",
			existingCCs: []client.Object{
				&cachev1alpha1.ContentConfiguration{
					ObjectMeta: apimachinery.ObjectMeta{Name: "definer"},
					Status: cachev1alpha1.ContentConfigurationStatus{
						ConfigurationResult: validJSONDefiningEntityType("mytype"),
					},
				},
			},
			inlineContent:          validJSONWithEntityType("mytype"),
			registry:               validation.NewEntityTypeRegistry(),
			expectValidCondition:   ConditionStatusTrue,
			expectValidReason:      ValidationConditionReasonSuccess,
			expectConfigResult:     true,
			expectRegistryContains: []string{"global", "mytype"},
		},
		{
			name: "initEntityTypeRegistry skips CC with empty ConfigurationResult",
			existingCCs: []client.Object{
				&cachev1alpha1.ContentConfiguration{
					ObjectMeta: apimachinery.ObjectMeta{Name: "empty-result"},
					Status:     cachev1alpha1.ContentConfigurationStatus{ConfigurationResult: ""},
				},
			},
			inlineContent:          validJSONWithEntityType("global"),
			registry:               validation.NewEntityTypeRegistry(),
			expectValidCondition:   ConditionStatusTrue,
			expectValidReason:      ValidationConditionReasonSuccess,
			expectConfigResult:     true,
			expectRegistryContains: []string{"global"},
		},
		{
			name: "initEntityTypeRegistry skips CC with unparseable ConfigurationResult",
			existingCCs: []client.Object{
				&cachev1alpha1.ContentConfiguration{
					ObjectMeta: apimachinery.ObjectMeta{Name: "bad-json"},
					Status: cachev1alpha1.ContentConfigurationStatus{
						ConfigurationResult: "{{not valid json at all",
					},
				},
			},
			inlineContent:          validJSONWithEntityType("global"),
			registry:               validation.NewEntityTypeRegistry(),
			expectValidCondition:   ConditionStatusTrue,
			expectValidReason:      ValidationConditionReasonSuccess,
			expectConfigResult:     true,
			expectRegistryContains: []string{"global"},
		},
		{
			name: "initEntityTypeRegistry skips unparseable but loads valid CCs",
			existingCCs: []client.Object{
				&cachev1alpha1.ContentConfiguration{
					ObjectMeta: apimachinery.ObjectMeta{Name: "bad"},
					Status: cachev1alpha1.ContentConfigurationStatus{
						ConfigurationResult: "not json",
					},
				},
				&cachev1alpha1.ContentConfiguration{
					ObjectMeta: apimachinery.ObjectMeta{Name: "good"},
					Status: cachev1alpha1.ContentConfigurationStatus{
						ConfigurationResult: validJSONDefiningEntityType("team"),
					},
				},
			},
			inlineContent:          validJSONWithEntityType("team"),
			registry:               validation.NewEntityTypeRegistry(),
			expectValidCondition:   ConditionStatusTrue,
			expectValidReason:      ValidationConditionReasonSuccess,
			expectConfigResult:     true,
			expectRegistryContains: []string{"global", "team"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reader client.Reader
			if tt.k8sReader != nil {
				reader = tt.k8sReader
			} else if tt.existingCCs != nil {
				reader = newFakeReader(tt.existingCCs...)
			}
			// tt.k8sReader == nil && tt.existingCCs == nil means nil reader (test for nil reader error path)

			sub := NewContentConfigurationSubroutine(
				validation.NewContentConfiguration(),
				http.DefaultClient,
				reader,
				tt.registry,
			)

			cc := &cachev1alpha1.ContentConfiguration{
				Spec: cachev1alpha1.ContentConfigurationSpec{
					InlineConfiguration: &cachev1alpha1.InlineConfiguration{
						Content:     tt.inlineContent,
						ContentType: "json",
					},
				},
			}

			_, err := sub.Process(context.Background(), cc)

			if tt.expectOperatorError {
				require.NotNil(t, err, "expected an OperatorError but got nil")
				return
			}

			require.Nil(t, err, "unexpected OperatorError: %v", err)

			cond := getCondition(cc.Status.Conditions, ValidationConditionType)
			assert.Equal(t, string(tt.expectValidCondition), string(cond.Status), "unexpected Valid condition status")
			assert.Equal(t, tt.expectValidReason, cond.Reason, "unexpected Valid condition reason")

			if tt.expectConfigResult {
				assert.NotEmpty(t, cc.Status.ConfigurationResult, "expected ConfigurationResult to be set")
			} else {
				assert.Empty(t, cc.Status.ConfigurationResult, "expected ConfigurationResult to be empty")
			}

			if tt.registry != nil && len(tt.expectRegistryContains) > 0 {
				known := tt.registry.KnownTypes()
				for _, et := range tt.expectRegistryContains {
					assert.True(t, known[et], "expected registry to contain entity type %q", et)
				}
			}
		})
	}
}

func TestProcess_RegistryInitOnlyRunsOnce(t *testing.T) {
	reader := newFakeReader(
		&cachev1alpha1.ContentConfiguration{
			ObjectMeta: apimachinery.ObjectMeta{Name: "seed"},
			Status: cachev1alpha1.ContentConfigurationStatus{
				ConfigurationResult: validJSONDefiningEntityType("project"),
			},
		},
	)

	registry := validation.NewEntityTypeRegistry()
	sub := NewContentConfigurationSubroutine(
		validation.NewContentConfiguration(),
		http.DefaultClient,
		reader,
		registry,
	)

	cc1 := &cachev1alpha1.ContentConfiguration{
		Spec: cachev1alpha1.ContentConfigurationSpec{
			InlineConfiguration: &cachev1alpha1.InlineConfiguration{
				Content:     validJSONWithEntityType("global"),
				ContentType: "json",
			},
		},
	}

	// First call: triggers registry init
	_, err := sub.Process(context.Background(), cc1)
	require.Nil(t, err)
	assert.True(t, registry.KnownTypes()["project"], "registry should contain 'project' after init")

	// Second call: should NOT re-init (registryInitDone is true)
	cc2 := &cachev1alpha1.ContentConfiguration{
		Spec: cachev1alpha1.ContentConfigurationSpec{
			InlineConfiguration: &cachev1alpha1.InlineConfiguration{
				Content:     validJSONWithEntityType("global"),
				ContentType: "json",
			},
		},
	}
	_, err = sub.Process(context.Background(), cc2)
	require.Nil(t, err)

	// "project" should still be present (not wiped by a second Bulkload)
	assert.True(t, registry.KnownTypes()["project"])
}

func TestProcess_NilRegistrySkipsEntityTypeValidation(t *testing.T) {
	// When registry is nil, entity type validation is skipped entirely
	sub := NewContentConfigurationSubroutine(
		validation.NewContentConfiguration(),
		http.DefaultClient,
		nil,
		nil,
	)

	// Use a CC that references a non-existent entity type -- should still pass
	// because registry is nil, so no entity type validation runs.
	cc := &cachev1alpha1.ContentConfiguration{
		Spec: cachev1alpha1.ContentConfigurationSpec{
			InlineConfiguration: &cachev1alpha1.InlineConfiguration{
				Content:     validJSONWithEntityType("nonexistent"),
				ContentType: "json",
			},
		},
	}

	_, err := sub.Process(context.Background(), cc)
	require.Nil(t, err)

	cond := getCondition(cc.Status.Conditions, ValidationConditionType)
	assert.Equal(t, string(ConditionStatusTrue), string(cond.Status))
	assert.NotEmpty(t, cc.Status.ConfigurationResult)
}

func TestProcess_EntityTypeValidationFailure_PreservesExistingConfigResult(t *testing.T) {
	reader := newFakeReader()
	registry := validation.NewEntityTypeRegistry()

	sub := NewContentConfigurationSubroutine(
		validation.NewContentConfiguration(),
		http.DefaultClient,
		reader,
		registry,
	)

	existingResult := validJSONWithEntityType("global")
	cc := &cachev1alpha1.ContentConfiguration{
		Spec: cachev1alpha1.ContentConfigurationSpec{
			InlineConfiguration: &cachev1alpha1.InlineConfiguration{
				Content:     validJSONWithEntityType("unknown-type"),
				ContentType: "json",
			},
		},
		Status: cachev1alpha1.ContentConfigurationStatus{
			ConfigurationResult: existingResult,
		},
	}

	_, err := sub.Process(context.Background(), cc)
	require.Nil(t, err)

	// ConfigurationResult should not have been overwritten
	assert.Equal(t, existingResult, cc.Status.ConfigurationResult)

	cond := getCondition(cc.Status.Conditions, ValidationConditionType)
	assert.Equal(t, string(ConditionStatusFalse), string(cond.Status))
	assert.Equal(t, ValidationConditionReasonFailed, cond.Reason)
	assert.Contains(t, cond.Message, "unknown-type")
}

func TestProcess_ValidCC_UpdatesRegistryWithDefinedEntityTypes(t *testing.T) {
	reader := newFakeReader()
	registry := validation.NewEntityTypeRegistry()

	sub := NewContentConfigurationSubroutine(
		validation.NewContentConfiguration(),
		http.DefaultClient,
		reader,
		registry,
	)

	// Process a CC that defines a new entity type
	cc := &cachev1alpha1.ContentConfiguration{
		Spec: cachev1alpha1.ContentConfigurationSpec{
			InlineConfiguration: &cachev1alpha1.InlineConfiguration{
				Content:     validJSONDefiningEntityType("newentity"),
				ContentType: "json",
			},
		},
	}

	_, err := sub.Process(context.Background(), cc)
	require.Nil(t, err)

	cond := getCondition(cc.Status.Conditions, ValidationConditionType)
	assert.Equal(t, string(ConditionStatusTrue), string(cond.Status))

	// After processing, the registry should now contain the newly defined entity type
	known := registry.KnownTypes()
	assert.True(t, known["newentity"], "registry should contain 'newentity' after processing CC that defines it")
	assert.True(t, known["global"], "registry should always contain 'global'")
}
