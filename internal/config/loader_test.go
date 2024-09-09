package config_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Ensono/taskctl/internal/config"
)

const sampleCfg = "{\"tasks\": {\"task1\": {\"command\": [\"true\"]}}}"

func TestLoader_Load(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	cl := config.NewConfigLoader(config.NewConfig())
	cfg, err := cl.Load(filepath.Join(cwd, "testdata", "test.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Tasks["task1"] == nil || cfg.Tasks["task1"].Commands[0] != "echo true" {
		t.Error("yaml parsing failed")
	}

	if cfg.Contexts["local_wth_quote"].Quote != "'" {
		t.Error("context's quote parsing failed")
	}

	cl = config.NewConfigLoader(config.NewConfig())
	cl.WithDir(filepath.Join(cwd, "testdata"))
	cfg, err = cl.Load("test.toml")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Tasks["task1"] == nil || cfg.Tasks["task1"].Commands[0] != "echo true" {
		t.Error("yaml parsing failed")
	}

	cl = config.NewConfigLoader(config.NewConfig())
	cl.WithDir(filepath.Join(cwd, "testdata", "nested"))
	cfg, err = cl.Load("")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.Tasks["test-task"]; !ok {
		t.Error("yaml parsing failed")
	}

	_, err = cl.LoadGlobalConfig()
	if err != nil {
		t.Fatal()
	}
}

func TestLoader_resolveDefaultConfigFile(t *testing.T) {
	cl := config.NewConfigLoader(config.NewConfig())
	cl.WithDir(filepath.Join(cl.Dir(), "testdata"))

	file, err := cl.ResolveDefaultConfigFile()
	if err != nil {
		t.Fatal(err)
	}

	if filepath.Base(file) != "tasks.yaml" {
		t.Error()
	}

	cl.WithDir("/")
	file, err = cl.ResolveDefaultConfigFile()
	if err == nil || file != "" {
		t.Error()
	}
}

func TestLoader_LoadDirImport(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	cl := config.NewConfigLoader(config.NewConfig())
	conf, err := cl.Load(filepath.Join(cwd, "testdata", "dir-dep-import.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	if len(conf.Tasks) != 5 {
		t.Error()
	}
}

func TestLoader_readURL(t *testing.T) {
	r := 0
	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "")
		if r == 0 {
			writer.Header().Set("Content-Type", "application/json")
			writer.Write([]byte(sampleCfg))
		}
		if r == 1 {
			writer.Header().Set("Content-Type", "application/x-yaml")
			writer.Write([]byte(`tasks:
  task1:
    command: 
      - true
`))
		}
		if r == 2 {
			writer.WriteHeader(500)
		}
		if r == 3 {
			writer.Header().Set("Content-Type", "application/toml")
			writer.Write([]byte(`[tasks.task1]
command = [ true ]
`))
		}
		if r == 4 {
			writer.Header().Set("Content-Type", "")
			writer.Write([]byte(sampleCfg))
		}
		if r == 5 {
			writer.Header().Set("Content-Type", "application/x-unknown")
			writer.Write([]byte(sampleCfg))
		}
		r++
	}))

	cl := config.NewConfigLoader(config.NewConfig())
	m, err := cl.ReadURL(srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	tasks := m["tasks"].(map[string]interface{})
	if len(tasks) != 1 {
		t.Error()
	}

	_, err = cl.ReadURL(srv.URL)
	if err != nil {
		t.Fatal()
	}

	_, err = cl.ReadURL(srv.URL)
	if err == nil {
		t.Fatal()
	}
	// toml test
	_, err = cl.ReadURL(srv.URL)
	if err != nil {
		t.Fatal()
	}
	// undefined test
	_, err = cl.ReadURL(srv.URL)
	if err == nil {
		t.Fatal("got nil, wanted err")
	}

	// unknown content-type
	//
	_, err = cl.ReadURL(srv.URL + "/config.json")
	if err != nil {
		t.Fatal()
	}
}

func TestLoader_LoadGlobalConfig(t *testing.T) {
	h := os.TempDir()
	originalHomeNix, originalHomeWin := os.Getenv("HOME"), os.Getenv("USERPROFILE")
	os.Setenv("HOME", h)
	// windows...
	os.Setenv("USERPROFILE", h)

	defer func() {
		_ = os.RemoveAll(filepath.Join(h, ".taskctl"))
		os.Setenv("HOME", originalHomeNix)
		// windows...
		os.Setenv("USERPROFILE", originalHomeWin)
	}()

	err := os.Mkdir(filepath.Join(h, ".taskctl"), 0744)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(filepath.Join(h, ".taskctl", "config.yaml"), []byte(sampleCfg), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cl := config.NewConfigLoader(config.NewConfig())
	// cl.homeDir = h
	cfg, err := cl.LoadGlobalConfig()
	if err != nil {
		t.Fatal()
	}

	if len(cfg.Tasks) == 0 {
		t.Error()
	}
}

func TestLoader_merging_env_with_user_supplied_envVars(t *testing.T) {
	// ttests := map[string]struct {
	// 	objType any
	// }{
	// 	"test1": {
	// 		objType: nil,
	// 	},
	// }
	// for name, tt := range ttests {
	// 	t.Run(name, func(t *testing.T) {

	// 	})
	// }
}
