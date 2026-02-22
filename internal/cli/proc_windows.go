//go:build windows
// +build windows

package cli

import "syscall"

// getSysProcAttr는 백그라운드 실행을 위한 SysProcAttr을 반환합니다.
// Windows에서는 기본값 사용
func getSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}
