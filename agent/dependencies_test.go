package agent

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/yurika0211/luckyagent-runtime/bootstrap"
	"github.com/yurika0211/luckyagent-runtime/config"
	"github.com/yurika0211/luckyagent-runtime/mcpclient"
	"github.com/yurika0211/luckyagent-tools/memory"
)

func preparedTestComponents(t *testing.T, endpoint string) *bootstrap.Components {
	t.Helper()
	mgr, err := config.NewManagerWithDir(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := mgr.Get()
	cfg.APIKey, cfg.LlmProvider.APIKey = "test-key", "test-key"
	cfg.Proactive.Enabled = true
	if endpoint != "" {
		cfg.MCP.Servers = map[string]mcpclient.ServerConfig{"fixture": {Transport: "http", URL: endpoint, TimeoutSeconds: 1}}
	}
	if err := mgr.Replace(cfg); err != nil {
		t.Fatal(err)
	}
	components, err := bootstrap.Open(mgr, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = components.Close() })
	return components
}

func treeNames(t *testing.T, directory string) []string {
	t.Helper()
	var result []string
	err := filepath.WalkDir(directory, func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		result = append(result, path)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestInjectedConstructionDoesNotActivateServicesOrCreateFiles(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	components := preparedTestComponents(t, server.URL)
	before := treeNames(t, components.Config.HomeDir())
	instance, err := NewWithDependencies(Options{ContextCacheSize: 4}, components)
	if err != nil {
		t.Fatal(err)
	}
	defer instance.Close()
	_ = instance.StartupHooks()
	if requests.Load() != 0 || components.ProactiveRuntime.Started() || components.Support.AutonomyKit.Status().Started {
		t.Fatal("injected construction activated an application service")
	}
	if after := treeNames(t, components.Config.HomeDir()); !reflect.DeepEqual(before, after) {
		t.Fatalf("constructor created files: before=%v after=%v", before, after)
	}
	if instance.Sessions() != components.Sessions || instance.Provider() != components.Provider.Provider {
		t.Fatal("injected resources were replaced")
	}
}

type countingMemoryStore struct {
	memory.RuntimeStore
	closed int
}

func (s *countingMemoryStore) Close() error {
	s.closed++
	return s.RuntimeStore.Close()
}

func TestInjectedResourceOwnershipAndRepeatedClose(t *testing.T) {
	for _, owned := range []bool{false, true} {
		t.Run(map[bool]string{false: "borrowed", true: "owned"}[owned], func(t *testing.T) {
			components := preparedTestComponents(t, "")
			store := &countingMemoryStore{RuntimeStore: components.Memory.Store}
			components.Memory.Store = store
			instance, err := NewWithDependencies(Options{TakeOwnership: owned}, components)
			if err != nil {
				t.Fatal(err)
			}
			if err := instance.Close(); err != nil {
				t.Fatal(err)
			}
			if err := instance.Close(); err != nil {
				t.Fatal(err)
			}
			want := 0
			if owned {
				want = 1
			}
			if store.closed != want {
				t.Fatalf("close calls=%d want=%d", store.closed, want)
			}
			if err := components.Close(); err != nil || store.closed != 1 {
				t.Fatalf("owner close=%d err=%v", store.closed, err)
			}
		})
	}
}

func TestInvalidInjectionDoesNotTakeOwnership(t *testing.T) {
	components := preparedTestComponents(t, "")
	store := &countingMemoryStore{RuntimeStore: components.Memory.Store}
	components.Memory.Store = store
	components.Provider.Provider = nil
	if _, err := NewWithDependencies(Options{TakeOwnership: true}, components); err == nil || store.closed != 0 {
		t.Fatal("invalid constructor accepted dependencies or closed caller resources")
	}
}
