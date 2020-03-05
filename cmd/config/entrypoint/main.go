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

// Implements /bin/build for config/entrypoint buildpack.
package main

import (
	"os"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) error {
	if os.Getenv(env.Entrypoint) == "" {
		ctx.OptOut("%s not set", env.Entrypoint)
	}
	return nil
}

func buildFn(ctx *gcp.Context) error {
	// Use exec because lifecycle/launcher will assume the whole command is a single executable.
	ctx.AddWebProcess([]string{"/bin/bash", "-c", "exec " + os.Getenv(env.Entrypoint)})
	return nil
}