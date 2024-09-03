#ifndef LUA_WRAPPER_H
#define LUA_WRAPPER_H

#include <lua.h>

typedef struct {
    const char *result;
    const char *error;
} LuaResult;

// 函数原型
LuaResult execute_lua_function(lua_State *L, const char *function_name, const char *json_str);
#endif // LUA_WRAPPER_H
