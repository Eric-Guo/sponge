// Package module provides the functions to merge the code of two files.
package module

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"

	"github.com/Eric-Guo/sponge/pkg/goast"
)

// CodeAst is the struct for code
type CodeAst struct {
	FilePath string
	SrcCode  string
	FileSize int
	AstInfos []*goast.AstInfo

	replaceCodeMap         map[string]string   // src code -> new code
	excludeReceiverNameMap map[string]struct{} // receiver name -> exclude or not
	appendCodes            []string
}

func NewCodeAst(filePath string) (*CodeAst, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	astInfos, err := goast.ParseGoCode(filePath, data)
	if err != nil {
		return nil, err
	}

	return &CodeAst{
		FilePath:               filePath,
		SrcCode:                string(data),
		FileSize:               len(data),
		AstInfos:               astInfos,
		replaceCodeMap:         make(map[string]string, 0),
		excludeReceiverNameMap: make(map[string]struct{}, 0),
	}, nil
}

func (a *CodeAst) compareExistedImportCode(genAst *CodeAst) error {
	srcImportInfos, err := parseImportCode(a.AstInfos)
	if err != nil {
		return err
	}
	genImportInfos, err := parseImportCode(genAst.AstInfos)
	if err != nil {
		return err
	}

	var pkgBody string
	for _, info := range a.AstInfos {
		if info.IsPackageType() {
			pkgBody = info.Body
		}
	}

	var nonExistedImports []string
	if len(srcImportInfos) == 0 {
		for _, info := range genAst.AstInfos {
			if info.IsImportType() {
				if info.Comment != "" {
					nonExistedImports = append(nonExistedImports, info.Comment)
				}
				nonExistedImports = append(nonExistedImports, info.Body)
			}
		}

		if pkgBody != "" {
			a.replaceCodeMap[pkgBody] = pkgBody + "\n\n" + strings.Join(nonExistedImports, "\n")
		}
		return nil
	}

	srcImportInfoMap := make(map[string]struct{}, len(srcImportInfos))
	for _, srcIfi := range srcImportInfos {
		srcImportInfoMap[srcIfi.Path] = struct{}{}
	}

	for _, genIfi := range genImportInfos {
		if _, ok := srcImportInfoMap[genIfi.Path]; !ok {
			nonExistedImports = append(nonExistedImports, genIfi.Body)
		}
	}
	if len(nonExistedImports) > 0 {
		srcLastImportLine, isGroup := lastImportPath(a.AstInfos)
		if srcLastImportLine == "" {
			if pkgBody != "" {
				a.replaceCodeMap[pkgBody] = pkgBody + "\n" + strings.Join(nonExistedImports, "\n")
			}
			return nil
		}
		if isGroup {
			a.replaceCodeMap[srcLastImportLine] = srcLastImportLine + "\n" + strings.Join(nonExistedImports, "\n")
		} else {
			targetStr := strings.Replace(srcLastImportLine, "import ", "import (\n", 1)
			a.replaceCodeMap[srcLastImportLine] = targetStr + "\n" + strings.Join(nonExistedImports, "\n") + "\n)\n"
		}
	}

	return nil
}

func (a *CodeAst) compareExistedStructMethodsCode(genAst *CodeAst) error {
	srcImportInfoMap := parseMethodFuncCode(a.AstInfos)
	genImportInfoMap := parseMethodFuncCode(genAst.AstInfos)

	for structName, genMethods := range genImportInfoMap {
		var nonExistedImports []string
		var lastMethodFuncCode string
		if srcMethods, ok := srcImportInfoMap[structName]; ok {
			var srcMethodMap = make(map[string]struct{}, len(srcMethods))
			for i, srcMethod := range srcMethods {
				srcMethodMap[srcMethod.Name] = struct{}{}
				if i == len(srcMethods)-1 {
					lastMethodFuncCode = srcMethod.Body
				}
			}
			for _, genMethod := range genMethods {
				if _, isExisted := srcMethodMap[genMethod.Name]; !isExisted {
					nonExistedImports = append(nonExistedImports, genMethod.Comment+"\n"+genMethod.Body)
				}
			}
		}
		if len(nonExistedImports) > 0 {
			a.excludeReceiverNameMap[structName] = struct{}{}
			a.replaceCodeMap[lastMethodFuncCode] = lastMethodFuncCode + "\n\n" + strings.Join(nonExistedImports, "\n\n")
		}
	}

	return nil
}

func (a *CodeAst) findNonExistedCode(genAsts []*goast.AstInfo) {
	srcNameAstMap := make(map[string]struct{}, len(a.AstInfos))
	for _, info := range a.AstInfos {
		name := strings.Join(info.Names, ",")
		srcNameAstMap[name] = struct{}{}
	}

	for _, genAst := range genAsts {
		if genAst.IsPackageType() || genAst.IsImportType() {
			continue
		}
		if genAst.IsFuncType() && len(genAst.Names) == 2 {
			if _, ok := a.excludeReceiverNameMap[genAst.Names[1]]; ok {
				continue
			}
		}

		isNeedAppend := false
		name := strings.Join(genAst.Names, ",")
		if _, ok := srcNameAstMap[name]; !ok {
			isNeedAppend = true
		} else {
			if name == "_" && !strings.Contains(a.SrcCode, genAst.Body) {
				isNeedAppend = true
			}
		}
		if isNeedAppend {
			comment := ""
			if genAst.Comment != "" {
				comment = genAst.Comment
			}
			a.appendCodes = append(a.appendCodes, comment+"\n"+genAst.Body)
		}
	}
}

type serviceMethods struct {
	methodNames    []string
	nameCodeBlocks map[string]string
}

func (s *serviceMethods) lastMethodCode() string {
	if len(s.methodNames) == 0 {
		return ""
	}
	return s.nameCodeBlocks[s.methodNames[len(s.methodNames)-1]]
}

// nolint
func (a *CodeAst) compareExistedGRPCMethodsTestCode(genAst *CodeAst) error {
	var srcServiceMethodsMap = make(map[string]*serviceMethods, 0)
	for _, srcAstInfo := range a.AstInfos {
		if !srcAstInfo.IsFuncType() || len(srcAstInfo.Names) != 1 {
			continue
		}
		name := srcAstInfo.Names[0]
		if (strings.HasPrefix(name, "Test_service_") && strings.HasSuffix(name, "_methods")) ||
			(strings.HasPrefix(name, "Test_service_") && strings.HasSuffix(name, "_benchmark")) {
			methodNames, m, err := parseGRPCMethodsTestCode(srcAstInfo.Body)
			if err != nil {
				return err
			}
			srcServiceMethodsMap[name] = &serviceMethods{
				methodNames:    methodNames,
				nameCodeBlocks: m,
			}
		}
	}

	var genServiceMethodsMap = make(map[string]*serviceMethods, 0)
	for _, genAstInfo := range genAst.AstInfos {
		if !genAstInfo.IsFuncType() || len(genAstInfo.Names) != 1 {
			continue
		}
		name := genAstInfo.Names[0]
		if (strings.HasPrefix(name, "Test_service_") && strings.HasSuffix(name, "_methods")) ||
			(strings.HasPrefix(name, "Test_service_") && strings.HasSuffix(name, "_benchmark")) {
			methodNames, m, err := parseGRPCMethodsTestCode(genAstInfo.Body)
			if err != nil {
				return err
			}
			genServiceMethodsMap[name] = &serviceMethods{
				methodNames:    methodNames,
				nameCodeBlocks: m,
			}
		}
	}

	// compare method names
	for name, genMethods := range genServiceMethodsMap {
		if srcMethods, ok := srcServiceMethodsMap[name]; ok {
			var nonExistedMethods []string
			for _, genMethodName := range genMethods.methodNames {
				if _, isExisted := srcMethods.nameCodeBlocks[genMethodName]; !isExisted {
					nonExistedMethods = append(nonExistedMethods, genMethods.nameCodeBlocks[genMethodName])
				}
			}
			if len(nonExistedMethods) > 0 {
				lastMethodCode := srcMethods.lastMethodCode()
				a.replaceCodeMap[lastMethodCode] = lastMethodCode + ",\n\n" + strings.Join(nonExistedMethods, ",\n\n")
			}
		}
	}

	return nil
}

func lastImportPath(astInfos []*goast.AstInfo) (string, bool) {
	lastPath := ""
	isGroup := false
	for _, info := range astInfos {
		if info.IsImportType() {
			l := len(info.Names)
			if l == 0 {
				continue
			}
			if strings.Count(info.Body, "(") == 1 && strings.Count(info.Body, ")") == 1 {
				isGroup = true
				lastPath = info.Names[l-1]
			} else {
				isGroup = false
				lastPath = info.Body
			}
		}
	}
	return lastPath, isGroup
}

func (a *CodeAst) replaceCode() {
	for oldStr, newStr := range a.replaceCodeMap {
		a.SrcCode = strings.ReplaceAll(a.SrcCode, oldStr, newStr)
	}
	if len(a.appendCodes) > 0 {
		a.SrcCode += "\n" + strings.Join(a.appendCodes, "\n\n") + "\n"
	}
}

func parseImportCode(astInfos []*goast.AstInfo) ([]*goast.ImportInfo, error) {
	var importInfos []*goast.ImportInfo

	for _, astInfo := range astInfos {
		if !astInfo.IsImportType() {
			continue
		}

		imports, err := goast.ParseImportGroup(astInfo.Body)
		if err != nil {
			continue
		}
		importInfos = append(importInfos, imports...)
	}

	return importInfos, nil
}

func parseMethodFuncCode(astInfos []*goast.AstInfo) map[string][]*goast.MethodInfo {
	return goast.ParseStructMethods(astInfos)
}

// parse grpc methods test code block from source code
func parseGRPCMethodsTestCode(body string) ([]string, map[string]string, error) {
	var src = body
	if !strings.Contains(body, "package ") {
		src = "package demo\n\n" + src
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse source: %v", err)
	}

	var testsSlice *ast.CompositeLit

	// find assignments to tests variables
	ast.Inspect(f, func(n ast.Node) bool {
		if assign, ok := n.(*ast.AssignStmt); ok {
			for i, lhs := range assign.Lhs {
				if ident, ok := lhs.(*ast.Ident); ok && ident.Name == "tests" {
					if cl, ok := assign.Rhs[i].(*ast.CompositeLit); ok {
						testsSlice = cl
						return false
					}
				}
			}
		}
		return true
	})

	if testsSlice == nil {
		return nil, nil, fmt.Errorf("could not find tests slice declaration")
	}

	var methodNames []string
	var nameCodeBlocks = map[string]string{}

	// traversing the elements of the tests slice
	for _, elt := range testsSlice.Elts {
		compLit, ok := elt.(*ast.CompositeLit)
		if !ok {
			continue
		}

		for _, e := range compLit.Elts {
			kv, ok := e.(*ast.KeyValueExpr)
			if !ok {
				continue
			}

			if key, ok := kv.Key.(*ast.Ident); ok && key.Name == "name" {
				if bl, ok := kv.Value.(*ast.BasicLit); ok {
					name := strings.Trim(bl.Value, `"`)
					methodNames = append(methodNames, name)
					start := fset.Position(compLit.Lbrace).Offset
					end := fset.Position(compLit.Rbrace).Offset + 1
					nameCodeBlocks[name] = "\t\t" + src[start:end]
				}
			}
		}
	}

	return methodNames, nameCodeBlocks, nil
}

// ParseHandlerAndServiceCode parses the source code and generated code of the handler and service file,
func ParseHandlerAndServiceCode(srcFile string, genFile string) (*CodeAst, error) {
	srcAst, err := NewCodeAst(srcFile)
	if err != nil {
		return nil, err
	}

	genAst, err := NewCodeAst(genFile)
	if err != nil {
		return nil, err
	}

	// compare existing import path
	err = srcAst.compareExistedImportCode(genAst)
	if err != nil {
		return nil, err
	}

	// compare existing struct methods
	err = srcAst.compareExistedStructMethodsCode(genAst)
	if err != nil {
		return nil, err
	}

	// get nonexistent variables
	srcAst.findNonExistedCode(genAst.AstInfos)

	// replace source code
	srcAst.replaceCode()

	return srcAst, nil
}

// ParseGRPCMethodsTestAndBenchmarkCode parses the source code and generated code of the service file
func ParseGRPCMethodsTestAndBenchmarkCode(srcFile string, genFile string) (*CodeAst, error) {
	srcAst, err := NewCodeAst(srcFile)
	if err != nil {
		return nil, err
	}

	genAst, err := NewCodeAst(genFile)
	if err != nil {
		return nil, err
	}

	// compare existing import path
	err = srcAst.compareExistedImportCode(genAst)
	if err != nil {
		return nil, err
	}

	// compare existing struct methods
	err = srcAst.compareExistedGRPCMethodsTestCode(genAst)
	if err != nil {
		return nil, err
	}

	// get nonexistent variables
	srcAst.findNonExistedCode(genAst.AstInfos)

	// replace source code
	srcAst.replaceCode()

	return srcAst, nil
}
