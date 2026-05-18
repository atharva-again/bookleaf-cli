package mcp

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const protocolVersion = "2024-11-05"

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type toolResult struct {
	Content []toolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type toolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type toolDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Client struct {
	baseURL    string
	authToken  string
	httpClient *http.Client
}

func NewClient(baseURL string, authToken string) *Client {
	return &Client{
		baseURL:   baseURL,
		authToken: authToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Initialize performs the MCP initialize handshake.
func (c *Client) Initialize() error {
	var result json.RawMessage
	if err := c.doRPC("initialize", nil, &result); err != nil {
		return fmt.Errorf("mcp init: %w", err)
	}
	_ = c.doRPC("notifications/initialized", nil, nil)
	return nil
}

// CallTool invokes an MCP tool and returns the parsed text content.
func (c *Client) CallTool(name string, args map[string]any) (string, error) {
	params := map[string]any{
		"name":      name,
		"arguments": args,
	}

	var raw json.RawMessage
	if err := c.doRPC("tools/call", params, &raw); err != nil {
		return "", err
	}

	var result toolResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("parse tool result: %w", err)
	}

	if result.IsError {
		msg := "unknown error"
		if len(result.Content) > 0 {
			msg = result.Content[0].Text
		}
		return "", fmt.Errorf("tool error: %s", msg)
	}

	if len(result.Content) == 0 {
		return "", fmt.Errorf("empty tool result")
	}

	return result.Content[0].Text, nil
}

// doRPC sends a JSON-RPC request and unmarshals the result.
func (c *Client) doRPC(method string, params any, result any) error {
	req := rpcRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal rpc: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/api/mcp", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.authToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("http call: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if httpResp.StatusCode == 401 {
		return fmt.Errorf("authentication required. Run 'bookleaf auth login'")
	}

	if httpResp.StatusCode != 200 {
		return fmt.Errorf("server error: %d %s", httpResp.StatusCode, string(respBody))
	}

	var rpcResp rpcResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return fmt.Errorf("parse rpc response: %w", err)
	}

	if rpcResp.Error != nil {
		return fmt.Errorf("rpc error (%d): %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	if result == nil {
		return nil
	}

	if err := json.Unmarshal(rpcResp.Result, result); err != nil {
		return fmt.Errorf("unmarshal result: %w (raw: %s)", err, string(rpcResp.Result))
	}

	return nil
}

// DeviceAuthClient handles communication with the device auth endpoints.
type DeviceAuthClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewDeviceAuthClient(baseURL string) *DeviceAuthClient {
	return &DeviceAuthClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

func (d *DeviceAuthClient) RequestDeviceCode() (*DeviceCodeResponse, error) {
	body := map[string]string{"client_id": "bookleaf-cli"}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal device code request: %w", err)
	}

	resp, err := d.httpClient.Post(d.baseURL+"/api/auth/device", "application/json", bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("request device code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("device code request failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var result DeviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parse device code: %w", err)
	}

	return &result, nil
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

type pollErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"error_description"`
}

func (d *DeviceAuthClient) PollForToken(deviceCode string, interval int) (*tokenResponse, error) {
	body := map[string]string{"device_code": deviceCode}
	bodyJSON, _ := json.Marshal(body)

	if interval < 1 {
		interval = 5
	}
	maxAttempts := 600 / interval // poll for up to 10 minutes (matches expires_in)

	for attempt := 0; attempt < maxAttempts; attempt++ {
		resp, err := d.httpClient.Post(d.baseURL+"/api/auth/device/poll", "application/json", bytes.NewReader(bodyJSON))
		if err != nil {
			return nil, fmt.Errorf("poll: %w", err)
		}

		if resp.StatusCode == 200 {
			var token tokenResponse
			if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
				resp.Body.Close()
				return nil, fmt.Errorf("parse token: %w", err)
			}
			resp.Body.Close()
			return &token, nil
		}

		var pollErr pollErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&pollErr); err == nil {
			resp.Body.Close()
			if pollErr.Error == "authorization_pending" {
				time.Sleep(time.Duration(interval) * time.Second)
				continue
			}
			return nil, fmt.Errorf("poll failed: %s", pollErr.Message)
		}
		resp.Body.Close()

		time.Sleep(time.Duration(interval) * time.Second)
	}

	return nil, fmt.Errorf("poll expired: authorization timed out after 10 minutes")
}

// TokenPayload represents the decoded access token payload.
type TokenPayload struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	Exp    int64  `json:"exp"`
}

// DecodeToken extracts the payload from an access token without verification.
// The token format is base64url(payload).hex(signature).
func DecodeToken(token string) (*TokenPayload, error) {
	dot := strings.LastIndex(token, ".")
	if dot == -1 {
		return nil, fmt.Errorf("invalid token format")
	}
	raw := token[:dot]
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("decode token payload: %w", err)
	}
	var payload TokenPayload
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return nil, fmt.Errorf("parse token payload: %w", err)
	}
	return &payload, nil
}
