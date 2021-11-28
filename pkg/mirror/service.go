package mirror

import (
	"context"
	"errors"
	"github.com/TierMobility/boring-registry/pkg/core"
	"github.com/TierMobility/boring-registry/pkg/storage"
)

// Service implements the Provider Network MirrorProtocol.
// For more information see: https://www.terraform.io/docs/internals/provider-network-mirror-protocol.html
type Service interface {
	// ListProviderVersions determines which versions are currently available for a particular provider
	// https://www.terraform.io/docs/internals/provider-network-mirror-protocol.html#list-available-versions
	ListProviderVersions(ctx context.Context, hostname, namespace, name string) (*ProviderVersions, error)

	// ListProviderInstallation returns download URLs and associated metadata for the distribution packages for a particular version of a provider
	// https://www.terraform.io/docs/internals/provider-network-mirror-protocol.html#list-available-installation-packages
	ListProviderInstallation(ctx context.Context, hostname, namespace, name, version string) (*core.Provider, error)

	//FetchUpstreamVersions(ctx context.Context)
}

type service struct {
	storage storage.Storage
}

func (s *service) ListProviderVersions(ctx context.Context, hostname, namespace, name string) (*ProviderVersions, error) {
	if hostname == "" || namespace == "" || name == "" {
		return nil, errors.New("invalid parameters")
	}

	opts := storage.ProviderOpts{
		Hostname:  hostname,
		Namespace: namespace,
		Name:      name,
	}

	// TODO(oliviermichaelis): fetch upstream providers
	providers, err := s.storage.GetMirroredProviders(ctx, opts)
	if err != nil {
		return nil, err
	}

	return newProviderVersions(providers), nil
}

func (s *service) ListProviderInstallation(ctx context.Context, hostname, namespace, name, version string) (*core.Provider, error) {
	panic("implement listproviderinstallation")
}

//func (s *service) FetchUpstreamVersions(ctx context.Context) {
//	panic("implement fetchupstream")
//}

func (s *service) findUpstreamProviders(ctx context.Context, opts storage.ProviderOpts) {

}

// NewService returns a fully initialized Service.
func NewService(storage storage.Storage) Service {
	return &service{
		storage: storage,
	}
}

// EmptyObject exists to return an `{}` JSON object to match the protocol spec
type EmptyObject struct {}

// TODO(oliviermichaelis): could be renamed as it clashes with the other core.ProviderVersion
// ProviderVersions holds the response that is passed up to the endpoint
type ProviderVersions struct {
	Versions map[string]EmptyObject `json:"versions"`
}

func newProviderVersions(providers *[]core.Provider) *ProviderVersions {
	p := &ProviderVersions{
		Versions: make(map[string]EmptyObject),
	}

	for _, provider := range *providers {
		p.Versions[provider.Version] = EmptyObject{}
	}
	return p
}

