//go:build windows

package sync

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

type diskPerformance struct {
	BytesRead           int64
	BytesWritten        int64
	ReadTime            int64
	WriteTime           int64
	IdleTime            int64
	ReadCount           uint32
	WriteCount          uint32
	QueueDepth          uint32
	SplitCount          uint32
	QueryTime           int64
	StorageDeviceNumber uint32
	StorageManagerName  [8]uint16
	alignmentPadding    uint32 // necessary for 32bit support, see https://github.com/elastic/beats/pull/16553
}

// reference: https://github.com/shirou/gopsutil/blob/master/disk/disk_windows.go
func getDiskQueue() ([]uint32, error) {
	lpBuffer := make([]uint16, 254)
	var diskPerformance diskPerformance
	var queueData []uint32
	lpBufferLen, err := windows.GetLogicalDriveStrings(uint32(len(lpBuffer)), &lpBuffer[0])
	if err != nil {
		return queueData, err
	}
	for _, v := range lpBuffer[:lpBufferLen] {
		if 'A' <= v && v <= 'Z' {
			path := string(rune(v)) + ":"
			typepath, _ := windows.UTF16PtrFromString(path)
			typeret := windows.GetDriveType(typepath)
			if typeret == 0 {
				return nil, windows.GetLastError()
			}
			if typeret != windows.DRIVE_FIXED {
				continue
			}
			szDevice := fmt.Sprintf(`\\.\%s`, path)
			const IOCTL_DISK_PERFORMANCE = 0x70020
			ptr, err := syscall.UTF16PtrFromString(szDevice)
			if err != nil {
				continue
			}

			h, err := windows.CreateFile(ptr, 0, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, 0, 0)
			if err != nil {
				if err == windows.ERROR_FILE_NOT_FOUND {
					continue
				}
				return nil, err
			}
			defer windows.CloseHandle(h)

			var diskPerformanceSize uint32
			err = windows.DeviceIoControl(h, IOCTL_DISK_PERFORMANCE, nil, 0,
				(*byte)(unsafe.Pointer(&diskPerformance)),
				uint32(unsafe.Sizeof(diskPerformance)), &diskPerformanceSize, nil)
			if err != nil {
				return queueData, err
			}
			queueData = append(queueData, diskPerformance.QueueDepth)
		}
	}
	return queueData, nil
}
