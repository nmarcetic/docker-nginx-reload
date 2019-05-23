package pids

import (
	"syscall"
)

// SendSignal sends Kill signal to each running process
func SendSignal(pids []int) error {
	for _, p := range pids {
		syscall.Kill(p, syscall.SIGHUP)
		if err := syscall.Kill(p, syscall.SIGHUP); err != nil {
			return err
		}
	}
	return nil
}
