package main

import (
	"fmt"

	"github.com/dop251/goja"
)

func main() {
	vm := goja.New()
	// 返回值
	v, _ := vm.RunString("2+2")
	fmt.Println(v.Export().(int64))

	// 注入方法
	vm.Set("add", func(call goja.FunctionCall) goja.Value {
		var a, b int64
		a = call.Argument(0).ToInteger()
		b = call.Argument(1).ToInteger()

		val := vm.ToValue(a + b)
		return val
	})

	v, _ = vm.RunString(`add(1,2)`)
	fmt.Println(v.Export().(int64))

	// 导出方法
	vm.RunString(`
function sub(a,b) {
	return a - b
}
	`)

	sub, _ := goja.AssertFunction(vm.Get("sub"))

	v, _ = sub(goja.Undefined(), vm.ToValue(10), vm.ToValue(1))

	fmt.Println(v.Export().(int64))
}
