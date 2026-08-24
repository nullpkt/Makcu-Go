package makcu

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

var Debug bool = false

var logger = slog.New(slog.NewTextHandler(os.Stdout, nil))

func DebugPrint(s string, a ...interface{}) {
	if Debug {
		logger.Debug(fmt.Sprintf(s, a...))
	}
}

func InfoPrint(s string, a ...interface{}) {
	logger.Info(fmt.Sprintf(s, a...))
}

func ErrorPrint(s string, a ...interface{}) {
	logger.Error(fmt.Sprintf(s, a...))
}

func (m *MakcuHandle) LeftDown() error {
	if m == nil {
		return fmt.Errorf("LeftDown: MakcuHandle is nil (no device connected)")
	}

	_, err := m.Write([]byte("km.left(1)\r"))
	if err != nil {
		DebugPrint("Failed to press mouse: Write Error: %v", err)
		return err
	}

	return nil
}

func (m *MakcuHandle) LeftUp() error {
	if m == nil {
		return fmt.Errorf("LeftUp: MakcuHandle is nil (no device connected)")
	}

	_, err := m.Write([]byte("km.left(0)\r"))
	if err != nil {
		DebugPrint("Failed to release mouse: Write Error: %v", err)
		return err
	}

	return nil
}

func (m *MakcuHandle) LeftClick() error {
	if m == nil {
		return fmt.Errorf("LeftClick: MakcuHandle is nil (no device connected)")
	}

	_, err := m.Write([]byte("km.left(1)\r km.left(0)\r"))
	if err != nil {
		DebugPrint("Failed to click mouse: %v", err)
		return err
	}

	return nil
}

func (m *MakcuHandle) RightDown() error {
	if m == nil {
		return fmt.Errorf("RightDown: MakcuHandle is nil (no device connected)")
	}

	_, err := m.Write([]byte("km.right(1)\r"))
	if err != nil {
		DebugPrint("Failed to press mouse: Write Error: %v", err)
		return err
	}

	return nil
}

func (m *MakcuHandle) RightUp() error {
	if m == nil {
		return fmt.Errorf("RightUp: MakcuHandle is nil (no device connected)")
	}

	_, err := m.Write([]byte("km.right(0)\r"))
	if err != nil {
		DebugPrint("Failed to release mouse: Write Error: %v", err)
		return err
	}

	return nil
}

func (m *MakcuHandle) RightClick() error {
	if m == nil {
		return fmt.Errorf("RightClick: MakcuHandle is nil (no device connected)")
	}

	_, err := m.Write([]byte("km.right(1)\r km.right(0)\r"))
	if err != nil {
		DebugPrint("Failed to right click mouse: %v", err)
		return err
	}

	return nil
}

func (m *MakcuHandle) MiddleDown() error {
	if m == nil {
		return fmt.Errorf("MiddleDown: MakcuHandle is nil (no device connected)")
	}

	_, err := m.Write([]byte("km.middle(1)\r"))
	if err != nil {
		DebugPrint("Failed to press middle mouse button: Write Error: %v", err)
		return err
	}

	return nil
}

func (m *MakcuHandle) MiddleUp() error {
	if m == nil {
		return fmt.Errorf("MiddleUp: MakcuHandle is nil (no device connected)")
	}

	_, err := m.Write([]byte("km.middle(0)\r"))
	if err != nil {
		DebugPrint("Failed to release middle mouse button: Write Error: %v", err)
		return err
	}

	return nil
}

func (m *MakcuHandle) MiddleClick() error {
	if m == nil {
		return fmt.Errorf("MiddleClick: MakcuHandle is nil (no device connected)")
	}

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
	MOUSE_BUTTON_SIDE1  = 4
	MOUSE_BUTTON_SIDE2  = 5
	MOUSE_X             = 6
	MOUSE_Y             = 7
)

func (m *MakcuHandle) Click(i int, delay time.Duration) error {
	if m == nil {
		return fmt.Errorf("Click: MakcuHandle is nil (no device connected)")
	}

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
		return fmt.Errorf("invalid mouse button: %d", i)
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

func (m *MakcuHandle) ClickMouse() error {
	if m == nil {
		return fmt.Errorf("ClickMouse: MakcuHandle is nil (no device connected)")
	}

	return m.Click(MOUSE_BUTTON_LEFT, 0)
}

func (m *MakcuHandle) LockMouse(Button int, lock int) error {
	if m == nil {
		return fmt.Errorf("MouseLock: MakcuHandle is nil (no device connected)")
	}

	if lock != 1 && lock != 0 {
		return fmt.Errorf("MouseLock: lock must be 1(lock) or 0(unlock)")
	}

	switch Button {
	case MOUSE_BUTTON_LEFT:
		_, err := m.Write([]byte("km.lock_ml(" + strconv.Itoa(lock) + ")\r"))
		if err != nil {
			DebugPrint("Failed to lock mouse: Write Error: %v", err)
			return err
		}
	case MOUSE_BUTTON_RIGHT:
		_, err := m.Write([]byte("km.lock_mr(" + strconv.Itoa(lock) + ")\r"))
		if err != nil {
			DebugPrint("Failed to lock mouse: Write Error: %v", err)
			return err
		}
	case MOUSE_BUTTON_MIDDLE:
		_, err := m.Write([]byte("km.lock_mm(" + strconv.Itoa(lock) + ")\r"))
		if err != nil {
			DebugPrint("Failed to lock mouse: Write Error: %v", err)
			return err
		}
	case MOUSE_BUTTON_SIDE1:
		_, err := m.Write([]byte("km.lock_ms1(" + strconv.Itoa(lock) + ")\r"))
		if err != nil {
			DebugPrint("Failed to lock mouse: Write Error: %v", err)
			return err
		}
	case MOUSE_BUTTON_SIDE2:
		_, err := m.Write([]byte("km.lock_ms2(" + strconv.Itoa(lock) + ")\r"))
		if err != nil {
			DebugPrint("Failed to lock mouse: Write Error: %v", err)
			return err
		}
	case MOUSE_X:
		_, err := m.Write([]byte("km.lock_mx(" + strconv.Itoa(lock) + ")\r"))
		if err != nil {
			DebugPrint("Failed to lock mouse: Write Error: %v", err)
			return err
		}
	case MOUSE_Y:
		_, err := m.Write([]byte("km.lock_my(" + strconv.Itoa(lock) + ")\r"))
		if err != nil {
			DebugPrint("Failed to lock mouse: Write Error: %v", err)
			return err
		}
	default:
		return fmt.Errorf("invalid mouse button: %d", Button)

	}

	return nil
}

func (m *MakcuHandle) GetButtonStatus() (int, error) {
	if m == nil {
		return -1, fmt.Errorf("MAKCU not connected")
	}

	_, err := m.Write([]byte("km.buttons()\r"))
	if err != nil {
		return -1, err
	}

	buf := make([]byte, 128)
	n, err := m.Read(buf)
	if err != nil {
		return -1, err
	}

	response := string(buf[:n])
	if strings.Contains(response, "1") {
		return 1, nil
	} else if strings.Contains(response, "0") {
		return 0, nil
	}

	return -1, fmt.Errorf("could not parse buttons status")
}

func (m *MakcuHandle) SetButtonStatus(enable bool) error {
	if m == nil {
		return fmt.Errorf("MAKCU not connected")
	}

	var cmd string
	if enable {
		cmd = "km.buttons(1)\r"
	} else {
		cmd = "km.buttons(0)\r"
	}

	_, err := m.Write([]byte(cmd))
	if err != nil {
		return fmt.Errorf("failed to set button status: %v", err)
	}

	DebugPrint("Button echo %s\n", map[bool]string{true: "enabled", false: "disabled"}[enable])
	return nil
}

func (m *MakcuHandle) ScrollMouse(amount int) error {
	if m == nil {
		return fmt.Errorf("ScrollMouse: MakcuHandle is nil (no device connected)")
	}

	_, err := m.Write([]byte(fmt.Sprintf("km.wheel(%d)\r", amount)))
	if err != nil {
		DebugPrint("Failed to scroll mouse: %v", err)
		return err
	}

	return nil
}

func (m *MakcuHandle) MoveMouse(x, y int) error {
	if m == nil {
		return fmt.Errorf("MoveMouse: MakcuHandle is nil (no device connected)")
	}

	_, err := m.Write([]byte(fmt.Sprintf("km.move(%d, %d)\r", x, y)))
	if err != nil {
		DebugPrint("Failed to move mouse: Write Error: %v", err)
		return err
	}

	return nil
}

// use a curve with the built in curve functionality from MAKCU... i THINK this is only on fw v3+ ??? idk don't care to fact check it rn either :)
// "It is common sense that the higher the number of the third parameter, the smoother the curve will be fitted" - from MAKCU/km box docs
func (m *MakcuHandle) MoveMouseWithCurve(x, y int, params ...int) error {
	if m == nil {
		return fmt.Errorf("MoveMouseWithCurve: MakcuHandle is nil (no device connected)")
	}

	var cmd string
	switch len(params) {
	case 0:
		cmd = fmt.Sprintf("km.move(%d, %d)\r", x, y)
	case 1:
		cmd = fmt.Sprintf("km.move(%d, %d, %d)\r", x, y, params[0])
	case 3:
		cmd = fmt.Sprintf("km.move(%d, %d, %d, %d, %d)\r", x, y, params[0], params[1], params[2])
	default:
		DebugPrint("Invalid number of parameters")
		return fmt.Errorf("invalid number of parameters")
	}

	_, err := m.Write([]byte(cmd))
	if err != nil {
		DebugPrint("Failed to move mouse with curve: Write Error: %v", err)
		return err
	}

	return nil
}