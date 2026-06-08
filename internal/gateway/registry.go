package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Capability is a registry capability row from GET /admin/capabilities.
type Capability struct {
	CapID        string `json:"capid"`
	FriendlyName string `json:"friendly_name"`
	Layer        string `json:"layer"`
	Description  string `json:"description,omitempty"`
	Deprecated   bool   `json:"deprecated"`
}

// DeviceFamily is a registry family row from GET /admin/device-families.
type DeviceFamily struct {
	DFID         string   `json:"dfid"`
	FriendlyName string   `json:"friendly_name"`
	Description  string   `json:"description,omitempty"`
	DeviceTypes  []string `json:"device_types,omitempty"`
	Deprecated   bool     `json:"deprecated"`
}

// DeviceType is a registry device type row from GET /admin/device-types.
type DeviceType struct {
	DTID         string   `json:"dtid"`
	DFID         string   `json:"dfid"`
	FriendlyName string   `json:"friendly_name"`
	Capabilities []string `json:"capabilities,omitempty"`
	Description  string   `json:"description,omitempty"`
	Deprecated   bool     `json:"deprecated"`
}

// ApplyRegistryResult is returned from POST /admin/device-registry/apply.
type ApplyRegistryResult struct {
	Added    []string `json:"added"`
	Updated  []string `json:"updated"`
	Rejected []string `json:"rejected"`
}

// ListCapabilities returns registry capabilities.
func (c *Client) ListCapabilities() ([]Capability, error) {
	data, status, err := c.do(http.MethodGet, "/admin/capabilities", nil, true)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, parseError(status, data)
	}
	var resp struct {
		Capabilities []Capability `json:"capabilities"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode capabilities: %w", err)
	}
	if resp.Capabilities == nil {
		return []Capability{}, nil
	}
	return resp.Capabilities, nil
}

// CreateCapability adds a capability to the registry.
func (c *Client) CreateCapability(cap Capability) (*Capability, error) {
	data, status, err := c.do(http.MethodPost, "/admin/capabilities", cap, true)
	if err != nil {
		return nil, err
	}
	if status != http.StatusCreated {
		return nil, parseError(status, data)
	}
	var out Capability
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("decode capability: %w", err)
	}
	return &out, nil
}

// ListDeviceFamilies returns registry families.
func (c *Client) ListDeviceFamilies() ([]DeviceFamily, error) {
	data, status, err := c.do(http.MethodGet, "/admin/device-families", nil, true)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, parseError(status, data)
	}
	var resp struct {
		Families []DeviceFamily `json:"families"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode families: %w", err)
	}
	if resp.Families == nil {
		return []DeviceFamily{}, nil
	}
	return resp.Families, nil
}

// CreateDeviceFamily adds a device family to the registry.
func (c *Client) CreateDeviceFamily(family DeviceFamily) (*DeviceFamily, error) {
	data, status, err := c.do(http.MethodPost, "/admin/device-families", family, true)
	if err != nil {
		return nil, err
	}
	if status != http.StatusCreated {
		return nil, parseError(status, data)
	}
	var out DeviceFamily
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("decode family: %w", err)
	}
	return &out, nil
}

// ListDeviceTypes returns registry types for the provision DTID picker.
func (c *Client) ListDeviceTypes() ([]DeviceType, error) {
	data, status, err := c.do(http.MethodGet, "/admin/device-types", nil, true)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, parseError(status, data)
	}
	var resp struct {
		Types []DeviceType `json:"types"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode device types: %w", err)
	}
	if resp.Types == nil {
		return []DeviceType{}, nil
	}
	return resp.Types, nil
}

// CreateDeviceType adds a device type to the registry.
func (c *Client) CreateDeviceType(typ DeviceType) (*DeviceType, error) {
	data, status, err := c.do(http.MethodPost, "/admin/device-types", typ, true)
	if err != nil {
		return nil, err
	}
	if status != http.StatusCreated {
		return nil, parseError(status, data)
	}
	var out DeviceType
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("decode device type: %w", err)
	}
	return &out, nil
}

// ApplyRegistry applies the server-side registry YAML bundle.
func (c *Client) ApplyRegistry(dryRun bool) (*ApplyRegistryResult, error) {
	path := "/admin/device-registry/apply"
	if dryRun {
		path += "?dry_run=true"
	}
	data, status, err := c.do(http.MethodPost, path, map[string]string{}, true)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK && status != http.StatusConflict {
		return nil, parseError(status, data)
	}
	var out ApplyRegistryResult
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("decode apply result: %w", err)
	}
	return &out, nil
}
