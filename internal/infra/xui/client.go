package xui

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	ErrNotFound       = errors.New("not found")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrAPIError       = errors.New("3x-ui api error")
	ErrInvalidRequest = errors.New("invalid request")
)

// ClientPayload represents client configuration for Xray inbounds.
// Strictly conforms to 3x-ui_openapi.json and docs/3xui_contract.md.
type ClientPayload struct {
	ID                  string              `json:"id,omitempty"`      // UUID v4
	Email               string              `json:"email"`             // Unique identifier
	SubID               string              `json:"subId,omitempty"`   // Subscription ID
	Password            string              `json:"password,omitempty"`
	Auth                string              `json:"auth,omitempty"`
	Flow                string              `json:"flow,omitempty"`     // e.g. "xtls-rprx-vision"
	TotalGB             int64               `json:"totalGB"`            // Traffic limit in bytes
	ExpiryTime          int64               `json:"expiryTime"`         // Unix timestamp in ms (0 = none)
	LimitIP             int                 `json:"limitIp"`            // Max concurrent IP limit
	LimitHwid           int                 `json:"limitHwid,omitempty"`
	TgID                int64               `json:"tgId,omitempty"`     // Numeric Telegram user ID
	Group               string              `json:"group,omitempty"`    // Group name
	Comment             string              `json:"comment,omitempty"`
	Enable              bool                `json:"enable"`
	Secret              string              `json:"secret,omitempty"`
	Security            string              `json:"security,omitempty"`
	PreSharedKey        string              `json:"preSharedKey,omitempty"`
	PrivateKey          string              `json:"privateKey,omitempty"`
	PublicKey           string              `json:"publicKey,omitempty"`
	AllowedIPs          []string            `json:"allowedIPs,omitempty"`
	AllowedIPsByInbound map[string][]string `json:"allowedIPsByInbound,omitempty"`
	KeepAlive           int                 `json:"keepAlive,omitempty"`
	AdTag               string              `json:"adTag,omitempty"`
	Reset               int                 `json:"reset,omitempty"`
	ResetDay            int                 `json:"resetDay,omitempty"`
	ResetMax            int                 `json:"resetMax,omitempty"`
	TrafficReset        string              `json:"trafficReset,omitempty"`
	TrafficResetDay     int                 `json:"trafficResetDay,omitempty"`
}

// AddClientRequest represents the payload required by POST /panel/api/clients/add.
type AddClientRequest struct {
	Client     ClientPayload `json:"client"`
	InboundIDs []int         `json:"inboundIds"`
}

// ClientTrafficStats represents the traffic statistics of a client.
type ClientTrafficStats struct {
	Up         int64 `json:"up"`
	Down       int64 `json:"down"`
	Total      int64 `json:"total"`
	ExpiryTime int64 `json:"expiryTime"`
}

// ClientResponse represents the object returned by GET /panel/api/clients/get/{email}.
type ClientResponse struct {
	Client     ClientPayload      `json:"client"`
	InboundIDs []int              `json:"inboundIds"`
	Traffic    ClientTrafficStats `json:"traffic"`
}

// ClientTraffic represents individual client traffic counters in an inbound.
type ClientTraffic struct {
	ID           int    `json:"id"`
	InboundID    int    `json:"inboundId"`
	Email        string `json:"email"`
	SubID        string `json:"subId"`
	UUID         string `json:"uuid"`
	Up           int64  `json:"up"`
	Down         int64  `json:"down"`
	Total        int64  `json:"total"`
	ExpiryTime   int64  `json:"expiryTime"`
	Enable       bool   `json:"enable"`
	LastOnline   int64  `json:"lastOnline"`
	LastSubFetch int64  `json:"lastSubFetch"`
}

// Inbound represents a 3x-ui inbound configuration.
type Inbound struct {
	ID                   int             `json:"id"`
	Up                   int64           `json:"up"`
	Down                 int64           `json:"down"`
	Total                int64           `json:"total"`
	Remark               string          `json:"remark"`
	Enable               bool            `json:"enable"`
	ExpiryTime           int64           `json:"expiryTime"`
	Listen               string          `json:"listen"`
	Port                 int             `json:"port"`
	Protocol             string          `json:"protocol"`
	Tag                  string          `json:"tag"`
	StreamSettings       any             `json:"streamSettings,omitempty"`
	Settings             any             `json:"settings,omitempty"`
	Sniffing             any             `json:"sniffing,omitempty"`
	ClientStats          []ClientTraffic `json:"clientStats,omitempty"`
	DisableFlow          bool            `json:"disableFlow,omitempty"`
	SubSortIndex         int             `json:"subSortIndex,omitempty"`
	TrafficReset         string          `json:"trafficReset,omitempty"`
	TrafficResetDay      int             `json:"trafficResetDay,omitempty"`
	LastTrafficResetTime int64           `json:"lastTrafficResetTime,omitempty"`
}

// AllSetting contains panel settings returned by POST /panel/api/setting/all.
type AllSetting struct {
	SubURI      string `json:"subURI"`
	SubPath     string `json:"subPath"`
	SubDomain   string `json:"subDomain,omitempty"`
	SubPort     int    `json:"subPort,omitempty"`
	SubEnable   bool   `json:"subEnable,omitempty"`
	WebDomain   string `json:"webDomain,omitempty"`
	WebPort     int    `json:"webPort,omitempty"`
	WebBasePath string `json:"webBasePath,omitempty"`
}

// BaseResponse represents the standard 3x-ui API envelope.
type BaseResponse struct {
	Success bool            `json:"success"`
	Msg     string          `json:"msg"`
	Obj     json.RawMessage `json:"obj,omitempty"`
}

// Client provides an HTTP client for communicating with 3x-ui via Bearer auth.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient overrides the default HTTP client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		if httpClient != nil {
			c.httpClient = httpClient
		}
	}
}

// NewClient creates a new 3x-ui API client.
func NewClient(baseURL, apiKey string, opts ...Option) *Client {
	c := &Client{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		apiKey:  strings.TrimSpace(apiKey),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Client) doRequest(ctx context.Context, method, path string, body any, targetObj any) error {
	fullURL := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request to %s failed: %w", fullURL, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("%w: status %d", ErrUnauthorized, resp.StatusCode)
	}
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("%w: status %d", ErrNotFound, resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%w: status %d, body: %s", ErrAPIError, resp.StatusCode, string(respBytes))
	}

	var baseResp BaseResponse
	if err := json.Unmarshal(respBytes, &baseResp); err != nil {
		return fmt.Errorf("failed to parse json response: %w", err)
	}

	if !baseResp.Success {
		return fmt.Errorf("%w: %s", ErrAPIError, baseResp.Msg)
	}

	if targetObj != nil && len(baseResp.Obj) > 0 && string(baseResp.Obj) != "null" {
		if err := json.Unmarshal(baseResp.Obj, targetObj); err != nil {
			return fmt.Errorf("failed to decode response obj: %w", err)
		}
	}

	return nil
}

// AddClient creates a new client and attaches it to one or more inbounds (POST /panel/api/clients/add).
func (c *Client) AddClient(ctx context.Context, req AddClientRequest) error {
	if req.Client.Email == "" {
		return fmt.Errorf("%w: email cannot be empty", ErrInvalidRequest)
	}
	if len(req.InboundIDs) == 0 {
		return fmt.Errorf("%w: at least one inboundId is required", ErrInvalidRequest)
	}
	return c.doRequest(ctx, http.MethodPost, "/panel/api/clients/add", req, nil)
}

// UpdateClient updates an existing client by email (POST /panel/api/clients/update/{email}).
func (c *Client) UpdateClient(ctx context.Context, email string, client ClientPayload) error {
	if email == "" {
		return fmt.Errorf("%w: email cannot be empty", ErrInvalidRequest)
	}
	path := "/panel/api/clients/update/" + url.PathEscape(email)
	return c.doRequest(ctx, http.MethodPost, path, client, nil)
}

// DeleteClient deletes a client by email (POST /panel/api/clients/del/{email}).
func (c *Client) DeleteClient(ctx context.Context, email string) error {
	if email == "" {
		return fmt.Errorf("%w: email cannot be empty", ErrInvalidRequest)
	}
	path := "/panel/api/clients/del/" + url.PathEscape(email)
	return c.doRequest(ctx, http.MethodPost, path, nil, nil)
}

// GetClient fetches one client by email including attached inbounds and traffic stats (GET /panel/api/clients/get/{email}).
func (c *Client) GetClient(ctx context.Context, email string) (*ClientResponse, error) {
	if email == "" {
		return nil, fmt.Errorf("%w: email cannot be empty", ErrInvalidRequest)
	}
	path := "/panel/api/clients/get/" + url.PathEscape(email)
	var clientResp ClientResponse
	if err := c.doRequest(ctx, http.MethodGet, path, nil, &clientResp); err != nil {
		return nil, err
	}
	return &clientResp, nil
}

// ListInbounds lists every inbound owned by the authenticated user (GET /panel/api/inbounds/list).
func (c *Client) ListInbounds(ctx context.Context) ([]Inbound, error) {
	var inbounds []Inbound
	if err := c.doRequest(ctx, http.MethodGet, "/panel/api/inbounds/list", nil, &inbounds); err != nil {
		return nil, err
	}
	return inbounds, nil
}

// GetSubLinks returns protocol URLs matching the subscription ID (GET /panel/api/clients/subLinks/{subId}).
func (c *Client) GetSubLinks(ctx context.Context, subId string) ([]string, error) {
	if subId == "" {
		return nil, fmt.Errorf("%w: subId cannot be empty", ErrInvalidRequest)
	}
	path := "/panel/api/clients/subLinks/" + url.PathEscape(subId)
	var links []string
	if err := c.doRequest(ctx, http.MethodGet, path, nil, &links); err != nil {
		return nil, err
	}
	return links, nil
}

// ResetTraffic zeroes out a client's up/down counters (POST /panel/api/clients/resetTraffic/{email}).
func (c *Client) ResetTraffic(ctx context.Context, email string) error {
	if email == "" {
		return fmt.Errorf("%w: email cannot be empty", ErrInvalidRequest)
	}
	path := "/panel/api/clients/resetTraffic/" + url.PathEscape(email)
	return c.doRequest(ctx, http.MethodPost, path, nil, nil)
}

// GetSettings retrieves all panel settings (POST /panel/api/setting/all).
func (c *Client) GetSettings(ctx context.Context) (*AllSetting, error) {
	var settings AllSetting
	if err := c.doRequest(ctx, http.MethodPost, "/panel/api/setting/all", nil, &settings); err != nil {
		return nil, err
	}
	return &settings, nil
}

// BuildSubscriptionURL constructs the universal subscription URL delivered to the customer.
// Strictly follows docs/3xui_contract.md:
// 1. If subURI is populated, link is <subURI><subId> (handling trailing slashes correctly).
// 2. If subURI is empty, link is <panelURL><subPath><subId> (default subPath is /sub/).
func BuildSubscriptionURL(panelURL, subURI, subPath, subId string) string {
	subURI = strings.TrimSpace(subURI)
	subId = strings.TrimSpace(subId)

	if subURI != "" {
		if !strings.HasSuffix(subURI, "/") {
			subURI += "/"
		}
		return subURI + subId
	}

	panelURL = strings.TrimRight(strings.TrimSpace(panelURL), "/")
	subPath = strings.TrimSpace(subPath)
	if subPath == "" {
		subPath = "/sub/"
	}
	if !strings.HasPrefix(subPath, "/") {
		subPath = "/" + subPath
	}
	if !strings.HasSuffix(subPath, "/") {
		subPath = subPath + "/"
	}

	return panelURL + subPath + subId
}
