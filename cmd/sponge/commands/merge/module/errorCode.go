package module

import (
	"fmt"

	"github.com/Eric-Guo/sponge/cmd/sponge/commands/patch"
)

// ParseErrorCode parse error code from srcFile and genFile, and merge them.
func ParseErrorCode(srcFile string, genFile string) (*patch.ErrorCodeOffsetAst, bool, error) {
	srcAst, err := patch.NewErrorCodeOffsetAst(srcFile)
	if err != nil {
		return nil, false, fmt.Errorf("NewErrorCodeOffsetAst error: %v, file: %s", err, srcFile)
	}

	genAst, err := patch.NewErrorCodeOffsetAst(genFile)
	if err != nil {
		return nil, false, fmt.Errorf("NewErrorCodeOffsetAst error: %v, file: %s", err, genFile)
	}

	isNeedSave := false

	for sn, codeInfo := range genAst.ErrorCodeNameInfoMap {
		if ci, ok := srcAst.ErrorCodeNameInfoMap[sn]; ok {
			var isModified bool
			srcAst.SrcCode, isModified = ci.CheckMergedItems(srcAst.SrcCode, codeInfo)
			isNeedSave = isModified || isNeedSave
		} else {
			srcAst.SrcCode += "\n" + codeInfo.Body + "\n"
			srcAst.ErrorCodeNameInfoMap[sn] = codeInfo
			isNeedSave = true
		}
	}

	for _, codeInfo := range srcAst.ErrorCodeNameInfoMap {
		var isModified bool
		srcAst.SrcCode, isModified = codeInfo.CheckDuplicateErrorCodeOffset(srcAst.SrcCode)
		isNeedSave = isModified || isNeedSave
	}

	return srcAst, isNeedSave, nil
}
