package validation

import "github.com/hashicorp/go-multierror"

type ExtensionConfiguration interface {
	Validate([]byte, string) (string, *multierror.Error)
	ValidateEntityTypes(input []byte, contentType string, registry *EntityTypeRegistry) *multierror.Error
	ParseContentConfiguration(input []byte, contentType string) (*ContentConfiguration, error)
	WithSchema([]byte) error
}
