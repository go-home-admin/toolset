package bean

import (
	f "fmt"
	"os"
)

// @Bean fdsaf
type con struct {
	orm  orm   `inject:""`
	orm2 *orm  `inject:""`
	orm3 []orm `inject:""`
	orm4 func(i int, f map[string]string)
}

// test api
func (c con) api() {
	f.Println("test")

	f.Println(testInt(1))

	os.Exit(0)
}

//
type testInt int
