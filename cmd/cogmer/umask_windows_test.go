//go:build windows

package main

func setUmask(m int) int { return m }
