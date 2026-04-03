package athena

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/ixxet/ashton-mcp-gateway/internal/manifest"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type Occupancy struct {
	FacilityID   string `json:"facility_id"`
	CurrentCount int    `json:"current_count"`
	ObservedAt   string `json:"observed_at"`
	Source       string `json:"source_service"`
}

type UpstreamError struct {
	StatusCode int
	Message    string
}

func (e *UpstreamError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("athena occupancy request failed with status %d", e.StatusCode)
	}
	return fmt.Sprintf("athena occupancy request failed with status %d: %s", e.StatusCode, e.Message)
}

type MalformedResponseError struct {
	Err error
}

func (e *MalformedResponseError) Error() string {
	return fmt.Sprintf("athena occupancy response is malformed: %v", e.Err)
}

func (e *MalformedResponseError) Unwrap() error {
	return e.Err
}

func NewClient(baseURL string, httpClient *http.Client) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
	}
}

func (c *Client) CurrentOccupancy(ctx context.Context, tool manifest.Tool, facilityID string) (Occupancy, error) {
	endpoint, err := url.Parse(c.baseURL + tool.Upstream.Path)
	if err != nil {
		return Occupancy{}, fmt.Errorf("build athena occupancy endpoint: %w", err)
	}

	query := endpoint.Query()
	for key, source := range tool.Upstream.Query {
		if source == "facility_id" {
			query.Set(key, facilityID)
		}
	}
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return Occupancy{}, fmt.Errorf("build athena occupancy request: %w", err)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return Occupancy{}, fmt.Errorf("athena occupancy request failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		var upstreamBody map[string]string
		if err := json.NewDecoder(response.Body).Decode(&upstreamBody); err == nil {
			return Occupancy{}, &UpstreamError{
				StatusCode: response.StatusCode,
				Message:    upstreamBody["error"],
			}
		}
		return Occupancy{}, &UpstreamError{StatusCode: response.StatusCode}
	}

	var result Occupancy
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return Occupancy{}, &MalformedResponseError{Err: err}
	}

	if result.FacilityID == "" {
		return Occupancy{}, &MalformedResponseError{Err: fmt.Errorf("facility_id is required")}
	}
	if result.ObservedAt == "" {
		return Occupancy{}, &MalformedResponseError{Err: fmt.Errorf("observed_at is required")}
	}
	result.Source = tool.Upstream.Service

	return result, nil
}
