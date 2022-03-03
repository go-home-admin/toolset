package bean

import (
	f "fmt"
	"os"
)

// @Bean fdsaf
type con struct {
	orm   orm   `inject:"" json:"orm"`
	orm2  *orm  `inject:""`
	orm3  []orm `inject:""`
	proto int   `protobuf:"bytes,1,opt,name=area,proto3" form:"area" json:"area,omitempty"`
	orm4  func(i int, f map[string]string)
}

// test api
func (c con) api() {
	f.Println("test")

	f.Println(testInt(1))

	os.Exit(0)
}

//
type testInt int
