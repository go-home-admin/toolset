package bean

import (
	f "fmt"
)

// @Bean
type orm struct {
}

// test api
func (c orm) api() {
	f.Println("test")
}
