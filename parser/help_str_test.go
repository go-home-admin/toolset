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
		{"t", wordT_word},
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
	fmt.Println(start, end)
}

func TestLastIsIdentifier(t *testing.T) {
	l := []*word{
		{"func", wordT_word},
		{" ", wordT_division},
		{"TestGetBrackets", wordT_word},
		{"(", wordT_division},
		{"t", wordT_word},
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
