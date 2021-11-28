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
)

type header string

// MakeHandler returns a fully initialized http.Handler.
func MakeHandler(svc Service, auth endpoint.Middleware, options ...httptransport.ServerOption) http.Handler {
	r := mux.NewRouter().StrictSlash(true)

	r.Methods("GET").Path(`/{hostname}/{namespace}/{name}/index.json`).Handler(
		httptransport.NewServer(
			auth(listVersionsEndpoint(svc)),
			decodeListRequest,
			httptransport.EncodeJSONResponse,
			append(
				options,
				httptransport.ServerBefore(extractMuxVars(varHostname, varNamespace, varName)),
				httptransport.ServerBefore(extractHeaders("Authorization")),
			)...,
		),
	)

	//r.Methods("GET").Path(`/{namespace}/{name}/{version}/download/{os}/{arch}`).Handler(
	//	httptransport.NewServer(
	//		auth(downloadEndpoint(svc)),
	//		decodeDownloadRequest,
	//		httptransport.EncodeJSONResponse,
	//		append(
	//			options,
	//			httptransport.ServerBefore(extractMuxVars(varNamespace, varName, varOS, varArch, varVersion)),
	//			httptransport.ServerBefore(extractHeaders("Authorization")),
	//		)...,
	//	),
	//)

	return r
}

func decodeListRequest(ctx context.Context, r *http.Request) (interface{}, error) {
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
		hostname:  hostname,
		namespace: namespace,
		name:      name,
	}, nil
}

//func decodeDownloadRequest(ctx context.Context, r *http.Request) (interface{}, error) {
//	namespace, ok := ctx.Value(varNamespace).(string)
//	if !ok {
//		return nil, errors.Wrap(ErrVarMissing, "namespace")
//	}
//
//	name, ok := ctx.Value(varName).(string)
//	if !ok {
//		return nil, errors.Wrap(ErrVarMissing, "name")
//	}
//
//	version, ok := ctx.Value(varVersion).(string)
//	if !ok {
//		return nil, errors.Wrap(ErrVarMissing, "version")
//	}
//
//	os, ok := ctx.Value(varOS).(string)
//	if !ok {
//		return nil, errors.Wrap(ErrVarMissing, "os")
//	}
//
//	arch, ok := ctx.Value(varArch).(string)
//	if !ok {
//		return nil, errors.Wrap(ErrVarMissing, "arch")
//	}
//
//	return downloadRequest{
//		namespace: namespace,
//		name:      name,
//		version:   version,
//		os:        os,
//		arch:      arch,
//	}, nil
//}

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

// TODO(oliviermichaelis): this function may be a generic function
func encodeUpstreamListProviderVersionsRequest(_ context.Context, r *http.Request, request interface{}) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(request); err != nil {
		return err
	}
	r.Body = ioutil.NopCloser(&buf)
	return nil
}

type listProviderVersionsResponse struct {
	Versions []string `json:"versions,omitempty"`
}

func decodeUpstreamListProviderVersionsResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response listProviderVersionsResponse
	//var response interface{}
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}
