package config

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Common CommonConfig `yaml:"common"`
	Serve  ServeConfig  `yaml:"serve"`
	Scan   ScanConfig   `yaml:"scan"`
}

type CommonConfig struct {
	Root string `yaml:"root"`
	Path string `yaml:"path"`
}

type ServeConfig struct {
	Port  int    `yaml:"port"`
	WebUI string `yaml:"webui"`
}

type ScanConfig struct {
	Ext []string `yaml:"ext"`
}

var defaultExt = []string{"mp3", "ogg", "m4a"}

func Load(filePath string) (Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return Config{}, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse yaml: %w", err)
	}

	applyDefaults(&cfg)
	if err := validate(&cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.Common.Path == "" {
		cfg.Common.Path = "/data"
	}
	if cfg.Serve.Port == 0 {
		cfg.Serve.Port = 8000
	}
	if len(cfg.Scan.Ext) == 0 {
		cfg.Scan.Ext = append([]string(nil), defaultExt...)
	}

	cfg.Common.Path = normalizeBasePath(cfg.Common.Path)
	cfg.Scan.Ext = normalizeExt(cfg.Scan.Ext)
}

func validate(cfg *Config) error {
	if strings.TrimSpace(cfg.Common.Root) == "" {
		return errors.New("common.root is required")
	}

	absRoot, err := filepath.Abs(cfg.Common.Root)
	if err != nil {
		return fmt.Errorf("resolve common.root: %w", err)
	}

	st, err := os.Stat(absRoot)
	if err != nil {
		return fmt.Errorf("stat common.root: %w", err)
	}
	if !st.IsDir() {
		return fmt.Errorf("common.root is not a directory: %s", absRoot)
	}
	cfg.Common.Root = absRoot

	if !strings.HasPrefix(cfg.Common.Path, "/") {
		return errors.New("common.path must start with /")
	}

	if len(cfg.Scan.Ext) == 0 {
		return errors.New("scan.ext must not be empty")
	}

	if strings.TrimSpace(cfg.Serve.WebUI) != "" {
		absWebUI, err := filepath.Abs(cfg.Serve.WebUI)
		if err != nil {
			return fmt.Errorf("resolve serve.webui: %w", err)
		}

		st, err := os.Stat(absWebUI)
		if err != nil {
			return fmt.Errorf("stat serve.webui: %w", err)
		}
		if !st.IsDir() {
			return fmt.Errorf("serve.webui is not a directory: %s", absWebUI)
		}
		cfg.Serve.WebUI = absWebUI
	}

	return nil
}

func normalizeBasePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return "/data"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	p = path.Clean(p)
	if p == "." {
		return "/"
	}
	if p != "/" {
		p = strings.TrimRight(p, "/")
	}
	return p
}

func normalizeExt(exts []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(exts))
	for _, ext := range exts {
		e := strings.ToLower(strings.TrimSpace(ext))
		e = strings.TrimPrefix(e, ".")
		if e == "" {
			continue
		}
		if _, ok := seen[e]; ok {
			continue
		}
		seen[e] = struct{}{}
		out = append(out, e)
	}
	sort.Strings(out)
	return out
}
