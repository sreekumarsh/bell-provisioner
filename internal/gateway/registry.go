package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// DeviceType is a registry device type row from GET /admin/device-types.
type DeviceType struct {
	DTID         string `json:"dtid"`
	DFID         string `json:"dfid"`
	FriendlyName string `json:"friendly_name"`
	Deprecated   bool   `json:"deprecated"`
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
