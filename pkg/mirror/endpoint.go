package mirror

import (
	"context"
	"fmt"

	"github.com/go-kit/kit/endpoint"
)

type listVersionsRequest struct {
	hostname  string `json:"hostname,omitempty"`
	namespace string `json:"namespace,omitempty"`
	name      string `json:"name,omitempty"`
}

type listVersionsResponse struct {
	Versions map[string]EmptyObject `json:"versions"`
}

func listVersionsEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(listVersionsRequest)
		if !ok {
			return nil, fmt.Errorf("type assertion failed for listVersionsRequest")
		}

		versions, err := svc.ListProviderVersions(ctx, req.hostname, req.namespace, req.name)
		if err != nil {
			return nil, err
		}

		return listVersionsResponse{
			Versions: versions.Versions,
		}, nil
	}
}

//type downloadRequest struct {
//	namespace string
//	name      string
//	version   string
//	os        string
//	arch      string
//}
//
//type downloadResponse struct {
//	OS                  string      `json:"os"`
//	Arch                string      `json:"arch"`
//	Filename            string      `json:"filename"`
//	DownloadURL         string      `json:"download_url"`
//	Shasum              string      `json:"shasum"`
//	ShasumsURL          string      `json:"shasums_url"`
//	ShasumsSignatureURL string      `json:"shasums_signature_url"`
//	SigningKeys         SigningKeys `json:"signing_keys"`
//}
//
//func downloadEndpoint(svc Service) endpoint.Endpoint {
//	return func(ctx context.Context, request interface{}) (interface{}, error) {
//		req := request.(downloadRequest)
//
//		res, err := svc.GetProvider(ctx, req.namespace, req.name, req.version, req.os, req.arch)
//		if err != nil {
//			return nil, err
//		}
//
//		return downloadResponse{
//			OS:                  res.OS,
//			Arch:                res.Arch,
//			DownloadURL:         res.DownloadURL,
//			Filename:            res.Filename,
//			Shasum:              res.Shasum,
//			SigningKeys:         res.SigningKeys,
//			ShasumsURL:          res.SHASumsURL,
//			ShasumsSignatureURL: res.SHASumsSignatureURL,
//		}, nil
//	}
//}
