//go:build linux

package makcu

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

type MakcuHandle struct {
	Port string
	f    *os.File
}

// Find searches for the MAKCU device by VID/PID and returns the port name.
func Find() (string, error) {
	// Method 1: Check /dev/serial/by-id/
	matches, _ := filepath.Glob("/dev/serial/by-id/*")
	for _, path := range matches {
		lowerPath := strings.ToLower(path)
		if strings.Contains(lowerPath, "1a86") && strings.Contains(lowerPath, "usb_single_serial") {
			realPath, err := filepath.EvalSymlinks(path)
			if err == nil {
				DebugPrint("Found device via by-id: %s -> %s\n", path, realPath)
				return realPath, nil
			}
		}
	}

	// Method 2: Iterate /sys/class/tty/ttyUSB* and /sys/class/tty/ttyACM*
	dirs, _ := filepath.Glob("/sys/class/tty/ttyUSB*")
	acmDirs, _ := filepath.Glob("/sys/class/tty/ttyACM*")
	dirs = append(dirs, acmDirs...)

	for _, dir := range dirs {
		devName := filepath.Base(dir)
		deviceLink := filepath.Join(dir, "device")
		
		realDevicePath, err := filepath.EvalSymlinks(deviceLink)
		if err != nil {
			continue
		}

		// Check both the device path and its parent (for interfaces like ttyACM)
		pathsToCheck := []string{realDevicePath, filepath.Dir(realDevicePath)}

		for _, p := range pathsToCheck {
			vendorPath := filepath.Join(p, "idVendor")
			productPath := filepath.Join(p, "idProduct")

			vendor, err := ioutil.ReadFile(vendorPath)
			if err != nil {
				continue
			}
			product, err := ioutil.ReadFile(productPath)
			if err != nil {
				continue
			}

			v := strings.TrimSpace(string(vendor))
			prod := strings.TrimSpace(string(product))

			if v == "1a86" && prod == "55d3" {
				DebugPrint("Found device via sysfs: %s (%s:%s)\n", devName, v, prod)
				return "/dev/" + devName, nil
			}
		}
	}

	fmt.Println("Failed to locate MAKCU!")
	return "", fmt.Errorf("Find: device not found")
}

func getBaudRateConst(baudRate uint32) (uint32, error) {
	switch baudRate {
	case 9600:
		return unix.B9600, nil
	case 19200:
		return unix.B19200, nil
	case 38400:
		return unix.B38400, nil
	case 57600:
		return unix.B57600, nil
	case 115200:
		return unix.B115200, nil
	case 230400:
		return unix.B230400, nil
	case 460800:
		return unix.B460800, nil
	case 500000:
		return unix.B500000, nil
	case 576000:
		return unix.B576000, nil
	case 921600:
		return unix.B921600, nil
	case 1000000:
		return unix.B1000000, nil
	case 1152000:
		return unix.B1152000, nil
	case 1500000:
		return unix.B1500000, nil
	case 2000000:
		return unix.B2000000, nil
	case 2500000:
		return unix.B2500000, nil
	case 3000000:
		return unix.B3000000, nil
	case 3500000:
		return unix.B3500000, nil
	case 4000000:
		return unix.B4000000, nil
	default:
		return 0, fmt.Errorf("unsupported baud rate: %d", baudRate)
	}
}

// Connect opens the serial port and sets the baud rate.
func Connect(portName string, baudRate uint32) (*MakcuHandle, error) {
	f, err := os.OpenFile(portName, syscall.O_RDWR|syscall.O_NOCTTY|syscall.O_NONBLOCK, 0666)
	if err != nil {
		return nil, fmt.Errorf("Connect: failed to open port: %w", err)
	}

	fd := int(f.Fd())

	// Set exclusive lock
	if err := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		f.Close()
		return nil, fmt.Errorf("Connect: failed to lock port: %w", err)
	}

	termios, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("Connect: failed to get termios: %w", err)
	}

	baudConst, err := getBaudRateConst(baudRate)
	if err != nil {
		// Attempt to use custom baud rate if supported, but for now just fail or fallback
		// For MAKCU, 4000000 is used, which is standard in Linux usually
		f.Close()
		return nil, err
	}

	// Set raw mode
	termios.Cflag &^= unix.CSIZE | unix.PARENB
	termios.Cflag |= unix.CS8 | unix.CREAD | unix.CLOCAL
	termios.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP | unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
	termios.Oflag &^= unix.OPOST
	termios.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN
	termios.Cc[unix.VMIN] = 0
	termios.Cc[unix.VTIME] = 5 // 0.5 seconds read timeout

	// Set input/output speed
	termios.Ispeed = baudConst
	termios.Ospeed = baudConst

	if err := unix.IoctlSetTermios(fd, unix.TCSETS, termios); err != nil {
		f.Close()
		return nil, fmt.Errorf("Connect: failed to set termios: %w", err)
	}

	// Flush
	unix.IoctlSetInt(fd, unix.TCFLSH, unix.TCIOFLUSH)

	DebugPrint("Successfully Connected to MAKCU! {Port %s | Baud Rate %d}\n", portName, baudRate)

	return &MakcuHandle{
		Port: portName,
		f:    f,
	},
		nil
}

func (m *MakcuHandle) Close() error {
	if m == nil || m.f == nil {
		return fmt.Errorf("Close: MakcuHandle is nil or file is nil")
	}
	// Unlock? handled by close usually
	unix.Flock(int(m.f.Fd()), unix.LOCK_UN)
	return m.f.Close()
}

func (m *MakcuHandle) Write(data []byte) (int, error) {
	if m == nil || m.f == nil {
		return -1, fmt.Errorf("Write: MakcuHandle is nil")
	}
    
    DebugPrint("Sending %s\r\n", data[:])
	return m.f.Write(data)
}

func (m *MakcuHandle) Read(buffer []byte) (int, error) {
	if m == nil || m.f == nil {
		return -1, fmt.Errorf("Read: MakcuHandle is nil")
	}
	return m.f.Read(buffer)
}

// ChangeBaudRate for Linux
func ChangeBaudRate(m *MakcuHandle) (*MakcuHandle, error) {
	if m == nil {
		return nil, fmt.Errorf("ChangeBaudRate: MakcuHandle is nil")
	}

	n, err := m.Write([]byte{0xDE, 0xAD, 0x05, 0x00, 0xA5, 0x00, 0x09, 0x3D, 0x00})
	if err != nil {
		_ = m.Close()
		return nil, fmt.Errorf("ChangeBaudRate: write error: %w", err)
	}

	if n != 9 {
		_ = m.Close()
		return nil, fmt.Errorf("ChangeBaudRate: wrong number of bytes written (got %d, want 9)", n)
	}

    // Wait for transmission to finish
    unix.IoctlSetInt(int(m.f.Fd()), unix.TCSBRK, 1)

	if err := m.Close(); err != nil {
		ErrorPrint("ChangeBaudRate: failed to close old connection: %v", err)
	}
    
    // Allow device to settle
	time.Sleep(100 * time.Millisecond)

	NewConn, err := Connect(m.Port, 4000000)
	if err != nil {
		return nil, fmt.Errorf("ChangeBaudRate: connect error: %w", err)
	}

	time.Sleep(1 * time.Second)

	_, err = NewConn.Write([]byte("km.version()\r"))
	if err != nil {
		_ = NewConn.Close()
		return nil, fmt.Errorf("ChangeBaudRate: write error after reconnect: %w", err)
	}

	ReadBuf := make([]byte, 32)
	n, err = NewConn.Read(ReadBuf)
	if err != nil {
		_ = NewConn.Close()
		return nil, fmt.Errorf("ChangeBaudRate: read error after reconnect: %w", err)
	}

	if !strings.Contains(string(ReadBuf[:n]), "MAKCU") {
		_ = NewConn.Close()
		return nil, fmt.Errorf("ChangeBaudRate: did not receive expected response, got: %q", string(ReadBuf[:n]))
	}

	time.Sleep(1 * time.Second)

	DebugPrint("Successfully Changed Baud Rate To %d!\n", 4000000)

	return NewConn, nil
}
