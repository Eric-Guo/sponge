package server

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/Eric-Guo/sponge/pkg/errcode"
	"github.com/Eric-Guo/sponge/pkg/gin/response"
	"github.com/Eric-Guo/sponge/pkg/gobash"
	"github.com/Eric-Guo/sponge/pkg/gofile"
	"github.com/Eric-Guo/sponge/pkg/krand"
	"github.com/Eric-Guo/sponge/pkg/mgo"
	"github.com/Eric-Guo/sponge/pkg/process"
	"github.com/Eric-Guo/sponge/pkg/sgorm"
	"github.com/Eric-Guo/sponge/pkg/sgorm/mysql"
	"github.com/Eric-Guo/sponge/pkg/sgorm/postgresql"
	"github.com/Eric-Guo/sponge/pkg/sgorm/sqlite"
	"github.com/Eric-Guo/sponge/pkg/utils"
)

var (
	recordDirName = "sponge_record"
	saveDir       = fmt.Sprintf("%s/.%s", getSpongeDir(), recordDirName)
)

type dbInfoForm struct {
	Dsn      string `json:"dsn" binding:"required"`
	DbDriver string `json:"dbDriver"`
}

type kv struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// ListDbDrivers list db drivers
func ListDbDrivers(c *gin.Context) {
	dbDrivers := []string{
		sgorm.DBDriverMysql,
		mgo.DBDriverName,
		sgorm.DBDriverPostgresql,
		sgorm.DBDriverTidb,
		sgorm.DBDriverSqlite,
	}

	data := []kv{}
	for _, driver := range dbDrivers {
		data = append(data, kv{
			Label: driver,
			Value: driver,
		})
	}

	response.Success(c, data)
}

// ListLLM list llm info
func ListLLM(c *gin.Context) {
	llmTypes := []string{
		"deepseek",
		"chatgpt",
		"gemini",
	}

	llmTypeOptions := []kv{}
	for _, driver := range llmTypes {
		llmTypeOptions = append(llmTypeOptions, kv{
			Label: driver,
			Value: driver,
		})
	}

	allLLMOptions := map[string][]kv{
		"deepseek": {
			{Label: "deepseek-chat", Value: "deepseek-chat"},
			{Label: "deepseek-reasoner", Value: "deepseek-reasoner"},
		},
		"chatgpt": {
			{Label: "gpt-5", Value: "gpt-5"},
			{Label: "gpt-5-thinking", Value: "gpt-5-thinking"},
			{Label: "gpt-4.1", Value: "gpt-4.1"},
			{Label: "gpt-4.1-mini", Value: "gpt-4.1-mini"},
			{Label: "gpt-4o", Value: "gpt-4o"},
			{Label: "gpt-4o-mini", Value: "gpt-4o-mini"},
		},
		"gemini": {
			{Label: "gemini-2.5-flash", Value: "gemini-2.5-flash"},
			{Label: "gemini-2.5-pro", Value: "gemini-2.5-pro"},
			{Label: "gemini-2.5-flash-lite", Value: "gemini-2.5-flash-lite"},
			{Label: "gemini-2.0-flash", Value: "gemini-2.0-flash"},
			{Label: "gemini-2.0-pro", Value: "gemini-2.0-pro"},
		},
	}

	response.Success(c, gin.H{
		"llmTypeOptions": llmTypeOptions,
		"allLLMOptions":  allLLMOptions,
	})
}

// ListTables list tables
func ListTables(c *gin.Context) {
	form := &dbInfoForm{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		response.Error(c, errcode.InvalidParams.RewriteMsg(err.Error()))
		return
	}

	var tables []string
	switch strings.ToLower(form.DbDriver) {
	case sgorm.DBDriverMysql, sgorm.DBDriverTidb:
		tables, err = getMysqlTables(form.Dsn)
	case sgorm.DBDriverPostgresql:
		tables, err = getPostgresqlTables(form.Dsn)
	case sgorm.DBDriverSqlite:
		tables, err = getSqliteTables(form.Dsn)
	case mgo.DBDriverName:
		tables, err = getMongodbTables(form.Dsn)
	case "":
		response.Error(c, errcode.InvalidParams.RewriteMsg("database type cannot be empty"))
		return
	default:
		response.Error(c, errcode.InvalidParams.RewriteMsg("unsupported database type: "+form.DbDriver))
		return
	}
	if err != nil {
		response.Error(c, errcode.InternalServerError.RewriteMsg(err.Error()))
		return
	}

	data := []kv{}
	for _, table := range tables {
		data = append(data, kv{
			Label: table,
			Value: table,
		})
	}

	response.Success(c, data)
}

// GenerateCodeForm generate code form
type GenerateCodeForm struct {
	Arg  string `json:"arg" binding:"required"`
	Path string `json:"path" binding:"required"`
}

// GenerateCode generate code
func GenerateCode(c *gin.Context) {
	// Allow getting the value of the request header when crossing domains
	c.Writer.Header().Set("Access-Control-Expose-Headers", "content-disposition, err-msg")

	form := &GenerateCodeForm{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		responseErr(c, err, errcode.InvalidParams)
		return
	}

	handleGenerateCode(c, form.Path, form.Arg)
}

// GetTemplateInfo get template info
func GetTemplateInfo(c *gin.Context) {
	form := &GenerateCodeForm{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		responseErr(c, err, errcode.InvalidParams)
		return
	}

	handleGenerateCode(c, form.Path, form.Arg)
}

// nolint
func handleGenerateCode(c *gin.Context, outPath string, arg string) {
	out := "-" + time.Now().Format("150405")
	if len(outPath) > 1 {
		if outPath[0] == '/' {
			out = outPath[1:] + out
		} else {
			out = outPath + out
		}
	}

	args := strings.Split(arg, " ")
	params := parseCommandArgs(args)
	if params.ServerName != "" {
		out = params.ServerName + "-" + out
	} else {
		if params.ModuleName != "" {
			out = params.ModuleName + "-" + out
		}
	}
	if params.SuitedMonoRepo {
		out += "-mono-repo"
	}

	out = os.TempDir() + gofile.GetPathDelimiter() + "sponge-generate-code" + gofile.GetPathDelimiter() + out
	args = append(args, fmt.Sprintf("--out=%s", out))

	ctx, _ := context.WithTimeout(context.Background(), time.Minute*2) // nolint
	result := gobash.Run(ctx, "sponge", args...)
	resultInfo := ""
	count := 0
	for v := range result.StdOut {
		count++
		if count == 1 { // first line is the command
			continue
		}
		resultInfo += v
	}
	if result.Err != nil {
		if params.OnlyPrint {
			response.Out(c, errcode.InternalServerError.RewriteMsg(result.Err.Error()))
		} else {
			responseErr(c, result.Err, errcode.InternalServerError)
		}
		return
	}

	if params.OnlyPrint {
		response.Success(c, resultInfo)
		return
	}

	zipFile := out + ".zip"
	var err error
	if gofile.IsWindows() {
		err = AdaptToWindowsZip(out, zipFile)
	} else {
		err = CompressPathToZip(out, zipFile)
	}
	if err != nil {
		responseErr(c, err, errcode.InternalServerError)
		return
	}

	if !gofile.IsExists(zipFile) {
		err = errors.New("no found file " + zipFile)
		responseErr(c, err, errcode.InternalServerError)
		return
	}

	c.Writer.Header().Set("Content-Type", "application/zip")
	c.Writer.Header().Set("Content-Disposition", gofile.GetFilename(zipFile))
	c.File(zipFile)

	recordObj().set(c.ClientIP(), outPath, params)

	go func() {
		ctx, _ := context.WithTimeout(context.Background(), time.Minute*10)

		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second * 5):
				err := os.RemoveAll(out)
				if err != nil {
					continue
				}
				err = os.RemoveAll(zipFile)
				if err != nil {
					continue
				}

				if params.ProtobufFile != "" && strings.Contains(params.ProtobufFile, recordDirName) {
					err = os.RemoveAll(gofile.GetFileDir(params.ProtobufFile))
					if err != nil {
						continue
					}
				}
				if params.YamlFile != "" && strings.Contains(params.YamlFile, recordDirName) {
					err = os.RemoveAll(gofile.GetFileDir(params.YamlFile))
					if err != nil {
						continue
					}
				}
				return
			}
		}
	}()
}

// HandleAssistant handle assistant generate and merge code
func HandleAssistant(c *gin.Context) {
	form := &GenerateCodeForm{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		responseErr(c, err, errcode.InvalidParams)
		return
	}

	args := strings.Split(form.Arg, " ")
	params := parseCommandArgs(args)

	ctx, _ := context.WithTimeout(context.Background(), time.Minute*60) // nolint
	result := gobash.Run(ctx, "sponge", args...)
	resultInfo := ""
	count := 0
	for v := range result.StdOut {
		count++
		if count == 1 { // first line is the command
			continue
		}
		if strings.Contains(v, "Waiting for assistant responses") {
			continue
		}
		resultInfo += v
	}
	if result.Err != nil {
		responseErr(c, result.Err, errcode.InternalServerError)
		return
	}

	recordObj().set(c.ClientIP(), form.Path, params)

	response.Success(c, resultInfo)
}

var processMap = sync.Map{}

func getCommand(args []string) string {
	if len(args) < 4 {
		return ""
	}
	commandArgs := []string{"sponge", args[0], args[1]}
	for _, arg := range args {
		if strings.Contains(arg, "--url") {
			commandArgs = append(commandArgs, arg)
		}
		if strings.Contains(arg, "--method") {
			commandArgs = append(commandArgs, arg)
		}
	}
	sort.Strings(commandArgs)
	return strings.Join(commandArgs, "&")
}

func addProcess(key string, pid int) {
	processMap.Store(key, pid)
}

func getPid(key string) (int, bool) {
	value, ok := processMap.Load(key)
	if !ok {
		return -1, false
	}
	valueInt, ok := value.(int)
	if !ok {
		return -1, false
	}
	return valueInt, true
}

func removeProcess(key string) {
	processMap.Delete(key)
}

// HandlePerformanceTest handle performance test
func HandlePerformanceTest(c *gin.Context) {
	form := &GenerateCodeForm{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		responseErr(c, err, errcode.InvalidParams)
		return
	}

	args := strings.Split(form.Arg, " ")
	params := parseCommandArgs(args)
	if len(args) > 2 {
		params.Protocol = args[1]
	}
	if params.TotalRequests > 0 {
		params.TestType = "requests"
	} else if params.Duration != "" {
		params.TestType = "duration"
	}
	if params.JobName != "" && params.PushURL != "" {
		params.PushType = "prometheus"
		params.PrometheusURL = params.PushURL
		params.PushURL = ""
	} else if params.PushURL != "" {
		params.PushType = "custom"
	}

	ctx, _ := context.WithTimeout(context.Background(), time.Hour*24) // nolint
	result := gobash.Run(ctx, "sponge", args...)
	key := ""
	pid := 0
	resultInfo := ""
	count := 0
	for v := range result.StdOut {
		count++
		if count == 1 { // first line is the command and pid value
			pid = gobash.ParsePid(v)
			if pid > 0 {
				key = getCommand(args)
				if result.Pid > 0 {
					addProcess(key, result.Pid)
				}
			}
			continue
		}
		if strings.Contains(v, "Waiting for assistant responses") {
			continue
		}
		resultInfo += v
	}
	if result.Err != nil {
		responseErr(c, result.Err, errcode.InternalServerError)
		return
	}

	recordObj().set(c.ClientIP(), form.Path, params)

	if result.Pid > 0 {
		removeProcess(key)
	}

	lineContent, reportStr := splitString(resultInfo, "Performance Test Report ==========")
	command := "sponge " + strings.Join(args, " ")
	insertStr := fmt.Sprintf(`

[Overview]
  • %-19s%s
  • %-19s%s`, "Command:", command,
		"End Time:", time.Now().Format(time.DateTime))
	reportStr = strings.ReplaceAll(reportStr, lineContent, lineContent+insertStr)

	response.Success(c, reportStr)
}

// HandleStopPerformanceTest handle stop performance test
func HandleStopPerformanceTest(c *gin.Context) {
	form := &GenerateCodeForm{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		responseErr(c, err, errcode.InvalidParams)
		return
	}

	args := strings.Split(form.Arg, " ")
	key := getCommand(args)
	pid, _ := getPid(key)
	if pid > 0 {
		err = process.Kill(pid)
		if err != nil {
			responseErr(c, err, errcode.InternalServerError)
			return
		}
		removeProcess(key)
	}

	response.Success(c)
}

func splitString(str string, sep string) (lineContent string, out string) {
	lines := strings.Split(str, "\n")
	startIndex := 0
	isFound := false

	for i, line := range lines {
		if strings.Contains(line, sep) {
			isFound = true
			startIndex = i
			lineContent = line
			break
		}
	}

	if isFound {
		return lineContent, strings.Join(lines[startIndex:], "\n")
	}
	return "", str
}

// GetRecord generate run command record
func GetRecord(c *gin.Context) {
	pathParam := c.Param("path")
	if pathParam == "" {
		response.Out(c, errcode.InvalidParams.RewriteMsg("path param is empty"))
		return
	}

	params := recordObj().get(c.ClientIP(), pathParam)
	if params == nil {
		params = &parameters{Embed: true}
	}

	response.Success(c, params)
}

func responseErr(c *gin.Context, err error, ec *errcode.Error) {
	k := "err-msg"
	e := ec.RewriteMsg(err.Error())
	c.Writer.Header().Set(k, e.Msg())
	response.Out(c, e)
}

// UploadFiles batch files upload
func UploadFiles(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		response.Error(c, errcode.InvalidParams.RewriteMsg(err.Error()))
		return
	}

	if len(form.File) == 0 {
		response.Error(c, errcode.InvalidParams.RewriteMsg("upload file is empty"))
		return
	}

	//spongeArg, err := getFormValue(form.Value, "spongeArg")
	//if err != nil {
	//	response.Error(c, errcode.InvalidParams.RewriteMsg("the field 'spongeArg' cannot be empty"))
	//	return
	//}

	hadSaveFiles := []string{}
	savePath := getSavePath()
	fileType := ""
	var filePath string
	for _, files := range form.File {
		for _, file := range files {
			filename := filepath.Base(file.Filename)
			fileType = path.Ext(filename)
			//if !checkFileType(fileType) {
			//	response.Error(c, errcode.InvalidParams.RewriteMsg("only .proto or yaml files are allowed to be uploaded"))
			//	return
			//}

			filePath = savePath + "/" + filename
			if checkSameFile(hadSaveFiles, filePath) {
				continue
			}
			if err = c.SaveUploadedFile(file, filePath); err != nil {
				response.Error(c, errcode.InternalServerError.RewriteMsg(err.Error()))
				return
			}

			hadSaveFiles = append(hadSaveFiles, filePath)
		}
	}

	if fileType == ".proto" {
		filePath = savePath + "/*.proto"
	}

	response.Success(c, filePath)
}

//func getFormValue(valueMap map[string][]string, key string) (string, error) {
//	valueSlice := valueMap[key]
//	if len(valueSlice) == 0 {
//		return "", fmt.Errorf("form '%s' is empty", key)
//	}
//
//	return valueSlice[0], nil
//}

//func checkFileType(typeName string) bool {
//	switch typeName {
//	case ".proto", ".yml", ".yaml", "json":
//		return true
//	}
//
//	return false
//}

func checkSameFile(files []string, file string) bool {
	for _, v := range files {
		if v == file {
			return true
		}
	}
	return false
}

func getSavePath() string {
	var dir = saveDir
	if gofile.IsWindows() {
		dir = strings.ReplaceAll(saveDir, "\\", "/")
	}
	dir += "/" + "s_" + krand.String(krand.R_NUM|krand.R_LOWER, 10)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		_ = os.MkdirAll(dir, 0766)
	}
	return dir
}

// CompressPathToZip compressed directory to zip file
func CompressPathToZip(outPath, targetFile string) error {
	d, err := os.Create(targetFile)
	if err != nil {
		return err
	}
	defer func() {
		_ = d.Close()
	}()
	w := zip.NewWriter(d)
	defer func() {
		_ = w.Close()
	}()

	f, err := os.Open(outPath)
	if err != nil {
		return err
	}

	return compress(f, "", w)
}

func compress(file *os.File, prefix string, zw *zip.Writer) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}

	if info.IsDir() {
		prefix = prefix + "/" + info.Name()
		fileInfos, err := file.Readdir(-1)
		if err != nil {
			return err
		}
		for _, fi := range fileInfos {
			f, err := os.Open(file.Name() + "/" + fi.Name())
			if err != nil {
				return err
			}
			err = compress(f, prefix, zw)
			if err != nil {
				return err
			}
		}
	} else {
		header, err := zip.FileInfoHeader(info)
		header.Name = prefix + "/" + header.Name
		if err != nil {
			return err
		}
		writer, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		_, err = io.Copy(writer, file)
		_ = file.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

func AdaptToWindowsZip(outPath, targetFile string) error {
	d, err := os.Create(targetFile)
	if err != nil {
		return err
	}
	defer d.Close()

	w := zip.NewWriter(d)
	defer w.Close()

	baseDir := filepath.Dir(outPath)

	return filepath.Walk(outPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relPath) // convert to slash path

		if info.IsDir() {
			header.Name += "/"
			_, err = w.CreateHeader(header)
			return err
		}

		writer, err := w.CreateHeader(header)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})
}

func getSpongeDir() string {
	dir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("can't get home directory'")
		return ""
	}

	return dir
}

func getMysqlTables(dsn string) ([]string, error) {
	dsn = utils.AdaptiveMysqlDsn(dsn)
	db, err := mysql.Init(dsn)
	if err != nil {
		return nil, err
	}
	defer mysql.Close(db) //nolint

	var tables []string
	err = db.Raw("show tables").Scan(&tables).Error
	if err != nil {
		return nil, err
	}

	return tables, nil
}

func getPostgresqlTables(dsn string) ([]string, error) {
	dsn = utils.AdaptivePostgresqlDsn(dsn)
	db, err := postgresql.Init(dsn)
	if err != nil {
		return nil, err
	}
	defer mysql.Close(db) //nolint

	schemas, err := getSchemas(db, dsn)
	if err != nil {
		return nil, err
	}

	return getSchemaTables(db, schemas)
}

type pgSchema struct {
	SchemaName string
}

type pgTable struct {
	TableName string
}

func getSchemas(db *sgorm.DB, dsn string) ([]pgSchema, error) {
	var schemas []pgSchema

	if strings.Contains(dsn, "search_path=") {
		ss := strings.Split(dsn, " ")
		for _, s := range ss {
			if strings.Contains(s, "search_path=") {
				schemaName := strings.Split(s, "=")[1]
				if schemaName != "" {
					schemas = append(schemas, pgSchema{SchemaName: schemaName})
				}
			}
		}
	}

	if len(schemas) != 0 {
		return schemas, nil
	}

	err := db.Raw("SELECT schema_name FROM information_schema.schemata").Scan(&schemas).Error
	if err != nil {
		return nil, err
	}
	return schemas, nil
}

func getSchemaTables(db *sgorm.DB, schemas []pgSchema) ([]string, error) {
	var schemaTables []string
	for _, schema := range schemas {
		if schema.SchemaName == "information_schema" || schema.SchemaName == "pg_catalog" || schema.SchemaName == "pg_toast" {
			continue
		}

		var tables []pgTable
		err := db.Raw("SELECT table_name FROM information_schema.tables WHERE table_schema = ?", schema.SchemaName).Scan(&tables).Error
		if err != nil {
			return nil, err
		}

		for _, table := range tables {
			schemaTables = append(schemaTables, table.TableName)
		}
	}
	return schemaTables, nil
}

func getSqliteTables(dbFile string) ([]string, error) {
	if !gofile.IsExists(dbFile) {
		return nil, fmt.Errorf("sqlite db file %s not found in local host", dbFile)
	}

	db, err := sqlite.Init(dbFile)
	if err != nil {
		return nil, err
	}
	defer sqlite.Close(db) //nolint

	var tables []string
	err = db.Raw("select name from sqlite_master where type = ?", "table").Scan(&tables).Error
	if err != nil {
		return nil, err
	}

	filteredTables := []string{}
	for _, table := range tables {
		if table == "sqlite_sequence" {
			continue
		}
		filteredTables = append(filteredTables, table)
	}

	return filteredTables, nil
}

func getMongodbTables(dsn string) ([]string, error) {
	dsn = utils.AdaptiveMongodbDsn(dsn)
	db, err := mgo.Init(dsn)
	if err != nil {
		return nil, err
	}
	defer mgo.Close(db) //nolint

	tables, err := db.ListCollectionNames(context.Background(), bson.M{})
	if err != nil {
		return nil, err
	}

	if len(tables) == 0 {
		u, _ := url.Parse(dsn)
		return nil, fmt.Errorf("mongodb db %s has no tables", strings.TrimLeft(u.Path, "/"))
	}

	return tables, nil
}
