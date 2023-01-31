package main

import (
	"fmt"

	v8 "rogchap.com/v8go"
)

func main() {
	iso := v8.NewIsolate()
	global := v8.NewObjectTemplate(iso)

	// 注入方法
	fn := v8.NewFunctionTemplate(iso, func(info *v8.FunctionCallbackInfo) *v8.Value {
		for _, v := range info.Args() {
			fmt.Println(v)
		}
		val, _ := v8.NewValue(iso, "something")
		return val
	})

	// 注入变量
	abc, _ := v8.NewValue(iso, int32(456))

	global.Set("abc", abc, v8.ReadOnly)
	global.Set("print", fn, v8.ReadOnly)
	ctx := v8.NewContext(iso, global)
	defer ctx.Close()
	ctx.RunScript(`
print("abc", 123, {"abc":123});
print(abc)
	`, "")

	//获取返回结果

	resp, err := ctx.RunScript(`abc + 321`, "")
	fmt.Println(resp.Int32(), err)
}
