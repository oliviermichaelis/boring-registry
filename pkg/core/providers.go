package core

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

const (
	ProviderPrefix    = "terraform-provider-"
	ProviderExtension = ".zip"
)

// Provider copied from provider.Provider
// Provider represents Terraform provider metadata.
type Provider struct {
	Hostname            string      `json:"hostname,omitempty"`
	Namespace           string      `json:"namespace,omitempty"`
	Name                string      `json:"name,omitempty"`
	Version             string      `json:"version,omitempty"`
	OS                  string      `json:"os,omitempty"`
	Arch                string      `json:"arch,omitempty"`
	Filename            string      `json:"filename,omitempty"`
	DownloadURL         string      `json:"download_url,omitempty"`
	Shasum              string      `json:"shasum,omitempty"`
	SHASumsURL          string      `json:"shasums_url,omitempty"`
	SHASumsSignatureURL string      `json:"shasums_signature_url,omitempty"`
	SigningKeys         SigningKeys `json:"signing_keys,omitempty"`
	Platforms           []Platform  `json:"platforms,omitempty"`
}

func (p *Provider) ArchiveFileName() (string, error) {
	// Validate the Provider struct
	if p.Name == "" {
		return "", errors.New("provider Name is empty")
	} else if p.Version == "" {
		return "", errors.New("provider Version is empty")
	} else if p.OS == "" {
		return "", errors.New("provider OS is empty")
	} else if p.Arch == "" {
		return "", errors.New("provider Arch is empty")
	}

	return fmt.Sprintf("%s%s_%s_%s_%s%s", ProviderPrefix, p.Name, p.Version, p.OS, p.Arch, ProviderExtension), nil
}

func (p *Provider) ShasumFileName() (string, error) {
	if p.Name == "" {
		return "", errors.New("provider Name is empty")
	} else if p.Version == "" {
		return "", errors.New("provider Version is empty")
	}

	return fmt.Sprintf("%s%s_%s_SHA256SUMS", ProviderPrefix, p.Name, p.Version), nil
}

func (p *Provider) ShasumSignatureFileName() (string, error) {
	if p.Name == "" {
		return "", errors.New("provider Name is empty")
	} else if p.Version == "" {
		return "", errors.New("provider Version is empty")
	}

	return fmt.Sprintf("%s%s_%s_SHA256SUMS.sig", ProviderPrefix, p.Name, p.Version), nil
}

func NewProviderFromArchive(filename string) (Provider, error) {
	// Criterias for terraform archives:
	// https://www.terraform.io/docs/registry/providers/publishing.html#manually-preparing-a-release
	f := filepath.Base(filename) // This is just a precaution
	trimmed := strings.TrimPrefix(f, ProviderPrefix)
	trimmed = strings.TrimSuffix(trimmed, ProviderExtension)
	tokens := strings.Split(trimmed, "_")
	if len(tokens) != 4 {
		return Provider{}, fmt.Errorf("couldn't parse provider file name: %s", filename)
	}

	return Provider{
		Name:     tokens[0],
		Version:  tokens[1],
		OS:       tokens[2],
		Arch:     tokens[3],
		Filename: f,
	}, nil
}

// NewProviderFromObjectPath should only be used for modules and custom providers
func NewProviderFromObjectPath(v string) (Provider, error) {
	m := make(map[string]string)

	for _, part := range strings.Split(v, "/") {
		parts := strings.SplitN(part, "=", 2)
		if len(parts) != 2 {
			continue
		}

		m[parts[0]] = parts[1]
	}

	provider := Provider{
		Namespace: m["namespace"],
		Name:      m["name"],
		Version:   m["version"],
		OS:        m["os"],
		Arch:      m["arch"],
	}

	if !provider.Valid() {
		return Provider{}, fmt.Errorf("%q is not a valid path", v)
	}

	return provider, nil
}

type SigningKeys struct {
	GPGPublicKeys []GPGPublicKey `json:"gpg_public_keys,omitempty"`
}

type GPGPublicKey struct {
	KeyID      string `json:"key_id,omitempty"`
	ASCIIArmor string `json:"ascii_armor,omitempty"`
	Source     string `json:"source,omitempty"`
	SourceURL  string `json:"source_url,omitempty"`
}

// The ProviderVersion is a copy from provider.ProviderVersion
type ProviderVersion struct {
	Namespace string     `json:"namespace,omitempty"`
	Name      string     `json:"name,omitempty"`
	Version   string     `json:"version,omitempty"`
	Platforms []Platform `json:"platforms,omitempty"`
}

// Platform is a copy from provider.Platform
type Platform struct {
	OS   string `json:"os,omitempty"`
	Arch string `json:"arch,omitempty"`
}
