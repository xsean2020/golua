#include <lua.h>
#include <lualib.h>
#include <lauxlib.h>
#include <string.h>
#include <stdlib.h>

#ifndef LUA_OK
#define LUA_OK 0
#endif

// 定义 LuaResult 结构体
typedef struct {
    const char *result;
    const char *error;
} LuaResult;



// 减少go 对cgo 的调用提升性能 
LuaResult execute_lua_function(lua_State *L, const char *function_name, const char *json_str) {
    LuaResult result;
    result.result = NULL;
    result.error = NULL;

    lua_getglobal(L, function_name); // 获取 Lua 函数
    if (!lua_isfunction(L, -1)) {
        result.error = "Function not found";
        return result;
    }

    // 将 JSON 字符串作为参数传递给 Lua 函数
    lua_pushstring(L, json_str);
    if (lua_pcall(L, 1, 1, 0) != LUA_OK) {
        result.error = lua_tostring(L, -1); // 如果调用失败，返回错误信息
        return result;
    }

    // 获取 Lua 函数的返回值
    if (!lua_isstring(L, -1)) {
        result.error = "Invalid return value";
        return result;
    }

    size_t len;
    result.result = lua_tolstring(L, -1, &len);
    lua_pop(L, 1); // 清理栈中的返回值
    return result;
}
