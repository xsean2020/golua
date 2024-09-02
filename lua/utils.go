package lua

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
)

type LuaFunction struct{}

// utils
func (L *State) SetGlobals(globals map[string]interface{}) {
	for k, v := range globals {
		L.PushAny(v)
		L.SetGlobal(k)
	}
}

func (L *State) GetGlobals(names ...string) map[string]interface{} {
	globals := make(map[string]interface{})
	for _, name := range names {
		L.GetGlobal(name)
		v := L.ReadAny(-1)
		globals[name] = v
		L.Pop(1)
	}
	return globals
}

func (L *State) GetFullStack() []interface{} {
	tip := L.GetTop()
	values := make([]interface{}, tip)
	for i := 1; i <= tip; i++ {
		v := L.ReadAny(i)
		values[i-1] = v
	}
	return values
}

// read stuff
func (L *State) ReadAny(pos int) interface{} {
	switch L.Type(pos) {
	case LUA_TNIL:
		return nil
	case LUA_TNUMBER:
		return L.ToNumber(pos)
	case LUA_TBOOLEAN:
		return L.ToBoolean(pos)
	case LUA_TSTRING:
		return L.ToString(pos)
	case LUA_TTABLE:
		ret, _ := L.ReadTable(pos)
		return ret
	case LUA_TFUNCTION:
		return &LuaFunction{}
	}
	return nil
}

func (L *State) ReadString(pos int) (v string) {
	switch L.Type(pos) {
	case LUA_TNUMBER:
		return fmt.Sprint(L.ToNumber(pos))
	case LUA_TBOOLEAN:
		return fmt.Sprint(L.ToBoolean(pos))
	case LUA_TSTRING:
		return L.ToString(pos)
	}
	return ""
}

// istable
func (L *State) ReadTable(pos int) (interface{}, bool) {
	if pos < 0 {
		pos = L.GetTop() + 1 + pos
	}

	var size = L.ObjLen(pos)
	var slice []interface{}
	var mp map[string]interface{}
	isTable := size == 0
	if !isTable {
		slice = make([]interface{}, size)
	} else {
		mp = make(map[string]interface{})
	}

	L.PushNil()
	for L.Next(pos) != 0 {
		val := L.ReadAny(-1)
		L.Pop(1)
		if isTable {
			key := L.ReadString(-1)
			mp[key] = val
			continue
		}

		index := L.ToInteger(-1)
		slice[index-1] = val
	}

	if !isTable {
		return slice, isTable
	}
	// special case for forcing an empty array
	if _, ok := mp["__emptyarray"]; len(mp) == 1 && ok {
		return make([]interface{}, 0), false
	}
	return mp, isTable
}

// push stuff
func (L *State) PushMap(m map[string]interface{}) {
	L.CreateTable(0, len(m))
	for k, v := range m {
		L.PushAny(k)
		L.PushAny(v)
		L.RawSet(-3)
	}
}

func (L *State) PushSlice(s []interface{}) {
	L.CreateTable(len(s), 0)
	for i, v := range s {
		L.PushAny(v)
		L.RawSeti(-2, i+1)
	}
}

func (L *State) PushAny(ival interface{}) {
	if ival == nil {
		L.PushNil()
		return
	}

	rv := reflect.ValueOf(ival)
	switch rv.Kind() {
	case reflect.Func:
		L.PushGoFunction(func(L *State) int {
			fnType := rv.Type()
			fnArgs := fnType.NumIn()           // includes a potential variadic argument
			givenArgs := L.GetTop()            // args passed to function
			variadic := rv.Type().IsVariadic() // means the last argument is ...
			var numArgs int
			if variadic {
				// when variadic we can ignore the last argument
				// or accept many of it
				if givenArgs+1 >= fnArgs {
					numArgs = givenArgs
				} else {
					numArgs = fnArgs
				}
			} else {
				// function is limited to the number of fnArgs
				numArgs = fnArgs
			}

			// when it's less there's nothing we can do
			if numArgs > givenArgs {
				L.RaiseError(fmt.Sprintf("got %d arguments, needed %d", numArgs, givenArgs))
			}

			args := make([]reflect.Value, numArgs)
			for i := 0; i < numArgs; i++ {
				arg := L.ReadAny(i + 1)
				av := reflect.ValueOf(arg)

				var requiredType reflect.Type
				if i >= fnArgs-1 && variadic {
					requiredType = fnType.In(fnArgs - 1).Elem()
				} else {
					requiredType = fnType.In(i)
				}

				if av.IsValid() {
					if !av.Type().ConvertibleTo(requiredType) {
						L.ArgError(i+1, fmt.Sprintf("wrong argument type: got %s, wanted %s",
							av.Kind().String(), requiredType.Kind().String()))
					}

					av = av.Convert(requiredType)
				} else {
					av = reflect.New(requiredType).Elem()

					if !av.Type().AssignableTo(requiredType) {
						L.ArgError(i+1, fmt.Sprintf("wrong argument type: got '%s', wanted %s",
							arg, requiredType.Kind().String()))
					}
				}

				args[i] = av
			}

			defer func() {
				// recover from panics during function run
				if err := recover(); err != nil {
					L.RaiseError(fmt.Sprintf("function panic: %s", err))
				}
			}()
			returned := rv.Call(args)

			for _, ret := range returned {
				kind := ret.Kind()
				if (kind == reflect.Slice || kind == reflect.Map || kind == reflect.Interface) && ret.IsNil() {
					L.PushNil()
				} else {
					L.PushAny(ret.Interface())
				}
			}

			return len(returned)
		})
	case reflect.String:
		L.PushString(rv.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		L.PushNumber(float64(rv.Int()))
	case reflect.Uint, reflect.Uintptr, reflect.Uint8,
		reflect.Uint16, reflect.Uint32, reflect.Uint64:
		L.PushNumber(float64(rv.Uint()))
	case reflect.Float32, reflect.Float64:
		L.PushNumber(rv.Float())
	case reflect.Bool:
		L.PushBoolean(rv.Bool())
	case reflect.Slice:
		size := rv.Len()
		slice := make([]interface{}, size)
		for i := 0; i < size; i++ {
			slice[i] = rv.Index(i).Interface()
		}
		L.PushSlice(slice)
	case reflect.Map:
		m := make(map[string]interface{}, rv.Len())
		for _, key := range rv.MapKeys() {
			m[fmt.Sprint(key)] = rv.MapIndex(key).Interface()
		}
		L.PushMap(m)
	case reflect.Ptr, reflect.Struct:
		out := &bytes.Buffer{}
		enc := json.NewEncoder(out)
		enc.SetEscapeHTML(false)

		// if it has an Error() or String() method, call these instead of pushing nil.
		method, ok := rv.Type().MethodByName("Error")
		if ok {
			goto callmethod
		}
		method, ok = rv.Type().MethodByName("String")
		if ok {
			goto callmethod
		}

		// try to convert the struct into an object using json
		if err := enc.Encode(rv.Interface()); err == nil {
			var value interface{}
			json.Unmarshal(out.Bytes(), &value)
			L.PushAny(value)
			break
		}

		goto justpushnil
	callmethod:
		if method.Type.NumIn() == 1 /* 1 because the struct itself is an argument */ &&
			method.Type.NumOut() == 1 &&
			method.Type.Out(0).Kind() == reflect.String {

			res := method.Func.Call([]reflect.Value{rv})
			L.PushString(res[0].String())
			break
		}
	justpushnil:
		L.PushNil()
	default:
		L.PushNil()
	}
}
