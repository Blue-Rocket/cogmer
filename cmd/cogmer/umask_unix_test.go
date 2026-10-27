//go:build !windows

package main

import "syscall"

func setUmask(m int) int { return syscall.Umask(m) }
