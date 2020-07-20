package gofield_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/henrylee2cn/gofield"
)

func TestMapper1(t *testing.T) {
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
	m := mapper.MustMake(&p)
	assert.Equal(t, "red", m["a"].String())
	assert.Equal(t, 7, int(m["b"].Int()))
}
