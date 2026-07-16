package node

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/railwayapp/railpack/core/app"
	"github.com/railwayapp/railpack/core/generate"
	"github.com/railwayapp/railpack/core/plan"
	testingUtils "github.com/railwayapp/railpack/core/testing"
	"github.com/stretchr/testify/require"
)

func TestNode(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		detected       bool
		packageManager PackageManager
		nodeVersion    string
		pnpmVersion    string
		envVars        map[string]string
	}{
		{
			name:           "npm",
			path:           "../../../examples/node-npm",
			detected:       true,
			packageManager: PackageManagerNpm,
			nodeVersion:    "23.5.0",
		},
		{
			name:           "bun",
			path:           "../../../examples/node-bun",
			detected:       true,
			packageManager: PackageManagerBun,
		},
		{
			name:           "pnpm",
			path:           "../../../examples/node-corepack",
			detected:       true,
			packageManager: PackageManagerPnpm,
			nodeVersion:    "20",
			pnpmVersion:    "10.4.1",
		},
		{
			name:           "pnpm",
			path:           "../../../examples/node-pnpm-workspaces",
			detected:       true,
			packageManager: PackageManagerPnpm,
			nodeVersion:    "22.2.0",
		},
		{
			name:           "pnpm from mise.toml",
			path:           "../../../examples/node-latest-pnpm-mise-native-deps",
			detected:       true,
			packageManager: PackageManagerPnpm,
			nodeVersion:    "latest",
			pnpmVersion:    "latest",
		},
		{
			name:           "pnpm",
			path:           "../../../examples/node-astro",
			detected:       true,
			packageManager: PackageManagerNpm,
		},
		{
			name:           "railpack node version overrides engines",
			path:           "../../../examples/node-version-precedence",
			detected:       true,
			packageManager: PackageManagerNpm,
			nodeVersion:    "22",
			envVars:        map[string]string{"RAILPACK_NODE_VERSION": "22"},
		},
		{
			name:     "golang",
			path:     "../../../examples/go-mod",
			detected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testingUtils.CreateGenerateContext(t, tt.path)
			if tt.envVars != nil {
				envVars := tt.envVars
				ctx.Env = app.NewEnvironment(&envVars)
			}
			provider := NodeProvider{}
			detected, err := provider.Detect(ctx)
			require.NoError(t, err)
			require.Equal(t, tt.detected, detected)

			if detected {
				err = provider.Initialize(ctx)
				require.NoError(t, err)

				packageManager := provider.getPackageManager(ctx.App)
				require.Equal(t, tt.packageManager, packageManager)

				err = provider.Plan(ctx)
				require.NoError(t, err)

				if tt.nodeVersion != "" {
					nodeVersion := ctx.Resolver.Get("node")
					if tt.nodeVersion == "latest" {
						require.NotEmpty(t, nodeVersion.Version)
					} else {
						require.Equal(t, tt.nodeVersion, nodeVersion.Version)
					}
				}

				if tt.pnpmVersion != "" {
					pnpmVersion := ctx.Resolver.Get("pnpm")
					require.NotNil(t, pnpmVersion)

					if tt.pnpmVersion == "latest" {
						require.NotEmpty(t, pnpmVersion.Version)
					} else {
						require.Equal(t, tt.pnpmVersion, pnpmVersion.Version)
					}
				}
			}
		})
	}
}

func TestNextStandalonePlan(t *testing.T) {
	tests := []struct {
		name             string
		appPath          string
		env              map[string]string
		configure        func(*generate.GenerateContext)
		wantStart        string
		wantDeployInput  string
		wantStandalone   bool
		wantBuildCommand string
	}{
		{
			name:             "root Next app",
			appPath:          "../../../examples/node-next",
			wantStart:        "node /railpack/next-standalone/server.js",
			wantDeployInput:  NextStandaloneDeployRoot,
			wantStandalone:   true,
			wantBuildCommand: "npm run build",
		},
		{
			name:             "workspace Next app",
			appPath:          "../../../examples/node-turborepo",
			wantStart:        "node /railpack/next-standalone/server.js",
			wantDeployInput:  NextStandaloneDeployRoot,
			wantStandalone:   true,
			wantBuildCommand: "npm run build",
		},
		{
			name:            "Next export remains SPA",
			appPath:         "../../../examples/node-next-spa",
			wantDeployInput: "out",
		},
		{
			name:            "forced SPA output remains SPA",
			appPath:         "../../../examples/node-next",
			env:             map[string]string{"RAILPACK_SPA_OUTPUT_DIR": "custom-out"},
			wantDeployInput: "custom-out",
		},
		{
			name:    "custom start command keeps full Node deploy",
			appPath: "../../../examples/node-next",
			configure: func(ctx *generate.GenerateContext) {
				ctx.Config.Deploy.StartCmd = "npm run custom-start"
			},
			wantStart:        "npm run custom-start",
			wantDeployInput:  ".",
			wantBuildCommand: "npm run build",
		},
		{
			name:    "custom build command keeps standalone setup",
			appPath: "../../../examples/node-next",
			configure: func(ctx *generate.GenerateContext) {
				build := ctx.Config.GetOrCreateStep("build")
				build.Commands = []plan.Command{
					plan.NewCopyCommand("."),
					plan.NewExecShellCommand("custom-next-build"),
				}
			},
			wantStart:        "node /railpack/next-standalone/server.js",
			wantDeployInput:  NextStandaloneDeployRoot,
			wantStandalone:   true,
			wantBuildCommand: "custom-next-build",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testingUtils.CreateGenerateContext(t, tt.appPath)
			if tt.env != nil {
				ctx.Env = app.NewEnvironment(&tt.env)
			}
			if tt.configure != nil {
				tt.configure(ctx)
			}

			provider := NodeProvider{}
			require.NoError(t, provider.Initialize(ctx))
			require.NoError(t, provider.Plan(ctx))

			if tt.wantStart != "" {
				require.Equal(t, tt.wantStart, ctx.Deploy.StartCmd)
			} else {
				require.Contains(t, ctx.Deploy.StartCmd, "caddy run")
			}

			var deployLayer *plan.Layer
			for i := range ctx.Deploy.DeployInputs {
				layer := &ctx.Deploy.DeployInputs[i]
				if slices.Contains(layer.Include, tt.wantDeployInput) {
					deployLayer = layer
					break
				}
			}
			require.NotNil(t, deployLayer)
			require.False(t, deployLayer.Spread)

			buildStepRef := ctx.GetStepByName("build")
			require.NotNil(t, buildStepRef)
			buildStep, ok := (*buildStepRef).(*generate.CommandStepBuilder)
			require.True(t, ok)

			scriptAsset, hasScriptAsset := buildStep.Assets[NextStandaloneConfigScriptAsset]
			require.Equal(t, tt.wantStandalone, hasScriptAsset)
			if tt.wantStandalone {
				require.Contains(t, scriptAsset, `output: "standalone"`)
				require.NotContains(t, scriptAsset, "railpack-original")
				commands := fmt.Sprint(buildStep.Commands)
				require.Contains(t, commands, NextStandaloneConfigScriptPath)
				if tt.configure == nil {
					require.Contains(t, commands, tt.wantBuildCommand)
				} else {
					configuredCommands := fmt.Sprint(ctx.Config.Steps["build"].Commands)
					require.Contains(t, configuredCommands, tt.wantBuildCommand)
					require.Contains(t, configuredCommands, "prepare Next.js standalone deploy")
				}
				require.Contains(t, commands, "prepare Next.js standalone deploy")
				require.Contains(t, commands, "sh -c")
			}
		})
	}
}

func TestNodeCorepack(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		wantCorepack bool
	}{
		{
			name:         "corepack project",
			path:         "../../../examples/node-corepack",
			wantCorepack: true,
		},
		{
			name:         "bun project",
			path:         "../../../examples/node-bun",
			wantCorepack: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testingUtils.CreateGenerateContext(t, tt.path)
			provider := NodeProvider{}
			err := provider.Initialize(ctx)
			require.NoError(t, err)

			usesCorepack := provider.usesCorepack()
			require.Equal(t, tt.wantCorepack, usesCorepack)
		})
	}
}

func TestGetNextApps(t *testing.T) {
	tests := []struct {
		name string
		path string
		want []string
	}{
		{
			name: "npm project",
			path: "../../../examples/node-npm",
			want: []string{},
		},
		{
			name: "bun project",
			path: "../../../examples/node-next",
			want: []string{""},
		},
		{
			name: "turbo with 2 next apps",
			path: "../../../examples/node-turborepo",
			want: []string{"apps/web"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testingUtils.CreateGenerateContext(t, tt.path)
			provider := NodeProvider{}
			err := provider.Initialize(ctx)
			require.NoError(t, err)

			nextPackages, err := provider.getPackagesWithFramework(ctx, func(pkg *WorkspacePackage, ctx *generate.GenerateContext) bool {
				if pkg.PackageJson.HasScript("build") {
					return strings.Contains(pkg.PackageJson.Scripts["build"], "next build")
				}
				return false
			})
			require.NoError(t, err)

			nextApps := make([]string, len(nextPackages))
			for i, pkg := range nextPackages {
				nextApps[i] = pkg.Path
			}
			require.Equal(t, tt.want, nextApps)
		})
	}
}

func TestPackageJsonRequiresBun(t *testing.T) {
	// Special cases
	t.Run("nil package.json", func(t *testing.T) {
		got := packageJsonRequiresBun(nil)
		require.False(t, got)
	})

	t.Run("no scripts", func(t *testing.T) {
		got := packageJsonRequiresBun(&PackageJson{})
		require.False(t, got)
	})

	// Scripts that should trigger bun detection
	bunScripts := []string{
		"bun run server.js",
		"bunx nodemon index.js",
		"bun test",
		"npm run clean && bun build.js",
		"echo 'Running tests' | bun test",
		"npm run build; bun run server.js",
		"cd src && bun install",
		"bun --version",
		"bunx prisma migrate",
	}

	t.Run("scripts requiring bun", func(t *testing.T) {
		packageJson := &PackageJson{
			Scripts: make(map[string]string),
		}
		for i, script := range bunScripts {
			packageJson.Scripts[fmt.Sprintf("script%d", i)] = script
		}
		got := packageJsonRequiresBun(packageJson)
		require.True(t, got)
	})

	// Scripts that should NOT trigger bun detection
	nonBunScripts := []string{
		"esbuild dev.ts ./src --bundle --outdir=dist --packages=external --platform=node --sourcemap --watch",
		"webpack --config webpack.bundle.config.js",
		"node src/bundle-manager.js",
		"jest --bundle-reporter",
		"eslint src/bundles/",
		"sh deploy-bundle.sh",
		"npm run bundle:production",
		"yarn bundle",
		"pnpm run unbundle",
	}

	t.Run("scripts not requiring bun", func(t *testing.T) {
		packageJson := &PackageJson{
			Scripts: make(map[string]string),
		}
		for i, script := range nonBunScripts {
			packageJson.Scripts[fmt.Sprintf("script%d", i)] = script
		}
		got := packageJsonRequiresBun(packageJson)
		require.False(t, got)
	})
}

func TestMetadata(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		runtime     string
		nodeRuntime string
		expose      string
	}{
		{name: "next server", path: "../../../examples/node-next", runtime: "nextjs", nodeRuntime: "next", expose: "3000"},
		{name: "next static", path: "../../../examples/node-next-spa", runtime: "nextjs", nodeRuntime: "next", expose: "80"},
		{name: "next workspace", path: "../../../examples/node-turborepo", runtime: "nextjs", nodeRuntime: "next", expose: "3000"},
		{name: "vite spa", path: "../../../examples/node-vite-react", runtime: "vite", nodeRuntime: "vite", expose: "80"},
		{name: "astro server", path: "../../../examples/node-astro-server", runtime: "astro", nodeRuntime: "astro", expose: "4321"},
		{name: "plain node", path: "../../../examples/node-npm", runtime: "nodejs", nodeRuntime: "node"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testingUtils.CreateGenerateContext(t, tt.path)
			provider := NodeProvider{}
			require.NoError(t, provider.Initialize(ctx))
			require.NoError(t, provider.Plan(ctx))

			metadata := provider.Metadata(ctx)
			require.Equal(t, tt.runtime, metadata.Runtime)
			require.Equal(t, tt.expose, metadata.Expose)
			require.Equal(t, tt.nodeRuntime, ctx.Metadata.Get("nodeRuntime"))
		})
	}
}

func TestUsesPnpmBinSubdir(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    bool
	}{
		{
			name:    "latest uses bin subdir",
			version: "latest",
			want:    true,
		},
		{
			name:    "major 11 uses bin subdir",
			version: "11",
			want:    true,
		},
		{
			name:    "pnpm 11 uses bin subdir",
			version: "11.0.0",
			want:    true,
		},
		{
			name:    "pnpm 10 does not use bin subdir",
			version: "10.9.0",
			want:    false,
		},
		{
			name:    "empty version does not use bin subdir",
			version: "",
			want:    false,
		},
		{
			name:    "invalid version does not use bin subdir",
			version: "workspace:^",
			want:    false,
		},
		{
			name:    "resolved mise version",
			version: "11.5.1",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, usesPnpmBinSubdir(tt.version))
		})
	}
}
