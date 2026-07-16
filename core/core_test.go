package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/railwayapp/railpack/core/app"
	"github.com/railwayapp/railpack/core/logger"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	v := m.Run()
	snaps.Clean(m, snaps.CleanOpts{Sort: true})
	os.Exit(v)
}

// generate snapshot plan JSON for each build example and assert against it
func TestGenerateBuildPlanForExamples(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	// Get all the examples
	examplesDir := filepath.Join(filepath.Dir(wd), "examples")
	entries, err := os.ReadDir(examplesDir)
	require.NoError(t, err)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// For each example, generate a build plan that we can snapshot test
		t.Run(entry.Name(), func(t *testing.T) {
			examplePath := filepath.Join(examplesDir, entry.Name())

			userApp, err := app.NewApp(examplePath)
			require.NoError(t, err)

			env := app.NewEnvironment(nil)
			buildResult := GenerateBuildPlan(userApp, env, &GenerateBuildPlanOptions{})

			if !buildResult.Success {
				t.Fatalf("failed to generate build plan for %s: %s", entry.Name(), buildResult.Logs)
			}

			plan := buildResult.Plan

			// Remove the generated-mise-toml asset since the versions may change between runs
			for _, step := range plan.Steps {
				for name := range step.Assets {
					if name == "generated-mise-toml" {
						step.Assets[name] = "[generated-mise-toml]"
					}
				}
			}

			snaps.MatchStandaloneJSON(t, plan)
		})
	}
}

func TestGenerateConfigFromFile_NotFound(t *testing.T) {
	// Use an existing example app directory so relative paths resolve
	appPath := "../examples/config-file"
	userApp, err := app.NewApp(appPath)
	require.NoError(t, err)

	env := app.NewEnvironment(nil)
	l := logger.NewLogger()

	options := &GenerateBuildPlanOptions{ConfigFilePath: "does-not-exist.railpack.json"}
	cfg, genErr := GenerateConfigFromFile(userApp, env, options, l)

	require.Error(t, genErr, "expected an error when explicit config file does not exist")
	require.Nil(t, cfg, "config should be nil on error")
}

func TestGenerateConfigFromFile_Malformed(t *testing.T) {
	appPath := "../examples/config-file"
	userApp, err := app.NewApp(appPath)
	require.NoError(t, err)

	env := app.NewEnvironment(nil)
	l := logger.NewLogger()

	options := &GenerateBuildPlanOptions{ConfigFilePath: "railpack.malformed.json"}
	cfg, genErr := GenerateConfigFromFile(userApp, env, options, l)

	require.Error(t, genErr, "expected an error for malformed JSON config file")
	require.Nil(t, cfg, "config should be nil on error")
}

func TestGenerateBuildPlan_ProviderMetadata(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		runtime     string
		expose      string
		legacyKey   string
		legacyValue string
	}{
		{name: "nextjs", path: "../examples/node-next", runtime: "nextjs", expose: "3000", legacyKey: "nodeRuntime", legacyValue: "next"},
		{name: "nodejs", path: "../examples/node-npm", runtime: "nodejs", legacyKey: "nodeRuntime", legacyValue: "node"},
		{name: "vite spa", path: "../examples/node-vite-react", runtime: "vite", expose: "80", legacyKey: "nodeRuntime", legacyValue: "vite"},
		{name: "django", path: "../examples/python-django", runtime: "django", expose: "8000", legacyKey: "pythonRuntime", legacyValue: "django"},
		{name: "go", path: "../examples/go-mod", runtime: "go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userApp, err := app.NewApp(tt.path)
			require.NoError(t, err)

			buildResult := GenerateBuildPlan(userApp, app.NewEnvironment(nil), &GenerateBuildPlanOptions{})
			require.True(t, buildResult.Success)
			require.Equal(t, tt.runtime, buildResult.Metadata["runtime"])
			require.Equal(t, tt.expose, buildResult.Metadata["expose"])
			if tt.legacyKey != "" {
				require.Equal(t, tt.legacyValue, buildResult.Metadata[tt.legacyKey])
			}
		})
	}
}

func TestGenerateBuildPlan_ExplicitProviderMetadata(t *testing.T) {
	userApp, err := app.NewApp("../examples/config-file")
	require.NoError(t, err)

	buildResult := GenerateBuildPlan(userApp, app.NewEnvironment(nil), &GenerateBuildPlanOptions{})
	require.True(t, buildResult.Success)
	require.Equal(t, "bun", buildResult.Metadata["runtime"])
	require.Equal(t, "node", buildResult.Metadata["providers"])
}

func TestGenerateBuildPlan_DockerignoreMetadata(t *testing.T) {
	appPath := "../examples/dockerignore"
	userApp, err := app.NewApp(appPath)
	require.NoError(t, err)

	env := app.NewEnvironment(nil)
	buildResult := GenerateBuildPlan(userApp, env, &GenerateBuildPlanOptions{})

	require.True(t, buildResult.Success)
	require.NotNil(t, buildResult.Metadata)
	require.Equal(t, "true", buildResult.Metadata["dockerIgnore"])
}
