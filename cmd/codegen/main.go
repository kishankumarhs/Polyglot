// Command codegen regenerates the C ABI header and language FFI bindings
// from api/abi.json — the single source of truth for the public C API.
//
// Usage (from repo root):
//
//	go run ./cmd/codegen
//
// Hand-written ergonomic wrappers (Logger classes) stay in bindings/*;
// they import the generated FFI layers. Implement new exports in
// native/export.go, add them to api/abi.json, then re-run codegen.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

type Spec struct {
	ABIVersion  int    `json:"abi_version"`
	Library     string `json:"library"`
	HeaderGuard string `json:"header_guard"`
	Enums       []Enum `json:"enums"`
	Functions   []Func `json:"functions"`
}

type Enum struct {
	Name    string      `json:"name"`
	CPrefix string      `json:"c_prefix"`
	Doc     string      `json:"doc"`
	Values  []EnumValue `json:"values"`
}

type EnumValue struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

type Func struct {
	Name    string `json:"name"`
	Doc     string `json:"doc"`
	Returns string `json:"returns"`
	Args    []Arg  `json:"args"`
}

type Arg struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
	Mutable  bool   `json:"mutable"`
}

func main() {
	root, err := repoRoot()
	if err != nil {
		fatal(err)
	}
	specPath := filepath.Join(root, "api", "abi.json")
	data, err := os.ReadFile(specPath)
	if err != nil {
		fatal(err)
	}
	var spec Spec
	if err := json.Unmarshal(data, &spec); err != nil {
		fatal(fmt.Errorf("parse %s: %w", specPath, err))
	}
	if err := validate(spec); err != nil {
		fatal(err)
	}
	// Verify before writing so a mismatch never leaves half-generated bindings
	// behind for someone to commit.
	if err := verifyExports(root, spec); err != nil {
		fatal(err)
	}

	writers := []struct {
		path string
		fn   func(Spec) ([]byte, error)
	}{
		{filepath.Join(root, "native", "include", "logger.h"), genHeader},
		{filepath.Join(root, "bindings", "python", "eximietas_logger", "_ffi_generated.py"), genPythonFFI},
		{filepath.Join(root, "bindings", "node", "src", "ffi.generated.ts"), genNodeFFI},
		{filepath.Join(root, "bindings", "dotnet", "Eximietas.Logger", "NativeMethods.Generated.cs"), genDotnetFFI},
		{filepath.Join(root, "native", "abi_exports.md"), genExportChecklist},
	}

	for _, w := range writers {
		out, err := w.fn(spec)
		if err != nil {
			fatal(fmt.Errorf("%s: %w", w.path, err))
		}
		if err := os.MkdirAll(filepath.Dir(w.path), 0o755); err != nil {
			fatal(err)
		}
		if err := os.WriteFile(w.path, out, 0o644); err != nil {
			fatal(err)
		}
		fmt.Println("wrote", rel(root, w.path))
	}

	fmt.Println("codegen ok (abi_version=", spec.ABIVersion, ")")
}

func repoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "api", "abi.json")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find repo root (api/abi.json) from %s", wd)
		}
		dir = parent
	}
}

func validate(spec Spec) error {
	if spec.ABIVersion < 1 {
		return fmt.Errorf("abi_version must be >= 1")
	}
	if len(spec.Functions) == 0 {
		return fmt.Errorf("functions list is empty")
	}
	seen := map[string]bool{}
	for _, f := range spec.Functions {
		if f.Name == "" {
			return fmt.Errorf("function with empty name")
		}
		if seen[f.Name] {
			return fmt.Errorf("duplicate function %q", f.Name)
		}
		seen[f.Name] = true
		if !validType(f.Returns) {
			return fmt.Errorf("%s: invalid return type %q", f.Name, f.Returns)
		}
		for _, a := range f.Args {
			if !validType(a.Type) {
				return fmt.Errorf("%s.%s: invalid type %q", f.Name, a.Name, a.Type)
			}
		}
	}
	return nil
}

func validType(t string) bool {
	switch t {
	case "void", "int", "string", "handle":
		return true
	default:
		return false
	}
}

// verifyExports checks that api/abi.json and native/export.go agree in both
// directions, and that each function's argument count matches. Without the
// reverse check an undeclared export (or a changed signature) would silently
// ship without bindings.
func verifyExports(root string, spec Spec) error {
	exportPath := filepath.Join(root, "native", "export.go")
	data, err := os.ReadFile(exportPath)
	if err != nil {
		return err
	}

	declared := map[string]Func{}
	for _, f := range spec.Functions {
		declared[f.Name] = f
	}
	exported, err := parseGoExports(string(data))
	if err != nil {
		return err
	}

	var problems []string
	for name := range declared {
		if _, ok := exported[name]; !ok {
			problems = append(problems, fmt.Sprintf("%s: declared in api/abi.json but no //export in native/export.go", name))
		}
	}
	for name, argCount := range exported {
		f, ok := declared[name]
		if !ok {
			problems = append(problems, fmt.Sprintf("%s: exported from native/export.go but missing from api/abi.json", name))
			continue
		}
		if argCount != len(f.Args) {
			problems = append(problems, fmt.Sprintf("%s: Go export takes %d arg(s), api/abi.json declares %d", name, argCount, len(f.Args)))
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("ABI mismatch:\n  - %s", strings.Join(problems, "\n  - "))
	}
	return nil
}

// parseGoExports maps each //export name to the argument count of the Go
// function that immediately follows it.
func parseGoExports(body string) (map[string]int, error) {
	result := map[string]int{}
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "//export ") {
			continue
		}
		name := strings.TrimSpace(strings.TrimPrefix(trimmed, "//export "))
		if name == "" {
			continue
		}
		sig := ""
		for j := i + 1; j < len(lines); j++ {
			candidate := strings.TrimSpace(lines[j])
			if candidate == "" || strings.HasPrefix(candidate, "//") {
				continue
			}
			sig = candidate
			break
		}
		if !strings.HasPrefix(sig, "func ") {
			return nil, fmt.Errorf("//export %s is not followed by a func declaration", name)
		}
		count, err := countGoParams(sig)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		result[name] = count
	}
	return result, nil
}

func countGoParams(sig string) (int, error) {
	open := strings.Index(sig, "(")
	if open < 0 {
		return 0, fmt.Errorf("malformed signature %q", sig)
	}
	depth := 0
	end := -1
	for i := open; i < len(sig); i++ {
		switch sig[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				end = i
			}
		}
		if end >= 0 {
			break
		}
	}
	if end < 0 {
		return 0, fmt.Errorf("unbalanced parentheses in %q", sig)
	}
	params := strings.TrimSpace(sig[open+1 : end])
	if params == "" {
		return 0, nil
	}
	// Go allows grouped params ("a, b *C.char"); count comma-separated names.
	return len(strings.Split(params, ",")), nil
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "codegen: %v\n", err)
	os.Exit(1)
}

func rel(root, path string) string {
	r, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(r)
}

func cType(t string, mutable, nullable bool) string {
	_ = nullable
	switch t {
	case "void":
		return "void"
	case "int":
		return "int"
	case "handle":
		return "logger_handle"
	case "string":
		if mutable {
			return "char*"
		}
		return "const char*"
	default:
		return "void"
	}
}

func genHeader(spec Spec) ([]byte, error) {
	var b bytes.Buffer
	b.WriteString("/* Code generated by go run ./cmd/codegen; DO NOT EDIT. */\n")
	b.WriteString("/* Source of truth: api/abi.json */\n\n")
	b.WriteString("#ifndef " + spec.HeaderGuard + "\n")
	b.WriteString("#define " + spec.HeaderGuard + "\n\n")
	b.WriteString("#ifdef __cplusplus\nextern \"C\" {\n#endif\n\n")
	b.WriteString("#ifdef _WIN32\n")
	b.WriteString("#  ifdef LOGGER_BUILD\n")
	b.WriteString("#    define LOGGER_API __declspec(dllexport)\n")
	b.WriteString("#  else\n")
	b.WriteString("#    define LOGGER_API __declspec(dllimport)\n")
	b.WriteString("#  endif\n")
	b.WriteString("#else\n")
	b.WriteString("#  define LOGGER_API __attribute__((visibility(\"default\")))\n")
	b.WriteString("#endif\n\n")
	b.WriteString("#include <stddef.h>\n\n")
	b.WriteString("/* Opaque logger handle (non-NULL when valid; never dereference it). */\n")
	b.WriteString("typedef void* logger_handle;\n\n")

	for _, e := range spec.Enums {
		if e.Doc != "" {
			b.WriteString("/* " + e.Doc + " */\n")
		}
		b.WriteString("enum " + e.Name + " {\n")
		for i, v := range e.Values {
			comma := ","
			if i == len(e.Values)-1 {
				comma = ""
			}
			fmt.Fprintf(&b, "  %s%s = %d%s\n", e.CPrefix, v.Name, v.Value, comma)
		}
		b.WriteString("};\n\n")
	}

	for _, f := range spec.Functions {
		writeCDoc(&b, f.Doc)
		fmt.Fprintf(&b, "LOGGER_API %s %s(", cType(f.Returns, false, false), f.Name)
		if len(f.Args) == 0 {
			b.WriteString("void")
		} else {
			for i, a := range f.Args {
				if i > 0 {
					b.WriteString(", ")
				}
				fmt.Fprintf(&b, "%s %s", cType(a.Type, a.Mutable, a.Nullable), a.Name)
			}
		}
		b.WriteString(");\n\n")
	}

	b.WriteString("#ifdef __cplusplus\n}\n#endif\n\n")
	b.WriteString("#endif /* " + spec.HeaderGuard + " */\n")
	return b.Bytes(), nil
}

func writeCDoc(b *bytes.Buffer, doc string) {
	doc = strings.TrimSpace(doc)
	if doc == "" {
		return
	}
	lines := strings.Split(doc, "\n")
	if len(lines) == 1 {
		fmt.Fprintf(b, "/* %s */\n", lines[0])
		return
	}
	b.WriteString("/*\n")
	for _, line := range lines {
		fmt.Fprintf(b, " * %s\n", line)
	}
	b.WriteString(" */\n")
}

func genPythonFFI(spec Spec) ([]byte, error) {
	var b bytes.Buffer
	b.WriteString("# Code generated by go run ./cmd/codegen; DO NOT EDIT.\n")
	b.WriteString("# Source of truth: api/abi.json\n\n")
	b.WriteString("from __future__ import annotations\n\n")
	b.WriteString("import ctypes\n")
	b.WriteString("from enum import IntEnum\n")
	b.WriteString("from typing import Any\n\n\n")

	for _, e := range spec.Enums {
		pyName := strings.TrimPrefix(e.Name, "Logger")
		if pyName == e.Name {
			pyName = e.Name
		}
		fmt.Fprintf(&b, "class %s(IntEnum):\n", pyName)
		for _, v := range e.Values {
			fmt.Fprintf(&b, "    %s = %d\n", v.Name, v.Value)
		}
		b.WriteString("\n\n")
	}

	b.WriteString("def bind(lib: Any) -> Any:\n")
	b.WriteString("    \"\"\"Configure ctypes signatures on a loaded CDLL/WinDLL.\"\"\"\n")
	for _, f := range spec.Functions {
		args := make([]string, 0, len(f.Args))
		for _, a := range f.Args {
			args = append(args, pyCType(a.Type))
		}
		fmt.Fprintf(&b, "    lib.%s.argtypes = [%s]\n", f.Name, strings.Join(args, ", "))
		fmt.Fprintf(&b, "    lib.%s.restype = %s\n", f.Name, pyRestype(f.Returns))
	}
	b.WriteString("    return lib\n")
	return b.Bytes(), nil
}

func pyCType(t string) string {
	switch t {
	case "handle":
		return "ctypes.c_void_p"
	case "int":
		return "ctypes.c_int"
	case "string":
		return "ctypes.c_char_p"
	default:
		return "ctypes.c_void_p"
	}
}

func pyRestype(t string) string {
	switch t {
	case "void":
		return "None"
	case "int":
		return "ctypes.c_int"
	case "handle":
		return "ctypes.c_void_p"
	case "string":
		return "ctypes.c_char_p"
	default:
		return "None"
	}
}

func genNodeFFI(spec Spec) ([]byte, error) {
	tmpl := `// Code generated by go run ./cmd/codegen; DO NOT EDIT.
// Source of truth: api/abi.json

{{range .Enums}}
export enum {{enumTS .Name}} {
{{range .Values}}  {{.Name}} = {{.Value}},
{{end}}}
{{end}}
export type NativeFns = {
{{range .Functions}}  {{.Name}}: {{nodeFnType .}};
{{end}}};

/** Bind C ABI symbols from a koffi-loaded library. */
export function bindNative(lib: { func: (name: string, result: string, args: string[]) => unknown }): NativeFns {
  return {
{{range .Functions}}    {{.Name}}: lib.func("{{.Name}}", "{{nodeRet .Returns}}", [{{nodeArgs .}}]) as NativeFns["{{.Name}}"],
{{end}}  };
}
`
	funcs := template.FuncMap{
		"enumTS": func(name string) string {
			if strings.HasPrefix(name, "Logger") {
				return strings.TrimPrefix(name, "Logger")
			}
			return name
		},
		"nodeRet": nodeRet,
		"nodeArgs": func(f Func) string {
			parts := make([]string, 0, len(f.Args))
			for _, a := range f.Args {
				parts = append(parts, fmt.Sprintf("%q", nodeArg(a.Type)))
			}
			return strings.Join(parts, ", ")
		},
		"nodeFnType": func(f Func) string {
			args := make([]string, 0, len(f.Args))
			for _, a := range f.Args {
				args = append(args, fmt.Sprintf("%s: %s", jsIdent(a.Name), nodeTSType(a.Type)))
			}
			return fmt.Sprintf("(%s) => %s", strings.Join(args, ", "), nodeTSRet(f.Returns))
		},
	}
	t, err := template.New("node").Funcs(funcs).Parse(tmpl)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	if err := t.Execute(&b, spec); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func jsIdent(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		if parts[i] == "" {
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return strings.Join(parts, "")
}

func nodeRet(t string) string {
	switch t {
	case "void":
		return "void"
	case "int":
		return "int"
	case "handle":
		return "void *"
	case "string":
		return "str"
	default:
		return "void"
	}
}

func nodeArg(t string) string {
	return nodeRet(t)
}

func nodeTSType(t string) string {
	switch t {
	case "handle":
		return "unknown"
	case "int":
		return "number"
	case "string":
		return "string | null"
	default:
		return "unknown"
	}
}

func nodeTSRet(t string) string {
	switch t {
	case "void":
		return "void"
	case "int":
		return "number"
	case "handle":
		return "unknown"
	case "string":
		return "string"
	default:
		return "unknown"
	}
}

func genDotnetFFI(spec Spec) ([]byte, error) {
	var b bytes.Buffer
	b.WriteString("// <auto-generated />\n")
	b.WriteString("// Code generated by go run ./cmd/codegen; DO NOT EDIT.\n")
	b.WriteString("// Source of truth: api/abi.json\n\n")
	b.WriteString("#nullable enable\n")
	b.WriteString("using System.Runtime.InteropServices;\n\n")
	b.WriteString("namespace Eximietas.Logger;\n\n")

	for _, e := range spec.Enums {
		name := strings.TrimPrefix(e.Name, "Logger")
		if name == e.Name {
			name = e.Name
		}
		fmt.Fprintf(&b, "public enum %s\n{\n", name)
		for _, v := range e.Values {
			cs := strings.ToUpper(v.Name[:1]) + strings.ToLower(v.Name[1:])
			// TRACE -> Trace, etc.
			cs = toPascal(v.Name)
			fmt.Fprintf(&b, "    %s = %d,\n", cs, v.Value)
		}
		b.WriteString("}\n\n")
	}

	b.WriteString("internal static partial class NativeMethods\n{\n")
	b.WriteString("    private const string LibraryName = \"logger\";\n\n")

	for _, f := range spec.Functions {
		writeCSDoc(&b, f.Doc)
		b.WriteString("    [DllImport(LibraryName, CallingConvention = CallingConvention.Cdecl)]\n")
		fmt.Fprintf(&b, "    internal static extern %s %s(", csRet(f.Returns), f.Name)
		for i, a := range f.Args {
			if i > 0 {
				b.WriteString(",\n        ")
			} else if len(f.Args) > 1 {
				b.WriteString("\n        ")
			}
			b.WriteString(csArg(a))
		}
		b.WriteString(");\n\n")
	}
	b.WriteString("}\n")
	return b.Bytes(), nil
}

func toPascal(s string) string {
	parts := strings.Split(strings.ToLower(s), "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, "")
}

func writeCSDoc(b *bytes.Buffer, doc string) {
	doc = strings.TrimSpace(doc)
	if doc == "" {
		return
	}
	for _, line := range strings.Split(doc, "\n") {
		fmt.Fprintf(b, "    /// <summary>%s</summary>\n", xmlEscape(line))
		break // one-line summary is enough for generated decls
	}
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func csRet(t string) string {
	switch t {
	case "void":
		return "void"
	case "int":
		return "int"
	case "handle":
		return "IntPtr"
	case "string":
		return "IntPtr"
	default:
		return "void"
	}
}

func csArg(a Arg) string {
	switch a.Type {
	case "handle":
		return "IntPtr " + csIdent(a.Name)
	case "int":
		return "int " + csIdent(a.Name)
	case "string":
		null := ""
		if a.Nullable {
			null = "?"
		}
		if a.Mutable {
			return "IntPtr " + csIdent(a.Name)
		}
		return "[MarshalAs(UnmanagedType.LPUTF8Str)] string" + null + " " + csIdent(a.Name)
	default:
		return "IntPtr " + csIdent(a.Name)
	}
}

func csIdent(s string) string {
	return jsIdent(s)
}

func genExportChecklist(spec Spec) ([]byte, error) {
	var b bytes.Buffer
	b.WriteString("# ABI export checklist\n\n")
	b.WriteString("Generated from `api/abi.json`. Each function must have a matching\n")
	b.WriteString("`//export <name>` implementation in [`native/export.go`](export.go).\n\n")
	b.WriteString("| Function | Returns | Args |\n")
	b.WriteString("|----------|---------|------|\n")
	for _, f := range spec.Functions {
		args := make([]string, 0, len(f.Args))
		for _, a := range f.Args {
			args = append(args, a.Type+" "+a.Name)
		}
		fmt.Fprintf(&b, "| `%s` | `%s` | %s |\n", f.Name, f.Returns, strings.Join(args, ", "))
	}
	b.WriteString("\nAfter editing `api/abi.json` or `native/export.go`, run:\n\n")
	b.WriteString("```bash\ngo run ./cmd/codegen\n```\n")
	return b.Bytes(), nil
}
