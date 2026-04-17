package validation

import "testing"

func FuzzValidate(f *testing.F) {
	f.Add([]byte(`{"name":"overview","luigiConfigFragment":{"data":{"nodes":[{"entityType":"global"}]}}}`), "json")
	f.Add([]byte(`name: overview`), "yaml")
	f.Add([]byte(`{}`), "json")
	f.Add([]byte(``), "json")
	f.Add([]byte(`not valid json`), "json")
	f.Add([]byte(`!invalid yaml`), "yaml")
	f.Add([]byte(`{"key":"value"}`), "xml")

	f.Fuzz(func(t *testing.T, input []byte, contentType string) {
		cC := NewContentConfiguration()
		// Must not panic — validation errors are expected
		_, _ = cC.Validate(input, contentType)
	})
}
