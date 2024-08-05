//go:build !windows

package sync

// Stub implementation
func getDiskQueue() ([]uint32, error) {
	return nil, nil
}
