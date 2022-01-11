package storage

import (
	"bufio"
	"context"
	"fmt"
	"github.com/TierMobility/boring-registry/pkg/core"
	"github.com/TierMobility/boring-registry/pkg/module"
	"github.com/TierMobility/boring-registry/pkg/provider"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var (
	mirrorPrefix          = "mirror"
	customProvidersPrefix = "providers"
)

type DirectoryStorage struct {
	path string
}

func (d DirectoryStorage) GetMirroredProviders(ctx context.Context, opts ProviderOpts) (*[]core.Provider, error) {
	return d.getProviders(ctx, mirrorPrefix, opts)
}

func (d DirectoryStorage) GetCustomProviders(ctx context.Context, opts ProviderOpts) (*[]core.Provider, error) {
	return d.getProviders(ctx, customProvidersPrefix, opts)
}

func (d DirectoryStorage) GetProviderArchive(ctx context.Context, hostname string, p core.Provider) (io.Reader, error) {
	f := fmt.Sprintf("%s/%s/%s/%s/%s/%s", d.path, mirrorPrefix, hostname, p.Namespace, p.Name, p.ArchiveFileName())
	file, err := os.Open(f)
	if err != nil {
		opts := ProviderOpts{
			Hostname:  hostname,
			Namespace: p.Namespace,
			Name:      p.Name,
			Version:   p.Version,
			OS:        p.OS,
			Arch:      p.Arch,
		}
		return nil, &ErrProviderNotMirrored{
			Err:  err,
			Opts: opts,
		}
	}

	r := bufio.NewReader(file)
	return r, nil
}

func (d DirectoryStorage) GetModule(ctx context.Context, namespace, name, provider, version string) (module.Module, error) {
	panic("implement me")
}

func (d DirectoryStorage) ListModuleVersions(ctx context.Context, namespace, name, provider string) ([]module.Module, error) {
	panic("implement me")
}

func (d DirectoryStorage) UploadModule(ctx context.Context, namespace, name, provider, version string, body io.Reader) (module.Module, error) {
	panic("implement me")
}

func (d DirectoryStorage) GetProvider(ctx context.Context, namespace, name, version, os, arch string) (provider.Provider, error) {
	panic("getProvider")
}

func (d DirectoryStorage) ListProviderVersions(ctx context.Context, namespace, name string) ([]provider.ProviderVersion, error) {
	providerDir := fmt.Sprintf("%s/providers", d.path)
	var files []string
	err := filepath.WalkDir(providerDir,
		func(path string, dir fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if !dir.IsDir() {
				files = append(files, path)
			}

			return nil
		})
	if err != nil {
		return nil, err
	}

	// Shorten the provider paths for further processing into provider
	collection := provider.NewCollection()
	for _, f := range files {
		trim := strings.TrimPrefix(f, providerDir)
		p, err := provider.Parse(trim)
		if err != nil {
			return nil, err
		}

		collection.Add(p)
	}

	return collection.List(), nil
}

func (d *DirectoryStorage) StoreMirroredProvider(ctx context.Context, hostname string, p core.Provider, reader io.Reader) error {
	panic("to be done")
}

func (d *DirectoryStorage) getProviders(ctx context.Context, prefix string, opts ProviderOpts) (*[]core.Provider, error) {
	p := fmt.Sprintf("%s/%s/%s/%s/%s", d.path, prefix, opts.Hostname, opts.Namespace, opts.Name)
	rootDir := filepath.Clean(p) // remove trailing path separators
	var archives []string
	err := filepath.Walk(rootDir,
		func(path string, file fs.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// skip directories
			if file.IsDir() {
				return nil
			}

			// skip if file extension does not end with `.zip`
			if filepath.Ext(path) != core.ProviderExtension {
				return nil
			}

			archives = append(archives, path)
			return nil
		})
	if err != nil {
		return nil, &ErrProviderNotMirrored{
			Opts: opts,
			Err:  err,
		}
	}

	var providers []core.Provider
	for _, a := range archives {
		p := core.NewProviderFromArchive(a)

		// Filter out providers that don't match the queried version
		if opts.Version != "" {
			if p.Version != opts.Version {
				continue
			}
		}
		providers = append(providers, p)
	}

	return &providers, nil
}

func NewDirectoryStorage(path string) (Storage, error) {
	p, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	// Check if directory exists
	if _, err := os.Stat(p); err != nil {
		return nil, err
	}

	return &DirectoryStorage{
		path: p,
	}, nil
}
