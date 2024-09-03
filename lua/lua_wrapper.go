package lua

/*

#include "lua_wrapper.h"
#include <lua.h>
#include <lauxlib.h>
#include <lualib.h>
#include <stdlib.h>

*/
import "C"
import (
	"errors"
	"unsafe"
)

// 减少go 与cgo之间的转换，直接封装一个完整的方法调用，这里传入的参数是json
func (L *State) CallLua(method string, json []byte) (result string, err error) {
	fn := C.CString(method)
	defer C.free(unsafe.Pointer(fn))
	// 调用 Lua 函数并获取返回结果
	ret := C.execute_lua_function(L.s, fn, (*C.char)(unsafe.Pointer(&json[0])))
	// 处理结果
	if ret.error != nil {
		err = errors.New(C.GoString(ret.error))
	}
	if ret.result != nil {
		result = C.GoString(ret.result)
	}
	return
}
