package fleetdb

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/coreos/go-oidc"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/metal-toolbox/bioscfg/internal/configuration"
	fleetdbapi "github.com/metal-toolbox/fleetdb/pkg/api/v1"
	"github.com/pkg/errors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"golang.org/x/oauth2/clientcredentials"
)

var (
	// timeout for requests made by this client.
	timeout   = 30 * time.Second
	ErrConfig = errors.New("error in fleetdb client configuration")
)

// NewFleetDBClient instantiates and returns a serverService client
func NewFleetDBClient(ctx context.Context, cfg *configuration.FleetDBOptions) (*fleetdbapi.Client, error) {
	if cfg == nil {
		return nil, errors.Wrap(ErrConfig, "configuration is nil")
	}

	if cfg.DisableOAuth {
		return newFleetDBClientWithOtel(cfg, cfg.Endpoint)
	}

	return newFleetDBClientWithOAuthOtel(ctx, cfg, cfg.Endpoint)
}

// returns a fleetdb retryable client with Otel
func newFleetDBClientWithOtel(cfg *configuration.FleetDBOptions, endpoint string) (*fleetdbapi.Client, error) {
	if cfg == nil {
		return nil, errors.Wrap(ErrConfig, "configuration is nil")
	}

	// init retryable http client
	retryableClient := retryablehttp.NewClient()

	// log hook fo 500 errors since the retryablehttp client masks them
	logHookFunc := func(l retryablehttp.Logger, r *http.Response) {
		if r.StatusCode == http.StatusInternalServerError {
			b, err := io.ReadAll(r.Body)
			if err != nil {
				slog.Warn("fleetdb query returned 500 status code; error reading body", "error", err)
				return
			}

			slog.Warn("fleetdb query returned 500 status code", "body", string(b))
		}
	}

	retryableClient.ResponseLogHook = logHookFunc

	// set retryable HTTP client to be the otel http client to collect telemetry
	retryableClient.HTTPClient = otelhttp.DefaultClient

	// requests taking longer than timeout value should be canceled.
	client := retryableClient.StandardClient()
	client.Timeout = timeout

	return fleetdbapi.NewClientWithToken(
		"dummy",
		endpoint,
		client,
	)
}

// returns a fleetdb retryable http client with Otel and Oauth wrapped in
func newFleetDBClientWithOAuthOtel(ctx context.Context, cfg *configuration.FleetDBOptions, endpoint string) (*fleetdbapi.Client, error) {
	if cfg == nil {
		return nil, errors.Wrap(ErrConfig, "configuration is nil")
	}

	slog.Info("fleetdb client ctor")

	// init retryable http client
	retryableClient := retryablehttp.NewClient()

	// set retryable HTTP client to be the otel http client to collect telemetry
	retryableClient.HTTPClient = otelhttp.DefaultClient

	// setup oidc provider
	provider, err := oidc.NewProvider(ctx, cfg.OidcIssuerEndpoint)
	if err != nil {
		return nil, err
	}

	// clientID defaults to 'bioscfg'
	clientID := "bioscfg"

	if cfg.OidcClientID != "" {
		clientID = cfg.OidcClientID
	}

	// setup oauth configuration
	oauthConfig := clientcredentials.Config{
		ClientID:       clientID,
		ClientSecret:   cfg.OidcClientSecret,
		TokenURL:       provider.Endpoint().TokenURL,
		Scopes:         cfg.OidcClientScopes,
		EndpointParams: url.Values{"audience": []string{cfg.OidcAudienceEndpoint}},
	}

	// wrap OAuth transport, cookie jar in the retryable client
	oAuthclient := oauthConfig.Client(ctx)

	retryableClient.HTTPClient.Transport = oAuthclient.Transport
	retryableClient.HTTPClient.Jar = oAuthclient.Jar

	// requests taking longer than timeout value should be canceled.
	client := retryableClient.StandardClient()
	client.Timeout = timeout

	return fleetdbapi.NewClientWithToken(
		cfg.OidcClientSecret,
		endpoint,
		client,
	)
}
