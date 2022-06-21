//go:build mage
// +build mage

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/sh"
)

const (
	name           = "gobl-org"
	runImage       = "klakegg/hugo:0.80.0-ext-alpine"
	goblSchemaPath = "../gobl/build/schemas"
	schemaOutPath  = "./static/draft-0"
)

func Start() error {
	return dockerRunCmd(name, "1313", "server", "--baseURL=https://gobl-org.invopop.dev/", "--liveReloadPort=443", "--appendPort=false", "-v", "--noHTTPCache", "--disableFastRender")
}

func dockerRunCmd(name, port string, cmd ...string) error {
	args := []string{
		"run",
		"--rm",
		"--name", name,
		"--network", "invopop-local",
		"-v", "$PWD:/src",
		"-w", "/src",
		"-it", // interactive
	}
	if port != "" {
		args = append(args,
			"--label", "traefik.enable=true",
			"--label", "traefik.http.routers."+name+".rule=Host(`"+name+".invopop.dev`)",
			"--label", "traefik.http.routers."+name+".tls=true",
			"--expose", port,
		)
	}
	args = append(args, runImage)
	args = append(args, cmd...)
	return sh.RunV("docker", args...)
}

func Schema() error {
	var files []string
	err := filepath.Walk(goblSchemaPath, func(path string, info os.FileInfo, err error) error {
		if filepath.Ext(path) == ".json" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	for _, f := range files {
		dst := strings.TrimSuffix(f, filepath.Ext(f))
		dst = strings.Replace(dst, goblSchemaPath, schemaOutPath, 1)
		os.Mkdir(filepath.Dir(dst), 0755)
		if err := sh.Copy(dst, f); err != nil {
			return err
		}
		fmt.Printf("FILE: %v\n", dst)
	}
	return nil
}

// Shell runs an interactive shell within a docker container.
func Shell() error {
	return dockerRunCmd(name+"-shell", "", "ash")
}
