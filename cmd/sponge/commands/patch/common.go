package patch

import (
	"bufio"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Eric-Guo/sponge/pkg/gofile"
)

const (
	httpType = "http"
	httpMark = "errcode.NewError"
	grpcType = "grpc"
	grpcMark = "errcode.NewRPCStatus"
)

// get moduleName and serverName from directory
func getNamesFromOutDir(dir string) (moduleName string, serverName string, suitedMonoRepo bool) {
	if dir == "" {
		return "", "", false
	}
	data, err := os.ReadFile(dir + "/docs/gen.info")
	if err != nil {
		return "", "", false
	}

	ms := strings.Split(string(data), ",")
	if len(ms) == 2 {
		return ms[0], ms[1], false
	} else if len(ms) >= 3 {
		return ms[0], ms[1], ms[2] == "true"
	}

	return "", "", false
}

func cutPath(srcProtoFile string) string {
	dirPath, _ := filepath.Abs("..")
	srcProtoFile = strings.ReplaceAll(srcProtoFile, dirPath, "..")
	return strings.ReplaceAll(srcProtoFile, "\\", "/")
}

func cutPathPrefix(srcProtoFile string) string {
	dirPath, _ := filepath.Abs(".")
	srcProtoFile = strings.ReplaceAll(srcProtoFile, dirPath, ".")
	return strings.ReplaceAll(srcProtoFile, "\\", "/")
}

func listErrorCodeFiles(dir string) (map[string][]string, error) {
	files, err := gofile.ListFiles(dir)
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, errors.New("not found files")
	}

	httpErrorCodeFiles := []string{}
	grpcErrorCodeFiles := []string{}
	for _, file := range files {
		if strings.Contains(file, "systemCode_http.go") || strings.Contains(file, "systemCode_rpc.go") {
			continue
		}
		if strings.HasSuffix(file, "_http.go") {
			httpErrorCodeFiles = append(httpErrorCodeFiles, file)
		} else if strings.HasSuffix(file, "_rpc.go") {
			grpcErrorCodeFiles = append(grpcErrorCodeFiles, file)
		}
	}

	return map[string][]string{
		httpType: httpErrorCodeFiles,
		grpcType: grpcErrorCodeFiles,
	}, nil
}

func getSubFiles(selectedFiles map[string][]string) []string {
	subFiles := []string{}
	for dir, files := range selectedFiles {
		for _, file := range files {
			subFiles = append(subFiles, dir+"/"+file)
		}
	}
	return subFiles
}

// ------------------------------------------------------------------------------------------

func getServiceName(varNames map[string]struct{}, body string) string {
	buf := bufio.NewReader(strings.NewReader(body))
	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimSpace(line)
		line = strings.ReplaceAll(line, " ", "")
		if strings.Contains(line, `Name="`) {
			sName := strings.TrimSuffix(strings.Split(line, `Name="`)[1], `"`)
			if _, ok := varNames[sName+"NO"]; ok {
				return sName
			}
		}
	}
	return ""
}

func findVarLineContent(varName string, src string) (srcLineCode string, valueStr string, err error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		return "", "", err
	}

	lines := strings.Split(src, "\n")

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.VAR {
			continue
		}

		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			for i, name := range valueSpec.Names {
				if name.Name != varName {
					continue
				}

				pos := name.Pos()
				position := fset.Position(pos)
				lineNumber := position.Line

				if lineNumber < 1 || lineNumber > len(lines) {
					return "", "", fmt.Errorf("invalid line number %d for variable %s", lineNumber, varName)
				}

				if i < len(valueSpec.Values) {
					if bl, ok := valueSpec.Values[i].(*ast.BasicLit); ok {
						valueStr = bl.Value
					}
				}

				return lines[lineNumber-1], valueStr, nil
			}
		}
	}

	return "", "", fmt.Errorf("variable %s not found", varName)
}

type Result struct {
	VarName        string
	BaseCodeOffset int
}

// Visitor is a visitor that walks the AST and extracts error code offsets.
type Visitor struct {
	PackageName  string
	CallFuncName string

	stack   []ast.Node
	results []Result
}

func (v *Visitor) parseParts(src string) ([]Result, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	ast.Walk(v, f)

	return v.results, nil
}

// Visit implements the ast.Visitor interface.
func (v *Visitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		if len(v.stack) > 0 {
			v.stack = v.stack[:len(v.stack)-1]
		}
		return nil
	}

	v.processNode(node)
	v.stack = append(v.stack, node)
	return v
}

func (v *Visitor) isNewErrorCall(callExpr *ast.CallExpr) bool {
	if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := selExpr.X.(*ast.Ident); ok && ident.Name == v.PackageName {
			return selExpr.Sel.Name == v.CallFuncName
		}
	}
	return false
}

// nolint
func (v *Visitor) processNode(node ast.Node) {
	// handling calls
	callExpr, ok := node.(*ast.CallExpr)
	if !ok || !v.isNewErrorCall(callExpr) {
		return
	}

	// get variable name
	var varName string
	for i := len(v.stack) - 1; i >= 0; i-- {
		switch n := v.stack[i].(type) {
		case *ast.ValueSpec:
			for _, value := range n.Values {
				if value == callExpr {
					if len(n.Names) > 0 {
						varName = n.Names[0].Name
					}
					break
				}
			}
		case *ast.AssignStmt:
			for _, rhs := range n.Rhs {
				if rhs == callExpr {
					if len(n.Lhs) > 0 {
						if ident, ok := n.Lhs[0].(*ast.Ident); ok {
							varName = ident.Name
						}
					}
					break
				}
			}
		}
		if varName != "" {
			break
		}
	}

	baseOffset := 0
	if len(callExpr.Args) > 0 {
		if binExpr, ok := callExpr.Args[0].(*ast.BinaryExpr); ok && binExpr.Op == token.ADD {
			if yLit, ok := binExpr.Y.(*ast.BasicLit); ok && yLit.Kind == token.INT {
				if val, err := strconv.Atoi(yLit.Value); err == nil {
					baseOffset = val
				}
			}
		}
	}

	if varName != "" && baseOffset != 0 {
		v.results = append(v.results, Result{
			VarName:        varName,
			BaseCodeOffset: baseOffset,
		})
	}
}
