//go:build mage
// +build mage

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/magefile/mage/sh"
)

const (
	name           = "gobl-org"
	runImage       = "ghcr.io/gohugoio/hugo:v0.169.0"
	goblSchemaPath = "../gobl/data/schemas"
	schemaOutPath  = "./static/draft-0"

	defaultGOBLDevURL = "https://gobl.dev"
	dataOutPath       = "./data"

	schemaDraft201909         = "https://json-schema.org/draft/2019-09/schema"
	schemaDraft202012         = "https://json-schema.org/draft/2020-12/schema"
	schemaDraft202012insecure = "http://json-schema.org/draft/2020-12/schema"
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

		// Revert to older schema supported by VSCode/Monaco
		if err := replaceTextInFile(dst, schemaDraft202012, schemaDraft202012insecure); err != nil {
			return err
		}

		fmt.Printf("FILE: %v\n", dst)
	}
	return nil
}

func replaceTextInFile(file, str, replace string) error {
	input, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("reading file contents: %w", err)
	}

	output := bytes.Replace(input, []byte(str), []byte(replace), 1)

	if err := os.WriteFile(file, output, 0666); err != nil {
		return fmt.Errorf("writing new file contents: %w", err)
	}

	return nil
}

// Shell runs an interactive shell within a docker container.
func Shell() error {
	return dockerRunCmd(name+"-shell", "", "ash")
}

// Data fetches the regimes, addons, and conversion formats from the GOBL.dev
// API and writes slimmed down copies into the Hugo data directory, so the
// site's coverage lists always reflect a real GOBL release. Set GOBL_DEV_URL
// to use another instance, such as a local `gobl serve` (http://localhost:8080).
func Data() error {
	base := strings.TrimSuffix(os.Getenv("GOBL_DEV_URL"), "/")
	if base == "" {
		base = defaultGOBLDevURL
	}
	client := &http.Client{Timeout: 30 * time.Second}

	var version struct {
		Version string `json:"version"`
	}
	if err := fetchJSON(client, base+"/v0/version", &version); err != nil {
		return err
	}

	var regimes struct {
		Regimes []struct {
			Country  string            `json:"country"`
			Name     map[string]string `json:"name"`
			Currency string            `json:"currency"`
		} `json:"regimes"`
	}
	if err := fetchJSON(client, base+"/v0/regimes", &regimes); err != nil {
		return err
	}
	type regime struct {
		Country  string `json:"country"`
		Name     string `json:"name"`
		Currency string `json:"currency"`
	}
	regimesOut := make([]regime, 0, len(regimes.Regimes))
	for _, r := range regimes.Regimes {
		regimesOut = append(regimesOut, regime{
			Country:  r.Country,
			Name:     r.Name["en"],
			Currency: r.Currency,
		})
	}
	sort.Slice(regimesOut, func(i, j int) bool { return regimesOut[i].Country < regimesOut[j].Country })

	var addons struct {
		Addons []struct {
			Key      string            `json:"key"`
			Name     map[string]string `json:"name"`
			Requires []string          `json:"requires"`
		} `json:"addons"`
	}
	if err := fetchJSON(client, base+"/v0/addons", &addons); err != nil {
		return err
	}
	type addon struct {
		Key      string   `json:"key"`
		Name     string   `json:"name"`
		Region   string   `json:"region"`
		Requires []string `json:"requires,omitempty"`
	}
	addonsOut := make([]addon, 0, len(addons.Addons))
	for _, a := range addons.Addons {
		addonsOut = append(addonsOut, addon{
			Key:      a.Key,
			Name:     a.Name["en"],
			Region:   addonRegion(a.Key),
			Requires: a.Requires,
		})
	}

	var formats struct {
		Formats []struct {
			Key         string            `json:"key"`
			Name        map[string]string `json:"name"`
			Syntax      string            `json:"syntax"`
			Countries   []string          `json:"countries"`
			Import      []string          `json:"import"`
			Export      []string          `json:"export"`
			Attachments bool              `json:"attachments"`
		} `json:"formats"`
	}
	if err := fetchJSON(client, base+"/v0/formats", &formats); err != nil {
		return err
	}
	type format struct {
		Key         string   `json:"key"`
		Name        string   `json:"name"`
		Syntax      string   `json:"syntax"`
		Countries   []string `json:"countries,omitempty"`
		Documents   []string `json:"documents"`
		Import      bool     `json:"import"`
		Export      bool     `json:"export"`
		Attachments bool     `json:"attachments,omitempty"`
	}
	formatsOut := make([]format, 0, len(formats.Formats))
	for _, f := range formats.Formats {
		formatsOut = append(formatsOut, format{
			Key:         f.Key,
			Name:        f.Name["en"],
			Syntax:      f.Syntax,
			Countries:   f.Countries,
			Documents:   schemaNames(f.Import, f.Export),
			Import:      len(f.Import) > 0,
			Export:      len(f.Export) > 0,
			Attachments: f.Attachments,
		})
	}

	files := map[string]any{
		"gobl.json": map[string]any{
			"version": version.Version,
			"source":  base,
			"regimes": len(regimesOut),
			"addons":  len(addonsOut),
			"formats": len(formatsOut),
		},
		"regimes.json": regimesOut,
		"addons.json":  addonsOut,
		"formats.json": formatsOut,
	}
	if err := os.MkdirAll(dataOutPath, 0755); err != nil {
		return err
	}
	for name, v := range files {
		out, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return fmt.Errorf("encoding %s: %w", name, err)
		}
		dst := filepath.Join(dataOutPath, name)
		if err := os.WriteFile(dst, append(out, '\n'), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", dst, err)
		}
		fmt.Printf("FILE: %v\n", dst)
	}
	fmt.Printf("GOBL %s: %d regimes, %d addons, %d formats\n",
		version.Version, len(regimesOut), len(addonsOut), len(formatsOut))
	return nil
}

func fetchJSON(client *http.Client, url string, v any) error {
	res, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("fetching %s: %w", url, err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return fmt.Errorf("fetching %s: %s: %s", url, res.Status, body)
	}
	if err := json.NewDecoder(res.Body).Decode(v); err != nil {
		return fmt.Errorf("decoding %s: %w", url, err)
	}
	return nil
}

// addonRegion groups addons by the prefix of their key: a country code such
// as "es" or "mx", or a wider scope such as "eu" or "peppol".
func addonRegion(key string) string {
	region, _, _ := strings.Cut(key, "-")
	return strings.ToUpper(region)
}

// schemaNames returns the short names of the document schemas, such as
// "invoice" for "https://gobl.org/draft-0/bill/invoice", without repeats.
func schemaNames(lists ...[]string) []string {
	seen := make(map[string]bool)
	var names []string
	for _, list := range lists {
		for _, s := range list {
			n := s[strings.LastIndex(s, "/")+1:]
			if !seen[n] {
				seen[n] = true
				names = append(names, n)
			}
		}
	}
	return names
}
