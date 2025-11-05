package willys

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

type Client struct {
	mu           sync.RWMutex
	httpClient   *http.Client
	baseURL      string
	csrfToken    string
	username     string
	password     string
	authAttempts atomic.Int32
}

const (
	DefaultTimeout       = 30 * time.Second
	DefaultPickingFee    = 59.0
	DefaultDeliveryFee   = 99.0
	MaxAuthRetryAttempts = 2

	maxIdleConns        = 100
	maxIdleConnsPerHost = 10
	idleConnTimeout     = 90 * time.Second
)

func newHTTPTransport() *http.Transport {
	return &http.Transport{
		MaxIdleConns:        maxIdleConns,
		MaxIdleConnsPerHost: maxIdleConnsPerHost,
		IdleConnTimeout:     idleConnTimeout,
		DisableKeepAlives:   false,
		DisableCompression:  false,
	}
}

func NewClient(baseURL, username, password string) (*Client, error) {
	if baseURL == "" {
		return nil, NewValidationError("base_url", "base URL cannot be empty")
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, NewValidationError("base_url", "invalid base URL format")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, NewValidationError("base_url", "base URL must use http or https scheme")
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	client := &Client{
		httpClient: &http.Client{
			Jar:       jar,
			Timeout:   DefaultTimeout,
			Transport: newHTTPTransport(),
		},
		baseURL:  baseURL,
		username: username,
		password: password,
	}
	client.authAttempts.Store(0)

	return client, nil
}

func (c *Client) GetCSRFToken() (string, error) {
	c.mu.RLock()
	token := c.csrfToken
	c.mu.RUnlock()

	if token != "" {
		return token, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.csrfToken != "" {
		return c.csrfToken, nil
	}

	return c.fetchCSRFTokenLocked()
}

func (c *Client) FetchCSRFToken() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.fetchCSRFTokenLocked()
}

func (c *Client) fetchCSRFTokenLocked() (string, error) {
	resp, err := c.httpClient.Get(c.baseURL + EndpointCSRFToken)
	if err != nil {
		return "", fmt.Errorf("failed to fetch CSRF token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("CSRF token request failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read CSRF token response: %w", err)
	}

	var token string
	if err := json.Unmarshal(body, &token); err != nil {
		var result struct {
			Token string `json:"token"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return "", fmt.Errorf("failed to parse CSRF token: %w", err)
		}
		token = result.Token
	}

	if token == "" {
		return "", fmt.Errorf("empty CSRF token")
	}

	c.csrfToken = token
	return token, nil
}

func (c *Client) createRequest(ctx context.Context, method, path string, bodyBytes []byte) (*http.Request, error) {
	reqURL := c.baseURL + path
	var req *http.Request
	var err error

	if ctx != nil {
		req, err = http.NewRequestWithContext(ctx, method, reqURL, bytes.NewReader(bodyBytes))
	} else {
		req, err = http.NewRequest(method, reqURL, bytes.NewReader(bodyBytes))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "sv-SE,sv;q=0.9,en;q=0.8")

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Origin", c.baseURL)
	req.Header.Set("Referer", c.baseURL+"/")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")

	return req, nil
}

func (c *Client) DoRequest(ctx context.Context, method, path string, body io.Reader, needsCSRF bool) (*http.Response, error) {
	if ctx != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}

	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
	}

	req, err := c.createRequest(ctx, method, path, bodyBytes)
	if err != nil {
		return nil, err
	}

	if needsCSRF {
		token, err := c.GetCSRFToken()
		if err != nil {
			return nil, fmt.Errorf("failed to get CSRF token: %w", err)
		}
		req.Header.Set("X-CSRF-TOKEN", token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized && needsCSRF {
		resp.Body.Close()

		if _, err := c.FetchCSRFToken(); err != nil {
			return nil, fmt.Errorf("failed to refresh CSRF token: %w", err)
		}

		req, err = c.createRequest(ctx, method, path, bodyBytes)
		if err != nil {
			return nil, err
		}

		token, err := c.GetCSRFToken()
		if err != nil {
			return nil, fmt.Errorf("failed to get updated CSRF token: %w", err)
		}
		req.Header.Set("X-CSRF-TOKEN", token)

		resp, err = c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("retry request failed: %w", err)
		}

		attempts := c.authAttempts.Load()
		c.mu.RLock()
		username := c.username
		password := c.password
		c.mu.RUnlock()

		if resp.StatusCode == http.StatusUnauthorized && username != "" && password != "" && attempts < MaxAuthRetryAttempts {
			resp.Body.Close()

			c.authAttempts.Add(1)

			if err := c.Login(ctx, username, password); err != nil {
				return nil, NewAuthenticationError("failed to re-authenticate", err)
			}

			req, err = c.createRequest(ctx, method, path, bodyBytes)
			if err != nil {
				return nil, err
			}

			token, err := c.GetCSRFToken()
			if err != nil {
				return nil, fmt.Errorf("failed to get CSRF token after re-auth: %w", err)
			}
			req.Header.Set("X-CSRF-TOKEN", token)

			resp, err = c.httpClient.Do(req)
			if err != nil {
				return nil, fmt.Errorf("final retry request failed: %w", err)
			}
		} else if resp.StatusCode == http.StatusUnauthorized && attempts >= MaxAuthRetryAttempts {
			resp.Body.Close()
			return nil, NewAuthenticationError("maximum authentication retry attempts exceeded", nil)
		}
	}

	return resp, nil
}

func (c *Client) GetCookies() []*http.Cookie {
	u, _ := url.Parse(c.baseURL)
	return c.httpClient.Jar.Cookies(u)
}

func (c *Client) SetCookies(cookies []*http.Cookie) {
	u, _ := url.Parse(c.baseURL)
	c.httpClient.Jar.SetCookies(u, cookies)
}
