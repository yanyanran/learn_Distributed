package test

import (
	"reflect"
	"testing"
)

type P int

func (c *P) Get(a int) {
	println("get", a)
}

func TestReflect(t *testing.T) {
	var test *P

	typ := reflect.TypeOf(test)
	val := reflect.ValueOf(test)
	a := 1

	p, _ := typ.MethodByName("Get")
	p.Func.Call([]reflect.Value{reflect.ValueOf(test), reflect.ValueOf(a)})
	val.MethodByName("Get").Call([]reflect.Value{reflect.ValueOf(a)})
}
