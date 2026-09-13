package oidc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
)

const (
	maxTokenResponseBytes = 32 << 10
	maxCodeBytes          = 4096
	transactionLifetime   = 5 * time.Minute
)

// ErrLoginFailed reveals no authorization code, token, or client credential.
var ErrLoginFailed = errors.New("OIDC login failed")

// LoginClient performs one confidential Authorization Code + PKCE S256 login.
// The consuming application owns secure browser binding and local sessions.
type LoginClient struct {
	configuration *Configuration
	clientID      string
	secret        string
	redirectURI   string
	httpClient    http.Client
	now           func() time.Time
}

// NewLoginClient requires a vetted Configuration and a confidential client
// secret. The redirect URI must exactly match the issuer's client registration.
func NewLoginClient(configuration *Configuration, clientID, clientSecret, redirectURI string, client *http.Client) (*LoginClient, error) {
	if configuration == nil || configuration.Verifier == nil || !validIssuer(configuration.Verifier.issuer) ||
		clientID == "" || clientID != configuration.Verifier.clientID || clientSecret == "" ||
		!sameOriginEndpoint(configuration.Verifier.issuer, configuration.AuthorizationEndpoint) ||
		!sameOriginEndpoint(configuration.Verifier.issuer, configuration.TokenEndpoint) {
		return nil, errors.New("invalid OIDC client configuration")
	}
	redirect, err := url.Parse(redirectURI)
	if err != nil || redirect.Scheme != "https" || redirect.Host == "" || redirect.User != nil || redirect.Path == "" || redirect.RawQuery != "" || redirect.ForceQuery || redirect.Fragment != "" || redirect.Opaque != "" {
		return nil, errors.New("redirect URI must be an exact HTTPS callback without query")
	}
	if client == nil {
		client = http.DefaultClient
	}
	copyClient := *client
	copyClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	snapshot := *configuration
	return &LoginClient{configuration: &snapshot, clientID: clientID, secret: clientSecret, redirectURI: redirectURI, httpClient: copyClient, now: time.Now}, nil
}

// Transaction contains one login's state, nonce, and PKCE verifier. Store the
// pointer server-side and bind its lookup to the initiating browser. Never log
// it, send it to the browser, or use it for more than one callback.
type Transaction struct {
	owner     *LoginClient
	state     string
	nonce     string
	verifier  string
	createdAt time.Time
	used      atomic.Bool
}

// State returns the opaque callback lookup key, not the PKCE verifier.
func (transaction *Transaction) State() string {
	if transaction == nil {
		return ""
	}
	return transaction.state
}

// Begin creates a fresh one-use browser transaction and its authorization URL.
// The caller must bind the transaction to a browser session before redirecting.
func (client *LoginClient) Begin() (string, *Transaction, error) {
	if client == nil {
		return "", nil, errors.New("OIDC client is required")
	}
	if !client.configuration.validUntil.IsZero() && !client.now().Before(client.configuration.validUntil) {
		return "", nil, ErrStaleConfiguration
	}
	state, err := randomURLString()
	if err != nil {
		return "", nil, err
	}
	nonce, err := randomURLString()
	if err != nil {
		return "", nil, err
	}
	verifier, err := randomURLString()
	if err != nil {
		return "", nil, err
	}
	challenge := sha256.Sum256([]byte(verifier))
	address, _ := url.Parse(client.configuration.AuthorizationEndpoint)
	query := address.Query()
	query.Set("response_type", "code")
	query.Set("client_id", client.clientID)
	query.Set("redirect_uri", client.redirectURI)
	query.Set("scope", "openid")
	query.Set("state", state)
	query.Set("nonce", nonce)
	query.Set("code_challenge", base64.RawURLEncoding.EncodeToString(challenge[:]))
	query.Set("code_challenge_method", "S256")
	address.RawQuery = query.Encode()
	return address.String(), &Transaction{owner: client, state: state, nonce: nonce, verifier: verifier, createdAt: client.now()}, nil
}

func randomURLString() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// Complete consumes a transaction, exchanges one authorization code, and
// returns only the verified external identity. Even a failed callback consumes
// the transaction. The caller must separately enforce browser binding and
// explicitly link the resulting identity to a local account.
func (client *LoginClient) Complete(ctx context.Context, transaction *Transaction, callback url.Values) (Identity, error) {
	if client == nil || ctx == nil || transaction == nil || transaction.owner != client || transaction.used.Swap(true) ||
		client.now().Sub(transaction.createdAt) > transactionLifetime || transaction.createdAt.After(client.now().Add(clockSkew)) {
		return Identity{}, ErrLoginFailed
	}
	if len(callback["state"]) != 1 || len(callback["code"]) != 1 || len(callback["error"]) != 0 ||
		callback.Get("code") == "" || len(callback.Get("code")) > maxCodeBytes ||
		subtle.ConstantTimeCompare([]byte(callback.Get("state")), []byte(transaction.state)) != 1 {
		return Identity{}, ErrLoginFailed
	}
	ctx, cancel := context.WithTimeout(ctx, discoveryTimeout)
	defer cancel()
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {callback.Get("code")},
		"redirect_uri":  {client.redirectURI},
		"code_verifier": {transaction.verifier},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.configuration.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return Identity{}, ErrLoginFailed
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")
	request.SetBasicAuth(client.clientID, client.secret)
	response, err := client.httpClient.Do(request)
	if err != nil {
		return Identity{}, ErrLoginFailed
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Identity{}, ErrLoginFailed
	}
	contents, err := io.ReadAll(io.LimitReader(response.Body, maxTokenResponseBytes+1))
	if err != nil || len(contents) > maxTokenResponseBytes {
		return Identity{}, ErrLoginFailed
	}
	var token struct {
		IDToken string `json:"id_token"`
	}
	if err := json.Unmarshal(contents, &token); err != nil {
		return Identity{}, ErrLoginFailed
	}
	identity, err := client.configuration.Verifier.VerifyIDToken(token.IDToken, transaction.nonce)
	if err != nil {
		return Identity{}, ErrLoginFailed
	}
	return identity, nil
}
