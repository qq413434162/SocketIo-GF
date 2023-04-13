// Package core 封装单元测试用到的公共方法
package core

import (
	"fmt"
	"os"
	"testing"
)

// TestMain 会在下面所有测试方法执行开始前先执行，一般用于初始化资源和执行完后释放资源
func _(m *testing.M) {
	fmt.Println("初始化资源")
	// 运行go的测试，相当于调用main方法
	result := m.Run()
	fmt.Println("释放资源")
	// 退出程序
	os.Exit(result)
}
