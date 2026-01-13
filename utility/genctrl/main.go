// Package main provides a utility to generate GoFrame-compatible API wrapper files
// from existing Protobuf service definitions.
//
// This tool parses .proto files, extracts RPC method definitions, and generates
// wrapper structs with GoFrame's g.Meta annotations, enabling compatibility with
// the `gf gen ctrl` command.
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"unicode"
)

// ProtoService represents a parsed protobuf service definition
type ProtoService struct {
	Package     string
	ServiceName string
	Methods     []ProtoMethod
}

// ProtoMethod represents a single RPC method in a protobuf service
type ProtoMethod struct {
	Name         string
	RequestType  string
	ResponseType string
	HTTPMethod   string
	HTTPPath     string
	Summary      string
}

// Config holds the generator configuration
type Config struct {
	ProtoDir   string // Directory containing .proto files
	OutputDir  string // Directory to output generated files
	ModulePath string // Go module path (e.g., "gaap-api")
}

func main() {
	// Default configuration
	cfg := Config{
		ProtoDir:   "manifest/protobuf",
		OutputDir:  "api",
		ModulePath: "gaap-api",
	}

	// Parse command line arguments
	if len(os.Args) > 1 {
		cfg.ProtoDir = os.Args[1]
	}
	if len(os.Args) > 2 {
		cfg.OutputDir = os.Args[2]
	}

	fmt.Println("=== Protobuf to GoFrame Wrapper Generator ===")
	fmt.Printf("Proto directory: %s\n", cfg.ProtoDir)
	fmt.Printf("Output directory: %s\n", cfg.OutputDir)

	// Find all .proto files
	protoFiles, err := findProtoFiles(cfg.ProtoDir)
	if err != nil {
		fmt.Printf("Error finding proto files: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d proto files\n\n", len(protoFiles))

	// Process each proto file
	for _, protoFile := range protoFiles {
		fmt.Printf("Processing: %s\n", protoFile)

		service, err := parseProtoFile(protoFile)
		if err != nil {
			fmt.Printf("  Warning: %v\n", err)
			continue
		}

		if service == nil || len(service.Methods) == 0 {
			fmt.Println("  No service methods found, skipping")
			continue
		}

		// Generate wrapper file
		err = generateWrapper(cfg, protoFile, service)
		if err != nil {
			fmt.Printf("  Error generating wrapper: %v\n", err)
			continue
		}

		fmt.Printf("  Generated wrapper with %d methods\n", len(service.Methods))
	}

	fmt.Println("\n=== Generation Complete ===")
}

// findProtoFiles recursively finds all .proto files in a directory
func findProtoFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".proto") {
			// Skip base.proto as it contains only common types
			if !strings.HasSuffix(path, "base.proto") {
				files = append(files, path)
			}
		}
		return nil
	})

	return files, err
}

// parseProtoFile parses a .proto file and extracts service definitions
func parseProtoFile(filename string) (*ProtoService, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	service := &ProtoService{}
	scanner := bufio.NewScanner(file)

	// Regex patterns
	packageRegex := regexp.MustCompile(`^package\s+([a-zA-Z0-9_.]+)\s*;`)
	serviceRegex := regexp.MustCompile(`^service\s+(\w+)\s*\{`)
	rpcRegex := regexp.MustCompile(`^\s*rpc\s+(\w+)\s*\(\s*(\w+)\s*\)\s*returns\s*\(\s*(\w+)\s*\)`)
	commentRegex := regexp.MustCompile(`^\s*//\s*(.+)`)

	var currentComment string
	inService := false
	braceCount := 0

	for scanner.Scan() {
		line := scanner.Text()

		// Parse package
		if matches := packageRegex.FindStringSubmatch(line); len(matches) > 1 {
			service.Package = matches[1]
		}

		// Parse service start
		if matches := serviceRegex.FindStringSubmatch(line); len(matches) > 1 {
			service.ServiceName = matches[1]
			inService = true
			braceCount = 1
			continue
		}

		// Track braces for service scope
		if inService {
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")
			if braceCount <= 0 {
				inService = false
			}
		}

		// Parse comments for method summaries
		if matches := commentRegex.FindStringSubmatch(line); len(matches) > 1 {
			currentComment = strings.TrimSpace(matches[1])
			continue
		}

		// Parse RPC methods
		if inService {
			if matches := rpcRegex.FindStringSubmatch(line); len(matches) > 3 {
				method := ProtoMethod{
					Name:         matches[1],
					RequestType:  matches[2],
					ResponseType: matches[3],
					Summary:      currentComment,
				}

				// Infer HTTP method and path
				method.HTTPMethod, method.HTTPPath = inferHTTPRoute(method.Name, service.Package)

				service.Methods = append(service.Methods, method)
				currentComment = ""
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return service, nil
}

// inferHTTPRoute determines the HTTP method and path based on RPC method name
func inferHTTPRoute(methodName, packageName string) (string, string) {
	// Extract module name from package (e.g., "account.v1" -> "account")
	parts := strings.Split(packageName, ".")
	module := parts[0]

	// Convert to lowercase path
	basePath := "/" + module

	// Determine HTTP method based on naming convention
	lowerName := strings.ToLower(methodName)

	switch {
	case strings.HasPrefix(lowerName, "list"):
		return "GET", basePath
	case strings.HasPrefix(lowerName, "get"):
		suffix := methodName[3:]
		if strings.EqualFold(suffix, module) {
			return "GET", basePath + "/:id"
		}
		return "GET", basePath + "/" + toKebabCase(suffix)
	case strings.HasPrefix(lowerName, "create"):
		return "POST", basePath
	case strings.HasPrefix(lowerName, "update"):
		suffix := methodName[6:]
		if strings.EqualFold(suffix, module) {
			return "PUT", basePath + "/:id"
		}
		return "PUT", basePath + "/" + toKebabCase(suffix)
	case strings.HasPrefix(lowerName, "delete"):
		suffix := methodName[6:]
		if strings.EqualFold(suffix, module) {
			return "DELETE", basePath + "/:id"
		}
		return "DELETE", basePath + "/" + toKebabCase(suffix)
	default:
		// For custom methods, use POST with method name in path
		methodPath := toKebabCase(methodName)
		return "POST", basePath + "/" + methodPath
	}
}

// toKebabCase converts PascalCase to kebab-case
func toKebabCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result.WriteRune('-')
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// generateWrapper generates a GoFrame-compatible wrapper file
func generateWrapper(cfg Config, protoFile string, service *ProtoService) error {
	// Determine output path
	// e.g., manifest/protobuf/account/v1/account.proto -> api/account/v1/account.go
	relPath, _ := filepath.Rel(cfg.ProtoDir, protoFile)
	dir := filepath.Dir(relPath)
	baseName := strings.TrimSuffix(filepath.Base(protoFile), ".proto")
	outputPath := filepath.Join(cfg.OutputDir, dir, baseName+".go")

	// Extract module name and version
	parts := strings.Split(dir, string(filepath.Separator))
	moduleName := parts[0]
	version := "v1"
	if len(parts) > 1 {
		version = parts[1]
	}

	// Prepare template data
	data := struct {
		PackageName string
		ModulePath  string
		ModuleName  string
		Version     string
		ServiceName string
		Methods     []ProtoMethod
	}{
		PackageName: version,
		ModulePath:  cfg.ModulePath,
		ModuleName:  moduleName,
		Version:     version,
		ServiceName: service.ServiceName,
		Methods:     service.Methods,
	}

	// Generate file content
	var buf strings.Builder
	tmpl := template.Must(template.New("wrapper").Parse(wrapperTemplate))
	if err := tmpl.Execute(&buf, data); err != nil {
		return err
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}

	// Write file
	return os.WriteFile(outputPath, []byte(buf.String()), 0644)
}

// wrapperTemplate is the Go template for generating wrapper files
const wrapperTemplate = `// Code generated by genctrl. DO NOT EDIT.
// Source: {{.ModuleName}}/{{.Version}}/{{.ModuleName}}.proto

package {{.PackageName}}

import (
	"github.com/gogf/gf/v2/frame/g"
)

// =============================================================================
// GoFrame API Wrappers for {{.ServiceName}}
// These wrapper types add g.Meta annotations to enable gf gen ctrl compatibility
// The wrapper types embed the original Protobuf types with a "Gf" prefix
// =============================================================================

{{range .Methods}}
// Gf{{.Name}}Req is the GoFrame-compatible request wrapper for {{.Name}}
type Gf{{.Name}}Req struct {
	g.Meta ` + "`" + `path:"{{.HTTPPath}}" method:"{{.HTTPMethod}}" tags:"{{$.ModuleName}}" summary:"{{.Summary}}"` + "`" + `
	*{{.RequestType}}
}

// Gf{{.Name}}Res is the GoFrame-compatible response wrapper for {{.Name}}
type Gf{{.Name}}Res = {{.ResponseType}}

{{end}}
`
