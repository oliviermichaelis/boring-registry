package mirror

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"golang.org/x/sync/errgroup"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/TierMobility/boring-registry/pkg/core"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// Middleware is a Service middleware.
type Middleware func(Service) Service

type loggingMiddleware struct {
	next   Service
	logger log.Logger
}

func (mw loggingMiddleware) ListProviderVersions(ctx context.Context, hostname, namespace, name string) (providerVersions *ProviderVersions, err error) {
	defer func(begin time.Time) {
		logger := level.Info(mw.logger)
		if err != nil {
			logger = level.Error(mw.logger)
		}

		_ = logger.Log(
			"op", "ListProviderVersions",
			"hostname", hostname,
			"namespace", namespace,
			"name", name,
			"took", time.Since(begin),
			"err", err,
		)

	}(time.Now())

	return mw.next.ListProviderVersions(ctx, hostname, namespace, name)
}

func (mw loggingMiddleware) ListProviderInstallation(ctx context.Context, hostname, namespace, name, version string) (provider *core.Provider, err error) {
	defer func(begin time.Time) {
		logger := level.Info(mw.logger)
		if err != nil {
			logger = level.Error(mw.logger)
		}

		_ = logger.Log(
			"op", "GetMirroredProviders",
			"hostname", hostname,
			"namespace", namespace,
			"name", name,
			"took", time.Since(begin),
			"err", err,
		)

	}(time.Now())

	return mw.next.ListProviderInstallation(ctx, hostname, namespace, name, version)
}

// LoggingMiddleware is a logging Service middleware.
func LoggingMiddleware(logger log.Logger) Middleware {
	return func(next Service) Service {
		return &loggingMiddleware{
			logger: logger,
			next:   next,
		}
	}
}

// TODO(oliviermichaelis): split out into a separate file
type proxyRegistry struct {
	next                 Service                      // serve most requests via this service
	listProviderVersions map[string]endpoint.Endpoint // except for Service.ListProviderVersions
}

// ListProviderVersions returns the available versions fetched from the upstream registry, as well as from the pull-through cache
func (p *proxyRegistry) ListProviderVersions(ctx context.Context, hostname, namespace, name string) (*ProviderVersions, error) {
	g, _ := errgroup.WithContext(ctx)

	// Get providers from the upstream registry if it is reachable
	upstreamVersions := &ProviderVersions{}
	g.Go(func() error {
		// Check if there is already an endpoint.Endpoint for the upstream registry, namespace and name
		id := fmt.Sprintf("%s/%s/%s", hostname, namespace, name)
		if _, ok := p.listProviderVersions[id]; !ok {
			upstreamUrl, err := url.Parse(fmt.Sprintf("https://%s/v1/providers/%s/%s/versions", hostname, namespace, name))
			if err != nil {
				return err
			}

			p.listProviderVersions[id] = httptransport.NewClient(http.MethodGet, upstreamUrl, encodeUpstreamListProviderVersionsRequest, decodeUpstreamListProviderVersionsResponse).Endpoint()
		}

		// TODO(oliviermichaelis): we pass the same context, even though the deadline should be slightly earlier
		e, exists := p.listProviderVersions[id]
		if !exists {
			return fmt.Errorf("the endpoint with id %s doesn't exist", id)
		}


		response, err := e(ctx, listVersionsRequest{}) // TODO(oliviermichaelis): The object is just a placeholder for now, as we don't have a payload
		if err != nil {
			return err
		}
		resp, ok := response.(listResponse)
		if !ok {
			return fmt.Errorf("failed type assertion for %v", response)
		}
		// Convert the response to the desired data format
		upstreamVersions = &ProviderVersions{Versions: make(map[string]EmptyObject)}
		for _, version := range resp.Versions {
			upstreamVersions.Versions[version.Version] = EmptyObject{}
		}
		return nil
	})

	// Get providers from the pull-through cache
	cachedVersions := &ProviderVersions{}
	g.Go(func() (err error) {
		cachedVersions, err = p.next.ListProviderVersions(ctx, hostname, namespace, name)
		if err != nil {
			return err
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		// Check for net.OpError, as that is an indication for network errors. There is likely a better solution to the problem
		var e *net.OpError
		if !errors.As(err, &e) {
			return nil, err
		}
		// TODO(oliviermichaelis): log this properly and expose a metric
		fmt.Println(fmt.Errorf("couldn't reach upstream registry: %v", err))
	}

	// Merge both maps together
	for k, v := range upstreamVersions.Versions {
		cachedVersions.Versions[k] = v
	}

	return cachedVersions, nil
}

func (p *proxyRegistry) ListProviderInstallation(ctx context.Context, hostname, namespace, name, version string) (*core.Provider, error) {
	return p.next.ListProviderInstallation(ctx, hostname, namespace, name, version)
}

func ProxyingMiddleware() Middleware {
	return func(next Service) Service {
		return &proxyRegistry{
			next:   next,
			listProviderVersions: make(map[string]endpoint.Endpoint),
		}
	}
}
