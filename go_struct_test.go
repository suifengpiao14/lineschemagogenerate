package lineschemagogenerate_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/suifengpiao14/lineschema"
	"github.com/suifengpiao14/lineschemagogenerate"
)

func TestA(t *testing.T) {
	l, err := lineschema.ParseLineschema(emptyFullnameSchema)
	require.NoError(t, err)
	fmt.Println(l.String())
	structs := lineschemagogenerate.NewSturct(*l)
	fmt.Println(structs)
}

var emptyFullnameSchema = `
version=http://json-schema.org/draft-07/schema#,id=out
fullname=,type=proto,required,allowEmptyValue,title=协议,comment=协议
fullname=proto.code,required,title=业务码,comment=业务码
fullname=proto.message,required,title=业务提示,comment=业务提示
`
