package parser

import (
	"fmt"
	"testing"
)

func TestIsIdentifier(t *testing.T) {
	fmt.Println('_')
}

func TestGetBrackets(t *testing.T) {
	l := []*word{
		{"func", wordT_word},
		{" ", wordT_division},
		{"TestGetBrackets", wordT_word},
		{"(", wordT_word},
		{"Ty", wordT_word},
		{" ", wordT_division},
		{")", wordT_division},
		{" ", wordT_division},
		{"{", wordT_division},
		{"{", wordT_division},
		{"fffff", wordT_word},
		{"}", wordT_division},
		{"}", wordT_division},
		{" ", wordT_division},
	}

	start, end := GetBrackets(l, "{", "}")
	if l[start].Str != "{" {
		t.Error("{解析失败")
	}
	if l[end].Str != "}" {
		t.Error("}解析失败")
	}
}

func TestLastIsIdentifier(t *testing.T) {
	l := []*word{
		{"func", wordT_word},
		{" ", wordT_division},
		{"TestGetBrackets", wordT_word},
		{"(", wordT_division},
		{"Ty", wordT_word},
		{" ", wordT_division},
		{")", wordT_division},
		{"(", wordT_division},
	}
	got, _ := GetLastIsIdentifier(l, "(")
	if got {
		t.Error("不是(结束的")
	}
	l = []*word{
		{"func", wordT_word},
		{" ", wordT_division},
		{"TestGetBrackets", wordT_word},
		{"(", wordT_division},
		{"// fdsafdsaf sa", wordT_doc},
	}
	got, _ = GetLastIsIdentifier(l, "(")
	if !got {
		t.Error("应该是(结束的")
	}
}

func Test_getWords(t *testing.T) {
	testStr := "protobuf:\"bytes,1,opt,name=area,proto3\" form:\"area\" json:\"area,omitempty\"" +
		"\n" +
		"inject:\"\" json:\"orm\""
	got := GetWords(testStr)

	if 19 != len(got) {
		t.Error("应该是(19)")
	}
}
