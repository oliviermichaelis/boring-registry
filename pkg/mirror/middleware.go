package mirror

import (
	"context"
	"errors"
	"fmt"
	"github.com/TierMobility/boring-registry/pkg/core"
	"github.com/TierMobility/boring-registry/pkg/storage"
	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"golang.org/x/sync/errgroup"
	"net"
	"net/http"
	"net/url"
	"time"

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

func (mw loggingMiddleware) ListProviderInstallation(ctx context.Context, hostname, namespace, name, version string) (provider *Archives, err error) {
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

func (mw loggingMiddleware) RetrieveProviderArchive(ctx context.Context, hostname string, provider core.Provider) (_ []byte, err error) {
	defer func(begin time.Time) {
		logger := level.Info(mw.logger)
		if err != nil {
			logger = level.Error(mw.logger)
		}

		_ = logger.Log(
			"op", "GetMirroredProviders",
			"hostname", hostname,
			"namespace", provider.Namespace,
			"name", provider.Name,
			"version", provider.Version,
			"os", provider.OS,
			"arch", provider.Arch,
			"took", time.Since(begin),
			"err", err,
		)

	}(time.Now())

	return mw.next.RetrieveProviderArchive(ctx, hostname, provider)
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
	next                     Service                      // serve most requests via this service
	listProviderVersions     map[string]endpoint.Endpoint // except for Service.ListProviderVersions
	listProviderInstallation map[string]endpoint.Endpoint
}

// ListProviderVersions returns the available versions fetched from the upstream registry, as well as from the pull-through cache
func (p *proxyRegistry) ListProviderVersions(ctx context.Context, hostname, namespace, name string) (*ProviderVersions, error) {
	// TODO(oliviermichaelis): the errgroup might not be right, as we want to return both errors in case upstream is down and cache does not contain the provider
	g, _ := errgroup.WithContext(ctx)

	// Get providers from the upstream registry if it is reachable
	upstreamVersions := &ProviderVersions{}
	g.Go(func() error {
		versions, err := p.getUpstreamProviders(ctx, hostname, namespace, name)
		if err != nil {
			return err
		}

		// Convert the response to the desired data format
		upstreamVersions = &ProviderVersions{Versions: make(map[string]EmptyObject)}
		for _, version := range versions {
			upstreamVersions.Versions[version.Version] = EmptyObject{}
		}
		return nil
	})

	// Get provider versions from the pull-through cache
	cachedVersions := &ProviderVersions{Versions: make(map[string]EmptyObject)}
	// TODO(oliviermichaelis): check for concurrency problems
	g.Go(func() (err error) {
		providerVersions, err := p.next.ListProviderVersions(ctx, hostname, namespace, name)
		if err != nil {
			return err
		}

		// We can only assign cachedVersions once we know that err is non-nil. Otherwise the map is not initialized
		cachedVersions = providerVersions
		return nil
	})

	if err := g.Wait(); err != nil {
		var opError *net.OpError
		var errProviderNotMirrored *storage.ErrProviderNotMirrored
		// Check for net.OpError, as that is an indication for network errors. There is likely a better solution to the problem
		if errors.As(err, &opError) {
			fmt.Println(fmt.Errorf("couldn't reach upstream registry: %v", err)) // TODO(oliviermichaelis): use proper logging
		} else if errors.As(err, &errProviderNotMirrored) {
			fmt.Println(err.Error())
		}
	}

	// Merge both maps together
	for k, v := range upstreamVersions.Versions {
		cachedVersions.Versions[k] = v
	}

	return cachedVersions, nil
}

func (p *proxyRegistry) ListProviderInstallation(ctx context.Context, hostname, namespace, name, version string) (*Archives, error) {
	g, _ := errgroup.WithContext(ctx)

	// Get archives from the pull-through cache
	cachedArchives := &Archives{}
	g.Go(func() (err error) {
		cachedArchives, err = p.next.ListProviderInstallation(ctx, hostname, namespace, name, version)
		if err != nil {
			return err
		}
		return nil
	})

	upstreamArchives := &Archives{
		Archives: make(map[string]Archive),
	}
	g.Go(func() error {
		// TODO(oliviermichaelis): pass short-living context here
		versions, err := p.getUpstreamProviders(ctx, hostname, namespace, name)
		if err != nil {
			return err
		}

		for _, v := range versions {
			if v.Version == version {
				for _, platform := range v.Platforms {
					provider := core.Provider{
						Namespace: namespace,
						Name:      name,
						Version:   version,
						OS:        platform.OS,
						Arch:      platform.Arch,
					}
					key := fmt.Sprintf("%s_%s", platform.OS, platform.Arch)
					upstreamArchives.Archives[key] = Archive{
						Url:    provider.ArchiveFileName(),
						Hashes: nil, // TODO(oliviermichaelis): hash is missing
					}
				}
			}
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

	// Warning, this is overwriting locally cached archives. In case a version was deleted from the upstream, we can't serve it locally anymore
	// This could be solved with a more complex merge
	for k, v := range upstreamArchives.Archives {
		cachedArchives.Archives[k] = v
	}

	return cachedArchives, nil
}

func (p *proxyRegistry) getUpstreamProviders(ctx context.Context, hostname, namespace, name string) ([]listResponseVersion, error) {
	// Check if there is already an endpoint.Endpoint for the upstream registry, namespace and name
	id := fmt.Sprintf("%s/%s/%s", hostname, namespace, name)
	if _, ok := p.listProviderVersions[id]; !ok {
		upstreamUrl, err := url.Parse(fmt.Sprintf("https://%s/v1/providers/%s/%s/versions", hostname, namespace, name))
		if err != nil {
			return nil, err
		}

		p.listProviderVersions[id] = httptransport.NewClient(http.MethodGet, upstreamUrl, encodeRequest, decodeUpstreamListProviderVersionsResponse).Endpoint()
	}

	// TODO(oliviermichaelis): we pass the same context, even though the deadline should be slightly earlier
	e, exists := p.listProviderVersions[id]
	if !exists {
		return nil, fmt.Errorf("the endpoint with id %s doesn't exist", id)
	}

	response, err := e(ctx, listVersionsRequest{}) // TODO(oliviermichaelis): The object is just a placeholder for now, as we don't have a payload
	if err != nil {
		return nil, err
	}
	resp, ok := response.(listResponse)
	if !ok {
		return nil, fmt.Errorf("failed type assertion for %v", response)
	}
	return resp.Versions, nil
}

func (p *proxyRegistry) RetrieveProviderArchive(ctx context.Context, hostname string, provider core.Provider) ([]byte, error) {
	// TODO(oliviermichaelis): get provider from upstream if not available locally
	return p.next.RetrieveProviderArchive(ctx, hostname, provider)
}

func ProxyingMiddleware() Middleware {
	return func(next Service) Service {
		return &proxyRegistry{
			next:                     next,
			listProviderVersions:     make(map[string]endpoint.Endpoint),
			listProviderInstallation: make(map[string]endpoint.Endpoint),
		}
	}
}
