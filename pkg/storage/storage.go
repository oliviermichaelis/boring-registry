package storage

import (
	"fmt"
	"path"
)

// Storage TODO(oliviermichaelis): refactor everything
//type Storage interface {
//GetModule(ctx context.Context, namespace, name, provider, version string) (module.Module, error)
//ListModuleVersions(ctx context.Context, namespace, name, provider string) ([]module.Module, error)
//UploadModule(ctx context.Context, namespace, name, provider, version string, body io.Reader) (module.Module, error)

//ListProviderVersions(ctx context.Context, namespace, name string) ([]provider.ProviderVersion, error)
//GetProvider(ctx context.Context, namespace, name, version, os, arch string) (provider.Provider, error)
//}

func storagePrefix(prefix, namespace, name string) string {
	return path.Join(
		prefix,
		fmt.Sprintf("namespace=%s", namespace),
		fmt.Sprintf("name=%s", name),
	)
}

func storagePath(prefix, namespace, name, version, os, arch string) string {
	return path.Join(
		prefix,
		fmt.Sprintf("namespace=%s", namespace),
		fmt.Sprintf("name=%s", name),
		fmt.Sprintf("version=%s", version),
		fmt.Sprintf("os=%s", os),
		fmt.Sprintf("arch=%s", arch),
		fmt.Sprintf("terraform-provider-%s_%s_%s_%s.zip", name, version, os, arch),
	)
}

func shasumsPath(prefix, namespace, name, version string) string {
	return path.Join(
		prefix,
		fmt.Sprintf("namespace=%s", namespace),
		fmt.Sprintf("name=%s", name),
		fmt.Sprintf("version=%s", version),
		fmt.Sprintf("terraform-provider-%s_%s_SHA256SUMS", name, version),
	)
}

func signingKeysPath(prefix, namespace string) string {
	return path.Join(
		prefix,
		fmt.Sprintf("namespace=%s", namespace),
		"signing-keys.json",
	)
}
