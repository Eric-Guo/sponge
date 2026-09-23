package generate

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi2conv"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/Eric-Guo/sponge/pkg/gofile"
)

// HandleSwaggerJSONCommand handle swagger json command
func HandleSwaggerJSONCommand() *cobra.Command {
	var (
		enableUniformResponse          bool
		enableConvertToOpenAPI3        bool
		enableTransformIntegerToString bool
		enableStringToInteger          bool

		jsonFile string
	)

	cmd := &cobra.Command{
		Use:   "swagger",
		Short: "Handle swagger json file",
		Long: "Handles Swagger JSON files by standardizing response format data, " +
			"converting specifications to OpenAPI 3, and transforming 64-bit fields string into integer (default is true).",
		Example: color.HiBlackString(`  # Standardize response format data in swagger.json
  sponge web swagger --enable-standardize-response --file=docs/swagger.json

  # Convert swagger2.0 to openapi3.0
  sponge web swagger --enable-to-openapi3 --file=docs/swagger.json

  # Cancel the default conversion of 64 bit integer type fields from strings to integers
  sponge web swagger --enable-string-to-integer=false --file=docs/swagger.json`),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error

			if enableStringToInteger {
				if err = handleSwaggerFieldStringToInteger(jsonFile); err != nil {
					return err
				}
				fmt.Printf("Successfully transform 64-bit fields string to integer in %s fields\n", jsonFile)
			}

			if enableUniformResponse {
				if err = handleStandardizeResponseAction(jsonFile); err != nil {
					return err
				}
				fmt.Printf("Successfully standardize response format data in %s\n", jsonFile)
			}

			if enableConvertToOpenAPI3 {
				if err = handleSwagger2ToOpenAPI3Action(jsonFile); err != nil {
					return err
				}
				outputJSONFile, outputYamlFile := getOutputFile(jsonFile)
				fmt.Printf("Successfully convert swagger2.0 to openapi3.0, output: %s, %s\n", outputJSONFile, outputYamlFile)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&jsonFile, "file", "f", "docs/apis.swagger.json", "input swagger json file")
	cmd.Flags().BoolVarP(&enableUniformResponse, "enable-standardize-response", "u", false, "standardize response format data in swagger json")
	cmd.Flags().BoolVarP(&enableConvertToOpenAPI3, "enable-to-openapi3", "o", false, "convert swagger2.0 to openapi3")
	cmd.Flags().BoolVarP(&enableStringToInteger, "enable-string-to-integer", "t", true, "transform string into 64-bit integer in swagger.json fields")
	cmd.Flags().BoolVarP(&enableTransformIntegerToString, "enable-integer-to-string", "s", false, "deprecated, instead use --enable-string-to-integer")

	return cmd
}

// -------------------------------------------------------------------------------------------

// nolint
func handleStandardizeResponseAction(inputPath string) error {
	outputPath := inputPath
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read input file %s: %w", inputPath, err)
	}

	var swaggerDocMap map[string]interface{}
	if err = json.Unmarshal(data, &swaggerDocMap); err != nil {
		return fmt.Errorf("failed to unmarshal swagger.json: %w", err)
	}

	definitions, ok := swaggerDocMap["definitions"].(map[string]interface{})
	if !ok {
		definitions = make(map[string]interface{})
		swaggerDocMap["definitions"] = definitions
	}

	paths, ok := swaggerDocMap["paths"].(map[string]interface{})
	if !ok {
		fmt.Println("Warning: 'paths' section not found in swagger.json.")
	} else {
		for pathKey, pathItem := range paths {
			pathItemMap, ok := pathItem.(map[string]interface{})
			if !ok {
				continue
			}

			for methodKey, methodItemUntyped := range pathItemMap {
				validMethods := map[string]bool{"get": true, "put": true, "post": true, "delete": true, "options": true, "head": true, "patch": true, "trace": true}
				if !validMethods[strings.ToLower(methodKey)] {
					continue // Skip non-HTTP method keys like "parameters" at path level
				}

				methodItemMap, ok := methodItemUntyped.(map[string]interface{})
				if !ok {
					continue
				}

				responses, ok := methodItemMap["responses"].(map[string]interface{})
				if !ok {
					continue
				}

				if _, defaultExists := responses["default"]; defaultExists { //nolint
					delete(responses, "default")
				}

				response200Untyped, response200Exists := responses["200"]
				if !response200Exists {
					continue
				}
				response200, ok := response200Untyped.(map[string]interface{})
				if !ok {
					continue
				}

				var newHTTPResponseDefName string
				var dataSchemaForNewHTTPResponseDef interface{}

				schemaUntyped, schemaExistsIn200 := response200["schema"]

				if !schemaExistsIn200 {
					baseName := generateBaseNameForNewDefinition(pathKey, methodKey, methodItemMap)
					newHTTPResponseDefName = adjustHTTPResponseName(baseName)
					dataSchemaForNewHTTPResponseDef = map[string]interface{}{"type": "object", "description": "Original data schema was not present."}
					response200["schema"] = map[string]interface{}{"$ref": "#/definitions/" + newHTTPResponseDefName}
				} else {
					currentSchemaMap, schemaIsMap := schemaUntyped.(map[string]interface{})
					if !schemaIsMap {
						baseName := generateBaseNameForNewDefinition(pathKey, methodKey, methodItemMap)
						adjustHTTPResponseName(baseName)
						dataSchemaForNewHTTPResponseDef = map[string]interface{}{
							"type":        "object",
							"description": "Original schema was not a JSON object/map.",
							// Optionally, you could try to embed schemaUntyped here if it's simple enough
							// "originalValue": schemaUntyped,
						}
						response200["schema"] = map[string]interface{}{"$ref": "#/definitions/" + newHTTPResponseDefName}
					} else {
						// Schema is a map, proceed with deep copy and ref checking
						copiedSchemaInterface, err := deepCopy(currentSchemaMap) //nolint
						if err != nil {
							return fmt.Errorf("failed to deep copy schema for %s %s: %w", strings.ToUpper(methodKey), pathKey, err)
						}
						originalSchemaContent, castOk := copiedSchemaInterface.(map[string]interface{})
						if !castOk {
							return fmt.Errorf("deepCopied schema for %s %s is not a map after copy, type: %T", strings.ToUpper(methodKey), pathKey, copiedSchemaInterface)
						}

						refValue, refExistsInSchema := currentSchemaMap["$ref"].(string)
						isValidRef := false
						originalDefNameFromRef := ""

						if refExistsInSchema && strings.HasPrefix(refValue, "#/definitions/") {
							originalDefNameFromRef = strings.TrimPrefix(refValue, "#/definitions/")
							if originalDefNameFromRef != "" {
								isValidRef = true
							}
						}

						if isValidRef {
							newHTTPResponseDefName = adjustHTTPResponseName(originalDefNameFromRef)
							dataSchemaForNewHTTPResponseDef = map[string]interface{}{"$ref": refValue}
							if isHTTPResponseStructure(definitions, originalDefNameFromRef) {
								continue
							}
							response200["schema"] = map[string]interface{}{"$ref": "#/definitions/" + newHTTPResponseDefName}
						} else {
							newHTTPResponseDefName = "emptyHTTPResponse"
							dataSchemaForNewHTTPResponseDef = originalSchemaContent // Embed the deep copied original schema
							if isHTTPResponseStructure(definitions, originalDefNameFromRef) {
								continue
							}
							response200["schema"] = map[string]interface{}{"$ref": "#/definitions/" + newHTTPResponseDefName}
						}
					}
				}

				if newHTTPResponseDefName != "" {
					definitions[newHTTPResponseDefName] = map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"code": map[string]interface{}{"type": "integer", "format": "int32", "description": "Business status code"},
							"msg":  map[string]interface{}{"type": "string", "description": "Response message description"},
							"data": dataSchemaForNewHTTPResponseDef,
						},
						//"required": []string{"code", "msg","data"},
					}
				}
			}
		}
	}

	orderedSwaggerDoc := convertToOrderedMap(swaggerDocMap, topLevelSortKeys)

	modifiedData, err := json.MarshalIndent(orderedSwaggerDoc, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal modified swagger data: %w", err)
	}

	outputDir := filepath.Dir(outputPath)
	if outputDir != "." && outputDir != "" {
		if err = os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
		}
	}

	if err = os.WriteFile(outputPath, modifiedData, 0644); err != nil {
		return fmt.Errorf("failed to write output file %s: %w", outputPath, err)
	}

	return nil
}

func adjustHTTPResponseName(name string) string {
	suffixes := []string{"Response", "Resp", "Res", "Reply", "Rep", "Request", "Req"}
	hrName := "HTTPResponse"

	for _, suffix := range suffixes {
		if strings.HasSuffix(name, suffix) {
			newName := strings.TrimSuffix(name, suffix)
			if len(newName) > 0 {
				return newName + hrName
			}
			break
		}
	}

	return name + hrName
}

func isHTTPResponseStructure(definitions map[string]interface{}, name string) bool {
	def, ok := definitions[name].(map[string]interface{})
	if !ok {
		return false
	}

	if defType, ok := def["type"].(string); !ok || defType != "object" { //nolint
		return false
	}
	properties, ok := def["properties"].(map[string]interface{})
	if !ok {
		return false
	}
	_, hasCode := properties["code"]
	_, hasMsg := properties["msg"]
	_, hasData := properties["data"]
	return hasCode && hasMsg && hasData
}

var topLevelSortKeys = []string{
	"swagger", "info", "host", "basePath", "tags", "schemes",
	"consumes", "produces", "paths", "definitions",
	"securityDefinitions", "security", "externalDocs",
}

// KeyValue represents a key-value pair for ordered JSON marshaling
type KeyValue struct {
	Key   string
	Value interface{}
}

// OrderedMap is a slice of KeyValue pairs, representing an ordered JSON object.
type OrderedMap []KeyValue

// MarshalJSON custom marshals OrderedMap to JSON, preserving key order.
func (om OrderedMap) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("{")
	for i, kv := range om {
		if i > 0 {
			buf.WriteString(",")
		}
		keyBytes, err := json.Marshal(kv.Key)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal key %s: %w", kv.Key, err)
		}
		buf.Write(keyBytes)
		buf.WriteString(":")

		valueBytes, err := json.Marshal(kv.Value)
		if err != nil {
			valStr := fmt.Sprintf("%v", kv.Value)
			if len(valStr) > 100 {
				valStr = valStr[:100] + "..."
			}
			return nil, fmt.Errorf("failed to marshal value for key '%s' (value snippet: %s): %w", kv.Key, valStr, err)
		}
		buf.Write(valueBytes)
	}
	buf.WriteString("}")
	return buf.Bytes(), nil
}

func convertToOrderedMap(data interface{}, preferredKeyOrder []string) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		om := make(OrderedMap, 0, len(v))
		processedKeys := make(map[string]bool)

		for _, key := range preferredKeyOrder {
			if val, ok := v[key]; ok {
				om = append(om, KeyValue{Key: key, Value: convertToOrderedMap(val, nil)})
				processedKeys[key] = true
			}
		}

		var remainingKeys []string
		for key := range v {
			if !processedKeys[key] {
				remainingKeys = append(remainingKeys, key)
			}
		}
		sort.Strings(remainingKeys)

		for _, key := range remainingKeys {
			om = append(om, KeyValue{Key: key, Value: convertToOrderedMap(v[key], nil)})
		}
		return om

	case []interface{}:
		s := make([]interface{}, len(v))
		for i, item := range v {
			s[i] = convertToOrderedMap(item, nil)
		}
		return s
	default:
		return v
	}
}

func deepCopy(source interface{}) (interface{}, error) {
	if source == nil {
		return nil, nil
	}
	bytesData, err := json.Marshal(source)
	if err != nil {
		return nil, fmt.Errorf("deepCopy: marshal error: %w", err)
	}
	var dest interface{}
	if err = json.Unmarshal(bytesData, &dest); err != nil {
		return nil, fmt.Errorf("deepCopy: unmarshal error: %w", err)
	}
	return dest, nil
}

func generateBaseNameForNewDefinition(pathKey, methodKey string, methodItem map[string]interface{}) string {
	if opID, ok := methodItem["operationId"].(string); ok && opID != "" {
		if len(opID) > 0 {
			return strings.ToUpper(string(opID[0])) + opID[1:]
		}
		return "UnnamedOperation" // Should be rare if opID is non-empty
	}

	methodNamePart := ""
	if len(methodKey) > 0 {
		methodNamePart = strings.ToUpper(string(methodKey[0])) + strings.ToLower(methodKey[1:])
	}

	var pathNameParts []string
	cleanedPath := strings.ReplaceAll(pathKey, "{", "")
	cleanedPath = strings.ReplaceAll(cleanedPath, "}", "")
	cleanedPath = strings.ReplaceAll(cleanedPath, "_", "-")

	for _, part := range strings.Split(cleanedPath, "/") {
		if part == "" {
			continue
		}
		subParts := strings.Split(part, "-")
		var capitalizedSubParts []string
		for _, sp := range subParts {
			if len(sp) > 0 {
				capitalizedSubParts = append(capitalizedSubParts, strings.ToUpper(string(sp[0]))+strings.ToLower(sp[1:]))
			}
		}
		pathNameParts = append(pathNameParts, strings.Join(capitalizedSubParts, ""))
	}
	if len(pathNameParts) == 0 {
		return methodNamePart + "Root"
	}

	return methodNamePart + strings.Join(pathNameParts, "")
}

// -------------------------------------------------------------------------------------------

func handleSwagger2ToOpenAPI3Action(inputFile string) error {
	if gofile.GetFileSuffixName(inputFile) != ".json" {
		return fmt.Errorf("input file must be a json file")
	}

	data, err := os.ReadFile(inputFile)
	if err != nil {
		return err
	}

	outputYAML, outputJSON := getOutputFile(inputFile)

	var swaggerDoc openapi2.T
	if err = json.Unmarshal(data, &swaggerDoc); err != nil {
		return fmt.Errorf("parse swagger json file failed: %v", err)
	}

	openapi3Doc, err := openapi2conv.ToV3(&swaggerDoc)
	if err != nil {
		return fmt.Errorf("convert to openapi3 failed: %v", err)
	}

	jsonData, err := json.MarshalIndent(openapi3Doc, "", "  ")
	if err != nil {
		return fmt.Errorf("serialize to json failed: %v", err)
	}
	if err = os.WriteFile(outputJSON, jsonData, 0644); err != nil {
		return fmt.Errorf("write json file failed: %v", err)
	}

	yamlData, err := yaml.Marshal(openapi3Doc)
	if err != nil {
		return fmt.Errorf("serialize to yaml failed: %v", err)
	}
	if err = os.WriteFile(outputYAML, yamlData, 0644); err != nil {
		return fmt.Errorf("write yaml file failed: %v", err)
	}

	return nil
}

func getOutputFile(filePath string) (yamlFile string, jsonFile string) {
	var suffix string
	if strings.HasSuffix(filePath, "swagger.json") {
		suffix = "swagger.json"
	} else {
		suffix = gofile.GetFilename(filePath)
	}
	yamlFile = strings.TrimSuffix(filePath, suffix) + "openapi3.yaml"
	jsonFile = strings.TrimSuffix(filePath, suffix) + "openapi3.json"
	return yamlFile, jsonFile
}

// -------------------------------------------------------------------------------------------

func handleSwaggerFieldStringToInteger(jsonFilePath string) error {
	newData, err := convertStringToInteger(jsonFilePath)
	if err != nil {
		return err
	}

	return saveJSONFile(newData, jsonFilePath)
}

func convertStringToInteger(jsonFilePath string) ([]byte, error) {
	f, err := os.Open(jsonFilePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()

	scanner := bufio.NewScanner(f)
	contents := []string{}
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, `"format": "uint64"`) || strings.Contains(line, `"format": "int64"`) {
			l := len(contents)
			previousLine := contents[l-1]
			if len(contents) > 0 && strings.Contains(previousLine, `"type": "string"`) {
				contents[l-1] = strings.ReplaceAll(previousLine, `"type": "string"`, `"type": "integer"`)
			}
		}
		contents = append(contents, line+"\n")
	}
	if err = scanner.Err(); err != nil {
		return nil, err
	}

	newData := []byte{}
	for _, v := range contents {
		newData = append(newData, []byte(v)...)
	}

	return newData, nil
}

func saveJSONFile(data []byte, jsonFilePath string) error {
	if gofile.IsExists(jsonFilePath) {
		tmpFile := jsonFilePath + ".tmp"
		err := os.WriteFile(tmpFile, data, 0666)
		if err != nil {
			return err
		}
		return os.Rename(tmpFile, jsonFilePath)
	}

	dir := gofile.GetFileDir(jsonFilePath)
	_ = os.MkdirAll(dir, 0766)
	return os.WriteFile(jsonFilePath, data, 0666)
}
