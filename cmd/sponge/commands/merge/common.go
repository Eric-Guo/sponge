// Package merge is merge the generated code into the template file, you don't worry about it affecting
// the logic code you have already written, in case of accidents, you can find the
// pre-merge code in the directory /tmp/sponge_merge_backup_code
package merge

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Eric-Guo/sponge/cmd/sponge/commands/merge/module"
	"github.com/Eric-Guo/sponge/pkg/gobash"
	"github.com/Eric-Guo/sponge/pkg/gofile"
)

const defaultFuzzyFilename = "*.go.gen20*"

// path end with "/" or "\" will be removed
func adaptDir(dir string) string {
	if dir == " " || dir == "./" || dir == ".\\" {
		return "."
	}
	l := len(dir)
	if l > 0 && dir[l-1] == '/' {
		return dir[:l-1]
	}
	if dir[l-1] == '\\' {
		return dir[:l-1]
	}
	return dir
}

func getRelativeFilePath(srcFile string) string {
	dirPath, _ := filepath.Abs(".")
	return strings.TrimPrefix(srcFile, dirPath+gofile.GetPathDelimiter())
}

func getRelativeDirAndFile(srcFile string) (dir string, file string) {
	filePath := getRelativeFilePath(srcFile)

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

func deleteGenFiles(files []string) {
	for _, file := range files {
		_ = os.Remove(file)
	}
}

// ---------------------------------------------------------------------------------

type mergeType int

const (
	errCodeType           mergeType = 1
	routersType           mergeType = 2
	handlerType           mergeType = 3
	serviceGRPCTmplType   mergeType = 4
	serviceGRPCClientType mergeType = 5
)

type mergeParams struct {
	serverDir     string    // specify the server directory
	Type          mergeType // type of code to be merged
	genCodeDir    string    // directory where code is generated
	fuzzyFilename string    // fuzzy matching file name
	backupDir     string    // backup Code Catalog
}

func newMergeParams(dir string, dirType mergeType) *mergeParams {
	var genCodeDir string
	switch dirType {
	case errCodeType:
		genCodeDir = "internal" + string(os.PathSeparator) + "ecode"
	case routersType:
		genCodeDir = "internal" + string(os.PathSeparator) + "routers"
	case handlerType:
		genCodeDir = "internal" + string(os.PathSeparator) + "handler"
	case serviceGRPCTmplType, serviceGRPCClientType:
		genCodeDir = "internal" + string(os.PathSeparator) + "service"
	}

	return &mergeParams{
		serverDir:     adaptDir(dir),
		Type:          dirType,
		genCodeDir:    genCodeDir,
		fuzzyFilename: defaultFuzzyFilename,
		backupDir:     getBackupDir(),
	}
}

func (m *mergeParams) runMerge() error {
	fuzzyFilename := m.serverDir + string(os.PathSeparator) + m.genCodeDir + string(os.PathSeparator) + m.fuzzyFilename
	genFiles := gofile.FuzzyMatchFiles(fuzzyFilename) // "*.go.gen20*"
	var groupFiles = make(map[string]string)
	for _, genFile := range genFiles {
		filePrefix := strings.Split(genFile, ".go.gen20")
		if len(filePrefix) != 2 {
			continue
		}
		srcFile := filePrefix[0] + ".go"
		if gFile, ok := groupFiles[genFile]; !ok {
			groupFiles[srcFile] = genFile
		} else {
			if gFile < genFile {
				gFile = genFile
			}
			groupFiles[srcFile] = gFile
		}
	}

	var err error
	var deleteFiles []string
	switch m.Type {
	case errCodeType:
		deleteFiles, err = m.mergeErrCodeFile(groupFiles)
	case routersType:
		deleteFiles, err = m.mergeRoutersFile(groupFiles)
	case handlerType, serviceGRPCTmplType:
		deleteFiles, err = m.mergeHandlerAndServiceFile(groupFiles)
	case serviceGRPCClientType:
		deleteFiles, err = m.mergeServiceGRPCClientFile(groupFiles)
	default:
		return fmt.Errorf("unsupported merge type: %d", m.Type)
	}
	if err != nil {
		return err
	}

	// delete generated files
	deleteGenFiles(deleteFiles)

	return nil
}

func (m *mergeParams) mergeErrCodeFile(groupFiles map[string]string) ([]string, error) {
	var deleteFiles []string
	for srcFile, genFile := range groupFiles {
		if genFile == "" {
			continue
		}

		srcAst, isNeedSave, err := module.ParseErrorCode(srcFile, genFile)
		if err != nil {
			return nil, err
		}

		if isNeedSave && srcAst.SrcCode != "" {
			backupFile(srcAst.FilePath, m.backupDir)
			if err = os.WriteFile(srcAst.FilePath, []byte(srcAst.SrcCode), 0666); err != nil {
				return nil, err
			}
		}
		deleteFiles = append(deleteFiles, genFile)
	}
	return deleteFiles, nil
}

func (m *mergeParams) mergeRoutersFile(groupFiles map[string]string) ([]string, error) {
	var deleteFiles []string
	for srcFile, genFile := range groupFiles {
		if !gofile.IsExists(srcFile) && !gofile.IsExists(genFile) {
			return nil, nil
		}

		srcAst, err := module.ParseRouterCode(srcFile, genFile)
		if err != nil {
			return nil, err
		}

		if len(srcAst.SrcCode) != srcAst.FileSize {
			backupFile(srcAst.FilePath, m.backupDir)
			if err = os.WriteFile(srcAst.FilePath, []byte(srcAst.SrcCode), 0666); err != nil {
				return nil, err
			}
		}
		deleteFiles = append(deleteFiles, genFile)
	}
	return deleteFiles, nil
}

func (m *mergeParams) mergeHandlerAndServiceFile(groupFiles map[string]string) ([]string, error) {
	var deleteFiles []string
	for srcFile, genFile := range groupFiles {
		if strings.HasSuffix(srcFile, "_test.go") {
			continue
		}
		if !gofile.IsExists(srcFile) && !gofile.IsExists(genFile) {
			return nil, nil
		}

		srcAst, err := module.ParseHandlerAndServiceCode(srcFile, genFile)
		if err != nil {
			return nil, err
		}

		if len(srcAst.SrcCode) != srcAst.FileSize {
			backupFile(srcAst.FilePath, m.backupDir)
			if err = os.WriteFile(srcAst.FilePath, []byte(srcAst.SrcCode), 0666); err != nil {
				return nil, err
			}
		}
		deleteFiles = append(deleteFiles, genFile)
	}
	return deleteFiles, nil
}

func (m *mergeParams) mergeServiceGRPCClientFile(groupFiles map[string]string) ([]string, error) {
	var deleteFiles []string
	for srcFile, genFile := range groupFiles {
		if !strings.HasSuffix(srcFile, "_test.go") {
			continue
		}
		if !gofile.IsExists(srcFile) && !gofile.IsExists(genFile) {
			return nil, nil
		}

		srcAst, err := module.ParseGRPCMethodsTestAndBenchmarkCode(srcFile, genFile)
		if err != nil {
			return nil, err
		}

		if len(srcAst.SrcCode) != srcAst.FileSize {
			backupFile(srcAst.FilePath, m.backupDir)
			if err = os.WriteFile(srcAst.FilePath, []byte(srcAst.SrcCode), 0666); err != nil {
				return nil, err
			}
		}
		deleteFiles = append(deleteFiles, genFile)
	}
	return deleteFiles, nil
}
