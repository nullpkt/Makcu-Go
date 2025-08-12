package makcu

import (
	"fmt"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var Debug = false

func DebugPrint(s string, a ...interface{}) {
	if Debug {
		fmt.Printf(s, a...)
	}
}

func utf16ToString(buf []uint16) string {
	for i, v := range buf {
		if v == 0 {
			return syscall.UTF16ToString(buf[:i])
		}
	}
	return syscall.UTF16ToString(buf)
}

func GetDeviceInfo(h uintptr, devInfo unsafe.Pointer, getProp *syscall.LazyProc, propertyCode uint32) string {
	buf := make([]uint16, 512)
	getProp.Call(h, uintptr(devInfo), uintptr(propertyCode), 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)*2), 0)
	return utf16ToString(buf)
}

func GetPortName(hDevInfo uintptr, devInfoData *byte) (string, error) {
	setupapi := syscall.NewLazyDLL("setupapi.dll")
	setupDiOpenDevRegKey := setupapi.NewProc("SetupDiOpenDevRegKey")

	regKey, _, err := setupDiOpenDevRegKey.Call(hDevInfo, uintptr(unsafe.Pointer(devInfoData)), 0x00000001, 0, 0x00000001, 0x20019)
	if regKey == 0 || regKey == ^uintptr(0) {
		return "", err
	}
	defer syscall.RegCloseKey(syscall.Handle(regKey))

	key := registry.Key(regKey)
	portName, _, err := key.GetStringValue("PortName")
	if err != nil {
		return "", err
	}
	return portName, nil
}

const (
	DeviceDescription = 0x0 // Device Description (ex: USB-Enhanced-SERIAL CH343)
	HardwareID        = 0x1 // Hardware ID (ex: USB\VID_1A86&PID_55D3&REV_0445)
	DeviceName        = 0xC // Friendly name of the device (ex: USB-Enhanced-SERIAL CH343 (COM3) )
)

// Find searches for the MAKCU device by name or VID/PID.
func Find() (makcuPort string, err error) {
	setupapi := syscall.NewLazyDLL("setupapi.dll")
	getClassDevs := setupapi.NewProc("SetupDiGetClassDevsW")
	enumDeviceInfo := setupapi.NewProc("SetupDiEnumDeviceInfo")
	getDeviceProperty := setupapi.NewProc("SetupDiGetDeviceRegistryPropertyW")
	destroyDeviceList := setupapi.NewProc("SetupDiDestroyDeviceInfoList")

	guid := windows.GUID{0x4d36e978, 0xe325, 0x11ce, [8]byte{0xbf, 0xc1, 0x08, 0x00, 0x2b, 0xe1, 0x03, 0x18}}

	h, _, _ := getClassDevs.Call(uintptr(unsafe.Pointer(&guid)), 0, 0, uintptr(0x2))
	if h == 0 || h == ^uintptr(0) {
		DebugPrint("Failed to get device list\n")
		return "", fmt.Errorf("Failed to get device list")
	}
	defer destroyDeviceList.Call(h)

	index := 0
	for {
		var devInfo struct {
			cbSize    uint32
			ClassGuid windows.GUID
			DevInst   uint32
			Reserved  uintptr
		}
		devInfo.cbSize = uint32(unsafe.Sizeof(devInfo))

		if ok, _, _ := enumDeviceInfo.Call(h, uintptr(index), uintptr(unsafe.Pointer(&devInfo))); ok == 0 {
			break
		}

		description := GetDeviceInfo(h, unsafe.Pointer(&devInfo), getDeviceProperty, DeviceDescription)
		hwid := GetDeviceInfo(h, unsafe.Pointer(&devInfo), getDeviceProperty, HardwareID)
		deviceName := GetDeviceInfo(h, unsafe.Pointer(&devInfo), getDeviceProperty, DeviceName)

		if deviceName == "" || description == "" || hwid == "" {
			index++
			continue
		}

		if strings.Contains(deviceName, "USB-Enhanced-SERIAL CH343") || strings.Contains(hwid, "VID_1A86&PID_55D3") {
			DebugPrint("--------\r\n")
			DebugPrint("Name: %s\r\n", deviceName)
			DebugPrint("Description: %s\r\n", description)
			DebugPrint("Hardware Info: %s\r\n", hwid)

			// Try to extract COM port from device name
			if strings.Contains(deviceName, "COM") {
				port := strings.Split(deviceName, "COM")[1]
				port = "COM" + port
				port = strings.ReplaceAll(port, "(", "")
				port = strings.ReplaceAll(port, ")", "")
				DebugPrint("Port Name: %s\r\n--------\r\n", port)
				return port, nil
			}

			// Fallback: try to get port from registry
			port, err := GetPortName(h, (*byte)(unsafe.Pointer(&devInfo)))
			if err != nil {
				DebugPrint("Failed to get port name: %v\n", err)
				return "", err
			}
			DebugPrint("Port Name: %s\n", port)
			DebugPrint("--------\r\n")
			if strings.Contains(port, "COM") {
				return port, nil
			}
			return "", nil
		}
		index++
	}
	return "", fmt.Errorf("Device not found")
}

// SetTimeouts sets the timeout settings for the COM port.
func SetTimeouts(handle windows.Handle) error {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	setCommTimeouts := kernel32.NewProc("SetCommTimeouts")

	var timeouts windows.CommTimeouts
	timeouts.ReadIntervalTimeout = 50
	timeouts.ReadTotalTimeoutMultiplier = 10
	timeouts.ReadTotalTimeoutConstant = 500
	timeouts.WriteTotalTimeoutMultiplier = 10
	timeouts.WriteTotalTimeoutConstant = 500

	ret, _, err := setCommTimeouts.Call(uintptr(handle), uintptr(unsafe.Pointer(&timeouts)))
	if ret == 0 {
		return fmt.Errorf("SetCommTimeouts Failed: %v", err)
	}
	return nil
}

type MakcuHandle struct {
	Port   string
	handle windows.Handle
	dcb    windows.DCB
}

// Connect opens a connection to the specified COM port at the given baud rate.
func Connect(portName string, baudRate uint32) (*MakcuHandle, error) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	openPort := kernel32.NewProc("CreateFileW")
	setCommState := kernel32.NewProc("SetCommState")

	if !strings.HasPrefix(portName, `\\.\`) {
		portName = `\\.\` + portName
	}

	path, err := windows.UTF16PtrFromString(portName)
	if err != nil {
		return nil, fmt.Errorf("Failed to convert port name to UTF16: %v", err)
	}

	handle, _, err := openPort.Call(uintptr(unsafe.Pointer(path)), syscall.GENERIC_READ|syscall.GENERIC_WRITE, 0, 0, 3, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if handle == uintptr(syscall.InvalidHandle) {
		return nil, fmt.Errorf("Failed to open port: %v", err)
	}

	portHandle := windows.Handle(handle)
	dcbOpts := &windows.DCB{}
	dcbOpts.DCBlength = uint32(unsafe.Sizeof(*dcbOpts))
	dcbOpts.BaudRate = baudRate
	dcbOpts.Flags = 0
	dcbOpts.ByteSize = 8
	dcbOpts.Parity = 0
	dcbOpts.StopBits = 1
	dcbOpts.Flags |= 0x00000400 | 0x00000800

	ret, _, err := setCommState.Call(uintptr(portHandle), uintptr(unsafe.Pointer(dcbOpts)))
	if ret == 0 {
		return nil, fmt.Errorf("Failed to set communication state: %v", err)
	}

	if err := SetTimeouts(portHandle); err != nil {
		return nil, fmt.Errorf("Failed to set timeouts: %v", err)
	}

	cleanPort := strings.TrimPrefix(portName, `\\.\`)
	DebugPrint("Successfully Connected to MAKCU! {Port %s | Baud Rate %d}\n", cleanPort, baudRate)

	return &MakcuHandle{
		Port:   cleanPort,
		handle: portHandle,
		dcb:    *dcbOpts,
	}, nil
}

// Close closes the connection to the MAKCU device.
func (m *MakcuHandle) Close() error {
	return windows.CloseHandle(m.handle)
}

// ChangeBaudRate sends the bytes needed to change the MAKCU baud rate to 4M, then returns a new connection at that rate.
// Note: This is NOT permanent and resets after power cycle.
func ChangeBaudRate(m *MakcuHandle) (*MakcuHandle, error) {
	n, err := m.Write([]byte{0xDE, 0xAD, 0x05, 0x00, 0xA5, 0x00, 0x09, 0x3D, 0x00})
	if err != nil {
		return nil, fmt.Errorf("Failed to change baud rate: Write Error: %v", err)
	}
	if n != 9 {
		return nil, fmt.Errorf("Failed to change baud rate: Wrong number of bytes written")
	}
	m.Close()
	newConn, err := Connect(m.Port, 4000000)
	if err != nil {
		return nil, fmt.Errorf("Failed to change baud rate: Connect Error: %v", err)
	}
	time.Sleep(1 * time.Second)
	_, err = newConn.Write([]byte("km.version()\r"))
	if err != nil {
		return nil, fmt.Errorf("Failed to change baud rate: Write Error: %v", err)
	}
	readBuf := make([]byte, 32)
	n, err = newConn.Read(readBuf)
	if err != nil {
		return nil, fmt.Errorf("Failed to change baud rate: Read Error: %v", err)
	}
	if !strings.Contains(string(readBuf[:n]), "MAKCU") {
		return nil, fmt.Errorf("Failed to change baud rate: Did Not Receive Expected Response...")
	}
	time.Sleep(1 * time.Second)
	DebugPrint("Successfully Changed Baud Rate To %d!\n", 4000000)
	return newConn, nil
}

// Write sends the given bytes to the MAKCU and returns the number of bytes written.
func (m *MakcuHandle) Write(data []byte) (int, error) {
	var bytesWritten uint32
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	writeFile := kernel32.NewProc("WriteFile")
	var overlapped windows.Overlapped
	overlapped.Offset = 0
	overlapped.OffsetHigh = 0
	DebugPrint("Sending %s\r\n", data)
	ret, _, err := writeFile.Call(uintptr(m.handle), uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)), uintptr(unsafe.Pointer(&bytesWritten)), uintptr(unsafe.Pointer(&overlapped)))
	if ret == 0 {
		return -1, fmt.Errorf("error writing to port: %v", err)
	}
	return int(bytesWritten), nil
}

// Read reads data from the MAKCU into the provided buffer and returns the number of bytes read.
func (m *MakcuHandle) Read(buffer []byte) (int, error) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	readFile := kernel32.NewProc("ReadFile")
	var bytesRead uint32
	ret, _, err := readFile.Call(uintptr(m.handle), uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)), uintptr(unsafe.Pointer(&bytesRead)), 0)
	if ret == 0 {
		return -1, fmt.Errorf("error reading from port: %v", err)
	}
	return int(bytesRead), nil
}

func (m *MakcuHandle) LeftDown() error {
	_, err := m.Write([]byte("km.left(1)\r"))
	if err != nil {
		DebugPrint("Failed to press mouse: Write Error: %v", err)
		return err
	}

	return nil
}

func (m *MakcuHandle) LeftUp() error {
	_, err := m.Write([]byte("km.left(0)\r"))
	if err != nil {
		DebugPrint("Failed to release mouse: Write Error: %v", err)
		return err
	}

	return nil
}

func (m *MakcuHandle) LeftClick() error {
	_, err := m.Write([]byte("km.left(1)\r km.left(0)\r"))
	if err != nil {
		DebugPrint("Failed to click mouse: %v", err)
		return err
	}

	return nil
}

func (m *MakcuHandle) RightDown() error {
	_, err := m.Write([]byte("km.right(1)\r"))
	if err != nil {
		DebugPrint("Failed to press mouse: Write Error: %v", err)
		return err
	}

	return nil
}

func (m *MakcuHandle) RightUp() error {
	_, err := m.Write([]byte("km.right(0)\r"))
	if err != nil {
		DebugPrint("Failed to release mouse: Write Error: %v", err)
		return err
	}

	return nil
}

func (m *MakcuHandle) RightClick() error {
	_, err := m.Write([]byte("km.right(1)\r km.right(0)\r"))
	if err != nil {
		DebugPrint("Failed to right click mouse: %v", err)
		return err
	}

	return nil
}

func (m *MakcuHandle) MiddleDown() error {
	_, err := m.Write([]byte("km.middle(1)\r"))
	if err != nil {
		DebugPrint("Failed to press middle mouse button: Write Error: %v", err)
		return err
	}

	return nil
}

func (m *MakcuHandle) MiddleUp() error {
	_, err := m.Write([]byte("km.middle(0)\r"))
	if err != nil {
		DebugPrint("Failed to release middle mouse button: Write Error: %v", err)
		return err
	}

	return nil
}

func (m *MakcuHandle) MiddleClick() error {
	_, err := m.Write([]byte("km.middle(1)\r km.middle(0)\r"))
	if err != nil {
		DebugPrint("Failed to middle click mouse: %v", err)
		return err
	}

	return nil
}

const (
	MOUSE_BUTTON_LEFT   = 1
	MOUSE_BUTTON_RIGHT  = 2
	MOUSE_BUTTON_MIDDLE = 3
)

func (m *MakcuHandle) ClickMouse(i int, delay time.Duration) error {
	// Basically, we create a function pointer which is just basically a variable that stores a function for us.
	// Then we can use that variable to call the function later on. :()
	type mouseAction func() error
	var down, up mouseAction

	switch i {
	case MOUSE_BUTTON_LEFT:
		down, up = m.LeftDown, m.LeftUp
	case MOUSE_BUTTON_RIGHT:
		down, up = m.RightDown, m.RightUp
	case MOUSE_BUTTON_MIDDLE:
		down, up = m.MiddleDown, m.MiddleUp
	default:
		return fmt.Errorf("Invalid mouse button: %d", i)
	}

	if err := down(); err != nil {
		return err
	}

	if delay > 0 {
		time.Sleep(delay)
	}

	if err := up(); err != nil {
		return err
	}

	return nil
}

func (m *MakcuHandle) ScrollMouse(amount int) error {
	_, err := m.Write([]byte(fmt.Sprintf("km.wheel(%d)\r", amount)))
	if err != nil {
		DebugPrint("Failed to scroll mouse: %v", err)
		return err
	}

	return nil
}

func (m *MakcuHandle) MoveMouse(x, y int) error {
	_, err := m.Write([]byte(fmt.Sprintf("km.move(%d, %d)\r", x, y)))
	if err != nil {
		DebugPrint("Failed to move mouse: Write Error: %v", err)
		return err
	}

	return nil
}

// MoveMouseWithCurve moves the mouse using MAKCU's built-in curve functionality.
// The higher the third parameter, the smoother the curve (per MAKCU docs).
func (m *MakcuHandle) MoveMouseWithCurve(x, y int, params ...int) error {
	var cmd string
	switch len(params) {
	case 0:
		cmd = fmt.Sprintf("km.move(%d, %d)\r", x, y)
	case 1:
		cmd = fmt.Sprintf("km.move(%d, %d, %d)\r", x, y, params[0])
	case 3:
		cmd = fmt.Sprintf("km.move(%d, %d, %d, %d, %d)\r", x, y, params[0], params[1], params[2])
	default:
		DebugPrint("Invalid number of parameters\n")
		return fmt.Errorf("Invalid number of parameters")
	}
	_, err := m.Write([]byte(cmd))
	if err != nil {
		DebugPrint("Failed to move mouse with curve: Write Error: %v\n", err)
		return err
	}
	return nil
}
