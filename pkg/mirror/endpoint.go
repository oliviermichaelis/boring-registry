package mirror

import (
	"context"
	"fmt"
	"github.com/TierMobility/boring-registry/pkg/core"
	"github.com/go-kit/kit/endpoint"
)

type listVersionsRequest struct {
	Hostname  string `json:"hostname,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
}

type listVersionsResponse struct {
	Versions map[string]EmptyObject `json:"versions"`
}

func listProviderVersionsEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(listVersionsRequest)
		if !ok {
			return nil, fmt.Errorf("type assertion failed for listVersionsRequest")
		}

		versions, err := svc.ListProviderVersions(ctx, req.Hostname, req.Namespace, req.Name)
		if err != nil {
			return nil, err
		}

		return listVersionsResponse{
			Versions: versions.Versions,
		}, nil
	}
}

type listProviderInstallationRequest struct {
	Hostname  string `json:"hostname,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
	Version   string `json:"version,omitempty"`
}

func listProviderInstallationEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(listProviderInstallationRequest)
		if !ok {
			return nil, fmt.Errorf("type assertion failed for listProviderInstallationRequest")
		}

		archives, err := svc.ListProviderInstallation(ctx, req.Hostname, req.Namespace, req.Name, req.Version)
		if err != nil {
			return nil, err
		}

		return archives, nil
	}
}

type retrieveProviderArchiveRequest struct {
	Hostname  string `json:"hostname,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
	Version   string `json:"version,omitempty"`
	OS 		string `json:"os,omitempty"`
	Architecture string `json:"architecture,omitempty"`
}

func retrieveProviderArchiveEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(retrieveProviderArchiveRequest)
		if !ok {
			return nil, fmt.Errorf("type assertion failed for retrieveProviderArchiveRequest")
		}

		provider := core.Provider{
			Namespace: req.Namespace,
			Name:     req.Name ,
			Version:   req.Version,
			OS:        req.OS,
			Arch:      req.Architecture,
		}

		// TODO(oliviermichaelis) should this be an io.Reader?
		archive, err := svc.RetrieveProviderArchive(ctx, req.Hostname, provider)
		if err != nil {
			return nil, err
		}

		return archive, nil
	}
}
