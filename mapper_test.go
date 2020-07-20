package gofield_test

import (
	"testing"

	"github.com/henrylee2cn/gofield"
)

func TestBind1(t *testing.T) {
	maker := func(ft *gofield.FieldType) (string, bool) {
		tag, err := ft.Subtags.Get("mapper")
		if err != nil {
			return "", false
		}
		return tag.Name, true
	}
	mapper := gofield.NewMapper(maker)

	type P struct {
		Apple  string `mapper:"a"`
		banana int    `mapper:"b"`
	}
	var p P
	p.Apple = "red"
	p.banana = 7
	m := mapper.MustGet(&p)
	for k, v := range m {
		t.Logf("key=%s, value=%v", k, v.Interface())
	}
}
