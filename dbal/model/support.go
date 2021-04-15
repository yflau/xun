package model

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/yaoapp/xun"
	"github.com/yaoapp/xun/dbal/query"
	"github.com/yaoapp/xun/dbal/schema"
)

// prepareRegisterNames parse the model name, return (fullname string, namespace string, name string)
func prepareRegisterNames(name string) (string, string, string) {
	sep := "."
	if strings.Contains(name, "/") {
		sep = "/"
	}
	name = strings.ToLower(strings.TrimPrefix(name, "*"))
	namer := strings.Split(name, sep)
	length := len(namer)
	if length <= 1 {
		return name, "", name
	}
	fullname := strings.Join(namer, ".")
	namespace := strings.Join(namer[0:length-1], ".")
	name = namer[length-1]
	return fullname, namespace, name
}

// prepareRegisterArgs parse the params for Register(), return (schema *Schema, flow *Flow)
func prepareRegisterArgs(args ...interface{}) (*Schema, *Flow) {
	var schema *Schema = nil
	var flow *Flow = nil

	if len(args) > 0 {
		content, ok := args[0].([]byte)
		if !ok {
			panic(fmt.Errorf("The schema type is %s, should be []byte", reflect.TypeOf(args[0]).String()))
		}

		schema = &Schema{}
		err := json.Unmarshal(content, schema)
		if err != nil {
			panic(fmt.Errorf("The parse schema error. %s ", err.Error()))
		}

	}

	if len(args) > 1 {
		content, ok := args[1].([]byte)
		if !ok {
			panic(fmt.Errorf("The flow type is %s, should be []byte", reflect.TypeOf(args[1]).String()))
		}

		flow = &Flow{}
		err := json.Unmarshal(content, flow)
		if err != nil {
			panic(fmt.Errorf("The parse flow error. %s ", err.Error()))
		}
	}

	return schema, flow
}

// prepareMigrateArgs parse the params for migrate, return (refresh bool, force bool)
func prepareMigrateArgs(args ...bool) (bool, bool) {
	refresh := false
	force := false
	if len(args) > 0 {
		refresh = args[0]
	}

	if len(args) > 1 {
		force = args[1]
	}

	return refresh, force
}

func prepareBlueprintArgs(method string, column *Column) []reflect.Value {
	in := []reflect.Value{reflect.ValueOf(column.Name)}
	switch method {
	case "String", "Char", "Binary":
		if column.Length > 0 {
			in = append(in, reflect.ValueOf(column.Length))
		}
		break
	case "Decimal", "UnsignedDecimal", "Float", "UnsignedFloat", "Double", "UnsignedDouble":
		args := []int{}
		if column.Precision > 0 {
			args = append(args, column.Precision)
		}
		if column.Scale > 0 {
			if len(args) == 0 {
				args = append(args, 10)
			}
			args = append(args, column.Scale)
		}
		if len(args) > 0 {
			for _, arg := range args {
				in = append(in, reflect.ValueOf(arg))
			}
		}
		break
	case "DateTime", "DateTimeTz", "Time", "TimeTz", "Timestamp", "TimestampTz", "Timestamps", "TimestampsTz", "SoftDeletes", "SoftDeletesTz":
		if column.Precision > 0 {
			in = append(in, reflect.ValueOf(column.Length))
		}
		break
	case "Enum":
		in = append(in, reflect.ValueOf(column.Option))
		break
	}
	return in
}

func getTypeName(v interface{}) string {
	return reflect.TypeOf(v).String()
}

// determine if the interface{} is json schema
func isSchema(reflectValue reflect.Value, args ...interface{}) bool {
	return reflectValue.Kind() == reflect.String && len(args) > 0
}

// determine if the interface{} is golang struct
func isStruct(reflectPtr reflect.Value, reflectValue reflect.Value) bool {
	return reflectPtr.Kind() == reflect.Ptr && reflectValue.Kind() == reflect.Struct && reflectValue.FieldByName("Model").Type() == typeOfModel
}

// register the model by given json schema
func registerSchema(v interface{}, args ...interface{}) {
	origin := v.(string)
	fullname, namespace, name := prepareRegisterNames(origin)
	schema, flow := prepareRegisterArgs(args...)
	model := Model{}
	model.namespace = namespace
	model.name = name
	setupAttributes(&model, schema)

	factory := &Factory{
		Namespace: namespace,
		Name:      name,
		Model:     &model,
		Schema:    schema,
		Flow:      flow,
	}
	modelsRegistered[origin] = factory
	modelsAlias[origin] = modelsRegistered[origin]
	modelsAlias[fullname] = modelsRegistered[origin]
}

// register the model by given golang struct pointer
func registerStruct(reflectPtr reflect.Value, reflectValue reflect.Value, v interface{}, args ...interface{}) {
	origin := reflectPtr.Type().String()
	fullname, namespace, name := prepareRegisterNames(origin)
	schema, flow := prepareRegisterArgs(args...)
	SetModel(v, func(model *Model) {
		model.namespace = namespace
		model.name = name
		setupAttributesStruct(model, schema, reflectValue)
	})

	factory := &Factory{
		Namespace: namespace,
		Name:      name,
		Model:     v,
		Schema:    schema,
		Flow:      flow,
	}
	modelsRegistered[origin] = factory
	modelsAlias[origin] = modelsRegistered[origin]
	modelsAlias[fullname] = modelsRegistered[origin]
}

func setupAttributesStruct(model *Model, schema *Schema, reflectValue reflect.Value) {

	columns := []Column{}
	for i := 0; i < reflectValue.NumField(); i++ {
		column := fieldToColumn(reflectValue.Type().Field(i))
		if column != nil {
			columns = append(columns, *column)
		}
	}

	columns = append(columns, schema.Columns...)

	// merge schema
	columnsMap := map[string]Column{}
	for _, column := range columns {
		if col, has := columnsMap[column.Name]; has {
			columnsMap[column.Name] = *col.merge(column)
		} else {
			columnsMap[column.Name] = column
		}
	}

	schema.Columns = []Column{}
	for _, column := range columnsMap {
		schema.Columns = append(schema.Columns, column)
	}

	setupAttributes(model, schema)
}

func fieldToColumn(field reflect.StructField) *Column {
	if field.Type == typeOfModel {
		return nil
	}

	column, has := StructMapping[field.Type.Kind()]
	if !has {
		return nil
	}

	ctag := parseFieldTag(string(field.Tag))
	if ctag != nil {
		column = *column.merge(*ctag)
	}

	if column.Name == "" {
		column.Name = xun.ToSnakeCase(field.Name)
	}
	return &column
}

func parseFieldTag(tag string) *Column {
	if !strings.Contains(tag, "x-") {
		return nil
	}

	params := map[string]string{}
	tagarr := strings.Split(tag, "x-")

	for _, tagstr := range tagarr {
		tagr := strings.Split(tagstr, ":")
		if len(tagr) == 2 {
			key := strings.Trim(tagr[0], " ")
			value := strings.Trim(strings.Trim(tagr[1], " "), "\"")
			key = strings.TrimPrefix(key, "x-")
			key = strings.ReplaceAll(key, "-", ".")
			if key == "json" {
				key = "name"
			}
			params[key] = value
		}
	}

	if len(params) == 0 {
		return nil
	}

	column := Column{}
	for name, value := range params {
		column.set(name, value)
	}

	return &column
}

func setupAttributes(model *Model, schema *Schema) {

	// init
	model.attributes = map[string]Attribute{}

	// set Columns
	for i, column := range schema.Columns {
		name := column.Name
		attr := Attribute{
			Name:         column.Name,
			Column:       &schema.Columns[i],
			Value:        nil,
			Relationship: nil,
		}
		model.attributes[name] = attr
	}

	// set Relationships
	for i, relation := range schema.Relationships {
		name := relation.Name
		attr := Attribute{
			Name:         relation.Name,
			Relationship: &schema.Relationships[i],
			Column:       nil,
			Value:        nil,
		}
		model.attributes[name] = attr
	}
}

// makeBySchema make a new xun model instance
func makeBySchema(query query.Query, schema schema.Schema, v interface{}, args ...interface{}) *Model {

	name, ok := v.(string)
	if !ok {
		panic(fmt.Errorf("the model name is not string"))
	}

	class, has := modelsRegistered[name]
	if !has {
		Register(name, args...)
		class, has = modelsRegistered[name]
		if !has {
			panic(fmt.Errorf("the model register failure"))
		}
	}
	model := class.New()
	model.schema = schema
	model.query = query
	return model
}

// makeByStruct make a new xun model instance
func makeByStruct(query query.Query, schema schema.Schema, v interface{}) {
	name := getTypeName(v)
	Class(name).New(v)
	SetModel(v, func(model *Model) {
		model.query = query
		model.schema = schema
	})
}