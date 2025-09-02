package tailscale

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// TailscaleClient provides methods to interact with the Tailscale API
type TailscaleClient struct {
	APIKey     string
	TailnetID  string
	HTTPClient *http.Client
}

// Device represents a device in the Tailscale tailnet
type Device struct {
	Name       string   `json:"name"`
	Addresses  []string `json:"addresses"`
	LastSeen   time.Time `json:"lastSeen"`
	ID         string    `json:"id"`
	User       string    `json:"user"`
	OS         string    `json:"os"`
	MachineKey string    `json:"machineKey"`
}

// DevicesResponse represents the response from the Tailscale API for devices
type DevicesResponse struct {
	Devices []Device `json:"devices"`
}

// NewTailscaleClient creates a new Tailscale client with the provided API key and tailnet ID
func NewTailscaleClient() (*TailscaleClient, error) {
	apiKey := os.Getenv("TAILSCALE_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("TAILSCALE_API_KEY environment variable not set")
	}

	tailnetID := os.Getenv("TAILSCALE_TAILNET_ID")
	if tailnetID == "" {
		return nil, fmt.Errorf("TAILSCALE_TAILNET_ID environment variable not set")
	}

	return &TailscaleClient{
		APIKey:     apiKey,
		TailnetID:  tailnetID,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}, nil
}

// GetDevices retrieves all devices in the tailnet from the Tailscale API
func (c *TailscaleClient) GetDevices() ([]Device, error) {
	url := fmt.Sprintf("https://api.tailscale.com/api/v2/tailnet/%s/devices", c.TailnetID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.APIKey, "")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}

	var devicesResp DevicesResponse
	if err := json.NewDecoder(resp.Body).Decode(&devicesResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return devicesResp.Devices, nil
}

// FindDeviceByIP searches for a device with the specified IP address
func (c *TailscaleClient) FindDeviceByIP(ipAddress string) (*Device, error) {
	devices, err := c.GetDevices()
	if err != nil {
		return nil, err
	}

	for _, device := range devices {
		for _, addr := range device.Addresses {
			if addr == ipAddress {
				return &device, nil
			}
		}
	}

	return nil, nil // No device found with the specified IP address
}