package module

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"strings"

	"github.com/Eric-Guo/sponge/pkg/goast"
)

// define the extracted information structure
type middlewareFuncInfo struct {
	funcLineSrcCode string
	singlePaths     []*singlePath
}

func (i *middlewareFuncInfo) String() string {
	var str string
	for _, path := range i.singlePaths {
		str += fmt.Sprintf("name --> %s\n", path.lineContent)
	}
	return str
}

type singlePath struct {
	name        string // method->path
	lineContent string // whole line content
}

// extract information about setSinglePath calls from the function body
func extractSinglePaths(src string, fset *token.FileSet, fn *ast.FuncDecl) *middlewareFuncInfo {
	var singlePaths []*singlePath
	var middlewareFuncCode string

	// gets the starting and ending line numbers of the function body
	lineStart := fset.Position(fn.Body.Lbrace).Line
	lineEnd := fset.Position(fn.Body.Rbrace).Line
	lines := strings.Split(src, "\n")
	funcLines := lines[lineStart-1 : lineEnd]

	// regular expression matches c. setSinglePath ("Method "," Path ",...)
	spRe := regexp.MustCompile(`^\s*(//)?\s*c\.setSinglePath\("(\w+)",\s*"([^"]+)",.*\)`)

	for i, line := range funcLines {
		if i == 0 {
			middlewareFuncCode = line
		}
		matches := spRe.FindStringSubmatch(line)
		if matches != nil {
			method := matches[2] // first parameter: HTTP method
			path := matches[3]   // second parameter: Path
			name := method + "->" + path
			srcCode := line // whole line content
			sp := &singlePath{
				name:        name,
				lineContent: srcCode,
			}
			singlePaths = append(singlePaths, sp)
		}
	}
	return &middlewareFuncInfo{
		funcLineSrcCode: middlewareFuncCode,
		singlePaths:     singlePaths,
	}
}

func findNonExistedSinglePaths(srcMiddlewareFunc *middlewareFuncInfo,
	genMiddlewareFunc *middlewareFuncInfo) (srcCode string, targetCode string) {
	if genMiddlewareFunc == nil || len(genMiddlewareFunc.singlePaths) == 0 || srcMiddlewareFunc == nil {
		return "", ""
	}

	l := len(srcMiddlewareFunc.singlePaths)
	existingPaths := make(map[string]struct{}, l)
	lastSrcCode := ""
	for i, path := range srcMiddlewareFunc.singlePaths {
		existingPaths[path.name] = struct{}{}
		if i == l-1 {
			lastSrcCode = path.lineContent
		}
	}

	var uniquePaths []string
	for _, path := range genMiddlewareFunc.singlePaths {
		if _, exists := existingPaths[path.name]; !exists {
			uniquePaths = append(uniquePaths, path.lineContent)
		}
	}
	if len(uniquePaths) == 0 {
		return "", ""
	}

	joinStr := strings.Join(uniquePaths, "\n")
	if lastSrcCode == "" {
		lastSrcCode = srcMiddlewareFunc.funcLineSrcCode
		targetCode = lastSrcCode + "\n\n" + joinStr + "\n"
	} else {
		targetCode = lastSrcCode + "\n" + joinStr
	}

	return lastSrcCode, targetCode
}

// RouterCodeAst is the struct for router code
type RouterCodeAst struct {
	FilePath string
	SrcCode  string
	FileSize int
	AstInfos []*goast.AstInfo

	moduleCodeMap map[string]string // src code -> new code
	appendCodes   []string
}

func NewRouterCodeAst(filePath string) (*RouterCodeAst, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	astInfos, err := goast.ParseGoCode(filePath, data)
	if err != nil {
		return nil, err
	}

	return &RouterCodeAst{
		FilePath:      filePath,
		SrcCode:       string(data),
		FileSize:      len(data),
		AstInfos:      astInfos,
		moduleCodeMap: make(map[string]string, 0),
	}, nil
}

func (a *RouterCodeAst) parseMiddlewareFunc() (map[string]*middlewareFuncInfo, error) {
	var funcSinglePaths = map[string]*middlewareFuncInfo{}

	var funcName string
	for _, astInfo := range a.AstInfos {
		if len(astInfo.Names) > 0 {
			funcName = astInfo.Names[0]
		}
		if astInfo.Type != goast.FuncType && !strings.HasSuffix(funcName, "Middlewares") {
			continue
		}

		fset := token.NewFileSet()
		src := "package routers\n\n" + astInfo.Body
		node, err := parser.ParseFile(fset, a.FilePath, src, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		// traversing declarations in AST
		for _, decl := range node.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok {
				if strings.HasSuffix(fn.Name.Name, "Middlewares") {
					if len(fn.Type.Params.List) == 1 {
						param := fn.Type.Params.List[0]
						if starExpr, ok := param.Type.(*ast.StarExpr); ok {
							if ident, ok := starExpr.X.(*ast.Ident); ok && ident.Name == "middlewareConfig" {
								funcSinglePaths[funcName] = extractSinglePaths(src, fset, fn)
							}
						}
					}
				}
			}
		}
	}

	return funcSinglePaths, nil
}

// CompareExistedMiddlewareFunc compare the existed middlewareFunc in the source code
// and generated code, and find the non-existed singlePaths
func (a *RouterCodeAst) CompareExistedMiddlewareFunc(genAst *RouterCodeAst) error {
	srcMiddlewareFuncInfo, err := a.parseMiddlewareFunc()
	if err != nil {
		return err
	}
	genMiddlewareFuncInfo, err := genAst.parseMiddlewareFunc()
	if err != nil {
		return err
	}
	for name, genMfi := range genMiddlewareFuncInfo {
		if srcMfi, ok := srcMiddlewareFuncInfo[name]; ok {
			srcLineCode, targetCode := findNonExistedSinglePaths(srcMfi, genMfi)
			if srcLineCode != "" && targetCode != "" {
				a.moduleCodeMap[srcLineCode] = targetCode
			}
		}
	}
	return nil
}

// FindNonExistedName find non-existed names in the src code
func (a *RouterCodeAst) FindNonExistedName(genAsts []*goast.AstInfo) {
	srcNameAstMap := make(map[string]struct{}, len(a.AstInfos))
	for _, info := range a.AstInfos {
		name := strings.Join(info.Names, ",")
		srcNameAstMap[name] = struct{}{}
	}

	for _, genAst := range genAsts {
		if genAst.IsPackageType() || genAst.IsImportType() {
			continue
		}

		name := strings.Join(genAst.Names, ",")
		if _, ok := srcNameAstMap[name]; !ok {
			comment := ""
			if genAst.Comment != "" {
				comment = genAst.Comment
			}
			a.appendCodes = append(a.appendCodes, comment+"\n"+genAst.Body)
		}
	}
}

// ParseRouterCode parse router code from source and generated file, and merge them.
func ParseRouterCode(srcFile string, genFile string) (*RouterCodeAst, error) {
	srcAst, err := NewRouterCodeAst(srcFile)
	if err != nil {
		return nil, err
	}

	genAst, err := NewRouterCodeAst(genFile)
	if err != nil {
		return nil, err
	}

	// compare existing middlewareFunc
	err = srcAst.CompareExistedMiddlewareFunc(genAst)
	if err != nil {
		return nil, err
	}

	// get nonexistent variables
	srcAst.FindNonExistedName(genAst.AstInfos)

	// replace source code
	for oldStr, newStr := range srcAst.moduleCodeMap {
		srcAst.SrcCode = strings.ReplaceAll(srcAst.SrcCode, oldStr, newStr)
	}
	if len(srcAst.appendCodes) > 0 {
		srcAst.SrcCode += "\n" + strings.Join(srcAst.appendCodes, "\n\n") + "\n"
	}

	return srcAst, nil
}
