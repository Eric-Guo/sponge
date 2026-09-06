package generate

import (
	"database/sql"
	"fmt"
	goparser "go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"

	"github.com/go-dev-frame/sponge/pkg/jy2struct"
	"github.com/go-dev-frame/sponge/pkg/sql2code"
)

func TestGeneratedGoVersions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the mock go command is a shell script")
	}

	tests := []struct {
		name       string
		version    string
		wantModule string
		wantImage  string
	}{
		{"missing toolchain", "", defaultGoModVersion, defaultImageGoModVersion},
		{"older toolchain", "go1.26.4", defaultGoModVersion, defaultImageGoModVersion},
		{"matching toolchain", "go1.27.1", "go 1.27.1", "golang:1.27.1-alpine"},
		{"newer patch", "go1.27.2", "go 1.27.2", "golang:1.27.2-alpine"},
		{"newer minor", "go1.28.0", "go 1.28.0", "golang:1.28.0-alpine"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.version != "" {
				script := "#!/bin/sh\necho 'go version " + tt.version + " linux/amd64'\n"
				require.NoError(t, os.WriteFile(filepath.Join(dir, "go"), []byte(script), 0755))
			}
			t.Setenv("PATH", dir)

			require.Equal(t, tt.wantModule, getLocalGoVersion())
			require.Equal(t, tt.wantImage, extractImageGoVersion())
		})
	}
}

func TestConfigKeepsDynamicMaps(t *testing.T) {
	for _, maps := range []string{"env: {}\n", "env:\n    RAILS_ENV: development\n"} {
		dir := t.TempDir()
		yamlPath := filepath.Join(dir, "service.yml")
		require.NoError(t, os.WriteFile(yamlPath, []byte("upstream:\n  "+maps+"http:\n  tls:\n    remoteAPI:\n      url: ''\n      headers: {}\n"), 0600))
		output := filepath.Join(dir, "config.go")
		require.NoError(t, runGenConfigCommand(map[string]configType{output: {configFile: yamlPath}}, jy2struct.Args{Tags: "yaml,json", SubStruct: true}))
		data, err := os.ReadFile(output)
		require.NoError(t, err)
		require.Regexp(t, `Env\s+map\[string\]string`, string(data))
		require.Regexp(t, `Headers\s+map\[string\]string`, string(data))
	}
}

func TestHTTPGenerationUsesRepositoryTemplates(t *testing.T) {
	repository, err := filepath.Abs("../../../..")
	require.NoError(t, err)
	previous := SpongeDir
	SpongeDir = repository
	t.Cleanup(func() { SpongeDir = previous })
	for _, embedded := range []bool{true, false} {
		t.Run(fmt.Sprintf("embed=%t", embedded), func(t *testing.T) {
			temp := t.TempDir()
			dbFile := filepath.Join(temp, "schema.db")
			require.NoError(t, os.WriteFile(dbFile, nil, 0600))
			codes, err := sql2code.Generate(&sql2code.Args{
				SQL:      "CREATE TABLE users (id bigint primary key, created_at datetime, updated_at datetime, deleted_at datetime, signed_in_at datetime, name varchar(100));",
				DBDriver: DBDriverSqlite, DBDsn: dbFile, Package: "model", GormType: true, JSONTag: true, JSONNamedType: 1, IsEmbed: embedded, IsExtendedAPI: embedded,
			})
			require.NoError(t, err)
			generator := &httpGenerator{moduleName: "example.com/service", serverName: "sample", projectName: "sample", dbDriver: DBDriverSqlite, dbDSN: dbFile,
				codes: codes, outPath: filepath.Join(temp, "output"), isEmbed: embedded, isExtendedAPI: embedded}
			output, err := generator.generateCode()
			require.NoError(t, err)
			require.NoError(t, filepath.WalkDir(output, func(path string, entry os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if !entry.IsDir() && strings.HasSuffix(path, ".go") {
					file, parseErr := goparser.ParseFile(token.NewFileSet(), path, nil, goparser.AllErrors)
					err = parseErr
					if parseErr == nil {
						for _, dependency := range file.Imports {
							require.NotContains(t, dependency.Path.Value, "test-user-server")
						}
					}
					require.NoError(t, err, path)
				}
				return nil
			}))
			read := func(name string) string {
				data, err := os.ReadFile(filepath.Join(output, name))
				require.NoError(t, err)
				return string(data)
			}
			require.Contains(t, read("cmd/sample/initial/createService.go"), "app.NewUpstreamServer")
			require.Contains(t, read("internal/routers/routers.go"), "proxy.RegisterFallback")
			require.Contains(t, read("internal/routers/routers.go"), "middleware.RailsCookieAuthMiddleware")
			require.Contains(t, read("internal/server/http.go"), "httpsrv.ModeRemoteAPI")
			if embedded {
				require.Contains(t, read("internal/model/users.go"), "sgorm.Model")
				require.NotContains(t, read("internal/dao/users.go"), "table.SignedInAt != nil")
			} else {
				require.Contains(t, read("internal/dao/users.go"), "table.SignedInAt != nil")
			}
		})
	}
}

func TestHTTPSoftDeleteOption(t *testing.T) {
	repository, err := filepath.Abs("../../../..")
	require.NoError(t, err)
	previous := SpongeDir
	SpongeDir = repository
	t.Cleanup(func() { SpongeDir = previous })
	dir := t.TempDir()
	dbFile := filepath.Join(dir, "schema.db")
	db, err := sql.Open("sqlite3", dbFile)
	require.NoError(t, err)
	for _, table := range []string{"users", "posts"} {
		_, err = db.Exec("CREATE TABLE " + table + " (id INTEGER PRIMARY KEY, created_at DATETIME, updated_at DATETIME, name TEXT)")
		require.NoError(t, err)
	}
	require.NoError(t, db.Close())
	for _, option := range []string{"default", "true", "false"} {
		t.Run(option, func(t *testing.T) {
			output := filepath.Join(dir, option)
			command := HTTPCommand()
			require.Equal(t, "true", command.Flags().Lookup("soft-delete").DefValue)
			args := []string{"--module-name=example.com/service", "--server-name=sample", "--project-name=sample",
				"--db-driver=sqlite", "--db-dsn=" + dbFile, "--db-table=users,posts", "--extended-api=true", "--embed=true", "--out=" + output}
			if option != "default" {
				args = append(args, "--soft-delete="+option)
			}
			command.SetArgs(args)
			require.NoError(t, command.Execute())
			for _, table := range []string{"users", "posts"} {
				model, err := os.ReadFile(filepath.Join(output, "internal/model", table+".go"))
				require.NoError(t, err)
				tests, err := os.ReadFile(filepath.Join(output, "internal/dao", table+"_test.go"))
				require.NoError(t, err)
				if option == "false" {
					require.Contains(t, string(model), "sgorm.BaseModel")
					require.NotContains(t, string(model), "deleted_at")
					require.Contains(t, string(tests), `expectedSQLForDeletion := "DELETE .*"`)
					require.NotContains(t, string(tests), "expectedArgsForDeletionTime")
				} else {
					require.Contains(t, string(model), "sgorm.Model")
					require.Contains(t, string(model), "deleted_at")
					require.Contains(t, string(tests), `expectedSQLForDeletion := "UPDATE .*"`)
				}
			}
		})
	}
}
