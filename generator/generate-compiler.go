// Copyright 2017 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/googleapis/gnostic/printer"
)

func (domain *Domain) GenerateCompiler(packageName string, license string, imports []string) string {
	code := &printer.Code{}
	code.Print(license)
	code.Print("// THIS FILE IS AUTOMATICALLY GENERATED.\n")

	// generate package declaration
	code.Print("package %s\n", packageName)

	code.Print("import (")
	for _, filename := range imports {
		code.Print("\"" + filename + "\"")
	}
	code.Print(")\n")

	// generate a simple Version() function
	code.Print("func Version() string {")
	code.Print("  return \"%s\"", packageName)
	code.Print("}\n")

	typeNames := domain.sortedTypeNames()

	// generate NewX() constructor functions for each type
	for _, typeName := range typeNames {
		domain.generateConstructorForType(code, typeName)
	}

	// generate ResolveReferences() methods for each type
	for _, typeName := range typeNames {
		domain.generateResolveReferencesMethodsForType(code, typeName)
	}

	return code.String()
}

func (domain *Domain) generateConstructorForType(code *printer.Code, typeName string) {
	code.Print("func New%s(in interface{}, context *compiler.Context) (*%s, error) {", typeName, typeName)
	code.Print("errors := make([]error, 0)")

	typeModel := domain.TypeModels[typeName]
	parentTypeName := typeName

	if typeModel.IsStringArray {
		code.Print("x := &TypeItem{}")
		code.Print("switch in := in.(type) {")
		code.Print("case string:")
		code.Print("  x.Value = make([]string, 0)")
		code.Print("  x.Value = append(x.Value, in)")
		code.Print("case []interface{}:")
		code.Print("  x.Value = make([]string, 0)")
		code.Print("  for _, v := range in {")
		code.Print("    value, ok := v.(string)")
		code.Print("    if ok {")
		code.Print("      x.Value = append(x.Value, value)")
		code.Print("    } else {")
		code.Print("      message := fmt.Sprintf(\"has unexpected value for string array element: %%+v (%%T)\", value, value)")
		code.Print("      errors = append(errors, compiler.NewError(context, message))")
		code.Print("    }")
		code.Print("  }")
		code.Print("default:")
		code.Print("  message := fmt.Sprintf(\"has unexpected value for string array: %%+v (%%T)\", in, in)")
		code.Print("  errors = append(errors, compiler.NewError(context, message))")
		code.Print("}")
	} else if typeModel.IsItemArray {
		if domain.Version == "v2" {
			code.Print("x := &ItemsItem{}")
			code.Print("m, ok := compiler.UnpackMap(in)")
			code.Print("if !ok {")
			code.Print("  message := fmt.Sprintf(\"has unexpected value for item array: %%+v (%%T)\", in, in)")
			code.Print("  errors = append(errors, compiler.NewError(context, message))")
			code.Print("} else {")
			code.Print("  x.Schema = make([]*Schema, 0)")
			code.Print("  y, err := NewSchema(m, compiler.NewContext(\"<array>\", context))")
			code.Print("  if err != nil {")
			code.Print("    return nil, err")
			code.Print("  }")
			code.Print("  x.Schema = append(x.Schema, y)")
			code.Print("}")
		} else if domain.Version == "v3" {
			code.Print("x := &ItemsItem{}")
			code.Print("m, ok := compiler.UnpackMap(in)")
			code.Print("if !ok {")
			code.Print("  message := fmt.Sprintf(\"has unexpected value for item array: %%+v (%%T)\", in, in)")
			code.Print("  errors = append(errors, compiler.NewError(context, message))")
			code.Print("} else {")
			code.Print("  x.SchemaOrReference = make([]*SchemaOrReference, 0)")
			code.Print("  y, err := NewSchemaOrReference(m, compiler.NewContext(\"<array>\", context))")
			code.Print("  if err != nil {")
			code.Print("    return nil, err")
			code.Print("  }")
			code.Print("  x.SchemaOrReference = append(x.SchemaOrReference, y)")
			code.Print("}")
		}
	} else if typeModel.IsBlob {
		code.Print("x := &Any{}")
		code.Print("bytes, _ := yaml.Marshal(in)")
		code.Print("x.Yaml = string(bytes)")
	} else if typeModel.Name == "StringArray" {
		code.Print("x := &StringArray{}")
		code.Print("a, ok := in.([]interface{})")
		code.Print("if !ok {")
		code.Print("  message := fmt.Sprintf(\"has unexpected value for StringArray: %%+v (%%T)\", in, in)")
		code.Print("  errors = append(errors, compiler.NewError(context, message))")
		code.Print("} else {")
		code.Print("  x.Value = make([]string, 0)")
		code.Print("  for _, s := range a {")
		code.Print("    x.Value = append(x.Value, s.(string))")
		code.Print("  }")
		code.Print("}")
	} else if typeModel.Name == "Primitive" {
		code.Print("	x := &Primitive{}")
		code.Print("	matched := false")
		code.Print("	switch in := in.(type) {")
		code.Print("	case bool:")
		code.Print("		x.Oneof = &Primitive_Boolean{Boolean: in}")
		code.Print("		matched = true")
		code.Print("	case string:")
		code.Print("		x.Oneof = &Primitive_String_{String_: in}")
		code.Print("		matched = true")
		code.Print("	case int64:")
		code.Print("		x.Oneof = &Primitive_Integer{Integer: in}")
		code.Print("		matched = true")
		code.Print("	case int32:")
		code.Print("		x.Oneof = &Primitive_Integer{Integer: int64(in)}")
		code.Print("		matched = true")
		code.Print("	case int:")
		code.Print("		x.Oneof = &Primitive_Integer{Integer: int64(in)}")
		code.Print("		matched = true")
		code.Print("	case float64:")
		code.Print("		x.Oneof = &Primitive_Number{Number: in}")
		code.Print("		matched = true")
		code.Print("	case float32:")
		code.Print("		x.Oneof = &Primitive_Number{Number: float64(in)}")
		code.Print("		matched = true")
		code.Print("	}")
		code.Print("	if matched {")
		code.Print("		// since the oneof matched one of its possibilities, discard any matching errors")
		code.Print("		errors = make([]error, 0)")
		code.Print("	}")
	} else if typeModel.Name == "SpecificationExtension" {
		code.Print("	x := &SpecificationExtension{}")
		code.Print("	matched := false")
		code.Print("	switch in := in.(type) {")
		code.Print("	case bool:")
		code.Print("		x.Oneof = &SpecificationExtension_Boolean{Boolean: in}")
		code.Print("		matched = true")
		code.Print("	case string:")
		code.Print("		x.Oneof = &SpecificationExtension_String_{String_: in}")
		code.Print("		matched = true")
		code.Print("	case int64:")
		code.Print("		x.Oneof = &SpecificationExtension_Integer{Integer: in}")
		code.Print("		matched = true")
		code.Print("	case int32:")
		code.Print("		x.Oneof = &SpecificationExtension_Integer{Integer: int64(in)}")
		code.Print("		matched = true")
		code.Print("	case int:")
		code.Print("		x.Oneof = &SpecificationExtension_Integer{Integer: int64(in)}")
		code.Print("		matched = true")
		code.Print("	case float64:")
		code.Print("		x.Oneof = &SpecificationExtension_Number{Number: in}")
		code.Print("		matched = true")
		code.Print("	case float32:")
		code.Print("		x.Oneof = &SpecificationExtension_Number{Number: float64(in)}")
		code.Print("		matched = true")
		code.Print("	}")
		code.Print("	if matched {")
		code.Print("		// since the oneof matched one of its possibilities, discard any matching errors")
		code.Print("		errors = make([]error, 0)")
		code.Print("	}")
	} else {
		oneOfWrapper := typeModel.OneOfWrapper

		code.Print("x := &%s{}", typeName)

		if oneOfWrapper {
			code.Print("matched := false")
		}

		unpackAtTop := !oneOfWrapper || len(typeModel.Required) > 0
		if unpackAtTop {
			code.Print("m, ok := compiler.UnpackMap(in)")
			code.Print("if !ok {")
			code.Print("  message := fmt.Sprintf(\"has unexpected value: %%+v (%%T)\", in, in)")
			code.Print("  errors = append(errors, compiler.NewError(context, message))")
			code.Print("} else {")
		}
		if len(typeModel.Required) > 0 {
			// verify that map includes all required keys
			keyString := ""
			sort.Strings(typeModel.Required)
			for _, k := range typeModel.Required {
				if keyString != "" {
					keyString += ","
				}
				keyString += "\""
				keyString += k
				keyString += "\""
			}
			code.Print("requiredKeys := []string{%s}", keyString)
			code.Print("missingKeys := compiler.MissingKeysInMap(m, requiredKeys)")
			code.Print("if len(missingKeys) > 0 {")
			code.Print("  message := fmt.Sprintf(\"is missing required %%s: %%+v\", compiler.PluralProperties(len(missingKeys)), strings.Join(missingKeys, \", \"))")
			code.Print("  errors = append(errors, compiler.NewError(context, message))")
			code.Print("}")
		}

		if !typeModel.Open {
			// verify that map has no unspecified keys
			allowedKeys := make([]string, 0)
			for _, property := range typeModel.Properties {
				if !property.Implicit {
					allowedKeys = append(allowedKeys, property.Name)
				}
			}
			sort.Strings(allowedKeys)
			allowedKeyString := ""
			for _, allowedKey := range allowedKeys {
				if allowedKeyString != "" {
					allowedKeyString += ","
				}
				allowedKeyString += "\""
				allowedKeyString += allowedKey
				allowedKeyString += "\""
			}
			allowedPatternString := ""
			if typeModel.OpenPatterns != nil {
				for _, pattern := range typeModel.OpenPatterns {
					if allowedPatternString != "" {
						allowedPatternString += ","
					}
					allowedPatternString += "\""
					allowedPatternString += pattern
					allowedPatternString += "\""
				}
			}
			// verify that map includes only allowed keys and patterns
			code.Print("allowedKeys := []string{%s}", allowedKeyString)
			code.Print("allowedPatterns := []string{%s}", allowedPatternString)
			code.Print("invalidKeys := compiler.InvalidKeysInMap(m, allowedKeys, allowedPatterns)")
			code.Print("if len(invalidKeys) > 0 {")
			code.Print("  message := fmt.Sprintf(\"has invalid %%s: %%+v\", compiler.PluralProperties(len(invalidKeys)), strings.Join(invalidKeys, \", \"))")
			code.Print("  errors = append(errors, compiler.NewError(context, message))")
			code.Print("}")
		}

		var fieldNumber = 0
		for _, propertyModel := range typeModel.Properties {
			propertyName := propertyModel.Name
			fieldNumber += 1
			propertyType := propertyModel.Type
			if propertyType == "int" {
				propertyType = "int64"
			}
			var displayName = propertyName
			if displayName == "$ref" {
				displayName = "_ref"
			}
			if displayName == "$schema" {
				displayName = "_schema"
			}
			displayName = camelCaseToSnakeCase(displayName)

			var line = fmt.Sprintf("%s %s = %d;", propertyType, displayName, fieldNumber)
			if propertyModel.Repeated {
				line = "repeated " + line
			}
			code.Print("// " + line)

			fieldName := strings.Title(propertyName)
			if propertyName == "$ref" {
				fieldName = "XRef"
			}

			typeModel, typeFound := domain.TypeModels[propertyType]
			if typeFound && !typeModel.IsPair {
				if propertyModel.Repeated {
					code.Print("v%d := compiler.MapValueForKey(m, \"%s\")", fieldNumber, propertyName)
					code.Print("if (v%d != nil) {", fieldNumber)
					code.Print("  // repeated %s", typeModel.Name)
					code.Print("  x.%s = make([]*%s, 0)", fieldName, typeModel.Name)
					code.Print("  a, ok := v%d.([]interface{})", fieldNumber)
					code.Print("  if ok {")
					code.Print("    for _, item := range a {")
					code.Print("      y, err := New%s(item, compiler.NewContext(\"%s\", context))", typeModel.Name, propertyName)
					code.Print("      if err != nil {")
					code.Print("        errors = append(errors, err)")
					code.Print("      }")
					code.Print("      x.%s = append(x.%s, y)", fieldName, fieldName)
					code.Print("    }")
					code.Print("  }")
					code.Print("}")
				} else {
					if oneOfWrapper {
						code.Print("{")
						if !unpackAtTop {
							code.Print("  m, ok := compiler.UnpackMap(in)")
							code.Print("  if ok {")
						}
						code.Print("    // errors might be ok here, they mean we just don't have the right subtype")
						code.Print("    t, matching_error := New%s(m, compiler.NewContext(\"%s\", context))", typeModel.Name, propertyName)
						code.Print("    if matching_error == nil {")
						code.Print("      x.Oneof = &%s_%s{%s: t}", parentTypeName, typeModel.Name, typeModel.Name)
						code.Print("      matched = true")
						code.Print("    } else {")
						code.Print("      errors = append(errors, matching_error)")
						code.Print("    }")
						if !unpackAtTop {
							code.Print("  }")
						}
						code.Print("}")
					} else {
						code.Print("v%d := compiler.MapValueForKey(m, \"%s\")", fieldNumber, propertyName)
						code.Print("if (v%d != nil) {", fieldNumber)
						code.Print("  var err error")
						code.Print("  x.%s, err = New%s(v%d, compiler.NewContext(\"%s\", context))",
							fieldName, typeModel.Name, fieldNumber, propertyName)
						code.Print("  if err != nil {")
						code.Print("    errors = append(errors, err)")
						code.Print("  }")
						code.Print("}")
					}
				}
			} else if propertyType == "string" {
				if propertyModel.Repeated {
					code.Print("v%d := compiler.MapValueForKey(m, \"%s\")", fieldNumber, propertyName)
					code.Print("if (v%d != nil) {", fieldNumber)
					code.Print("  v, ok := v%d.([]interface{})", fieldNumber)
					code.Print("  if ok {")
					code.Print("    x.%s = compiler.ConvertInterfaceArrayToStringArray(v)", fieldName)
					code.Print("  } else {")
					code.Print("    message := fmt.Sprintf(\"has unexpected value for %s: %%+v (%%T)\", v%d, v%d)", propertyName, fieldNumber, fieldNumber)
					code.Print("    errors = append(errors, compiler.NewError(context, message))")
					code.Print("}")
					code.Print("}")
				} else {
					code.Print("v%d := compiler.MapValueForKey(m, \"%s\")", fieldNumber, propertyName)
					code.Print("if (v%d != nil) {", fieldNumber)
					code.Print("  x.%s, ok = v%d.(string)", fieldName, fieldNumber)
					code.Print("  if !ok {")
					code.Print("    message := fmt.Sprintf(\"has unexpected value for %s: %%+v (%%T)\", v%d, v%d)", propertyName, fieldNumber, fieldNumber)
					code.Print("    errors = append(errors, compiler.NewError(context, message))")
					code.Print("  }")
					code.Print("}")
				}
			} else if propertyType == "float" {
				code.Print("v%d := compiler.MapValueForKey(m, \"%s\")", fieldNumber, propertyName)
				code.Print("if (v%d != nil) {", fieldNumber)
				code.Print("  switch v%d := v%d.(type) {", fieldNumber, fieldNumber)
				code.Print("  case float64:")
				code.Print("    x.%s = v%d", fieldName, fieldNumber)
				code.Print("  case float32:")
				code.Print("    x.%s = float64(v%d)", fieldName, fieldNumber)
				code.Print("  case uint64:")
				code.Print("    x.%s = float64(v%d)", fieldName, fieldNumber)
				code.Print("  case uint32:")
				code.Print("    x.%s = float64(v%d)", fieldName, fieldNumber)
				code.Print("  case int64:")
				code.Print("    x.%s = float64(v%d)", fieldName, fieldNumber)
				code.Print("  case int32:")
				code.Print("    x.%s = float64(v%d)", fieldName, fieldNumber)
				code.Print("  case int:")
				code.Print("    x.%s = float64(v%d)", fieldName, fieldNumber)
				code.Print("  default:")
				code.Print("    message := fmt.Sprintf(\"has unexpected value for %s: %%+v (%%T)\", v%d, v%d)", propertyName, fieldNumber, fieldNumber)
				code.Print("    errors = append(errors, compiler.NewError(context, message))")
				code.Print("  }")
				code.Print("}")
			} else if propertyType == "int64" {
				code.Print("v%d := compiler.MapValueForKey(m, \"%s\")", fieldNumber, propertyName)
				code.Print("if (v%d != nil) {", fieldNumber)
				code.Print("  t, ok := v%d.(int)", fieldNumber)
				code.Print("  if ok {")
				code.Print("    x.%s = int64(t)", fieldName)
				code.Print("  } else {")
				code.Print("    message := fmt.Sprintf(\"has unexpected value for %s: %%+v (%%T)\", v%d, v%d)", propertyName, fieldNumber, fieldNumber)
				code.Print("    errors = append(errors, compiler.NewError(context, message))")
				code.Print("  }")
				code.Print("}")
			} else if propertyType == "bool" {
				if oneOfWrapper {
					propertyName := "Boolean"
					code.Print("boolValue, ok := in.(bool)")
					code.Print("if ok {")
					code.Print("  x.Oneof = &%s_%s{%s: boolValue}", parentTypeName, propertyName, propertyName)
					code.Print("}")
				} else {
					code.Print("v%d := compiler.MapValueForKey(m, \"%s\")", fieldNumber, propertyName)
					code.Print("if (v%d != nil) {", fieldNumber)
					code.Print("  x.%s, ok = v%d.(bool)", fieldName, fieldNumber)
					code.Print("  if !ok {")
					code.Print("    message := fmt.Sprintf(\"has unexpected value for %s: %%+v (%%T)\", v%d, v%d)", propertyName, fieldNumber, fieldNumber)
					code.Print("    errors = append(errors, compiler.NewError(context, message))")
					code.Print("  }")
					code.Print("}")
				}
			} else {
				mapTypeName := propertyModel.MapType
				if mapTypeName != "" {
					code.Print("// MAP: %s %s", mapTypeName, propertyModel.Pattern)
					if mapTypeName == "string" {
						code.Print("x.%s = make([]*NamedString, 0)", fieldName)
					} else {
						code.Print("x.%s = make([]*Named%s, 0)", fieldName, mapTypeName)
					}
					code.Print("for _, item := range m {")
					code.Print("k, ok := item.Key.(string)")
					code.Print("if ok {")
					code.Print("v := item.Value")
					if propertyModel.Pattern != "" {
						code.Print("if compiler.PatternMatches(\"%s\", k) {", propertyModel.Pattern)
					}

					code.Print("pair := &Named" + strings.Title(mapTypeName) + "{}")
					code.Print("pair.Name = k")

					if mapTypeName == "string" {
						code.Print("pair.Value = v.(string)")
					} else if mapTypeName == "Any" {
						code.Print("result := &Any{}")
						code.Print("handled, resultFromExt, err := compiler.HandleExtension(context, v, k)")
						code.Print("if handled {")
						code.Print("	if err != nil {")
						code.Print("		errors = append(errors, err)")
						code.Print("	} else {")
						code.Print("		bytes, _ := yaml.Marshal(v)")
						code.Print("		result.Yaml = string(bytes)")
						code.Print("		result.Value = resultFromExt")
						code.Print("		pair.Value = result")
						code.Print("	}")
						code.Print("} else {")
						code.Print("	pair.Value, err = NewAny(v, compiler.NewContext(k, context))")
						code.Print("	if err != nil {")
						code.Print("		errors = append(errors, err)")
						code.Print("	}")
						code.Print("}")

					} else {
						code.Print("var err error")
						code.Print("pair.Value, err = New%s(v, compiler.NewContext(k, context))", mapTypeName)
						code.Print("if err != nil {")
						code.Print("  errors = append(errors, err)")
						code.Print("}")
					}
					code.Print("x.%s = append(x.%s, pair)", fieldName, fieldName)
					if propertyModel.Pattern != "" {
						code.Print("}")
					}
					code.Print("}")
					code.Print("}")
				} else {
					code.Print("// TODO: %s", propertyType)
				}
			}
		}
		if unpackAtTop {
			code.Print("}")
		}
		if oneOfWrapper {
			code.Print("if matched {")
			code.Print("    // since the oneof matched one of its possibilities, discard any matching errors")
			code.Print("	errors = make([]error, 0)")
			code.Print("}")
		}
	}

	// assumes that the return value is in a variable named "x"
	code.Print("  return x, compiler.NewErrorGroupOrNil(errors)")
	code.Print("}\n")
}

// ResolveReferences() methods
func (domain *Domain) generateResolveReferencesMethodsForType(code *printer.Code, typeName string) {
	code.Print("func (m *%s) ResolveReferences(root string) (interface{}, error) {", typeName)
	code.Print("errors := make([]error, 0)")

	typeModel := domain.TypeModels[typeName]
	if typeModel.OneOfWrapper {
		// call ResolveReferences on whatever is in the Oneof.
		for _, propertyModel := range typeModel.Properties {
			propertyType := propertyModel.Type
			_, typeFound := domain.TypeModels[propertyType]
			if typeFound {
				code.Print("{")
				code.Print("p, ok := m.Oneof.(*%s_%s)", typeName, propertyType)
				code.Print("if ok {")
				if propertyType == "JsonReference" { // Special case for OpenAPI
					code.Print("info, err := p.%s.ResolveReferences(root)", propertyType)
					code.Print("if err != nil {")
					code.Print("  return nil, err")
					code.Print("} else if info != nil {")
					code.Print("  n, err := New%s(info, nil)", typeName)
					code.Print("  if err != nil {")
					code.Print("    return nil, err")
					code.Print("  } else if n != nil {")
					code.Print("    *m = *n")
					code.Print("    return nil, nil")
					code.Print("  }")
					code.Print("}")
				} else {
					code.Print("_, err := p.%s.ResolveReferences(root)", propertyType)
					code.Print("if err != nil {")
					code.Print("	return nil, err")
					code.Print("}")
				}
				code.Print("}")
				code.Print("}")
			}
		}
	} else {
		for _, propertyModel := range typeModel.Properties {
			propertyName := propertyModel.Name
			var displayName = propertyName
			if displayName == "$ref" {
				displayName = "_ref"
			}
			if displayName == "$schema" {
				displayName = "_schema"
			}
			displayName = camelCaseToSnakeCase(displayName)

			fieldName := strings.Title(propertyName)
			if propertyName == "$ref" {
				fieldName = "XRef"
				code.Print("if m.XRef != \"\" {")
				//code.Print("log.Printf(\"%s reference to resolve %%+v\", m.XRef)", typeName)
				code.Print("info, err := compiler.ReadInfoForRef(root, m.XRef)")

				code.Print("if err != nil {")
				code.Print("	return nil, err")
				code.Print("}")
				//code.Print("log.Printf(\"%%+v\", info)")

				if len(typeModel.Properties) > 1 {
					code.Print("if info != nil {")
					code.Print("  replacement, err := New%s(info, nil)", typeName)
					code.Print("  if err == nil {")
					code.Print("    *m = *replacement")
					code.Print("    return m.ResolveReferences(root)")
					code.Print("  }")
					code.Print("}")
				}

				code.Print("return info, nil")
				code.Print("}")
			}

			if !propertyModel.Repeated {
				propertyType := propertyModel.Type
				typeModel, typeFound := domain.TypeModels[propertyType]
				if typeFound && !typeModel.IsPair {
					code.Print("if m.%s != nil {", fieldName)
					code.Print("    _, err := m.%s.ResolveReferences(root)", fieldName)
					code.Print("    if err != nil {")
					code.Print("       errors = append(errors, err)")
					code.Print("    }")
					code.Print("}")
				}
			} else {
				propertyType := propertyModel.Type
				_, typeFound := domain.TypeModels[propertyType]
				if typeFound {
					code.Print("for _, item := range m.%s {", fieldName)
					code.Print("if item != nil {")
					code.Print("  _, err := item.ResolveReferences(root)")
					code.Print("  if err != nil {")
					code.Print("     errors = append(errors, err)")
					code.Print("  }")
					code.Print("}")
					code.Print("}")
				}

			}
		}
	}
	code.Print("  return nil, compiler.NewErrorGroupOrNil(errors)")
	code.Print("}\n")
}
