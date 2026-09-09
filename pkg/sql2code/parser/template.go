package parser

import (
	"sync"
	"text/template"

	"github.com/pkg/errors"
)

var (
	modelStructTmpl    *template.Template
	modelStructTmplRaw = `
{{- if .Comment -}}
// {{.TableName}} {{.Comment}}
{{end -}}
type {{.TableName}} struct {
{{- range .Fields}}
	{{.Name}} {{.GoType}} {{if .Tag}}` + "`{{.Tag}}`" + `{{end}}{{if .Comment}} // {{.Comment}}{{end}}
{{- end}}
}
{{if .NameFunc}}
// TableName table name
func (m *{{.TableName}}) TableName() string {
	return "{{.RawTableName}}"
}
{{end}}
`

	tableColumnsTmpl    *template.Template
	tableColumnsTmplRaw = `
// {{.TableName}}ColumnNames Whitelist for custom query fields to prevent sql injection attacks
var {{.TableName}}ColumnNames = map[string]bool{
{{- range .Fields}}
	"{{.ColName}}": true,
{{- end}}
}
`

	modelTmpl    *template.Template
	modelTmplRaw = `package {{.Package}}
{{if .ImportPath}}
import (
	{{- range .ImportPath}}
	"{{.}}"
	{{- end}}
)
{{- end}}
{{range .StructCode}}
{{.}}
{{end}}`

	updateFieldTmpl    *template.Template
	updateFieldTmplRaw = `
{{- range .Fields}}
	if {{.NonZeroCondition}} {
		update["{{.ColName}}"] = table.{{.Name}}
	}
{{- end}}`

	handlerCreateStructTmpl    *template.Template
	handlerCreateStructTmplRaw = `
// Create{{.TableName}}Request request params
type Create{{.TableName}}Request struct {
{{- range .Fields}}
	{{.Name}}  {{.GoType}} ` + "`" + `json:"{{.JSONName}}" binding:""` + "`" + `{{if .Comment}} // {{.Comment}}{{end}}
{{- end}}
}
`

	handlerUpdateStructTmpl    *template.Template
	handlerUpdateStructTmplRaw = `
// Update{{.TableName}}ByIDRequest request params
type Update{{.TableName}}ByIDRequest struct {
{{- range .Fields}}
	{{.Name}}  {{.GoType}} ` + "`" + `json:"{{.JSONName}}" binding:""` + "`" + `{{if .Comment}} // {{.Comment}}{{end}}
{{- end}}
}
`

	handlerDetailStructTmpl    *template.Template
	handlerDetailStructTmplRaw = `
// {{.TableName}}ObjDetail detail
type {{.TableName}}ObjDetail struct {
{{- range .Fields}}
	{{.Name}}  {{.GoType}} ` + "`" + `json:"{{.JSONName}}"` + "`" + `{{if .Comment}} // {{.Comment}}{{end}}
{{- end}}
}`

	modelJSONTmpl    *template.Template
	modelJSONTmplRaw = `{
{{- range .Fields}}
	"{{.ColName}}" {{.GoZero}}
{{- end}}
}
`

	protoFileTmpl    *template.Template
	protoFileTmplRaw = `syntax = "proto3";

package api.serverNameExample.v1;

import "api/types/types.proto";
import "validate/validate.proto";

option go_package = "github.com/go-dev-frame/sponge/api/serverNameExample/v1;v1";

service {{.TName}} {
  // Create a new {{.TName}}
  rpc Create(Create{{.TableName}}Request) returns (Create{{.TableName}}Reply) {}

  // Delete a {{.TName}} by id
  rpc DeleteByID(Delete{{.TableName}}ByIDRequest) returns (Delete{{.TableName}}ByIDReply) {}

  // Update a {{.TName}} by id
  rpc UpdateByID(Update{{.TableName}}ByIDRequest) returns (Update{{.TableName}}ByIDReply) {}

  // Get a {{.TName}} by id
  rpc GetByID(Get{{.TableName}}ByIDRequest) returns (Get{{.TableName}}ByIDReply) {}

  // Get a paginated list of {{.TName}} by custom conditions
  rpc List(List{{.TableName}}Request) returns (List{{.TableName}}Reply) {}

  // Batch delete {{.TName}} by ids
  rpc DeleteByIDs(Delete{{.TableName}}ByIDsRequest) returns (Delete{{.TableName}}ByIDsReply) {}

  // Get a {{.TName}} by custom condition
  rpc GetByCondition(Get{{.TableName}}ByConditionRequest) returns (Get{{.TableName}}ByConditionReply) {}

  // Batch get {{.TName}} by ids
  rpc ListByIDs(List{{.TableName}}ByIDsRequest) returns (List{{.TableName}}ByIDsReply) {}

  // Get a paginated list of {{.TName}} by last id
  rpc ListByLastID(List{{.TableName}}ByLastIDRequest) returns (List{{.TableName}}ByLastIDReply) {}
}


/*
Notes for defining message fields:
    1. Suggest using camel case style naming for message field names, such as firstName, lastName, etc.
    2. If the message field name ending in 'id', it is recommended to use xxxID naming format, such as userID, orderID, etc.
    3. Add validate rules https://github.com/envoyproxy/protoc-gen-validate#constraint-rules, such as:
        uint64 id = 1 [(validate.rules).uint64.gte  = 1];
*/


// protoMessageCreateCode

message Create{{.TableName}}Reply {
  // createTableReplyFieldCode
}

message Delete{{.TableName}}ByIDRequest {
  // deleteTableByIDRequestFieldCode
}

message Delete{{.TableName}}ByIDReply {

}

// protoMessageUpdateCode

message Update{{.TableName}}ByIDReply {

}

// protoMessageDetailCode

message Get{{.TableName}}ByIDRequest {
  // getTableByIDRequestFieldCode
}

message Get{{.TableName}}ByIDReply {
  {{.TableName}} {{.TName}} = 1;
}

message List{{.TableName}}Request {
  api.types.Params params = 1;
}

message List{{.TableName}}Reply {
  int64 total = 1;
  repeated {{.TableName}} {{.TName}}s = 2;
}

message Delete{{.TableName}}ByIDsRequest {
  // deleteTableByIDsRequestFieldCode
}

message Delete{{.TableName}}ByIDsReply {

}

message Get{{.TableName}}ByConditionRequest {
  types.Conditions conditions = 1;
}

message Get{{.TableName}}ByConditionReply {
  {{.TableName}} {{.TName}} = 1;
}

message List{{.TableName}}ByIDsRequest {
  // getTableByIDsRequestFieldCode
}

message List{{.TableName}}ByIDsReply {
  repeated {{.TableName}} {{.TName}}s = 1;
}

message List{{.TableName}}ByLastIDRequest {
  // listTableByLastIDRequestFieldCode
  uint32 limit = 2 [(validate.rules).uint32.gt = 0]; // limit size per page
  string sort = 3; // sort by column name of table, default is -id, the - sign indicates descending order.
}

message List{{.TableName}}ByLastIDReply {
  repeated {{.TableName}} {{.TName}}s = 1;
}
`

	protoFileSimpleTmpl    *template.Template
	protoFileSimpleTmplRaw = `syntax = "proto3";

package api.serverNameExample.v1;

import "api/types/types.proto";
import "validate/validate.proto";

option go_package = "github.com/go-dev-frame/sponge/api/serverNameExample/v1;v1";

service {{.TName}} {
  // Create a new {{.TName}}
  rpc Create(Create{{.TableName}}Request) returns (Create{{.TableName}}Reply) {}

  // Delete a {{.TName}} by id
  rpc DeleteByID(Delete{{.TableName}}ByIDRequest) returns (Delete{{.TableName}}ByIDReply) {}

  // Update a {{.TName}} by id
  rpc UpdateByID(Update{{.TableName}}ByIDRequest) returns (Update{{.TableName}}ByIDReply) {}

  // Get a {{.TName}} by id
  rpc GetByID(Get{{.TableName}}ByIDRequest) returns (Get{{.TableName}}ByIDReply) {}

  // Get a paginated list of {{.TName}} by custom conditions
  rpc List(List{{.TableName}}Request) returns (List{{.TableName}}Reply) {}
}


/*
Notes for defining message fields:
    1. Suggest using camel case style naming for message field names, such as firstName, lastName, etc.
    2. If the message field name ending in 'id', it is recommended to use xxxID naming format, such as userID, orderID, etc.
    3. Add validate rules https://github.com/envoyproxy/protoc-gen-validate#constraint-rules, such as:
        uint64 id = 1 [(validate.rules).uint64.gte  = 1];
*/


// protoMessageCreateCode

message Create{{.TableName}}Reply {
  // createTableReplyFieldCode
}

message Delete{{.TableName}}ByIDRequest {
  // deleteTableByIDRequestFieldCode
}

message Delete{{.TableName}}ByIDReply {

}

// protoMessageUpdateCode

message Update{{.TableName}}ByIDReply {

}

// protoMessageDetailCode

message Get{{.TableName}}ByIDRequest {
  // getTableByIDRequestFieldCode
}

message Get{{.TableName}}ByIDReply {
  {{.TableName}} {{.TName}} = 1;
}

message List{{.TableName}}Request {
  api.types.Params params = 1;
}

message List{{.TableName}}Reply {
  int64 total = 1;
  repeated {{.TableName}} {{.TName}}s = 2;
}
`

	protoFileForWebTmpl    *template.Template
	protoFileForWebTmplRaw = `syntax = "proto3";

package api.serverNameExample.v1;

import "api/types/types.proto";
import "google/api/annotations.proto";
import "protoc-gen-openapiv2/options/annotations.proto";
import "tagger/tagger.proto";
import "validate/validate.proto";

option go_package = "github.com/go-dev-frame/sponge/api/serverNameExample/v1;v1";

/*
Default settings for generating *.swagger.json documents. For reference, see: https://bit.ly/4dE5jj7
Tip: To enhance the generated Swagger documentation, you can add the openapiv2_operation option to your RPC method. For example:
    option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_operation) = {
      summary: "get user by id",
      description: "get user by id",
      security: {
        security_requirement: {
          key: "BearerAuth";
          value: {}
        }
      }
    };
*/
option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_swagger) = {
  host: "localhost:8080"
  base_path: ""
  info: {
    title: "serverNameExample api docs";
    version: "v1.0.0";
  }
  schemes: HTTP;
  schemes: HTTPS;
  consumes: "application/json";
  produces: "application/json";
  security_definitions: {
    security: {
      key: "BearerAuth";
      value: {
        type: TYPE_API_KEY;
        in: IN_HEADER;
        name: "Authorization";
        description: "Type Bearer your-jwt-token to Value";
      }
    }
  }
};

service {{.TName}} {
  // Create a new {{.TName}}
  rpc Create(Create{{.TableName}}Request) returns (Create{{.TableName}}Reply) {
    option (google.api.http) = {
      post: "/api/v1/{{.TName}}"
      body: "*"
    };
  }

  // Delete a {{.TName}} by id
  rpc DeleteByID(Delete{{.TableName}}ByIDRequest) returns (Delete{{.TableName}}ByIDReply) {
    option (google.api.http) = {
      delete: "/api/v1/{{.TName}}/{id}"
    };
  }

  // Update a {{.TName}} by id
  rpc UpdateByID(Update{{.TableName}}ByIDRequest) returns (Update{{.TableName}}ByIDReply) {
    option (google.api.http) = {
      put: "/api/v1/{{.TName}}/{id}"
      body: "*"
    };
  }

  // Get a {{.TName}} by id
  rpc GetByID(Get{{.TableName}}ByIDRequest) returns (Get{{.TableName}}ByIDReply) {
    option (google.api.http) = {
      get: "/api/v1/{{.TName}}/{id}"
    };
  }

  // Get a paginated list of {{.TName}} by custom conditions
  rpc List(List{{.TableName}}Request) returns (List{{.TableName}}Reply) {
    option (google.api.http) = {
      post: "/api/v1/{{.TName}}/list"
      body: "*"
    };
  }

  // Batch delete {{.TName}} by ids
  rpc DeleteByIDs(Delete{{.TableName}}ByIDsRequest) returns (Delete{{.TableName}}ByIDsReply) {
    option (google.api.http) = {
      post: "/api/v1/{{.TName}}/delete/ids"
      body: "*"
    };
  }

  // Get a {{.TName}} by custom condition
  rpc GetByCondition(Get{{.TableName}}ByConditionRequest) returns (Get{{.TableName}}ByConditionReply) {
    option (google.api.http) = {
      post: "/api/v1/{{.TName}}/condition"
      body: "*"
    };
  }

  // Batch get {{.TName}} by ids
  rpc ListByIDs(List{{.TableName}}ByIDsRequest) returns (List{{.TableName}}ByIDsReply) {
    option (google.api.http) = {
      post: "/api/v1/{{.TName}}/list/ids"
      body: "*"
    };
  }

  // Get a paginated list of {{.TName}} by last id
  rpc ListByLastID(List{{.TableName}}ByLastIDRequest) returns (List{{.TableName}}ByLastIDReply) {
    option (google.api.http) = {
      get: "/api/v1/{{.TName}}/list"
    };
  }
}


/*
Notes for defining message fields:
    1. Suggest using camel case style naming for message field names, such as firstName, lastName, etc.
    2. If the message field name ending in 'id', it is recommended to use xxxID naming format, such as userID, orderID, etc.
    3. Add validate rules https://github.com/envoyproxy/protoc-gen-validate#constraint-rules, such as:
        uint64 id = 1 [(validate.rules).uint64.gte  = 1];

If used to generate code that supports the HTTP protocol, notes for defining message fields:
    1. If the route contains the path parameter, such as /api/v1/userExample/{id}, the defined
        message must contain the name of the path parameter and the name should be added
        with a new tag, such as int64 id = 1 [(tagger.tags) = "uri:\"id\""];
    2. If the request url is followed by a query parameter, such as /api/v1/getUserExample?name=Tom,
        a form tag must be added when defining the query parameter in the message, such as:
        string name = 1 [(tagger.tags) = "form:\"name\""].
    3. When the message fields use snake_case naming (e.g., order_id), the generated swagger.json file
        will use camelCase (e.g., orderId) instead of the expected snake_case. This behavior aligns with
        the JSON tag names used by gRPC, but it can cause the Gin framework to fail to correctly bind and
        retrieve parameter values. There are two ways to resolve this issue:
            (1) Explicitly specify the JSON tag name using the json_name option, such as:
                 string order_id = 1 [json_name = "order_id"];
            (2) If you want to switch to camelCase naming and update the JSON tag name accordingly, such as:
                 string order_id = 1 [json_name = "orderID", (tagger.tags) = "json:\"orderID\""];
*/


// protoMessageCreateCode

message Create{{.TableName}}Reply {
  // createTableReplyFieldCode
}

message Delete{{.TableName}}ByIDRequest {
  // deleteTableByIDRequestFieldCode
}

message Delete{{.TableName}}ByIDReply {

}

// protoMessageUpdateCode

message Update{{.TableName}}ByIDReply {

}

// protoMessageDetailCode

message Get{{.TableName}}ByIDRequest {
  // getTableByIDRequestFieldCode
}

message Get{{.TableName}}ByIDReply {
  {{.TableName}} {{.TName}} = 1;
}

message List{{.TableName}}Request {
  api.types.Params params = 1;
}

message List{{.TableName}}Reply {
  int64 total = 1;
  repeated {{.TableName}} {{.TName}}s = 2;
}

message Delete{{.TableName}}ByIDsRequest {
  // deleteTableByIDsRequestFieldCode
}

message Delete{{.TableName}}ByIDsReply {

}

message Get{{.TableName}}ByConditionRequest {
  types.Conditions conditions = 1;
}

message Get{{.TableName}}ByConditionReply {
  {{.TableName}} {{.TName}} = 1;
}

message List{{.TableName}}ByIDsRequest {
  // getTableByIDsRequestFieldCode
}

message List{{.TableName}}ByIDsReply {
  repeated {{.TableName}} {{.TName}}s = 1;
}

message List{{.TableName}}ByLastIDRequest {
  // listTableByLastIDRequestFieldCode
  uint32 limit = 2 [(validate.rules).uint32.gt = 0, (tagger.tags) = "form:\"limit\""]; // limit size per page
  string sort = 3 [(tagger.tags) = "form:\"sort\""]; // sort by column name of table, default is -id, the - sign indicates descending order.
}

message List{{.TableName}}ByLastIDReply {
  repeated {{.TableName}} {{.TName}}s = 1;
}
`

	protoFileForSimpleWebTmpl    *template.Template
	protoFileForSimpleWebTmplRaw = `syntax = "proto3";

package api.serverNameExample.v1;

import "api/types/types.proto";
import "google/api/annotations.proto";
import "protoc-gen-openapiv2/options/annotations.proto";
import "tagger/tagger.proto";
import "validate/validate.proto";

option go_package = "github.com/go-dev-frame/sponge/api/serverNameExample/v1;v1";

/*
Default settings for generating *.swagger.json documents. For reference, see: https://bit.ly/4dE5jj7
Tip: To enhance the generated Swagger documentation, you can add the openapiv2_operation option to your RPC method. For example:
    option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_operation) = {
      summary: "get user by id",
      description: "get user by id",
      security: {
        security_requirement: {
          key: "BearerAuth";
          value: {}
        }
      }
    };
*/
option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_swagger) = {
  host: "localhost:8080"
  base_path: ""
  info: {
    title: "serverNameExample api docs";
    version: "v1.0.0";
  }
  schemes: HTTP;
  schemes: HTTPS;
  consumes: "application/json";
  produces: "application/json";
  security_definitions: {
    security: {
      key: "BearerAuth";
      value: {
        type: TYPE_API_KEY;
        in: IN_HEADER;
        name: "Authorization";
        description: "Type Bearer your-jwt-token to Value";
      }
    }
  }
};

service {{.TName}} {
  // Create a new {{.TName}}
  rpc Create(Create{{.TableName}}Request) returns (Create{{.TableName}}Reply) {
    option (google.api.http) = {
      post: "/api/v1/{{.TName}}"
      body: "*"
    };
  }

  // Delete a {{.TName}} by id
  rpc DeleteByID(Delete{{.TableName}}ByIDRequest) returns (Delete{{.TableName}}ByIDReply) {
    option (google.api.http) = {
      delete: "/api/v1/{{.TName}}/{id}"
    };
  }

  // Update a {{.TName}} by id
  rpc UpdateByID(Update{{.TableName}}ByIDRequest) returns (Update{{.TableName}}ByIDReply) {
    option (google.api.http) = {
      put: "/api/v1/{{.TName}}/{id}"
      body: "*"
    };
  }

  // Get a {{.TName}} by id
  rpc GetByID(Get{{.TableName}}ByIDRequest) returns (Get{{.TableName}}ByIDReply) {
    option (google.api.http) = {
      get: "/api/v1/{{.TName}}/{id}"
    };
  }

  // Get a paginated list of {{.TName}} by custom conditions
  rpc List(List{{.TableName}}Request) returns (List{{.TableName}}Reply) {
    option (google.api.http) = {
      post: "/api/v1/{{.TName}}/list"
      body: "*"
    };
  }
}


/*
Notes for defining message fields:
    1. Suggest using camel case style naming for message field names, such as firstName, lastName, etc.
    2. If the message field name ending in 'id', it is recommended to use xxxID naming format, such as userID, orderID, etc.
    3. Add validate rules https://github.com/envoyproxy/protoc-gen-validate#constraint-rules, such as:
        uint64 id = 1 [(validate.rules).uint64.gte  = 1];

If used to generate code that supports the HTTP protocol, notes for defining message fields:
    1. If the route contains the path parameter, such as /api/v1/userExample/{id}, the defined
        message must contain the name of the path parameter and the name should be added
        with a new tag, such as int64 id = 1 [(tagger.tags) = "uri:\"id\""];
    2. If the request url is followed by a query parameter, such as /api/v1/getUserExample?name=Tom,
        a form tag must be added when defining the query parameter in the message, such as:
        string name = 1 [(tagger.tags) = "form:\"name\""].
    3. When the message fields use snake_case naming (e.g., order_id), the generated swagger.json file
        will use camelCase (e.g., orderId) instead of the expected snake_case. This behavior aligns with
        the JSON tag names used by gRPC, but it can cause the Gin framework to fail to correctly bind and
        retrieve parameter values. There are two ways to resolve this issue:
            (1) Explicitly specify the JSON tag name using the json_name option, such as:
                 string order_id = 1 [json_name = "order_id"];
            (2) If you want to switch to camelCase naming and update the JSON tag name accordingly, such as:
                 string order_id = 1 [json_name = "orderID", (tagger.tags) = "json:\"orderID\""];
*/


// protoMessageCreateCode

message Create{{.TableName}}Reply {
  // createTableReplyFieldCode
}

message Delete{{.TableName}}ByIDRequest {
  // deleteTableByIDRequestFieldCode
}

message Delete{{.TableName}}ByIDReply {

}

// protoMessageUpdateCode

message Update{{.TableName}}ByIDReply {

}

// protoMessageDetailCode

message Get{{.TableName}}ByIDRequest {
  // getTableByIDRequestFieldCode
}

message Get{{.TableName}}ByIDReply {
  {{.TableName}} {{.TName}} = 1;
}

message List{{.TableName}}Request {
  api.types.Params params = 1;
}

message List{{.TableName}}Reply {
  int64 total = 1;
  repeated {{.TableName}} {{.TName}}s = 2;
}
`

	protoMessageCreateTmpl    *template.Template
	protoMessageCreateTmplRaw = `message Create{{.TableName}}Request {
{{- range $i, $v := .Fields}}
	{{$v.GoType}} {{$v.JSONName}} = {{$v.AddOne $i}}; {{if $v.Comment}} // {{$v.Comment}}{{end}}
{{- end}}
}`

	protoMessageUpdateTmpl    *template.Template
	protoMessageUpdateTmplRaw = `message Update{{.TableName}}ByIDRequest {
{{- range $i, $v := .Fields}}
	{{$v.GoType}} {{$v.JSONName}} = {{$v.AddOneWithTag $i}}; {{if $v.Comment}} // {{$v.Comment}}{{end}}
{{- end}}
}`

	protoMessageDetailTmpl    *template.Template
	protoMessageDetailTmplRaw = `message {{.TableName}} {
{{- range $i, $v := .Fields}}
	{{$v.GoType}} {{$v.JSONName}} = {{$v.AddOne $i}}; {{if $v.Comment}} // {{$v.Comment}}{{end}}
{{- end}}
}`

	serviceStructTmpl    *template.Template
	serviceStructTmplRaw = `
		{
			name: "Create",
			fn: func() (interface{}, error) {
				// todo enter parameters before testing
// serviceCreateStructCode
			},
			wantErr: false,
		},

		{
			name: "UpdateByID",
			fn: func() (interface{}, error) {
				// todo enter parameters before testing
// serviceUpdateStructCode
			},
			wantErr: false,
		},
`

	serviceCreateStructTmpl    *template.Template
	serviceCreateStructTmplRaw = `				req := &serverNameExampleV1.Create{{.TableName}}Request{
					{{- range .Fields}}
						{{.Name}}:  {{.GoTypeZero}}, {{if .Comment}} // {{.Comment}}{{end}}
					{{- end}}
				}
				return cli.Create(ctx, req)`

	serviceUpdateStructTmpl    *template.Template
	serviceUpdateStructTmplRaw = `				req := &serverNameExampleV1.Update{{.TableName}}ByIDRequest{
					{{- range .Fields}}
						{{.Name}}:  {{.GoTypeZero}}, {{if .Comment}} // {{.Comment}}{{end}}
					{{- end}}
				}
				return cli.UpdateByID(ctx, req)`

	tmplParseOnce sync.Once
)

func initTemplate() {
	tmplParseOnce.Do(func() {
		var err, errSum error

		modelStructTmpl, err = template.New("goStruct").Parse(modelStructTmplRaw)
		if err != nil {
			errSum = errors.Wrap(err, "modelStructTmplRaw")
		}
		tableColumnsTmpl, err = template.New("tableColumns").Parse(tableColumnsTmplRaw)
		if err != nil {
			errSum = errors.Wrap(err, "tableColumnsTmplRaw")
		}
		modelTmpl, err = template.New("goFile").Parse(modelTmplRaw)
		if err != nil {
			errSum = errors.Wrap(errSum, "modelTmplRaw:"+err.Error())
		}
		updateFieldTmpl, err = template.New("goUpdateField").Parse(updateFieldTmplRaw)
		if err != nil {
			errSum = errors.Wrap(errSum, "updateFieldTmplRaw:"+err.Error())
		}
		handlerCreateStructTmpl, err = template.New("goPostStruct").Parse(handlerCreateStructTmplRaw)
		if err != nil {
			errSum = errors.Wrap(errSum, "handlerCreateStructTmplRaw:"+err.Error())
		}
		handlerUpdateStructTmpl, err = template.New("goPutStruct").Parse(handlerUpdateStructTmplRaw)
		if err != nil {
			errSum = errors.Wrap(errSum, "handlerUpdateStructTmplRaw:"+err.Error())
		}
		handlerDetailStructTmpl, err = template.New("goGetStruct").Parse(handlerDetailStructTmplRaw)
		if err != nil {
			errSum = errors.Wrap(errSum, "handlerDetailStructTmplRaw:"+err.Error())
		}
		modelJSONTmpl, err = template.New("modelJSON").Parse(modelJSONTmplRaw)
		if err != nil {
			errSum = errors.Wrap(errSum, "modelJSONTmplRaw:"+err.Error())
		}
		protoFileTmpl, err = template.New("protoFile").Parse(protoFileTmplRaw)
		if err != nil {
			errSum = errors.Wrap(errSum, "protoFileTmplRaw:"+err.Error())
		}
		protoFileSimpleTmpl, err = template.New("protoFileSimple").Parse(protoFileSimpleTmplRaw)
		if err != nil {
			errSum = errors.Wrap(errSum, "protoFileSimpleTmplRaw:"+err.Error())
		}
		protoFileForWebTmpl, err = template.New("protoFileForWeb").Parse(protoFileForWebTmplRaw)
		if err != nil {
			errSum = errors.Wrap(errSum, "protoFileForWebTmplRaw:"+err.Error())
		}
		protoFileForSimpleWebTmpl, err = template.New("protoFileForSimpleWeb").Parse(protoFileForSimpleWebTmplRaw)
		if err != nil {
			errSum = errors.Wrap(errSum, "protoFileForSimpleWebTmplRaw:"+err.Error())
		}
		protoMessageCreateTmpl, err = template.New("protoMessageCreate").Parse(protoMessageCreateTmplRaw)
		if err != nil {
			errSum = errors.Wrap(errSum, "protoMessageCreateTmplRaw:"+err.Error())
		}
		protoMessageUpdateTmpl, err = template.New("protoMessageUpdate").Parse(protoMessageUpdateTmplRaw)
		if err != nil {
			errSum = errors.Wrap(errSum, "protoMessageUpdateTmplRaw:"+err.Error())
		}
		protoMessageDetailTmpl, err = template.New("protoMessageDetail").Parse(protoMessageDetailTmplRaw)
		if err != nil {
			errSum = errors.Wrap(errSum, "protoMessageDetailTmplRaw:"+err.Error())
		}
		serviceCreateStructTmpl, err = template.New("serviceCreateStruct").Parse(serviceCreateStructTmplRaw)
		if err != nil {
			errSum = errors.Wrap(errSum, "serviceCreateStructTmplRaw:"+err.Error())
		}
		serviceUpdateStructTmpl, err = template.New("serviceUpdateStruct").Parse(serviceUpdateStructTmplRaw)
		if err != nil {
			errSum = errors.Wrap(errSum, "serviceUpdateStructTmplRaw:"+err.Error())
		}
		serviceStructTmpl, err = template.New("serviceStruct").Parse(serviceStructTmplRaw)
		if err != nil {
			errSum = errors.Wrap(errSum, "serviceStructTmplRaw:"+err.Error())
		}

		if errSum != nil {
			panic(errSum)
		}
	})
}
