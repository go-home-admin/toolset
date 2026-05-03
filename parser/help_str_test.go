package parser

import (
	"testing"
)

func TestIsIdentifier(t *testing.T) {
	_ = IsIdentifier('_')
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
	// 行末为 `)` 且此前 `(` 与参数已闭合，不应再视为「仍停在 ( 处」的场景
	l := []*word{
		{"func", wordT_word},
		{" ", wordT_division},
		{"TestGetBrackets", wordT_word},
		{"(", wordT_division},
		{"Ty", wordT_word},
		{" ", wordT_division},
		{")", wordT_division},
		{"", wordT_line},
	}
	got, _ := GetLastIsIdentifier(l, "(")
	if got {
		t.Error("期望 false：参数列表已闭合，不应判定为仍以 ( 结束")
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
