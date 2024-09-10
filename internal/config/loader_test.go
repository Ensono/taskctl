package config_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Ensono/taskctl/internal/config"
)

var sampleCfg = []byte(`{"tasks": {"task1": {"command": ["true"]}}}`)

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

func TestLoader_ReadConfigFromURL(t *testing.T) {
	ttests := map[string]struct {
		contentType    string
		responseBytes  []byte
		wantError      bool
		taskCount      int
		additionalPath string
	}{
		"correct json": {
			"application/json",
			sampleCfg, false, 1, "",
		},
		"correct json from file": {
			"application/x-unknown",
			sampleCfg, false, 1, "/config.json",
		},
		"correct toml": {
			"application/toml",
			[]byte(`[tasks.task1]
command = [ true ]
`),
			false, 1, ""},
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				writer.Header().Set("Content-Type", tt.contentType)
				// fmt.Println(string(tt.responseBytes))
				_, err := writer.Write([]byte(tt.responseBytes))
				if err != nil {
					t.Errorf("failed to write bytes to response stream")
				}
			}))

			cl := config.NewConfigLoader(config.NewConfig())
			// cl.WithStrictDecoder()
			m, err := cl.ReadURL(srv.URL + tt.additionalPath)
			if err != nil && !tt.wantError {
				t.Error("got error, wanted nil")
			}
			if tt.wantError && err == nil {
				t.Error("got nil, wanted error")
			}
			tasks, ok := m["tasks"].(map[string]any)
			if !ok {
				t.Fatal("unable to cast into type")
			}
			if len(tasks) != tt.taskCount {
				t.Errorf("got %v count, wanted %v task count", len(tasks), tt.taskCount)
			}
		})
	}

	// yaml needs to be run separately "¯\_(ツ)_/¯"
	t.Run("yaml parsed correctly", func(t *testing.T) {
		t.Skip()
		srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Content-Type", "application/x-yaml")
			_, err := writer.Write([]byte(`
tasks:
  task1:
    command:
      - true
`))
			if err != nil {
				t.Errorf("failed to write bytes to response stream")
			}
		}))

		cl := config.NewConfigLoader(config.NewConfig())
		m, err := cl.ReadURL(srv.URL)
		if err != nil {
			t.Error("got error, wanted nil")
		}
		tasks, ok := m["tasks"].(map[string]any)
		if !ok {
			t.Fatal("unable to cast into type")
		}
		if len(tasks) != 1 {
			t.Errorf("got %v count, wanted %v task count", len(tasks), 1)
		}
	})
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

			bInput := []byte(`tasks:
  task1:
    command:
      - true
`)
			fmt.Println(string(bInput))
			writer.Write(bInput)
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
			writer.Write(sampleCfg)
		}
		if r == 5 {
			writer.Header().Set("Content-Type", "application/x-unknown")
			writer.Write(sampleCfg)
		}
		r++
	}))

	cl := config.NewConfigLoader(config.NewConfig())
	m, err := cl.ReadURL(srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	tasks := m["tasks"].(map[string]any)
	if len(tasks) != 1 {
		t.Error()
	}

	_, err = cl.ReadURL(srv.URL)
	if err != nil {
		t.Fatal()
	}
	yamlTasks := m["tasks"].(map[string]any)
	if len(yamlTasks) != 1 {
		t.Error()
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
	tomlTasks := m["tasks"].(map[string]any)
	if len(tomlTasks) != 1 {
		t.Error()
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
	jsonFileTasks := m["tasks"].(map[string]any)
	if len(jsonFileTasks) != 1 {
		t.Error()
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

	// loader := config.NewConfigLoader(config.NewConfig())
	// loader.WithStrictDecoder()
	// cwd, _ := os.Getwd()
	// def, err := loader.Load(filepath.Join(cwd, "testdata", "tasks.yaml"))
	// if err != nil {
	// 	t.Fatal(err)
	// }
}
