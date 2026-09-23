package patch

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Eric-Guo/sponge/pkg/goast"
)

// ModifyDuplicateErrorCodeNumCommand Command modify duplicate error code numbers
func ModifyDuplicateErrorCodeNumCommand() *cobra.Command {
	var dir string

	cmd := &cobra.Command{
		Use:   "modify-dup-num",
		Short: "Modify duplicate error code numbers",
		Long:  "Modify duplicate error code numbers.",
		Example: color.HiBlackString(`  # Modify duplicate error code numbers
  sponge patch modify-dup-num --dir=internal/ecode`),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			files, err := listErrorCodeFiles(dir)
			if err != nil {
				fmt.Println("listErrCodeFiles:", err)
				return nil
			}

			err = checkAndModifyGoFileErrorCodeNO(files[httpType])
			if err != nil {
				fmt.Println("checkAndModifyGoFileErrorCodeNO[http]:", err)
			}
			err = checkAndModifyGoFileErrorCodeNO(files[grpcType])
			if err != nil {
				fmt.Println("checkAndModifyGoFileErrorCodeNO[grpc]:", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&dir, "dir", "d", "internal/ecode", "input directory")

	return cmd
}

// distinguish between _http.go and _rpc.go files, check and correct duplicate error code NO
func checkAndModifyGoFileErrorCodeNO(files []string) error {
	if len(files) < 1 {
		return nil
	}

	var noCodeAsts []*ErrorCodeNOAst
	for _, file := range files {
		asts, err := NewErrorCodeNOAst(file)
		if err != nil {
			return err
		}
		noCodeAsts = append(noCodeAsts, asts...)
	}

	return CheckAndModifyDuplicateErrorCodeNO(noCodeAsts)
}

// ErrorCodeNOAst is the struct for error code NO
type ErrorCodeNOAst struct {
	FilePath string
	SrcCode  string
	fileSize int

	VarName    string // example: userExampleNO
	VarValue   int    // example: 1
	VarSrcCode string // example: "userExampleNO = 1"
}

// SaveToFile save the modified source code to file
func (e *ErrorCodeNOAst) SaveToFile(maxErrorCodeNO int) error {
	oldStr := e.VarSrcCode
	newStr := "\t" + e.VarName + " = " + strconv.Itoa(maxErrorCodeNO)
	e.SrcCode = strings.Replace(e.SrcCode, oldStr, newStr, 1)
	if e.SrcCode == "" {
		return nil
	}

	return os.WriteFile(e.FilePath, []byte(e.SrcCode), 0666)
}

// CheckAndModifyDuplicateErrorCodeNO check and modify duplicate error code NO
func CheckAndModifyDuplicateErrorCodeNO(errorCodeNOs []*ErrorCodeNOAst) error {
	varValueMap := make(map[int][]*ErrorCodeNOAst)
	duplicateNOMap := make(map[int][]*ErrorCodeNOAst)
	maxErrorCodeNO := 0

	for _, e := range errorCodeNOs {
		varValueMap[e.VarValue] = append(varValueMap[e.VarValue], e)
		if e.VarValue > maxErrorCodeNO {
			maxErrorCodeNO = e.VarValue
		}
	}

	for varNOValue, es := range varValueMap {
		if len(es) > 1 {
			duplicateNOMap[varNOValue] = es
		}
	}

	if len(duplicateNOMap) == 0 {
		return nil
	}

	for _, es := range duplicateNOMap {
		sort.Slice(es, func(i, j int) bool {
			return es[i].VarValue > es[j].VarValue
		})
		for i, e := range es {
			if i > 0 {
				maxErrorCodeNO++
				if maxErrorCodeNO > 999 {
					maxErrorCodeNO = 999
				}
				if err := e.SaveToFile(maxErrorCodeNO); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// NewErrorCodeNOAst create a ErrorCodeNOAst object
func NewErrorCodeNOAst(filePath string) ([]*ErrorCodeNOAst, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	astInfos, err := goast.ParseGoCode(filePath, data)
	if err != nil {
		return nil, err
	}

	var errorCodeNOAsts []*ErrorCodeNOAst

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
				varNOName := serviceName + "NO"
				if spliceType == 2 {
					varNOName = "_" + varNOName
				}
				infoBody := "package ecode\n\nimport (\"github.com/Eric-Guo/sponge/pkg/errcode\")\n\n" + info.Body
				varNOCodeSrc, varNOValueStr, err := findVarLineContent(varNOName, infoBody)
				if err != nil {
					continue
				}
				varNOValue, _ := strconv.Atoi(varNOValueStr)

				errorCodeNOAsts = append(errorCodeNOAsts, &ErrorCodeNOAst{
					FilePath:   filePath,
					SrcCode:    string(data),
					fileSize:   len(data),
					VarName:    varNOName,
					VarValue:   varNOValue,
					VarSrcCode: varNOCodeSrc,
				})
			}
		}
	}

	return errorCodeNOAsts, nil
}
