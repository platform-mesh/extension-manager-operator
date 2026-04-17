package v1alpha1

import (
	"encoding/json"
	"testing"

	"k8s.io/apimachinery/pkg/api/equality"
)

func FuzzContentConfigurationRoundTrip(f *testing.F) {
	f.Add([]byte(`{"spec":{"inlineConfiguration":{"contentType":"json","content":"{}"}}}`))
	f.Add([]byte(`{"spec":{"remoteConfiguration":{"contentType":"yaml","url":"https://example.com/config.yaml","authentication":{"type":"bearer","secretRef":{"name":"my-secret"}}}}}`))
	f.Add([]byte(`{}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		fuzzRoundTrip(t, data, &ContentConfiguration{}, &ContentConfiguration{})
	})
}

func FuzzProviderMetadataRoundTrip(f *testing.F) {
	f.Add([]byte(`{"spec":{"displayName":"My Provider","description":"A provider","tags":["tag1","tag2"],"contacts":[{"displayName":"Admin","email":"admin@example.com","role":["owner"]}]}}`))
	f.Add([]byte(`{"spec":{"displayName":"","documentation":[{"displayName":"Docs","url":"https://example.com"}],"icon":{"light":{"url":"https://example.com/light.png"},"dark":{"url":"https://example.com/dark.png"}}}}`))
	f.Add([]byte(`{}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		fuzzRoundTrip(t, data, &ProviderMetadata{}, &ProviderMetadata{})
	})
}

// fuzzRoundTrip unmarshals arbitrary JSON into obj, marshals it back, unmarshals
// into obj2, and checks semantic equality. We use equality.Semantic.DeepEqual from
// k8s.io/apimachinery which treats nil and empty slices/maps as equivalent — the
// standard Kubernetes comparison semantic for API objects.
func fuzzRoundTrip[T any](t *testing.T, data []byte, obj *T, obj2 *T) {
	t.Helper()

	if err := json.Unmarshal(data, obj); err != nil {
		return
	}

	roundtripped, err := json.Marshal(obj)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	if err := json.Unmarshal(roundtripped, obj2); err != nil {
		t.Fatalf("failed to unmarshal roundtripped data: %v", err)
	}

	if !equality.Semantic.DeepEqual(obj, obj2) {
		t.Errorf("roundtrip mismatch for %T", obj)
	}
}
