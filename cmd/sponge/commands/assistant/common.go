package assistant

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/fatih/color"

	"github.com/Eric-Guo/sponge/pkg/aicli"
	"github.com/Eric-Guo/sponge/pkg/aicli/chatgpt"
	"github.com/Eric-Guo/sponge/pkg/aicli/deepseek"
	"github.com/Eric-Guo/sponge/pkg/aicli/gemini"
	"github.com/Eric-Guo/sponge/pkg/goast"
	"github.com/Eric-Guo/sponge/pkg/gobash"
	"github.com/Eric-Guo/sponge/pkg/gofile"
	"github.com/Eric-Guo/sponge/pkg/utils"
)

const (
	typeChatGPT  = "chatgpt"
	typeDeepSeek = "deepseek"
	typeGemini   = "gemini"

	gopherRoleDescCN = aicli.GopherRoleDescCN
	gopherRoleDescEN = aicli.GopherRoleDescEN

	successSymbol = "✓"
)

var assistantTypeMap = map[string]string{
	typeChatGPT:  "ChatGPT",
	typeDeepSeek: "DeepSeek",
	typeGemini:   "Gemini",
}

var defaultModelMap = map[string]string{
	typeChatGPT:  chatgpt.DefaultModel,
	typeDeepSeek: deepseek.DefaultModel,
	typeGemini:   gemini.DefaultModel,
}

type assistantParams struct {
	Type string

	apiKey        string
	model         string
	enableContext bool

	// only for chatgpt and deepseek
	roleDesc    string
	maxToken    int
	temperature float32
}

func (a *assistantParams) newClient() (aicli.Assistanter, error) {
	asstType := strings.ToLower(a.Type)
	switch asstType {
	case typeChatGPT, typeDeepSeek:
		var opts []chatgpt.ClientOption
		if a.model != "" {
			opts = append(opts, chatgpt.WithModel(a.model))
		}
		if a.enableContext {
			opts = append(opts, chatgpt.WithEnableContext())
		}
		if a.roleDesc != "" {
			opts = append(opts, chatgpt.WithInitialRole(a.roleDesc))
		}
		if a.maxToken > 0 {
			opts = append(opts, chatgpt.WithMaxTokens(a.maxToken))
		}
		if a.temperature > 0 {
			opts = append(opts, chatgpt.WithTemperature(a.temperature))
		}

		if asstType == typeChatGPT {
			return chatgpt.NewClient(a.apiKey, opts...)
		} else if asstType == typeDeepSeek {
			return deepseek.NewClient(a.apiKey, opts...)
		}

	case typeGemini:
		var opts []gemini.ClientOption
		if a.model != "" {
			opts = append(opts, gemini.WithModel(a.model))
		}
		if a.enableContext {
			opts = append(opts, gemini.WithEnableContext())
		}
		return gemini.NewClient(a.apiKey, opts...)
	}

	return nil, fmt.Errorf("unsupported assistant type: %s", a.Type)
}

// --------------------------------------------------------------------------

const (
	gormDao = `package dao

import (
	"context"
	"errors"

	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"

	"github.com/Eric-Guo/sponge/pkg/logger"
	"github.com/Eric-Guo/sponge/pkg/sgorm/query"
	"github.com/Eric-Guo/sponge/pkg/utils"

	"github.com/Eric-Guo/sponge/internal/cache"
	"github.com/Eric-Guo/sponge/internal/database"
	"github.com/Eric-Guo/sponge/internal/model"
)

var _ UserExampleDao = (*userExampleDao)(nil)

// UserExampleDao defining the dao interface
type UserExampleDao interface {
	Create(ctx context.Context, table *model.UserExample) error
	UpdateByID(ctx context.Context, table *model.UserExample) error
	GetByID(ctx context.Context, id uint64) (*model.UserExample, error)

	CreateByTx(ctx context.Context, tx *gorm.DB, table *model.UserExample) (uint64, error)
	DeleteByTx(ctx context.Context, tx *gorm.DB, id uint64) error
	UpdateByTx(ctx context.Context, tx *gorm.DB, table *model.UserExample) error
}

type userExampleDao struct {
	db    *gorm.DB
	cache cache.UserExampleCache // if nil, the cache is not used.
	sfg   *singleflight.Group    // if cache is nil, the sfg is not used.
}

// NewUserExampleDao creating the dao interface
func NewUserExampleDao(db *gorm.DB, xCache cache.UserExampleCache) UserExampleDao {
	if xCache == nil {
		return &userExampleDao{db: db}
	}
	return &userExampleDao{
		db:    db,
		cache: xCache,
		sfg:   new(singleflight.Group),
	}
}

`

	mongoDao = `package dao

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/sync/singleflight"

	"github.com/Eric-Guo/sponge/pkg/logger"
	"github.com/Eric-Guo/sponge/pkg/mgo"
	"github.com/Eric-Guo/sponge/pkg/mgo/query"

	"github.com/Eric-Guo/sponge/internal/cache"
	"github.com/Eric-Guo/sponge/internal/database"
	"github.com/Eric-Guo/sponge/internal/model"
)

var _ UserExampleDao = (*userExampleDao)(nil)

// UserExampleDao defining the dao interface
type UserExampleDao interface {
	Create(ctx context.Context, record *model.UserExample) error
	UpdateByID(ctx context.Context, record *model.UserExample) error
	GetByID(ctx context.Context, id string) (*model.UserExample, error)
}

type userExampleDao struct {
	collection *mongo.Collection
	cache      cache.UserExampleCache // if nil, the cache is not used.
	sfg        *singleflight.Group    // if cache is nil, the sfg is not used.
}

// NewUserExampleDao creating the dao interface
func NewUserExampleDao(collection *mongo.Collection, xCache cache.UserExampleCache) UserExampleDao {
	if xCache == nil {
		return &userExampleDao{collection: collection}
	}
	return &userExampleDao{
		collection: collection,
		cache:      xCache,
		sfg:        new(singleflight.Group),
	}
}

`

	codeDelimiterMarker = "/**code-delimiter**/"
)

// nolint
var ErrnoAssistantMarker = fmt.Errorf("\n%s%s\n\n",
	`No Go code requiring AI assistant generation was detected. To trigger the AI assistant, ensure the following conditions are met:
    1. Define a function in Go code.
    2. Add detailed function comments (used as AI prompts).
    3. Add panic("implement me") inside the function body.

Example:`, fmt.Sprintf(`

    %s
    func FunctionName() {
        %s
    }`, color.HiCyanString("// Describe the specific functionality of the function"), color.HiCyanString(`panic("implement me")`)))

// get dao file path
func getDaoFilePath(path string) string {
	path = filepath.ToSlash(path)
	parts := strings.Split(path, "/")

	for i := len(parts) - 2; i >= 0; i-- {
		if parts[i] == "handler" || parts[i] == "service" || parts[i] == "biz" || parts[i] == "logic" {
			parts[i] = "dao"
			break
		}
	}

	return strings.Join(parts, "/")
}

type daoCodeInfo struct {
	structName    string
	interfaceName string
	methodNames   string
	code          string
}

func parseDaoCode(filePath string) (*daoCodeInfo, error) {
	astInfos, err := goast.ParseFile(filePath)
	if err != nil {
		return nil, err
	}

	var daoCode []string
	var methodNames []string
	var structNames []string

	for _, astInfo := range astInfos {
		if astInfo.IsFuncType() {
			if len(astInfo.Names) == 2 {
				methodNames = append(methodNames, astInfo.Names[0])
				if len(structNames) == 0 {
					structNames = append(structNames, astInfo.Names[1])
				} else {
					if structNames[len(structNames)-1] != astInfo.Names[1] {
						structNames = append(structNames, astInfo.Names[1])
					}
				}
			}
			if !(len(astInfo.Names) == 1 && strings.HasPrefix(astInfo.Names[0], "New")) {
				continue
			}
		}

		if strings.Contains(astInfo.Body, "var total int64") {
			continue
		}

		if astInfo.Comment != "" {
			daoCode = append(daoCode, astInfo.Comment+"\n"+astInfo.Body)
		} else {
			daoCode = append(daoCode, astInfo.Body)
		}
	}

	if len(daoCode) == 0 {
		return nil, fmt.Errorf("no dao code found in %s", filePath)
	}

	var interfaceNames []string
	for _, structName := range structNames {
		interfaceNames = append(interfaceNames, capitalize(structName))
	}

	return &daoCodeInfo{
		code:          strings.Join(daoCode, "\n\n"),
		structName:    strings.Join(structNames, ", "),
		methodNames:   `"` + strings.Join(methodNames, `", "`) + `"`,
		interfaceName: strings.Join(interfaceNames, ", "),
	}, nil
}

type daoInfo struct {
	code          string
	structName    string
	methodNames   string
	interfaceName string
}

func newDaoInfo(daoFile string, isMongo bool, objName string, isChinese bool) *daoInfo {
	var (
		isUseDefaultDao  = true
		daoCode          string
		daoStructName    = objName + "Dao"
		daoInterfaceName = capitalize(objName) + "Dao"
		daoMethodNames   = getDaoDefaultMethodNames(isMongo, isChinese)
	)

	if gofile.IsExists(daoFile) {
		dci, err := parseDaoCode(daoFile)
		if err == nil {
			isUseDefaultDao = false
			daoCode = dci.code
			daoStructName = dci.structName
			daoInterfaceName = dci.interfaceName
			daoMethodNames = dci.methodNames
		}
	}
	if isUseDefaultDao {
		if isMongo {
			daoCode = mongoDao
		} else {
			daoCode = gormDao
		}
		daoCode = strings.Replace(daoCode, "UserExample", capitalize(objName), -1)
		daoCode = strings.Replace(daoCode, "userExample", objName, -1)
	}

	decs := ""
	if isChinese {
		decs = fmt.Sprintf(`

// 已折叠隐藏 %s 方法函数的代码块。
// 由于 "%s" 结构体已实现了 "%s" 接口中定义的所有方法，因此无需再额外创建或实现这些方法。
// 如需使用相关功能，直接调用对应方法即可。`,
			daoMethodNames, daoStructName, daoInterfaceName)
	} else {
		decs = fmt.Sprintf(`

// The code block for the method %s has been collapsed and hidden.
// Since the struct "%s" already implements all methods defined in the "%s" interface, there is no need to create or implement these methods again.
// If needed, you can directly call the corresponding methods.`,
			daoMethodNames, daoStructName, daoInterfaceName)
	}

	return &daoInfo{
		code:          daoCode + decs + "\n\n",
		structName:    daoStructName,
		methodNames:   daoMethodNames,
		interfaceName: daoInterfaceName,
	}
}

func getDaoDefaultMethodNames(isMongo bool, isChinese bool) string {
	baseFuncNames := `"Create", "DeleteByID", "UpdateByID", "GetByID"`
	if !isMongo {
		baseFuncNames += `, "CreateByTx", "DeleteByTx", "UpdateByTx"`
	}
	if isChinese {
		baseFuncNames = strings.ReplaceAll(baseFuncNames, ", ", "、")
	}
	return baseFuncNames
}

func getModelCode(file string, dirName string, fileName string, isChinese bool) string {
	modelFile := strings.TrimSuffix(gofile.GetDir(file), dirName) + "model" + string(filepath.Separator) + fileName
	data, err := os.ReadFile(modelFile)
	if err != nil {
		return ""
	}

	modelCode := ""
	modelFile = "internal" + string(filepath.Separator) + "model" + string(filepath.Separator) + fileName
	if isChinese {
		modelCode = fmt.Sprintf("\n\n原始文件 %s 代码如下：", modelFile)
	} else {
		modelCode = fmt.Sprintf("\n\nThe original file %s code is as follows:", modelFile)
	}

	return modelCode + fmt.Sprintf("\n```go\n%s\n```", string(data))
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// extractGoCode extracts the Go code blocks from the given markdown string.
func extractGoCode(markdown string) []string {
	var goCodeBlocks []string
	scanner := bufio.NewScanner(strings.NewReader(markdown))
	inCodeBlock := false
	var currentBlock strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		// dealing with new ```go
		if strings.HasPrefix(line, "```go") {
			// if it is already in the code block, it means that the previous code block is missing the close identifier and stores it first.
			if inCodeBlock {
				goCodeBlocks = append(goCodeBlocks, currentBlock.String())
				currentBlock.Reset()
			}
			inCodeBlock = true
			continue
		}

		// processing ``` close code block
		if inCodeBlock && strings.HasPrefix(line, "```") {
			inCodeBlock = false
			goCodeBlocks = append(goCodeBlocks, currentBlock.String())
			currentBlock.Reset()
			continue
		}

		// record code content
		if inCodeBlock {
			currentBlock.WriteString(line)
			currentBlock.WriteString("\n")
		}
	}

	// prevents the last code block from not closing
	if inCodeBlock && currentBlock.Len() > 0 {
		goCodeBlocks = append(goCodeBlocks, currentBlock.String())
	}

	return goCodeBlocks
}

func parseCode(code string) []string {
	n := strings.Count(code, "```go")
	switch n {
	case 0:
		if strings.Contains(code, codeDelimiterMarker) {
			ss := strings.Split(code, codeDelimiterMarker)
			var codes []string
			for _, s := range ss {
				if strings.Contains(s, "package ") {
					codes = append(codes, s)
				}
			}
			if len(codes) > 0 {
				return reassembleGoMarkdown(codes)
			}
		}
		if strings.Contains(code, "\npackage ") {
			return reassembleGoMarkdown([]string{code})
		}
		return []string{code}
	case 1:
		goCodes := extractGoCode(code)
		var codes []string
		for _, goCode := range goCodes {
			codes = append(codes, strings.Split(goCode, codeDelimiterMarker)...)
		}
		return reassembleGoMarkdown(codes)
	}

	code = strings.ReplaceAll(code, codeDelimiterMarker, "\n\n")
	return reassembleGoMarkdown(extractGoCode(code))
}

func reassembleGoMarkdown(codes []string) []string {
	for i, c := range codes {
		codes[i] = "```go\n" + strings.TrimSpace(c) + "\n```\n"
	}
	return codes
}

func cutFilePath(fullPath string) string {
	cwd, _ := os.Getwd()
	if strings.HasPrefix(fullPath, cwd) {
		return strings.TrimPrefix(fullPath, cwd+string(filepath.Separator))
	}
	return fullPath
}

func newPrintLog(t ...time.Duration) *utils.WaitPrinter {
	if len(t) > 0 {
		time.Sleep(t[0])
	}
	p := utils.NewWaitPrinter(time.Millisecond * 250)
	p.LoopPrint("Waiting for assistant responses ")
	return p
}

func getAssistantSuffixed(assistantType string) string {
	return ".go." + assistantType + ".md"
}

func deleteGenFiles(files []string, assistantType string) {
	var doneFiles []string
	for _, file := range files {
		err := os.Remove(file)
		if err != nil {
			continue
		}
		doneFiles = append(doneFiles, cutFilePath(file))
	}

	if len(doneFiles) > 0 {
		fmt.Printf("Removed temporary files generated by AI Assistant (%s):\n", assistantType)
		for _, file := range doneFiles {
			fmt.Printf("    %s\n", color.HiGreenString(cutFilePath(file)))
		}
		fmt.Println()
	}
}

func getRelativeDirAndFile(srcFile string) (dir string, file string) {
	dirPath, _ := filepath.Abs(".")
	filePath := strings.TrimPrefix(srcFile, dirPath+gofile.GetPathDelimiter())

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return srcFile, ""
	}

	if !fileInfo.IsDir() {
		dir = gofile.GetDir(filePath)
		file = gofile.GetFilename(filePath)
		return dir, file
	}
	return filePath, ""
}

func getBackupDir() string {
	var backupDir = os.TempDir() + string(os.PathSeparator) + "sponge_merge_backup_code"
	return backupDir + string(os.PathSeparator) + time.Now().Format("20060102T150405")
}

func backupFile(file string, backupDir string) {
	relPath, _ := getRelativeDirAndFile(file)
	bkDir := backupDir + string(os.PathSeparator) + relPath + string(os.PathSeparator)
	_ = os.MkdirAll(bkDir, 0744)
	_, _ = gobash.Exec("cp", file, bkDir)
}
