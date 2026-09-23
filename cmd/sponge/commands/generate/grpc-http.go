package generate

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Eric-Guo/sponge/pkg/replacer"
	"github.com/Eric-Guo/sponge/pkg/sql2code"
)

// GRPCAndHTTPCommand generate grpc+http servers code based on sql
func GRPCAndHTTPCommand() *cobra.Command {
	var (
		moduleName  string // module name for go.mod
		serverName  string // server name
		projectName string // project name for deployment name
		repoAddr    string // image repo address
		outPath     string // output directory
		dbTables    string // table names
		sqlArgs     = sql2code.Args{
			Package:    "model",
			JSONTag:    true,
			GormType:   true,
			IsWebProto: true,
		}

		suitedMonoRepo bool // whether the generated code is suitable for mono-repo
	)

	//nolint
	cmd := &cobra.Command{
		Use:   "grpc-http",
		Short: "Generate grpc+http servers code based on sql",
		Long:  "Generate grpc+http servers code based on sql.",
		Example: color.HiBlackString(`  # Generate grpc+http servers code.
  sponge micro grpc-http --module-name=yourModuleName --server-name=yourServerName --project-name=yourProjectName --db-driver=mysql --db-dsn=root:123456@(192.168.3.37:3306)/test --db-table=user

  # Generate grpc+http servers code with multiple table names.
  sponge micro grpc-http --module-name=yourModuleName --server-name=yourServerName --project-name=yourProjectName --db-driver=mysql --db-dsn=root:123456@(192.168.3.37:3306)/test --db-table=t1,t2

  # Generate grpc+http servers code with extended api.
  sponge micro grpc-http --module-name=yourModuleName --server-name=yourServerName --project-name=yourProjectName --db-driver=mysql --db-dsn=root:123456@(192.168.3.37:3306)/test --db-table=user --extended-api=true

  # Generate grpc+http servers code and specify the output directory, Note: code generation will be canceled when the latest generated file already exists.
  sponge micro grpc-http --module-name=yourModuleName --server-name=yourServerName --project-name=yourProjectName --db-driver=mysql --db-dsn=root:123456@(192.168.3.37:3306)/test --db-table=user --out=./yourServerDir

  # Generate grpc+http servers code and specify the docker image repository address.
  sponge micro grpc-http --module-name=yourModuleName --server-name=yourServerName --project-name=yourProjectName --repo-addr=192.168.3.37:9443/user-name --db-driver=mysql --db-dsn=root:123456@(192.168.3.37:3306)/test --db-table=user

  # If you want the generated code to suited to mono-repo, you need to set the parameter --suited-mono-repo=true`),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			projectName, serverName, err = convertProjectAndServerName(projectName, serverName)
			if err != nil {
				return err
			}

			if suitedMonoRepo {
				outPath = changeOutPath(outPath, serverName)
			}

			g := &httpAndGRPCPbGenerator{
				moduleName:        moduleName,
				serverName:        serverName,
				projectName:       projectName,
				repoAddr:          repoAddr,
				outPath:           outPath,
				suitedMonoRepo:    suitedMonoRepo,
				isHandleProtoFile: false,

				isAddDBInitCode:    true,
				dbDriver:           sqlArgs.DBDriver,
				extraReplaceFields: extraFields(sqlArgs.DBDriver, sqlArgs.DBDsn),
			}
			outPath, err = g.generateCode()
			if err != nil {
				return err
			}

			_ = generateConfigmap(serverName, outPath)

			tableNames := strings.Split(dbTables, ",")
			for _, tableName := range tableNames {
				if tableName == "" {
					continue
				}

				sqlArgs.DBTable = tableName
				codes, err := sql2code.Generate(&sqlArgs)
				if err != nil {
					return err
				}

				g := &serviceAndHandlerGenerator{
					moduleName:     moduleName,
					serverName:     serverName,
					dbDriver:       sqlArgs.DBDriver,
					isEmbed:        sqlArgs.IsEmbed,
					isExtendedAPI:  sqlArgs.IsExtendedAPI,
					codes:          codes,
					outPath:        outPath,
					suitedMonoRepo: suitedMonoRepo,
				}
				outPath, err = g.generateCode()
				if err != nil {
					return err
				}
			}

			fmt.Printf(`
using help:
  1. open a terminal and execute the command to generate code:  make proto
  2. compile and run server: make run
  3. access http://localhost:8080/apis/swagger/index.html in your browser, and test the http api.
     open the file "internal/service/xxx_client_test.go" using Goland or VSCode, and test the grpc api.

`)
			fmt.Printf("generate %s's grpc+http servers code successfully, out = %s\n", serverName, outPath)

			_ = generateConfigmap(serverName, outPath)
			// generate database
			return nil
		},
	}

	cmd.Flags().StringVarP(&moduleName, "module-name", "m", "", "module-name is the name of the module in the go.mod file")
	_ = cmd.MarkFlagRequired("module-name")
	cmd.Flags().StringVarP(&serverName, "server-name", "s", "", "server name")
	_ = cmd.MarkFlagRequired("server-name")
	cmd.Flags().StringVarP(&projectName, "project-name", "p", "", "project name")
	_ = cmd.MarkFlagRequired("project-name")
	cmd.Flags().StringVarP(&sqlArgs.DBDriver, "db-driver", "k", "mysql", "database driver, support mysql, mongodb, postgresql, sqlite")
	cmd.Flags().StringVarP(&sqlArgs.DBDsn, "db-dsn", "d", "", "database content address, e.g. user:password@(host:port)/database. Note: if db-driver=sqlite, db-dsn must be a local sqlite db file, e.g. --db-dsn=/tmp/sponge_sqlite.db") //nolint
	_ = cmd.MarkFlagRequired("db-dsn")
	cmd.Flags().StringVarP(&dbTables, "db-table", "t", "", "table name, multiple names separated by commas")
	_ = cmd.MarkFlagRequired("db-table")
	cmd.Flags().BoolVarP(&sqlArgs.IsEmbed, "embed", "e", false, "whether to embed gorm.model struct")
	cmd.Flags().BoolVarP(&sqlArgs.IsExtendedAPI, "extended-api", "a", false, "whether to generate extended crud api, additional includes: DeleteByIDs, GetByCondition, ListByIDs, ListByLatestID")
	cmd.Flags().BoolVarP(&suitedMonoRepo, "suited-mono-repo", "l", false, "whether the generated code is suitable for mono-repo")
	cmd.Flags().IntVarP(&sqlArgs.JSONNamedType, "json-name-type", "j", 1, "json tags name type, 0:snake case, 1:camel case")
	cmd.Flags().StringVarP(&repoAddr, "repo-addr", "r", "", "docker image repository address, excluding http and repository names")
	cmd.Flags().StringVarP(&outPath, "out", "o", "", "output directory, default is ./serverName_rpc_<time>")

	return cmd
}

func extraFields(dbDriver string, dbDSN string) []replacer.Field {
	var fields []replacer.Field

	return append(fields, []replacer.Field{
		{ // replace the contents of the database/init.go file
			Old: databaseInitDBFileMark,
			New: getInitDBCode(dbDriver),
		},
		{
			Old: showDbNameMark,
			New: CurrentDbDriver(dbDriver),
		},
		{
			Old: "init.go.mgo",
			New: "init.go",
		},
		{
			Old: "mongodb.go.mgo",
			New: "mongodb.go",
		},

		// replace config file content
		{
			Old: "root:123456@(192.168.3.37:3306)/account",
			New: dbDSN,
		},
		{
			Old: "root:123456@192.168.3.37:27017/account",
			New: dbDSN,
		},
		{
			Old: "root:123456@192.168.3.37:5432/account?sslmode=disable",
			New: adaptPgDsn(dbDSN),
		},
		{
			Old: "test/sql/sqlite/sponge.db",
			New: sqliteDSNAdaptation(dbDriver, dbDSN),
		},
	}...)
}
