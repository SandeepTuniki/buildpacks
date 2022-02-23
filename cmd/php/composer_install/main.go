// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Implements php/composer-install buildpack.
// The composer-install buildpack installs the composer dependency manager.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb"
)

const (
	composerLayer    = "composer"
	composerJSON     = "composer.json"
	composerSetup    = "composer-setup"
	composerVer      = "2.1.3"
	versionKey       = "version"
	composerSigURL   = "https://composer.github.io/installer.sig"
	composerSetupURL = "https://getcomposer.org/installer"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if !ctx.FileExists(composerJSON) {
		return gcp.OptOutFileNotFound(composerJSON), nil
	}
	return gcp.OptInFileFound(composerJSON), nil
}

func buildFn(ctx *gcp.Context) error {
	l := ctx.Layer(composerLayer, gcp.BuildLayer, gcp.CacheLayer)

	ctx.AddBOMEntry(libcnb.BOMEntry{
		Name:     composerLayer,
		Metadata: map[string]interface{}{"version": composerVer},
		Build:    true,
	})

	// Check the metadata in the cache layer to determine if we need to proceed.
	metaVersion := ctx.GetMetadata(l, versionKey)
	if composerVer == metaVersion {
		ctx.CacheHit(composerLayer)
		ctx.Logf("composer binary cache hit, skipping installation.")
		return nil
	}
	ctx.CacheMiss(composerLayer)
	ctx.ClearLayer(l)

	// download the installer
	installer, err := os.CreateTemp(l.Path, fmt.Sprintf("%s-*.php", composerSetup))
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(installer.Name())

	fetchCmd := fmt.Sprintf("curl --fail --show-error --silent --location --retry 3 --output %s %s", installer.Name(), composerSetupURL)
	ctx.Exec([]string{"bash", "-c", fetchCmd}, gcp.WithUserAttribution)

	// verify the installer hash
	expectedSHACmd := fmt.Sprintf("curl --fail --show-error --silent --location --retry 3 %s", composerSigURL)
	expectedSHA := ctx.Exec([]string{"bash", "-c", expectedSHACmd}).Stdout
	actualSHACmd := fmt.Sprintf("php -r \"echo hash_file('sha384', '%s');\"", installer.Name())
	actualSHA := ctx.Exec([]string{"bash", "-c", actualSHACmd}).Stdout
	if actualSHA != expectedSHA {
		return fmt.Errorf("invalid composer installer found at %q: checksum for composer installer, %q, does not match expected checksum of %q", composerSetupURL, actualSHA, expectedSHA)
	}

	// run the installer
	ctx.Logf("Installing Composer v%s", composerVer)
	clBin := filepath.Join(l.Path, "bin")
	ctx.MkdirAll(clBin, 0755)
	installCmd := fmt.Sprintf("php %s --install-dir %s --filename composer --version %s", installer.Name(), clBin, composerVer)
	ctx.Exec([]string{"bash", "-c", installCmd})

	ctx.SetMetadata(l, versionKey, composerVer)
	return nil
}