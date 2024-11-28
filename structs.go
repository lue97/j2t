package main

import (
	"regexp"

	"github.com/valyala/fastjson"
)

var decimalRegex = regexp.MustCompile(`^([-+])?\d+$`)

type typeMap map[string]string

const (
	typeString      = "string"
	typeNumber      = "number"
	typeNumberFloat = "number_float"
	typeNumberInt   = "number_int"
	typeBool        = "bool"
	typeUnknown     = "unknown"
)

func (m typeMap) Merge(other typeMap) typeMap {
	if other == nil {
		return m
	}
	for k, v := range other {
		m[k] = v
	}
	hasNull := len(m["null"]) > 0
	if hasNull && len(m) > 1 {
		delete(m, "null")
	}
	if len(m[typeNumberFloat]) > 0 || len(other[typeNumberFloat]) > 0 {
		delete(m, typeNumberInt)
	}
	return m
}

func Parse(prefix string, val *fastjson.Value, out map[string]typeMap, categorizeNumeric bool) {
	parse(prefix, val, out, categorizeNumeric)
}

func parse(prefix string, val *fastjson.Value, out map[string]typeMap, categorizeNumeric bool) {
	if val.Type() == fastjson.TypeString {
		m := typeMap{typeString: val.String()}
		out[prefix] = m.Merge(out[prefix])
		return
	}

	if val.Type() == fastjson.TypeNull {
		m := typeMap{typeUnknown: val.String()}
		out[prefix] = m.Merge(out[prefix])
		return
	}

	if val.Type() == fastjson.TypeTrue || val.Type() == fastjson.TypeFalse {
		m := typeMap{typeBool: val.String()}
		out[prefix] = m.Merge(out[prefix])
		return
	}

	if val.Type() == fastjson.TypeNumber {
		rawString := val.String()
		if !categorizeNumeric {
			m := typeMap{typeNumber: val.String()}
			out[prefix] = m.Merge(out[prefix])
			return
		}
		if decimalRegex.MatchString(rawString) {
			m := typeMap{typeNumberInt: val.String()}
			out[prefix] = m.Merge(out[prefix])
			return
		}
		m := typeMap{typeNumberFloat: val.String()}
		out[prefix] = m.Merge(out[prefix])
		return
	}

	if val.Type() == fastjson.TypeObject {
		obj, _ := val.Object()
		obj.Visit(func(k []byte, v *fastjson.Value) {
			parse(prefix+"."+string(k), v, out, categorizeNumeric)
		})
		return
	}

	if val.Type() == fastjson.TypeArray {
		arr, _ := val.Array()
		for _, v := range arr {
			parse(prefix+"[]", v, out, categorizeNumeric)
		}
	}
}
