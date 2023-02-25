package main

import (
	"fmt"

	"github.com/dop251/goja"
	"github.com/robertkrimen/otto"
	v8 "rogchap.com/v8go"
)

var iso *v8.Isolate
var global *v8.ObjectTemplate

func init() {
	iso = v8.NewIsolate()
	global = v8.NewObjectTemplate(iso)
	fnAdd := v8.NewFunctionTemplate(iso, func(info *v8.FunctionCallbackInfo) *v8.Value {
		v, _ := v8.NewValue(iso, info.Args()[0].Integer()+info.Args()[1].Integer())
		return v
	})
	global.Set("Add", fnAdd, v8.ReadOnly)
}

func RunOtto(a, b int64) int64 {
	vm := otto.New()
	vm.Set("Add", func(call otto.FunctionCall) otto.Value {
		var a, b int64
		a, _ = call.Argument(0).ToInteger()
		b, _ = call.Argument(1).ToInteger()

		val, _ := vm.ToValue(a + b)
		return val
	})
	val, _ := vm.Run(`Add(1,2)`)
	iVal, _ := val.ToInteger()
	return iVal

}

func OttoSum() int64 {
	vm := otto.New()

	val, _ := vm.Run(`
  var i = 0, sum = 0;
	for (; i < 100000; i ++){
		sum += i;
	}
	sum
	`)
	iVal, _ := val.ToInteger()
	return iVal
}
func RunGoja(a, b int64) int64 {
	vm := goja.New()
	vm.Set("Add", func(call goja.FunctionCall) goja.Value {
		var a, b int64
		a = call.Argument(0).ToInteger()
		b = call.Argument(1).ToInteger()

		return vm.ToValue(a + b)
	})

	v, _ := vm.RunString(`Add(1,2)`)
	return v.ToInteger()
}

func GojaSum() int64 {
	vm := goja.New()
	v, _ := vm.RunString(`
		let i = 0, sum = 0;
		for (; i < 100000; i ++){
			sum += i;
		}
		sum
	`)
	return v.ToInteger()
}

func RunV8(a, b int64) int64 {
	ctx := v8.NewContext(iso, global)
	defer ctx.Close()
	v, _ := ctx.RunScript(`Add(1,2)`, "")
	return v.BigInt().Int64()
}

func V8Sum() int64 {
	ctx := v8.NewContext(iso, global)
	defer ctx.Close()
	v, _ := ctx.RunScript(`
		var i = 0, sum = 0;
		for (; i < 100000; i ++){
			sum += i;
		}
		sum
	`, "")
	return v.Integer()
}

func main() {
	fmt.Println(RunOtto(1, 2))
	fmt.Println(RunGoja(1, 2))
	fmt.Println(RunV8(1, 2))
	fmt.Println(V8Sum())
	fmt.Println(OttoSum())
	fmt.Println(GojaSum())
}
