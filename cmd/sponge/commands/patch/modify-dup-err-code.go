package patch

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Eric-Guo/sponge/pkg/goast"
)

// ModifyDuplicateErrorCodeOffsetCommand modify duplicate error code offset command
func ModifyDuplicateErrorCodeOffsetCommand() *cobra.Command {
	var dir string

	cmd := &cobra.Command{
		Use:   "modify-dup-err-code",
		Short: "Modify duplicate error code offset",
		Long:  "Modify duplicate error code offset.",
		Example: color.HiBlackString(`  # Modify duplicate error code offset
  sponge patch modify-dup-err-code --dir=internal/ecode`),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			filesMap, err := listErrorCodeFiles(dir)
			if err != nil {
				fmt.Println("listErrCodeFiles:", err)
				return nil
			}

			for _, files := range filesMap {
				for _, file := range files {
					err = checkAndModifyDuplicateErrorCodeOffset(file)
					if err != nil {
						fmt.Println(err)
						return nil
					}
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&dir, "dir", "d", "internal/ecode", "input directory")

	return cmd
}

func checkAndModifyDuplicateErrorCodeOffset(file string) error {
	codeAst, err := NewErrorCodeOffsetAst(file)
	if err != nil {
		return fmt.Errorf("NewErrorCodeOffsetAst error: %v, file: %s", err, file)
	}

	isNeedSave := false
	for _, codeInfo := range codeAst.ErrorCodeNameInfoMap {
		codeAst.SrcCode, isNeedSave = codeInfo.CheckDuplicateErrorCodeOffset(codeAst.SrcCode)
	}

	if isNeedSave && codeAst.SrcCode != "" {
		return os.WriteFile(codeAst.FilePath, []byte(codeAst.SrcCode), 0666)
	}

	return nil
}

// ErrorCodeOffsetAst is the struct for error code offset
type ErrorCodeOffsetAst struct {
	FilePath string
	SrcCode  string
	FileSize int

	ErrorCodeNameInfoMap map[string]*ErrorCodeOffset
}

// ErrorCodeOffset is the struct for error code offset
type ErrorCodeOffset struct {
	Name string

	OffsetInfos []*Deconstruction
	offsetIndex int

	VarNameCodeSrcMap map[string]string // varName -> varErrCodeSrc
	Body              string
}

// Deconstruction is the struct for error code offset deconstruction
type Deconstruction struct {
	VarName    string
	VarValue   int
	VarSrcCode string
}

// NewErrorCodeOffsetAst create a ErrorCodeOffsetAst object
func NewErrorCodeOffsetAst(filePath string) (*ErrorCodeOffsetAst, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	astInfos, err := goast.ParseGoCode(filePath, data)
	if err != nil {
		return nil, err
	}

	var errorCodeOffsetInfoMap = map[string]*ErrorCodeOffset{}

	for _, info := range astInfos {
		if info.Type == goast.VarType {
			spliceType := 0
			if strings.Contains(info.Body, httpMark) {
				spliceType = 1
			} else if strings.Contains(info.Body, grpcMark) {
				spliceType = 2
			}

			if spliceType > 0 {
				var varNames = map[string]struct{}{}
				for _, varName := range info.Names {
					if strings.HasSuffix(varName, "NO") {
						if spliceType == 2 {
							varName = strings.TrimPrefix(varName, "_")
						}
						varNames[varName] = struct{}{}
					}
				}

				serviceName := getServiceName(varNames, info.Body)

				if serviceName == "" {
					continue
				}

				//varNOName := serviceName + "NO"
				callFuncName := "NewError"
				infoBody := "package ecode\n\nimport (\"github.com/Eric-Guo/sponge/pkg/errcode\")\n\n" + info.Body
				if spliceType == 2 {
					//varNOName = "_" + varNOName
					callFuncName = "NewRPCStatus"
				}

				vis := &Visitor{PackageName: "errcode", CallFuncName: callFuncName}
				results, err := vis.parseParts(infoBody)
				if err != nil {
					return nil, fmt.Errorf("parseParts error: %v", err)
				}

				var errCodes []*Deconstruction
				var varErrMap = make(map[string]string)
				for _, result := range results {
					varErrCodeSrc, _, err := findVarLineContent(result.VarName, infoBody)
					if err != nil {
						return nil, fmt.Errorf("findVarLineContent error: %v", err)
					}
					errCodes = append(errCodes, &Deconstruction{
						VarName:    result.VarName,
						VarValue:   result.BaseCodeOffset,
						VarSrcCode: varErrCodeSrc,
					})
					varErrMap[result.VarName] = varErrCodeSrc
				}
				if len(errCodes) != len(results) || len(errCodes) == 0 {
					continue
				}
				errCodesIndex := len(errCodes) - 1

				errorCodeOffsetInfoMap[serviceName] = &ErrorCodeOffset{
					Name:              serviceName,
					OffsetInfos:       errCodes,
					offsetIndex:       errCodesIndex,
					VarNameCodeSrcMap: varErrMap,
					Body:              info.Body,
				}
			}
		}
	}

	if len(errorCodeOffsetInfoMap) == 0 {
		return nil, fmt.Errorf("no error code found")
	}

	return &ErrorCodeOffsetAst{
		FilePath:             filePath,
		SrcCode:              string(data),
		FileSize:             len(data),
		ErrorCodeNameInfoMap: errorCodeOffsetInfoMap,
	}, nil
}

// CheckDuplicateErrorCodeOffset check if the error code offset has duplicate value
func (c *ErrorCodeOffset) CheckDuplicateErrorCodeOffset(srcCode string) (string, bool) {
	varValueMap := make(map[int][]string)
	duplicateNOMap := make(map[int][]string)
	maxVarValue := 0

	for _, r := range c.OffsetInfos {
		varValueMap[r.VarValue] = append(varValueMap[r.VarValue], r.VarName)
		if r.VarValue > maxVarValue {
			maxVarValue = r.VarValue
		}
	}

	for offset, names := range varValueMap {
		if len(names) > 1 {
			duplicateNOMap[offset] = names
		}
	}

	if len(duplicateNOMap) == 0 {
		return srcCode, false
	}

	return c.replaceErrorCodeOffset(srcCode, duplicateNOMap, maxVarValue)
}

func (c *ErrorCodeOffset) replaceErrorCodeOffset(srcCode string, duplicateNOMap map[int][]string, maxVarValue int) (string, bool) {
	isModified := false

	var sortNOs []int
	for varValue := range duplicateNOMap {
		sortNOs = append(sortNOs, varValue)
	}
	sort.Ints(sortNOs)

	for _, varValue := range sortNOs {
		varNames := duplicateNOMap[varValue]
		for i, name := range varNames {
			if i > 0 {
				oldStr := c.VarNameCodeSrcMap[name]
				maxVarValue++
				if maxVarValue > 99 {
					maxVarValue = 99
				}
				newStr := strings.ReplaceAll(oldStr, fmt.Sprintf("BaseCode+%d,", varValue), fmt.Sprintf("BaseCode+%d,", maxVarValue))
				srcCode = strings.ReplaceAll(srcCode, oldStr, newStr)
				isModified = true
			}
		}
	}

	return srcCode, isModified
}

// CheckMergedItems check if the merged items have duplicate error code offset
func (c *ErrorCodeOffset) CheckMergedItems(srcCode string, codeInfo *ErrorCodeOffset) (string, bool) {
	if c.Name != codeInfo.Name {
		return srcCode, false
	}

	var newErrCodes []*Deconstruction
	if codeInfo != nil && len(codeInfo.OffsetInfos) > 0 {
		for _, errCode := range codeInfo.OffsetInfos {
			if _, ok := c.VarNameCodeSrcMap[errCode.VarName]; !ok {
				newErrCodes = append(newErrCodes, errCode)
			}
		}
	}
	if len(newErrCodes) == 0 {
		return srcCode, false
	}

	var varSrcCodes []string
	for _, errCode := range newErrCodes {
		c.OffsetInfos = append(c.OffsetInfos, errCode)
		c.VarNameCodeSrcMap[errCode.VarName] = errCode.VarSrcCode
		varSrcCodes = append(varSrcCodes, errCode.VarSrcCode)
	}

	markerStr := c.OffsetInfos[c.offsetIndex].VarSrcCode
	joinStr := markerStr + "\n" + strings.Join(varSrcCodes, "\n")
	srcCode = strings.ReplaceAll(srcCode, markerStr, joinStr)

	return srcCode, true
}
