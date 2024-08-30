package llm

import (
	"encoding/json"
	"reflect"
	"strings"
)

type CallableFunctionImpl[Req any, Resp any] struct {
	Name        string
	Description string
	Handler     func(*Req) *Resp

	parameters  *ParamDef
}

func NewCallableFunction[Req any, Resp any](name string, description string, handler func(*Req) *Resp) *CallableFunctionImpl[Req, Resp] {
	return &CallableFunctionImpl[Req, Resp]{
		Name:        name,
		Description: description,
		Handler:     handler,
	}
}

type ErrorResp struct {
	Error string `json:"error"`
}

func (c *CallableFunctionImpl[Req, Resp]) Call(args string) string {
	req := new(Req)
	if err := json.Unmarshal([]byte(args), &req); err != nil {
		resp := ErrorResp{
			Error: "Invalid arguments",
		}

		respJSON, _ := json.Marshal(resp)
		return string(respJSON)
	}

	resp := c.Handler(req)

	respJSON, _ := json.Marshal(resp)
	return string(respJSON)
}

func (c *CallableFunctionImpl[Req, Resp]) GetName() string {
	return c.Name
}

func (c *CallableFunctionImpl[Req, Resp]) GetDescription() string {
	return c.Description
}

func (c *CallableFunctionImpl[Req, Resp]) GetParameters() ParamDef {
	if c.parameters == nil {
		c.makeParameters()
	}
	return *c.parameters
}

func (c *CallableFunctionImpl[Req, Resp]) makeParameters() {
	ty := reflect.TypeOf(*new(Req))

	def := getParamDef(ty)
	c.parameters = def
}

func getParamDef(ty reflect.Type) *ParamDef {
	for {
		if ty.Kind() == reflect.Ptr {
			ty = ty.Elem()
		} else {
			break
		}
	}

	if ty.Kind() == reflect.Slice {
		return &ParamDef{
			Type: "array",
			Items: getParamDef(ty.Elem()),
		}
	}

	if ty.Kind() != reflect.Struct {
		return &ParamDef{
			Type: getTypeName(ty),
		}
	}

	props := make(map[string]interface{})
	required := make([]string, 0)

	for i := 0; i < ty.NumField(); i++ {
		field := ty.Field(i)
		jsonTag := field.Tag.Get("json")
		descTag := field.Tag.Get("desc")

		jsonOpts := strings.Split(jsonTag, ",")
		jsonName := jsonOpts[0]

		fieldRequired := true
		for _, opt := range jsonOpts[1:] {
			if opt == "omitempty" {
				fieldRequired = false
				break
			}
		}
	
		props[jsonName] = getParamDef(field.Type)

		if descTag != "" {
			props[jsonName].(*ParamDef).Description = descTag
		}

		if fieldRequired {
			required = append(required, jsonName)
		}
	}

	return &ParamDef{
		Type:       "object",
		Properties: props,
		Required:   required,
	}
}

func getTypeName(ty reflect.Type) string {
	switch ty.Kind() {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "integer"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Bool:
		return "boolean"
	default:
		return "object"
	}
}