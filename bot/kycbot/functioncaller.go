package bot

import (
	"fmt"
	"reflect"

	"runtime/debug"

	"bitbucket.org/magmeng/go-utils/log"
)

func isFunction(f reflect.Value) bool {
	return f.Kind() == reflect.Func
}

func changeToFunction(function interface{}) (reflect.Value, bool) {
	fv := reflect.ValueOf(function)
	if isFunction(fv) {
		return fv, true
	}

	return fv, false
}

func changeToParams(params ...interface{}) []reflect.Value {
	var res []reflect.Value
	for _, p := range params {
		res = append(res, reflect.ValueOf(p))
	}
	return res
}

type functionCaller struct {
	f reflect.Value
	p []reflect.Value
}

func NewCaller(function interface{}, params ...interface{}) *functionCaller {
	var f functionCaller
	var ok bool
	f.f, ok = changeToFunction(function)
	if !ok {
		return nil
	}
	f.p = changeToParams(params...)
	return &f
}

func (c *functionCaller) Call(mustRecover bool) []reflect.Value {
	if c == nil {
		return nil
	}

	if mustRecover {
		defer func() {
			if r := recover(); r != nil {
				log.Errorfln("Panic", fmt.Errorf("panic: %v\r\n\r\n%s", r, debug.Stack()))
			}
		}()
	}

	return c.f.Call(c.p)
}
