package node

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNextStandaloneConfigScript(t *testing.T) {
	tests := []struct {
		name       string
		filename   string
		input      string
		want       string
		wantError  bool
		fileExists bool
	}{
		{
			name:       "typescript object",
			filename:   "next.config.ts",
			input:      "const nextConfig: NextConfig = { reactStrictMode: true };\nexport default nextConfig;\n",
			want:       `output: "standalone"`,
			fileExists: true,
		},
		{
			name:       "esm direct object",
			filename:   "next.config.mjs",
			input:      "export default { reactStrictMode: true };\n",
			want:       `output: "standalone"`,
			fileExists: true,
		},
		{
			name:       "commonjs object",
			filename:   "next.config.js",
			input:      "module.exports = { reactStrictMode: true };\n",
			want:       `output: "standalone"`,
			fileExists: true,
		},
		{
			name:       "replace existing output",
			filename:   "next.config.mjs",
			input:      "export default { output: 'custom', reactStrictMode: true };\n",
			want:       `output: "standalone"`,
			fileExists: true,
		},
		{
			name:       "create missing config",
			filename:   "next.config.mjs",
			want:       `output: "standalone"`,
			fileExists: true,
		},
		{
			name:       "reject dynamic config",
			filename:   "next.config.mjs",
			input:      "export default withSentryConfig(nextConfig);\n",
			wantError:  true,
			fileExists: true,
		},
		{
			name:       "reject plugin wrapped object",
			filename:   "next.config.mjs",
			input:      "const nextConfig = { reactStrictMode: true };\nexport default withSentryConfig(nextConfig);\n",
			wantError:  true,
			fileExists: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			scriptPath := filepath.Join(tempDir, NextStandaloneConfigScriptAsset)
			require.NoError(t, os.WriteFile(scriptPath, []byte(getNextStandaloneConfigScript()), 0644))
			configPath := filepath.Join(tempDir, tt.filename)
			if tt.input != "" {
				require.NoError(t, os.WriteFile(configPath, []byte(tt.input), 0644))
			}

			output, err := exec.Command("node", scriptPath, configPath).CombinedOutput()
			if tt.wantError {
				require.Error(t, err, string(output))
				contents, readErr := os.ReadFile(configPath)
				require.NoError(t, readErr)
				require.Equal(t, tt.input, string(contents))
				return
			}
			require.NoError(t, err, string(output))
			contents, err := os.ReadFile(configPath)
			require.NoError(t, err)
			require.Contains(t, string(contents), tt.want)
			require.NotContains(t, string(contents), "railpack-original")
		})
	}
}

func TestNextOutputDetection(t *testing.T) {
	require.True(t, nextExportOutputPattern.MatchString(`const config = { output: "export" }`))
	require.True(t, nextExportOutputPattern.MatchString(`const config = { output : 'export' }`))
	require.True(t, nextExportOutputPattern.MatchString("const config = { output:\n  \"export\" }"))
	require.False(t, nextExportOutputPattern.MatchString(`const config = { output: "standalone" }`))

	withoutComments := stripJavaScriptComments("// output: \"export\"\nconst url = \"https://example.com\";\n/* output: 'export' */\nconst config = {};")
	require.NotContains(t, withoutComments, `output: "export"`)
	require.Contains(t, withoutComments, `https://example.com`)
	require.True(t, nextOutputPropertyPattern.MatchString(`const config = { output: mode }`))
	require.False(t, nextStaticOutputPattern.MatchString(`const config = { output: mode }`))
}

func TestNextDistDirPattern(t *testing.T) {
	tests := map[string]string{
		`distDir:"site"`:       "site",
		`distDir : 'site'`:     "site",
		"distDir:\n  \"site\"": "site",
	}
	for input, want := range tests {
		matches := nextDistDirPattern.FindStringSubmatch(input)
		require.Len(t, matches, 3)
		require.Equal(t, want, matches[1]+matches[2])
	}

	withoutComments := stripJavaScriptComments("// distDir: \"ignored\"\nconst config = { distDir: \"site\" }")
	matches := nextDistDirPattern.FindStringSubmatch(withoutComments)
	require.Len(t, matches, 3)
	require.Equal(t, "site", matches[1]+matches[2])
}

func TestGetNextStandaloneStartCommand(t *testing.T) {
	require.Equal(t, "node /railpack/next-standalone/server.js", getNextStandaloneStartCommand(""))
	require.Equal(t, "node /railpack/next-standalone/server.js", getNextStandaloneStartCommand("apps/web"))
}

func TestGetNextStandalonePrepareCommand(t *testing.T) {
	command := getNextStandalonePrepareCommand("apps/web", true)
	require.Contains(t, command, `cd "apps/web"`)
	require.Contains(t, command, "-type d -name standalone")
	require.Contains(t, command, `app_server_rel="apps/web/server.js"`)
	require.Contains(t, command, `-f "$standalone_dir/server.js"`)
	require.Contains(t, command, `-f "$standalone_dir/$app_server_rel"`)
	require.NotContains(t, command, "-type f -name server.js")
	require.NotContains(t, command, "'")
	require.Contains(t, command, `cp -a "$standalone_dir/." "/railpack/next-standalone"`)
	require.Equal(t, 1, strings.Count(command, `cp -a public "/railpack/next-standalone"/"$server_dir"/public`))
	require.Contains(t, command, `ln -s "$server_rel" "/railpack/next-standalone"/server.js`)
	require.NoError(t, exec.Command("sh", "-n", "-c", command).Run())

	withoutPublic := getNextStandalonePrepareCommand("", false)
	require.Contains(t, withoutPublic, `cd "."`)
	require.NotContains(t, withoutPublic, "cp -a public")
}
