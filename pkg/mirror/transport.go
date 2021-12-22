package mirror

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/TierMobility/boring-registry/pkg/auth"
	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

type muxVar string

const (
	varHostname  muxVar = "hostname"
	varNamespace muxVar = "namespace"
	varName      muxVar = "name"
	varVersion   muxVar = "version"
	varOS 		 muxVar = "os"
	varArchitecture muxVar = "architecture"
)

type header string

// MakeHandler returns a fully initialized http.Handler.
func MakeHandler(svc Service, auth endpoint.Middleware, options ...httptransport.ServerOption) http.Handler {
	r := mux.NewRouter().StrictSlash(true)

	r.Methods("GET").Path(`/{hostname}/{namespace}/{name}/index.json`).Handler(
		httptransport.NewServer(
			auth(listProviderVersionsEndpoint(svc)),
			decodeListVersionsRequest,
			httptransport.EncodeJSONResponse,
			append(
				options,
				httptransport.ServerBefore(extractMuxVars(varHostname, varNamespace, varName)),
				httptransport.ServerBefore(extractHeaders("Authorization")),
			)...,
		),
	)

	r.Methods("GET").Path(`/{hostname}/{namespace}/{name}/{version}.json`).Handler(
		httptransport.NewServer(
			auth(listProviderInstallationEndpoint(svc)),
			decodeListInstallationRequest,
			httptransport.EncodeJSONResponse,
			append(
				options,
				httptransport.ServerBefore(extractMuxVars(varHostname, varNamespace, varName, varVersion)),
				httptransport.ServerBefore(extractHeaders("Authorization")),
			)...,
		),
	)

	// TODO(oliviermichaelis): create handler for downloads
	// registry.terraform.io/hashicorp/random/terraform-provider-random_2.0.0_darwin_amd64.zip
	r.Methods("GET").Path(`/{hostname}/{namespace}/{name}/terraform-provider-{nameplaceholder}_{version}_{os}_{architecture}.zip`).Handler(
		httptransport.NewServer(
			auth(retrieveProviderArchiveEndpoint(svc)),
			decodeRetrieveProviderArchiveRequest,
			httptransport.EncodeJSONResponse,
			append(
				options,
				httptransport.ServerBefore(extractMuxVars(varHostname, varNamespace, varName, varVersion, varOS, varArchitecture)),
				httptransport.ServerBefore(extractHeaders("Authorization")),
			)...,
		),
	)
	return r
}

func decodeListVersionsRequest(ctx context.Context, _ *http.Request) (interface{}, error) {
	hostname, ok := ctx.Value(varHostname).(string)
	if !ok {
		return nil, errors.Wrap(ErrVarMissing, "hostname")
	}

	namespace, ok := ctx.Value(varNamespace).(string)
	if !ok {
		return nil, errors.Wrap(ErrVarMissing, "namespace")
	}

	name, ok := ctx.Value(varName).(string)
	if !ok {
		return nil, errors.Wrap(ErrVarMissing, "name")
	}

	return listVersionsRequest{
		Hostname:  hostname,
		Namespace: namespace,
		Name:      name,
	}, nil
}

func decodeListInstallationRequest(ctx context.Context, _ *http.Request) (interface{}, error) {
	hostname, ok := ctx.Value(varHostname).(string)
	if !ok {
		return nil, errors.Wrap(ErrVarMissing, "hostname")
	}

	namespace, ok := ctx.Value(varNamespace).(string)
	if !ok {
		return nil, errors.Wrap(ErrVarMissing, "namespace")
	}

	name, ok := ctx.Value(varName).(string)
	if !ok {
		return nil, errors.Wrap(ErrVarMissing, "name")
	}

	version, ok := ctx.Value(varVersion).(string)
	if !ok {
		return nil, errors.Wrap(ErrVarMissing, "version")
	}

	return listProviderInstallationRequest{
		Hostname:  hostname,
		Namespace: namespace,
		Name:      name,
		Version:   version,
	}, nil
}

func decodeRetrieveProviderArchiveRequest(ctx context.Context, _ *http.Request) (interface{}, error) {
	hostname, ok := ctx.Value(varHostname).(string)
	if !ok {
		return nil, errors.Wrap(ErrVarMissing, "hostname")
	}

	namespace, ok := ctx.Value(varNamespace).(string)
	if !ok {
		return nil, errors.Wrap(ErrVarMissing, "namespace")
	}

	name, ok := ctx.Value(varName).(string)
	if !ok {
		return nil, errors.Wrap(ErrVarMissing, "name")
	}

	version, ok := ctx.Value(varVersion).(string)
	if !ok {
		return nil, errors.Wrap(ErrVarMissing, "version")
	}

	os, ok := ctx.Value(varOS).(string)
	if !ok {
		return nil, errors.Wrap(ErrVarMissing, string(varOS))
	}

	architecture, ok := ctx.Value(varArchitecture).(string)
	if !ok {
		return nil, errors.Wrap(ErrVarMissing, string(varArchitecture))
	}

	return retrieveProviderArchiveRequest{
		Hostname:  hostname,
		Namespace: namespace,
		Name:      name,
		Version:   version,
		OS: os,
		Architecture: architecture,
	}, nil

}

// ErrorEncoder translates domain specific errors to HTTP status codes.
func ErrorEncoder(_ context.Context, err error, w http.ResponseWriter) {
	switch errors.Cause(err) {
	case ErrVarMissing:
		w.WriteHeader(http.StatusBadRequest)
	case auth.ErrInvalidKey:
		w.WriteHeader(http.StatusUnauthorized)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	_ = json.NewEncoder(w).Encode(struct {
		Error string `json:"error"`
	}{
		Error: err.Error(),
	})
}

func extractHeaders(keys ...header) httptransport.RequestFunc {
	return func(ctx context.Context, r *http.Request) context.Context {
		for _, k := range keys {
			if v := r.Header.Get(string(k)); v != "" {
				ctx = context.WithValue(ctx, k, v)
			}
		}

		return ctx
	}
}

func extractMuxVars(keys ...muxVar) httptransport.RequestFunc {
	return func(ctx context.Context, r *http.Request) context.Context {
		for _, k := range keys {
			if v, ok := mux.Vars(r)[string(k)]; ok {
				ctx = context.WithValue(ctx, k, v)
			}
		}

		return ctx
	}
}

func encodeRequest(_ context.Context, r *http.Request, request interface{}) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(request); err != nil {
		return err
	}
	r.Body = ioutil.NopCloser(&buf)
	return nil
}

type listResponse struct {
	Versions []listResponseVersion `json:"versions,omitempty"`
}

type listResponseVersion struct {
	Version   string     `json:"version,omitempty"`
	Protocols []string   `json:"protocols,omitempty"`
	Platforms []platform `json:"platforms,omitempty"`
}

type platform struct {
	OS   string `json:"os,omitempty"`
	Arch string `json:"arch,omitempty"`
}

func decodeUpstreamListProviderVersionsResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response listResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

//func decodeProviderResponse(_ context.Context, r *http.Response) (interface{}, error) {
//	var p core.Provider
//	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
//		return nil, err
//	}
//	return p, nil
//}
