// Implements php/flex buildpack.
// The flex buildpack installs the config needed for PHP runtime.
package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/appyaml"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/fileutil"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nginx"
)

const (
	// pid1
	appSocket = "app.sock"
	pid1Log   = "pid1.log"

	// nginx
	defaultFrontController = "index.php"
	defaultNginxBinary     = "nginx"
	defaultNginxPort       = 8080
	defaultRoot            = "/workspace"
	nginxConf              = "nginx.conf"
	nginxLog               = "nginx.log"

	// php-fpm
	defaultDynamicWorkers = false
	defaultFPMBinary      = "php-fpm"
	defaultFPMWorkers     = 2
	phpFpmPid             = "php-fpm.pid"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if env.IsFlex() {
		return gcp.OptInEnvSet(env.XGoogleTargetPlatform), nil
	}

	return gcp.OptOutEnvNotSet(env.XGoogleTargetPlatform), nil
}

func buildFn(ctx *gcp.Context) error {
	l, err := ctx.Layer("flex", gcp.CacheLayer, gcp.LaunchLayerUnlessSkipRuntimeLaunch)
	if err != nil {
		return err
	}
	runtimeConfig, err := appyaml.PhpConfiguration(ctx.ApplicationRoot())
	if err != nil {
		return err
	}

	fpmConfFile, err := writeFpmConfig(l.Path, runtimeConfig)
	if err != nil {
		return err
	}

	cmd := []string{
		"pid1",
		"--nginxBinaryPath", defaultNginxBinary,
		"--nginxErrLogFilePath", filepath.Join(l.Path, nginxLog),
		"--customAppCmd", fmt.Sprintf("%q", fmt.Sprintf("%s -R --nodaemonize --fpm-config %s", defaultFPMBinary, fpmConfFile.Name())),
		"--pid1LogFilePath", filepath.Join(l.Path, pid1Log),
		// Ideally, we should be able to use the path of the nginx layer and not hardcode it here.
		// This needs some investigation on how to pass values between build steps of buildpacks.
		"--mimeTypesPath", filepath.Join("/layers/google.utils.nginx/nginx", "conf/mime.types"),
		"--customAppSocket", filepath.Join(l.Path, appSocket),
	}
	nginxArgs, err := nginxConfCmdArgs(ctx.ApplicationRoot(), l.Path, runtimeConfig)
	if err != nil {
		return err
	}
	cmd = append(cmd, nginxArgs...)

	ctx.AddProcess(gcp.WebProcess, cmd, gcp.AsDefaultProcess())

	return nil
}

func nginxConfig(l string, runtimeConfig appyaml.RuntimeConfig) nginx.Config {
	frontController := defaultFrontController
	if runtimeConfig.FrontControllerFile != "" {
		frontController = runtimeConfig.FrontControllerFile
	}

	nginx := nginx.Config{
		Port:                  defaultNginxPort,
		FrontControllerScript: frontController,
		Root:                  filepath.Join(defaultRoot, runtimeConfig.DocumentRoot),
		AppListenAddress:      filepath.Join(l, appSocket),
	}

	return nginx
}

func writeNginxServerConfig(root, path string, runtimeConfig appyaml.RuntimeConfig) (string, error) {

	nginxConf := nginxConfig(path, runtimeConfig)
	nginxConfFile, err := nginx.WriteNginxConfigToPath(path, nginxConf)
	if err != nil {
		return "", err
	}
	return nginxConfFile.Name(), nil
}

func fpmConfig(l string) (nginx.FPMConfig, error) {
	user, err := user.Current()
	if err != nil {
		return nginx.FPMConfig{}, fmt.Errorf("getting current user: %w", err)
	}

	fpm := nginx.FPMConfig{
		PidPath:              filepath.Join(l, phpFpmPid),
		NumWorkers:           defaultFPMWorkers,
		ListenAddress:        filepath.Join(l, appSocket),
		DynamicWorkers:       defaultDynamicWorkers,
		Username:             user.Username,
		AddNoDecorateWorkers: true,
	}

	return fpm, nil
}
func writeFpmConfig(path string, runtimeConfig appyaml.RuntimeConfig) (*os.File, error) {
	conf, err := fpmConfig(path)
	if err != nil {
		return nil, err
	}
	return nginx.WriteFpmConfigToPath(path, conf)
}

func nginxConfCmdArgs(root, path string, runtimeConfig appyaml.RuntimeConfig) ([]string, error) {
	if overrideConf := runtimeConfig.NginxConfOverride; overrideConf != "" {
		dest := filepath.Join(path, overrideConf)
		err := fileutil.MaybeCopyPathContents(filepath.Join(root, overrideConf), dest, fileutil.AllPaths)
		if err != nil {
			return nil, err
		}
		return []string{"--nginxConfigPath", dest}, nil

	}
	nginxServerConfFileName, err := writeNginxServerConfig(root, path, runtimeConfig)
	if err != nil {
		return nil, err
	}
	return []string{
		"--nginxConfigPath", filepath.Join(path, nginxConf),
		"--serverConfigPath", nginxServerConfFileName,
	}, nil
}
