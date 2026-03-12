package validation

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEntityTypeRegistry(t *testing.T) {
	registry := NewEntityTypeRegistry()
	types := registry.KnownTypes()

	assert.True(t, types["global"])
	assert.Len(t, types, 1)
}

func TestCollectDefinedEntityTypes(t *testing.T) {
	tests := []struct {
		name     string
		cc       ContentConfiguration
		expected map[string]bool
	}{
		{
			name: "simple defineEntity under global",
			cc: ContentConfiguration{
				LuigiConfigFragment: LuigiConfigFragment{
					Data: LuigiConfigData{
						Nodes: []Node{
							{
								EntityType:   "global",
								DefineEntity: &DefineEntity{Id: "main"},
							},
						},
					},
				},
			},
			expected: map[string]bool{"main": true},
		},
		{
			name: "defineEntity under non-global entityType",
			cc: ContentConfiguration{
				LuigiConfigFragment: LuigiConfigFragment{
					Data: LuigiConfigData{
						Nodes: []Node{
							{
								EntityType:   "project",
								DefineEntity: &DefineEntity{Id: "custom"},
							},
						},
					},
				},
			},
			expected: map[string]bool{"project.custom": true},
		},
		{
			name: "nested defineEntity",
			cc: ContentConfiguration{
				LuigiConfigFragment: LuigiConfigFragment{
					Data: LuigiConfigData{
						Nodes: []Node{
							{
								EntityType:   "project",
								DefineEntity: &DefineEntity{Id: "component"},
								Children: []Node{
									{
										DefineEntity: &DefineEntity{Id: "overview"},
									},
								},
							},
						},
					},
				},
			},
			expected: map[string]bool{
				"project.component":          true,
				"project.component.overview": true,
			},
		},
		{
			name: "no defineEntity",
			cc: ContentConfiguration{
				LuigiConfigFragment: LuigiConfigFragment{
					Data: LuigiConfigData{
						Nodes: []Node{
							{
								EntityType: "global",
							},
						},
					},
				},
			},
			expected: map[string]bool{},
		},
		{
			name: "defineEntity with nodeDefaults entityType",
			cc: ContentConfiguration{
				LuigiConfigFragment: LuigiConfigFragment{
					Data: LuigiConfigData{
						NodeDefaults: &NodeDefaults{EntityType: "project"},
						Nodes: []Node{
							{
								DefineEntity: &DefineEntity{Id: "custom"},
							},
						},
					},
				},
			},
			expected: map[string]bool{"project.custom": true},
		},
		{
			name: "defineEntity with empty id is ignored",
			cc: ContentConfiguration{
				LuigiConfigFragment: LuigiConfigFragment{
					Data: LuigiConfigData{
						Nodes: []Node{
							{
								EntityType:   "global",
								DefineEntity: &DefineEntity{Id: ""},
							},
						},
					},
				},
			},
			expected: map[string]bool{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collectDefinedEntityTypes(tt.cc)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCollectReferencedEntityTypes(t *testing.T) {
	tests := []struct {
		name     string
		cc       ContentConfiguration
		expected []string
	}{
		{
			name: "single node",
			cc: ContentConfiguration{
				LuigiConfigFragment: LuigiConfigFragment{
					Data: LuigiConfigData{
						Nodes: []Node{
							{EntityType: "global"},
						},
					},
				},
			},
			expected: []string{"global"},
		},
		{
			name: "nodeDefaults and node",
			cc: ContentConfiguration{
				LuigiConfigFragment: LuigiConfigFragment{
					Data: LuigiConfigData{
						NodeDefaults: &NodeDefaults{EntityType: "project"},
						Nodes: []Node{
							{EntityType: "team"},
						},
					},
				},
			},
			expected: []string{"project", "team"},
		},
		{
			name: "nested children",
			cc: ContentConfiguration{
				LuigiConfigFragment: LuigiConfigFragment{
					Data: LuigiConfigData{
						Nodes: []Node{
							{
								EntityType: "project",
								Children: []Node{
									{EntityType: "project.component"},
								},
							},
						},
					},
				},
			},
			expected: []string{"project", "project.component"},
		},
		{
			name: "empty entityType is not collected",
			cc: ContentConfiguration{
				LuigiConfigFragment: LuigiConfigFragment{
					Data: LuigiConfigData{
						Nodes: []Node{
							{EntityType: ""},
						},
					},
				},
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collectReferencedEntityTypes(tt.cc)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEntityTypeRegistry_Load(t *testing.T) {
	registry := NewEntityTypeRegistry()

	configs := []ContentConfiguration{
		{
			LuigiConfigFragment: LuigiConfigFragment{
				Data: LuigiConfigData{
					Nodes: []Node{
						{
							EntityType:   "global",
							DefineEntity: &DefineEntity{Id: "project"},
						},
					},
				},
			},
		},
		{
			LuigiConfigFragment: LuigiConfigFragment{
				Data: LuigiConfigData{
					Nodes: []Node{
						{
							EntityType:   "global",
							DefineEntity: &DefineEntity{Id: "team"},
						},
					},
				},
			},
		},
	}

	registry.Bulkload(configs)
	types := registry.KnownTypes()

	assert.True(t, types["global"])
	assert.True(t, types["project"])
	assert.True(t, types["team"])
	assert.Len(t, types, 3)
}

func TestEntityTypeRegistry_Update(t *testing.T) {
	registry := NewEntityTypeRegistry()

	cc := ContentConfiguration{
		LuigiConfigFragment: LuigiConfigFragment{
			Data: LuigiConfigData{
				Nodes: []Node{
					{
						EntityType:   "project",
						DefineEntity: &DefineEntity{Id: "component"},
					},
				},
			},
		},
	}

	registry.Load(cc)
	types := registry.KnownTypes()

	assert.True(t, types["global"])
	assert.True(t, types["project.component"])
	assert.Len(t, types, 2)
}

func TestEntityTypeRegistry_Remove(t *testing.T) {
	registry := NewEntityTypeRegistry()

	cc := ContentConfiguration{
		LuigiConfigFragment: LuigiConfigFragment{
			Data: LuigiConfigData{
				Nodes: []Node{
					{
						EntityType:   "global",
						DefineEntity: &DefineEntity{Id: "project"},
					},
				},
			},
		},
	}

	registry.Load(cc)
	require.True(t, registry.KnownTypes()["project"])

	registry.Remove(cc)
	assert.False(t, registry.KnownTypes()["project"])
	assert.True(t, registry.KnownTypes()["global"])
}

func TestEntityTypeRegistry_Remove_RefCounting(t *testing.T) {
	registry := NewEntityTypeRegistry()

	cc1 := ContentConfiguration{
		LuigiConfigFragment: LuigiConfigFragment{
			Data: LuigiConfigData{
				Nodes: []Node{
					{EntityType: "global", DefineEntity: &DefineEntity{Id: "project"}},
				},
			},
		},
	}
	cc2 := ContentConfiguration{
		LuigiConfigFragment: LuigiConfigFragment{
			Data: LuigiConfigData{
				Nodes: []Node{
					{EntityType: "global", DefineEntity: &DefineEntity{Id: "project"}},
				},
			},
		},
	}

	registry.Load(cc1)
	registry.Load(cc2)
	require.True(t, registry.KnownTypes()["project"])

	// Remove one — "project" should still be known
	registry.Remove(cc1)
	assert.True(t, registry.KnownTypes()["project"])

	// Validate still passes
	errs := registry.Validate(ContentConfiguration{
		LuigiConfigFragment: LuigiConfigFragment{
			Data: LuigiConfigData{
				Nodes: []Node{{EntityType: "project"}},
			},
		},
	})
	assert.Empty(t, errs)

	// Remove the second — now "project" should be gone
	registry.Remove(cc2)
	assert.False(t, registry.KnownTypes()["project"])

	// Validate now fails
	errs = registry.Validate(ContentConfiguration{
		LuigiConfigFragment: LuigiConfigFragment{
			Data: LuigiConfigData{
				Nodes: []Node{{EntityType: "project"}},
			},
		},
	})
	assert.Len(t, errs, 1)
}

func TestEntityTypeRegistry_Validate(t *testing.T) {
	registry := NewEntityTypeRegistry()
	registry.Bulkload([]ContentConfiguration{
		{
			LuigiConfigFragment: LuigiConfigFragment{
				Data: LuigiConfigData{
					Nodes: []Node{
						{EntityType: "global", DefineEntity: &DefineEntity{Id: "project"}},
						{EntityType: "global", DefineEntity: &DefineEntity{Id: "team"}},
						{EntityType: "project", DefineEntity: &DefineEntity{Id: "component"}},
						{EntityType: "project", DefineEntity: &DefineEntity{Id: "team"}},
					},
				},
			},
		},
	})

	tests := []struct {
		name        string
		cc          ContentConfiguration
		expectCount int
	}{
		{
			name: "all valid",
			cc: ContentConfiguration{
				LuigiConfigFragment: LuigiConfigFragment{
					Data: LuigiConfigData{
						Nodes: []Node{
							{EntityType: "global"},
							{EntityType: "project"},
							{EntityType: "team"},
							{EntityType: "project.component"},
							{EntityType: "project.team"},
						},
					},
				},
			},
			expectCount: 0,
		},
		{
			name: "unknown entityType",
			cc: ContentConfiguration{
				LuigiConfigFragment: LuigiConfigFragment{
					Data: LuigiConfigData{
						Nodes: []Node{
							{EntityType: "nonexistent"},
						},
					},
				},
			},
			expectCount: 1,
		},
		{
			name: "compound suffix is stripped",
			cc: ContentConfiguration{
				LuigiConfigFragment: LuigiConfigFragment{
					Data: LuigiConfigData{
						Nodes: []Node{
							{EntityType: "project.component::compound"},
						},
					},
				},
			},
			expectCount: 0,
		},
		{
			name: "unknown compound base",
			cc: ContentConfiguration{
				LuigiConfigFragment: LuigiConfigFragment{
					Data: LuigiConfigData{
						Nodes: []Node{
							{EntityType: "nonexistent::compound"},
						},
					},
				},
			},
			expectCount: 1,
		},
		{
			name: "empty entityType is skipped",
			cc: ContentConfiguration{
				LuigiConfigFragment: LuigiConfigFragment{
					Data: LuigiConfigData{
						Nodes: []Node{
							{EntityType: ""},
						},
					},
				},
			},
			expectCount: 0,
		},
		{
			name: "global is always valid",
			cc: ContentConfiguration{
				LuigiConfigFragment: LuigiConfigFragment{
					Data: LuigiConfigData{
						NodeDefaults: &NodeDefaults{EntityType: "global"},
						Nodes: []Node{
							{EntityType: "global"},
						},
					},
				},
			},
			expectCount: 0,
		},
		{
			name: "multiple unknown",
			cc: ContentConfiguration{
				LuigiConfigFragment: LuigiConfigFragment{
					Data: LuigiConfigData{
						Nodes: []Node{
							{EntityType: "foo"},
							{EntityType: "bar"},
						},
					},
				},
			},
			expectCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := registry.Validate(tt.cc)
			assert.Len(t, errs, tt.expectCount)
		})
	}
}

func TestNormalizeEntityType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"global", "global"},
		{"project", "project"},
		{"project.component", "project.component"},
		{"project.overview::compound", "project.overview"},
		{"team.overview::compound", "team.overview"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizeEntityType(tt.input))
		})
	}
}

func TestBuildEntityTypeName(t *testing.T) {
	tests := []struct {
		name           string
		parentType     string
		defineEntityId string
		expected       string
	}{
		{"global parent", "global", "main", "main"},
		{"empty parent", "", "main", "main"},
		{"non-global parent", "project", "component", "project.component"},
		{"nested parent", "project.component", "overview", "project.component.overview"},
		{"global compound parent", "global::compound", "test", "test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, buildEntityTypeName(tt.parentType, tt.defineEntityId))
		})
	}
}

func TestEntityTypeRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewEntityTypeRegistry()

	var wg sync.WaitGroup
	for range 100 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			cc := ContentConfiguration{
				LuigiConfigFragment: LuigiConfigFragment{
					Data: LuigiConfigData{
						Nodes: []Node{
							{EntityType: "global", DefineEntity: &DefineEntity{Id: "project"}},
						},
					},
				},
			}
			registry.Load(cc)
		}()
		go func() {
			defer wg.Done()
			cc := ContentConfiguration{
				LuigiConfigFragment: LuigiConfigFragment{
					Data: LuigiConfigData{
						Nodes: []Node{
							{EntityType: "project"},
						},
					},
				},
			}
			registry.Validate(cc)
		}()
	}
	wg.Wait()
}

func TestValidateEntityTypes_Integration(t *testing.T) {
	cC := NewContentConfiguration()
	registry := NewEntityTypeRegistry()
	registry.Bulkload([]ContentConfiguration{
		{
			LuigiConfigFragment: LuigiConfigFragment{
				Data: LuigiConfigData{
					Nodes: []Node{
						{EntityType: "global", DefineEntity: &DefineEntity{Id: "project"}},
						{EntityType: "global", DefineEntity: &DefineEntity{Id: "team"}},
					},
				},
			},
		},
	})

	input := []byte(`{
		"name": "test",
		"luigiConfigFragment": {
			"data": {
				"nodes": [{
					"entityType": "project",
					"pathSegment": "test"
				}]
			}
		}
	}`)

	merr := cC.ValidateEntityTypes(input, "json", registry)
	assert.Nil(t, merr)

	invalidInput := []byte(`{
		"name": "test",
		"luigiConfigFragment": {
			"data": {
				"nodes": [{
					"entityType": "nonexistent",
					"pathSegment": "test"
				}]
			}
		}
	}`)

	merr = cC.ValidateEntityTypes(invalidInput, "json", registry)
	require.NotNil(t, merr)
	assert.GreaterOrEqual(t, merr.Len(), 1)
	assert.Contains(t, merr.Error(), "nonexistent")
}
