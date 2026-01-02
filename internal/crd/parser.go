package crd

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// Parser handles parsing of CRD resources from YAML
type Parser struct{}

// NewParser creates a new CRD parser
func NewParser() *Parser {
	return &Parser{}
}

// ParseFile parses a CRD resource from a YAML file
func (p *Parser) ParseFile(filepath string) (Resource, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return p.Parse(data)
}

// Parse parses a CRD resource from YAML bytes
func (p *Parser) Parse(data []byte) (Resource, error) {
	// First, parse to get the kind
	var meta struct {
		APIVersion string       `yaml:"apiVersion"`
		Kind       ResourceKind `yaml:"kind"`
	}

	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	// Validate API version
	if meta.APIVersion != APIVersion {
		return nil, fmt.Errorf("unsupported API version: %s", meta.APIVersion)
	}

	// Parse based on kind
	var resource Resource
	switch meta.Kind {
	case KindSoul:
		var soul Soul
		if err := yaml.Unmarshal(data, &soul); err != nil {
			return nil, fmt.Errorf("failed to parse Soul: %w", err)
		}
		resource = &soul
	case KindMind:
		var mind Mind
		if err := yaml.Unmarshal(data, &mind); err != nil {
			return nil, fmt.Errorf("failed to parse Mind: %w", err)
		}
		resource = &mind
	case KindCraft:
		var craft Craft
		if err := yaml.Unmarshal(data, &craft); err != nil {
			return nil, fmt.Errorf("failed to parse Craft: %w", err)
		}
		resource = &craft
	case KindRobot:
		var robot Robot
		if err := yaml.Unmarshal(data, &robot); err != nil {
			return nil, fmt.Errorf("failed to parse Robot: %w", err)
		}
		resource = &robot
	case KindTeam:
		var team Team
		if err := yaml.Unmarshal(data, &team); err != nil {
			return nil, fmt.Errorf("failed to parse Team: %w", err)
		}
		resource = &team
	case KindCollaboration:
		var collab Collaboration
		if err := yaml.Unmarshal(data, &collab); err != nil {
			return nil, fmt.Errorf("failed to parse Collaboration: %w", err)
		}
		resource = &collab
	default:
		return nil, fmt.Errorf("unknown resource kind: %s", meta.Kind)
	}

	// Validate the resource
	if err := resource.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return resource, nil
}

// ParseMultiple parses multiple resources from a YAML file with --- separators
func (p *Parser) ParseMultiple(data []byte) ([]Resource, error) {
	// Split by --- separator
	decoder := yaml.NewDecoder(bytes.NewReader(data))

	var resources []Resource
	for {
		var doc interface{}
		err := decoder.Decode(&doc)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to decode YAML document: %w", err)
		}

		// Re-marshal and parse individual document
		docBytes, err := yaml.Marshal(doc)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal document: %w", err)
		}

		resource, err := p.Parse(docBytes)
		if err != nil {
			return nil, err
		}

		resources = append(resources, resource)
	}

	return resources, nil
}

// Marshal converts a resource back to YAML
func (p *Parser) Marshal(resource Resource) ([]byte, error) {
	return yaml.Marshal(resource)
}
