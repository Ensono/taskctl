package utils_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Ensono/taskctl/internal/utils"
	"github.com/Ensono/taskctl/variables"
)

func TestConvertEnv(t *testing.T) {
	type args struct {
		env map[string]string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{args: args{env: map[string]string{"key1": "val1"}}, want: []string{"key1=val1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.ConvertEnv(tt.args.env); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	type args struct {
		file string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{args: args{file: filepath.Join(cwd, "utils.go")}, want: true, name: "file exists"},
		{args: args{file: filepath.Join(cwd, "utils_test.go")}, want: true, name: "test file exists"},
		{args: args{file: filepath.Join(cwd, "manifesto.txt")}, want: false, name: "file does not exist"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.FileExists(tt.args.file); got != tt.want {
				t.Errorf("FileExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsExitError(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{args: args{err: &exec.ExitError{}}, want: true},
		{args: args{err: fmt.Errorf("%w", &exec.ExitError{})}, want: true},
		{args: args{err: os.ErrNotExist}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.IsExitError(tt.args.err); got != tt.want {
				t.Errorf("IsExitError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsURL(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "HTTP URL", args: args{s: "http://github.com/"}, want: true},
		{name: "HTTPS URL", args: args{s: "https://github.com/"}, want: true},
		{name: "Windows path", args: args{s: "C:\\Windows"}, want: false},
		{name: "Mailto", args: args{s: "mailto:admin@example.org"}, want: false},
		{name: "Invalid", args: args{s: "::::::::not-parsed"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.IsURL(tt.args.s); got != tt.want {
				t.Errorf("IsURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLastLine(t *testing.T) {
	type args struct {
		r io.Reader
	}
	tests := []struct {
		name  string
		args  args
		wantL string
	}{
		{args: args{r: strings.NewReader("line1\nline2")}, wantL: "line2"},
		{args: args{r: strings.NewReader("\nline1")}, wantL: "line1"},
		{args: args{r: strings.NewReader("line1\n")}, wantL: "line1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotL := utils.LastLine(tt.args.r); gotL != tt.wantL {
				t.Errorf("LastLine() = %v, want %v", gotL, tt.wantL)
			}
		})
	}
}

func TestMapKeys(t *testing.T) {
	type args struct {
		m interface{}
	}
	tests := []struct {
		name     string
		args     args
		wantKeys []string
	}{
		{args: args{m: map[string]bool{"a": true, "b": false}}, wantKeys: []string{"a", "b"}},
		{args: args{m: []string{"a", "b"}}, wantKeys: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotKeys := utils.MapKeys(tt.args.m)
			for _, v := range tt.wantKeys {
				var found bool
				for _, vv := range gotKeys {
					if v == vv {
						found = true
						break
					}
				}
				if found == false {
					t.Errorf("MapKeys() = %v, want %v", gotKeys, tt.wantKeys)
				}
			}
		})
	}
}

func TestRenderString(t *testing.T) {
	type args struct {
		tmpl      string
		variables map[string]interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{args: args{tmpl: "hello, {{ .Name }}!", variables: map[string]interface{}{"Name": "world"}}, want: "hello, world!"},
		{args: args{tmpl: "hello, {{ .Name | default \"John\" }}!", variables: map[string]interface{}{"Name": ""}}, want: "hello, John!"},
		{args: args{tmpl: "hello, {{ .Name }}!", variables: make(map[string]interface{})}, wantErr: true},
		{args: args{tmpl: "hello, {{ .Name", variables: make(map[string]interface{})}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := utils.RenderString(tt.args.tmpl, tt.args.variables)
			if (err != nil) != tt.wantErr {
				t.Errorf("RenderString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("RenderString() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMustGetwd(t *testing.T) {
	wd, _ := os.Getwd()
	if wd != utils.MustGetwd() {
		t.Error()
	}

}

func TestMustGetUserHomeDir(t *testing.T) {
	err := os.Setenv("HOME", "/test")
	if err != nil {
		t.Fatal(err)
	}
	hd := utils.MustGetUserHomeDir()
	if hd != "/test" {
		t.Error()
	}

}

// Test envfile

func TestUtils_Envfile(t *testing.T) {

	envfile := utils.NewEnvFile(func(e *utils.Envfile) {
		// e.Delay =
		e.Exclude = []string{}
		e.Include = []string{}
		// e.Path = def.Envfile.Path
		e.Modify = []utils.ModifyEnv{
			{Pattern: "", Operation: "lower"},
		}
		e.Quote = false
	})

	if err := envfile.Validate(); err == nil {
		t.Error("failed to validate")
	}

	if envfile.GeneratedDir != ".taskctl" {
		t.Error("generated dir not set correctly")
	}
}

func TestUtils_ConvertFromEnv(t *testing.T) {
	ttests := map[string]struct {
		envPairs   []string
		expectKeys []string
		expectVals []string
	}{
		"with vars with =": {
			envPairs:   []string{"=somestt", "key=val", "SOM_LONG=region=qradf,sdfsfd=84hndsfdsf;off=true"},
			expectKeys: []string{"", "key", "SOM_LONG"},
			expectVals: []string{"somestt", "val", "region=qradf,sdfsfd=84hndsfdsf;off=true"},
		},
		"with vars with newlines": {
			envPairs: []string{"=", "key=val", `SOM_LONG=rdffsdfsdfsdgbew23r44fr3435f
f5g5rtrdf;sdf094wsdf
truedsf sf sdf sdff sd
sdf sdsfdsfd fds sdf f sd
sdfds dfsg w45 546rth ghfdsr ht hrt
fdsggfd gdf`},
			expectKeys: []string{"", "key", "SOM_LONG"},
			expectVals: []string{"", "val", `rdffsdfsdfsdgbew23r44fr3435f
f5g5rtrdf;sdf094wsdf
truedsf sf sdf sdff sd
sdf sdsfdsfd fds sdf f sd
sdfds dfsg w45 546rth ghfdsr ht hrt
fdsggfd gdf`},
		},
	}

	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			got := utils.ConvertFromEnv(tt.envPairs)
			for _, k := range tt.expectKeys {
				val, ok := got[k]
				if !ok {
					t.Fatalf("got %s\nnot in wanted keys output: %v", k, tt.expectKeys)
				}
				if !slices.Contains(tt.expectVals, val) {
					t.Fatalf("got %s\nnot in wanted values output: %v", val, tt.expectVals)
				}
			}
		})
	}
}

func TestUtils_ConvertToMapOfStrings(t *testing.T) {
	t.Parallel()
	in := make(map[string]any)
	in["str"] = "string"
	in["int"] = 123
	in["bool"] = true
	got := utils.ConvertToMapOfStrings(in)

	if got["str"] != "string" {
		t.Fatal("str incorrect")
	}
	if got["int"] != "123" {
		t.Fatal("int incorrect")
	}

	if got["bool"] != "true" {
		t.Fatal("bool incorrect")
	}
}

func TestUtils_ConvertToMachineFriendly(t *testing.T) {
	ttests := map[string]struct {
		input  string
		expect string
	}{
		"with :": {
			"task:123",
			"task__e__123",
		},
		"with space": {
			"task name with space",
			"task__f__name__f__with__f__space",
		},
		"with existing _": {
			"task123:with space and _",
			"task123__e__with__f__space__f__and__f___",
		},
		"with existing _ -> pipeline pointer": {
			"pipeline1->task123:with space and _",
			"pipeline1__a__task123__e__with__f__space__f__and__f___",
		},
		"with existing _ -> pipeline pointer in the middle": {
			"pipeline1->task123:with space and _->task:567",
			"pipeline1__a__task123__e__with__f__space__f__and__f_____a__task__e__567",
		},
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			got := utils.ConvertToMachineFriendly(tt.input)
			if got != tt.expect {
				t.Errorf("got %s\nwanted %q\n", got, tt.expect)
			}
		})
	}
}

func TestUtils_TailExtractName(t *testing.T) {
	t.Parallel()
	ttests := map[string]struct {
		input  string
		expect string
	}{
		"one level": {
			"foo->1l",
			"1l",
		},
		"no level": {
			"foo",
			"foo",
		},
		"5 level": {
			"foo->one->-two->three->four->five",
			"five",
		},
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			got := utils.TailExtract(tt.input)
			if got != tt.expect {
				t.Errorf("TailExtract error: got %s, wanted %s\n", got, tt.expect)
			}
		})
	}
}

func TestUtils_CascadeName(t *testing.T) {
	t.Parallel()
	ttests := map[string]struct {
		parents []string
		curr    string
		expect  string
	}{
		"one": {
			[]string{"foo"}, "qux", "foo->qux",
		},
		"two": {
			[]string{"foo", "bar"}, "qux", "foo->bar->qux",
		},
		"5": {
			[]string{"foo", "bar", "bar1", "bar2", "bar3", "bar4"}, "qux", "foo->bar->bar1->bar2->bar3->bar4->qux",
		},
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			got := utils.CascadeName(tt.parents, tt.curr)
			if got != tt.expect {
				t.Errorf("CascadeName error: got %s, wanted %s\n", got, tt.expect)
			}
		})
	}
}

func TestUtils_DefaultTaskctlEnv(t *testing.T) {
	t.Run("path not exist - returns initialized empty vars", func(t *testing.T) {
		got := utils.DefaultTaskctlEnv()
		if got == nil {
			t.Errorf("got nil, wanted %T", &variables.Variables{})
		}
		if len(got.Map()) != 0 {
			t.Errorf("got %q, wanted %q", got.Map(), (&variables.Variables{}).Map())
		}
	})
	t.Run("path exists and is correctly ingested", func(t *testing.T) {
		err := os.WriteFile(utils.TASKCTL_ENV_FILE, []byte(`FOO=bar`), 0o777)
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(utils.TASKCTL_ENV_FILE)
		got := utils.DefaultTaskctlEnv()
		if got == nil {
			t.Errorf("got nil, wanted %T", &variables.Variables{})
		}
		if len(got.Map()) == 0 {
			t.Errorf("got %q, wanted at least one key\n", got.Map())
		}
		if !got.Has("FOO") {
			t.Errorf("got %q, wanted FOO to be in the map\n", got.Map())

		}
	})
}

func TestUtils_ReaderFromPath(t *testing.T) {
	t.Parallel()
	tf, _ := os.CreateTemp("", "test-reader-*.env")
	_, err := tf.Write([]byte(`FOO=bar`))
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tf.Name())
	ef := utils.NewEnvFile()
	ef.WithPath(tf.Name())
	r, success := utils.ReaderFromPath(ef)
	if !success {
		t.Error("reader failed to create")
	}
	if r == nil {
		t.Fatal("reader empty")
	}
	b, err := io.ReadAll(r)
	if string(b) != `FOO=bar` {
		t.Error("wrong data written")
	}
}

type tRCloser struct {
	io.Reader
}

func (trc *tRCloser) Close() error {
	return nil
}
func TestUtils_Generated(t *testing.T) {
	tf, _ := os.CreateTemp("", "test-generated-*.env")
	_, err := tf.Write([]byte(`FOO=bar`))
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tf.Name())
	ef := utils.NewEnvFile()
	ef.WithGeneratedPath(tf.Name())

	b, err := os.ReadFile(ef.GeneratedPath())
	if err != nil {
		t.Error(err)
	}
	m, _ := utils.ReadEnvFile(&tRCloser{bytes.NewReader(b)})

	if len(m) == 0 {
		t.Error("nothing written")
	}
}
func TestUtils_B62Encode_Decode(t *testing.T) {
	t.Parallel()
	ttests := map[string]struct {
		input string
	}{
		"with :": {
			"task:123",
		},
		"with space": {
			"task name with space",
		},
		"with existing _": {
			"task123:with space and _",
		},
		"with existing _ -> pipeline pointer": {
			"pipeline1->task123:with space and _",
		},
		"with existing _ -> pipeline pointer in the middle": {
			"pipeline1->task123:with space and _->task:567",
		},
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			got := utils.EncodeBase62(tt.input)
			inverseGot := utils.DecodeBase62(got)
			if inverseGot != tt.input {
				t.Errorf("got: %s\nwanted: %s", inverseGot, tt.input)
			}
		})
	}
}

func TestUtils_Binary(t *testing.T) {
	ttests := map[string]struct {
		binary        string
		args          []string
		baseArgs      []string
		shellArgs     []string
		containerArgs []string
		envFile       string
		expect        []string
		isContainer   bool
	}{
		"legacy docker with envfile specified": {
			"docker",
			[]string{"run", "--rm", "--env-file", "ignored-env.file"},
			[]string{},
			[]string{},
			[]string{},
			"envfile.env",
			[]string{"run", "--rm", "--env-file", "envfile.env"},
			false,
		},
		"legacy docker without envfile specified": {
			"docker",
			[]string{"run", "--rm"},
			[]string{},
			[]string{},
			[]string{},
			"envfile.env",
			[]string{"run", "--env-file", "envfile.env", "--rm"},
			false,
		},
		"other executable - passthrough only": {
			"someshell",
			[]string{"--out", "-c"},
			[]string{},
			[]string{},
			[]string{},
			"envfile.env",
			[]string{"--out", "-c"},
			false,
		},
		"container executable - with base args only": {
			"docker",
			[]string{"--out", "-c"},
			[]string{"run", "--rm", "other"},
			[]string{},
			[]string{},
			"envfile.env",
			[]string{"run", "--rm", "other", "envfile.env"},
			true,
		},
		"container executable - with base shell and container": {
			"docker",
			[]string{"--out", "-c"},
			[]string{"run", "--rm", "--env-file"},
			[]string{"sh", "--shellArg", "s1"},
			[]string{"--containerArg1", "c1"},
			"envfile.env",
			[]string{"run", "--rm", "--env-file", "envfile.env", "--containerArg1", "c1", "sh", "--shellArg", "s1"},
			true,
		},
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			executable := &utils.Binary{
				IsContainer: tt.isContainer,
				Args:        tt.args,
				Bin:         tt.binary,
			}

			executable.WithBaseArgs(tt.baseArgs)
			executable.WithContainerArgs(tt.containerArgs)
			executable.WithShellArgs(tt.shellArgs)

			got := executable.BuildArgsWithEnvFile(tt.envFile)
			if !slices.Equal(got, tt.expect) {
				t.Errorf("got: %v\nwanted: %v\n", got, tt.expect)
			}
		})
	}
}

// Borrow from stdlib
type alwaysError struct{}

func (alwaysError) Read(p []byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

type closerWrapper struct {
	io.Reader
}

func (closerWrapper) Close() error {
	return nil
}
func TestReadEvnFile(t *testing.T) {
	t.Parallel()
	ttests := map[string]struct {
		readCloser io.ReadCloser
		expectKeys []string
		expectVals []string
	}{
		"no unset vars": {
			closerWrapper{bytes.NewReader([]byte(`FOO=bar
BAZ=qux`))},
			[]string{"FOO", "BAZ"},
			[]string{"bar", "qux"},
		},
		"with unset vars": {
			closerWrapper{bytes.NewReader([]byte(`FOO=bar
BAZ=`))},
			[]string{"FOO", "BAZ"},
			[]string{"bar", ""},
		},
		"with vars that include =": {
			closerWrapper{bytes.NewReader([]byte(`FOO=bar
BAZ=
MULTI=somekey=someval
ANOTHER=region=123,foo=bar;colon=true|pipe=fhass`))},
			[]string{"FOO", "BAZ", "MULTI", "ANOTHER"},
			[]string{"bar", "", "somekey=someval", "region=123,foo=bar;colon=true|pipe=fhass"},
		},
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			got, err := utils.ReadEnvFile(tt.readCloser)
			if err != nil {
				t.Fatal("unable to read file for env")
			}
			for _, k := range tt.expectKeys {
				val, found := got[k]
				if !found {
					t.Errorf("key (%s) not found in map (%v)\n", k, got)
				}
				if !slices.Contains(tt.expectVals, val) {
					t.Errorf("val (%s) not found in map (%v)\n", val, got)
				}
			}
		})
	}

	t.Run("errors on bad input", func(t *testing.T) {
		if _, err := utils.ReadEnvFile(closerWrapper{alwaysError{}}); err == nil {
			t.Fatal("got nil, expected error")
		}
	})
}
