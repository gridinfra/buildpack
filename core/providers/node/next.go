package node

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/railwayapp/railpack/core/generate"
)

const (
	DefaultNextOutputDirectory      = "out"
	DefaultNextStartCommand         = "next start"
	NextStandaloneConfigScriptAsset = "configure-next-standalone.mjs"
	NextStandaloneConfigScriptPath  = "/railpack/scripts/configure-next-standalone.mjs"
	NextStandaloneDeployRoot        = "/railpack/next-standalone"
	NextStandaloneServerFile        = "server.js"
)

var (
	nextConfigFiles = []string{
		"next.config.js",
		"next.config.mjs",
		"next.config.ts",
	}
	nextOutputPropertyPattern = regexp.MustCompile(`(?m)\boutput\s*:`)
	nextStaticOutputPattern   = regexp.MustCompile(`(?m)\boutput\s*:\s*["'][^"']+["']`)
	nextExportOutputPattern   = regexp.MustCompile(`(?m)\boutput\s*:\s*["']export["']`)
	nextDistDirPattern        = regexp.MustCompile(`(?m)\bdistDir\s*:\s*(?:'([^']+)'|"([^"]+)")`)
)

func stripJavaScriptComments(contents string) string {
	var result strings.Builder
	var quote rune
	for i := 0; i < len(contents); {
		current := rune(contents[i])
		if quote != 0 {
			result.WriteByte(contents[i])
			if current == '\\' && i+1 < len(contents) {
				i++
				result.WriteByte(contents[i])
			} else if current == quote {
				quote = 0
			}
			i++
			continue
		}

		if current == '\'' || current == '"' || current == '`' {
			quote = current
			result.WriteByte(contents[i])
			i++
			continue
		}
		if current == '/' && i+1 < len(contents) && contents[i+1] == '/' {
			for i < len(contents) && contents[i] != '\n' {
				i++
			}
			continue
		}
		if current == '/' && i+1 < len(contents) && contents[i+1] == '*' {
			i += 2
			for i+1 < len(contents) && (contents[i] != '*' || contents[i+1] != '/') {
				i++
			}
			if i+1 < len(contents) {
				i += 2
			}
			continue
		}
		result.WriteByte(contents[i])
		i++
	}
	return result.String()
}

func (p *NodeProvider) isNextSPA(ctx *generate.GenerateContext) bool {
	nextPackage, err := p.getNextPackage(ctx)
	if err != nil {
		return false
	}

	configFileContents := stripJavaScriptComments(p.getNextConfigFileContents(ctx))
	if nextExportOutputPattern.MatchString(configFileContents) {
		return true
	}

	buildScript := nextPackage.PackageJson.Scripts["build"]
	if strings.Contains(buildScript, "next export") {
		return true
	}

	if strings.Contains(nextPackage.PackageJson.GetScript("export"), "next export") {
		return true
	}

	return false
}

func (p *NodeProvider) hasDynamicNextOutput(ctx *generate.GenerateContext) bool {
	contents := stripJavaScriptComments(p.getNextConfigFileContents(ctx))
	return nextOutputPropertyPattern.MatchString(contents) && !nextStaticOutputPattern.MatchString(contents)
}

func (p *NodeProvider) getNextPackage(ctx *generate.GenerateContext) (*WorkspacePackage, error) {
	packages, err := p.getPackagesWithFramework(ctx, func(pkg *WorkspacePackage, _ *generate.GenerateContext) bool {
		return pkg.PackageJson.hasDependency("next")
	})
	if err != nil {
		return nil, err
	}
	if len(packages) == 0 {
		return nil, fmt.Errorf("next.js application not found")
	}
	if len(packages) > 1 {
		return nil, fmt.Errorf("multiple Next.js applications found; select a single application before using standalone output")
	}
	return packages[0], nil
}

func (p *NodeProvider) getNextAppPath(ctx *generate.GenerateContext) (string, error) {
	nextPackage, err := p.getNextPackage(ctx)
	if err != nil {
		return "", err
	}
	return nextPackage.Path, nil
}

func (p *NodeProvider) getNextConfigFile(ctx *generate.GenerateContext, appPath string) string {
	for _, configFile := range nextConfigFiles {
		candidate := path.Join(appPath, configFile)
		if ctx.App.HasFile(candidate) {
			return candidate
		}
	}
	return ""
}

func (p *NodeProvider) getNextConfigFileContents(ctx *generate.GenerateContext) string {
	appPath, err := p.getNextAppPath(ctx)
	if err != nil {
		return ""
	}
	configFile := p.getNextConfigFile(ctx, appPath)
	if configFile == "" {
		return ""
	}
	contents, err := ctx.App.ReadFile(configFile)
	if err != nil {
		return ""
	}
	return contents
}

func getNextStandaloneConfigScript() string {
	return `import { readFile, writeFile } from "node:fs/promises";
import path from "node:path";

const configPath = process.argv[2];
if (!configPath || ![".js", ".mjs", ".ts"].includes(path.extname(configPath))) {
  throw new Error("expected a next.config.js, next.config.mjs, or next.config.ts path");
}

let source;
try {
  source = await readFile(configPath, "utf8");
} catch (error) {
  if (error.code !== "ENOENT") throw error;
  source = "const nextConfig = {};\nexport default nextConfig;\n";
}

function maskComments(value) {
  let result = "";
  let quote = "";
  for (let index = 0; index < value.length; index += 1) {
    const current = value[index];
    const next = value[index + 1];
    if (quote) {
      result += current;
      if (current === "\\" && next) {
        index += 1;
        result += value[index];
      } else if (current === quote) {
        quote = "";
      }
      continue;
    }
    if (current === "'" || current === '"') {
      quote = current;
      result += current;
      continue;
    }
    if (current === "/" && next === "/") {
      result += "  ";
      index += 1;
      while (index + 1 < value.length && value[index + 1] !== "\n") {
        index += 1;
        result += " ";
      }
      continue;
    }
    if (current === "/" && next === "*") {
      result += "  ";
      index += 1;
      while (index + 1 < value.length && !(value[index] === "*" && value[index + 1] === "/")) {
        index += 1;
        result += value[index] === "\n" ? "\n" : " ";
      }
      if (index + 1 < value.length) {
        index += 1;
        result += " ";
      }
      continue;
    }
    result += current;
  }
  return result;
}

const masked = maskComments(source);
const outputPattern = /\boutput\s*:\s*(["'])[^"']*\1/g;
const outputMatches = [...masked.matchAll(outputPattern)];
if (outputMatches.length > 1) {
  throw new Error("multiple static output properties found in Next.js config");
}

if (outputMatches.length === 1) {
  const match = outputMatches[0];
  source = source.slice(0, match.index) + 'output: "standalone"' + source.slice(match.index + match[0].length);
} else if (/\boutput\s*:/.test(masked)) {
  throw new Error("dynamic output properties are not supported in Next.js config");
} else {
  const nextConfigObject = /\b(?:const|let|var)\s+nextConfig(?:\s*:[^=]+)?\s*=\s*\{/g;
  const exportsNextConfig =
    /\bexport\s+default\s+nextConfig\b/.test(masked) ||
    /\bmodule\.exports\s*=\s*nextConfig\b/.test(masked);
  const objectPatterns = [
    ...(exportsNextConfig ? [nextConfigObject] : []),
    /\bexport\s+default\s*\{/g,
    /\bmodule\.exports\s*=\s*\{/g,
  ];
  const matches = objectPatterns.flatMap((pattern) => [...masked.matchAll(pattern)]);
  if (matches.length !== 1) {
    throw new Error("unable to locate a unique Next.js config object");
  }
  const insertAt = matches[0].index + matches[0][0].length;
  source = source.slice(0, insertAt) + '\n  output: "standalone",' + source.slice(insertAt);
}

await writeFile(configPath, source);
console.log("Configured " + configPath + " for Next.js standalone output");
`
}

func (p *NodeProvider) getNextStandaloneConfigPath(ctx *generate.GenerateContext) (string, error) {
	appPath, err := p.getNextAppPath(ctx)
	if err != nil {
		return "", err
	}
	configPath := p.getNextConfigFile(ctx, appPath)
	if configPath == "" {
		configPath = path.Join(appPath, "next.config.mjs")
	}
	return configPath, nil
}

func getNextStandaloneConfigCommand(configPath string) string {
	return fmt.Sprintf("node %s %s", shellQuote(NextStandaloneConfigScriptPath), shellQuote(configPath))
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func shellDoubleQuote(value string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		`$`, `\$`,
		"`", "\\`",
	)
	return `"` + replacer.Replace(value) + `"`
}

func getNextStandalonePrepareCommand(appPath string, hasPublic bool) string {
	appDir := appPath
	if appDir == "" {
		appDir = "."
	}

	deployRoot := shellDoubleQuote(NextStandaloneDeployRoot)
	commands := []string{
		fmt.Sprintf("cd %s", shellDoubleQuote(appDir)),
		`standalone_dirs="$(find . -path "*/node_modules" -prune -o -type d -name standalone -print)"`,
		`[ "$(printf "%s\n" "$standalone_dirs" | sed "/^$/d" | wc -l)" -eq 1 ]`,
		`standalone_dir="$standalone_dirs"`,
		fmt.Sprintf("app_server_rel=%s", shellDoubleQuote(path.Join(appPath, NextStandaloneServerFile))),
		`if [ -f "$standalone_dir/server.js" ]; then server_rel=server.js; ` +
			`elif [ -f "$standalone_dir/$app_server_rel" ]; then server_rel="$app_server_rel"; ` +
			`else exit 1; fi`,
		`server_dir="$(dirname "$server_rel")"`,
		`dist_dir="${standalone_dir%/standalone}"`,
		fmt.Sprintf("rm -rf %s", deployRoot),
		fmt.Sprintf("mkdir -p %s", deployRoot),
		fmt.Sprintf(`cp -a "$standalone_dir/." %s`, deployRoot),
		fmt.Sprintf(`mkdir -p %s/"$server_dir"/"$dist_dir"`, deployRoot),
		fmt.Sprintf(`cp -a "$dist_dir/static" %s/"$server_dir"/"$dist_dir"`, deployRoot),
	}
	if hasPublic {
		commands = append(commands, fmt.Sprintf(`cp -a public %s/"$server_dir"/public`, deployRoot))
	}
	// A stable root entrypoint hides root and monorepo standalone layout differences.
	commands = append(commands,
		fmt.Sprintf(
			`if [ "$server_rel" != server.js ]; then ln -s "$server_rel" %s/server.js; fi`,
			deployRoot,
		),
	)

	return strings.Join(commands, " && ")
}

func getNextStandaloneStartCommand(_ string) string {
	return fmt.Sprintf("node %s", path.Join(NextStandaloneDeployRoot, NextStandaloneServerFile))
}

func (p *NodeProvider) getNextOutputDirectory(ctx *generate.GenerateContext) string {
	configFileContents := stripJavaScriptComments(p.getNextConfigFileContents(ctx))
	matches := nextDistDirPattern.FindStringSubmatch(configFileContents)
	if len(matches) == 3 {
		if matches[1] != "" {
			return matches[1]
		}
		return matches[2]
	}

	return DefaultNextOutputDirectory
}
