package main

import (
	"fmt"

	"github.com/robertkrimen/otto"
)

func main() {
	vm := otto.New()

	// 注入变量
	vm.Set("def", map[string]interface{}{"abc": 123})
	// 注入方法
	vm.Set("Add", func(call otto.FunctionCall) otto.Value {
		var a, b int64
		a, _ = call.Argument(0).ToInteger()
		b, _ = call.Argument(1).ToInteger()

		val, _ := vm.ToValue(a + b)
		return val
	})
	vm.Run(`
		abc = Add(1,2);
		console.log("The value of abc is " + abc);
		console.log("The value of def is " , def.abc);
	`)

	// 变量取值
	if value, err := vm.Get("abc"); err == nil {
		if intVal, err := value.ToInteger(); err == nil {
			fmt.Println(intVal)
		}
	}
}
