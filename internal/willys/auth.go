package willys

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

type (
	LoginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	CustomerInfo struct {
		CustomerID   string `json:"customerId"`
		Email        string `json:"email"`
		FirstName    string `json:"firstName"`
		LastName     string `json:"lastName"`
		PhoneNumber  string `json:"phoneNumber"`
		PlusCustomer bool   `json:"plusCustomer"`
	}
)

// LoginWithBrowser uses headless browser automation because Willys requires cookie consent
// and some dynamic page loading before login. The time.Sleep calls are necessary since
// the page doesn't always reliably signal when elements are ready.
func (c *Client) LoginWithBrowser(ctx context.Context, username, password string) error {
	if username == "" {
		return NewValidationError("username", "username cannot be empty")
	}
	if password == "" {
		return NewValidationError("password", "password cannot be empty")
	}
	if len(password) < 6 {
		return NewValidationError("password", "password must be at least 6 characters")
	}

	path, exists := launcher.LookPath()
	if !exists {
		path = launcher.NewBrowser().MustGet()
	}

	u := launcher.New().
		Bin(path).
		Headless(true).
		Devtools(false).
		MustLaunch()

	browser := rod.New().ControlURL(u)
	if err := browser.Connect(); err != nil {
		return NewAuthenticationError("failed to connect to browser", err)
	}
	defer browser.MustClose()

	page, err := browser.Timeout(30 * time.Second).Page(proto.TargetCreateTarget{URL: c.baseURL})
	if err != nil {
		return NewAuthenticationError("failed to create page", err)
	}
	defer page.MustClose()

	if err := page.WaitLoad(); err != nil {
		return NewAuthenticationError("page failed to load", err)
	}

	time.Sleep(2 * time.Second) // wait for page to settle

	// Try to accept cookies if the banner appears
	acceptCookieBtn, err := page.Timeout(3*time.Second).ElementR("button", "Acceptera")
	if err == nil {
		if err := acceptCookieBtn.Click(proto.InputMouseButtonLeft, 1); err == nil {
			time.Sleep(500 * time.Millisecond)
		}
	}

	loginLink, err := page.Timeout(5*time.Second).ElementR("a", "Logga in")
	if err != nil {
		return NewAuthenticationError("failed to find login link", err)
	}

	if err := loginLink.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return NewAuthenticationError("failed to click login link", err)
	}

	time.Sleep(1 * time.Second) // dialog animation

	dialog, err := page.Timeout(5 * time.Second).Element("dialog, [role='dialog']")
	if err != nil {
		return NewAuthenticationError("failed to find login dialog", err)
	}

	usernameInput, err := dialog.Timeout(5 * time.Second).Element("input[type='text']")
	if err != nil {
		return NewAuthenticationError("failed to find username input field", err)
	}
	if err := usernameInput.Input(username); err != nil {
		return NewAuthenticationError("failed to input username", err)
	}

	passwordInput, err := dialog.Timeout(5 * time.Second).Element("input[type='password']")
	if err != nil {
		return NewAuthenticationError("failed to find password input field", err)
	}
	if err := passwordInput.Input(password); err != nil {
		return NewAuthenticationError("failed to input password", err)
	}

	time.Sleep(500 * time.Millisecond) // let form validate

	loginButton, err := page.Timeout(5*time.Second).ElementR("button", "^Logga in$")
	if err != nil {
		return NewAuthenticationError("failed to find login button", err)
	}
	if err := loginButton.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return NewAuthenticationError("failed to click login button", err)
	}

	time.Sleep(2 * time.Second) // wait for login response

	// Check for error indicators (they use different class names)
	hasError1, _, _ := page.Has("*[class*='error']")
	hasError2, _, _ := page.Has("*[class*='Error']")
	if hasError1 || hasError2 {
		return NewAuthenticationError("invalid username or password", nil)
	}

	cookies, err := page.Cookies(nil)
	if err != nil {
		return NewAuthenticationError("failed to extract cookies", err)
	}

	parsedURL, _ := url.Parse(c.baseURL)
	httpCookies := make([]*http.Cookie, 0, len(cookies))

	for _, cookie := range cookies {
		httpCookie := &http.Cookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Path:     cookie.Path,
			Domain:   cookie.Domain,
			Expires:  time.Unix(int64(cookie.Expires), 0),
			Secure:   cookie.Secure,
			HttpOnly: cookie.HTTPOnly,
			SameSite: http.SameSiteNoneMode,
		}
		httpCookies = append(httpCookies, httpCookie)
	}

	c.httpClient.Jar.SetCookies(parsedURL, httpCookies)

	c.mu.Lock()
	c.username = username
	c.password = password
	c.mu.Unlock()

	c.authAttempts.Store(0)

	_, err = c.FetchCSRFToken()
	if err != nil {
		return NewAuthenticationError("failed to fetch CSRF token after login", err)
	}

	return nil
}

func (c *Client) InitializeSession(ctx context.Context) error {
	resp, err := c.httpClient.Get(c.baseURL)
	if err != nil {
		return fmt.Errorf("failed to initialize session: %w", err)
	}
	defer resp.Body.Close()

	// Drain body to allow connection reuse
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) Login(ctx context.Context, username, password string) error {
	if username == "" {
		return NewValidationError("username", "username cannot be empty")
	}
	if password == "" {
		return NewValidationError("password", "password cannot be empty")
	}
	if len(password) < 6 {
		return NewValidationError("password", "password must be at least 6 characters")
	}

	if err := c.InitializeSession(ctx); err != nil {
		return NewAuthenticationError("failed to initialize session", err)
	}

	loginReq := LoginRequest{
		username,
		password,
	}

	jsonData, err := json.Marshal(loginReq)
	if err != nil {
		return NewAuthenticationError("failed to prepare login request", err)
	}

	resp, err := c.DoRequest(ctx, "POST", EndpointLogin, bytes.NewReader(jsonData), false)
	if err != nil {
		return NewAuthenticationError("login request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return NewAuthenticationError("invalid username or password", nil)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		errorDetail := string(bodyBytes)
		if errorDetail == "" {
			errorDetail = "no additional details provided"
		}
		return NewAPIError(resp.StatusCode, EndpointLogin, fmt.Sprintf("login failed - %s", errorDetail), nil)
	}

	c.mu.Lock()
	c.username = username
	c.password = password
	c.mu.Unlock()

	c.authAttempts.Store(0)

	_, err = c.FetchCSRFToken()
	if err != nil {
		return NewAuthenticationError("failed to fetch CSRF token after login", err)
	}

	return nil
}

func (c *Client) GetCustomerInfo(ctx context.Context) (*CustomerInfo, error) {
	resp, err := c.DoRequest(ctx, "GET", EndpointCustomer, nil, false)
	if err != nil {
		return nil, NewAPIError(0, EndpointCustomer, "failed to get customer info", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, NewAuthenticationError("not authenticated", nil)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, NewAPIError(resp.StatusCode, EndpointCustomer, "get customer info failed", nil)
	}

	var customerInfo CustomerInfo
	if err := json.NewDecoder(resp.Body).Decode(&customerInfo); err != nil {
		return nil, NewAPIError(resp.StatusCode, EndpointCustomer, "failed to decode customer info", err)
	}

	return &customerInfo, nil
}

func (c *Client) IsAuthenticated() bool {
	cookies := c.GetCookies()
	return len(cookies) > 0
}
