package main

import (
	"fmt"
	"math"
	"time"

	makcu "github.com/nullpkt/Makcu-Go"
)

func main() {
	// Find the MAKCU COM port
	comPort, err := makcu.Find()
	if err != nil {
		fmt.Printf("Error finding MAKCU: %v\n", err)
		return
	}

	// Connect at default baud rate
	makcuConn, err := makcu.Connect(comPort, 115200)
	if err != nil {
		fmt.Printf("Error connecting: %v\n", err)
		return
	}
	defer makcuConn.Close()

	time.Sleep(1 * time.Second)

	// Switch to high baud rate for precision
	makcuConn, err = makcu.ChangeBaudRate(makcuConn)
	if err != nil {
		fmt.Printf("Error changing baud rate: %v\n", err)
		return
	}

	time.Sleep(2 * time.Second)
	fmt.Printf("\033[2J\033[HMoving mouse in a spiral with smooth curves...\n")
	time.Sleep(1 * time.Second)

	// Parameters for spiral
	centerX, centerY := 0, 0
	maxRadius := 100
	turns := 3
	stepsPerTurn := 100
	smoothness := 20 // Higher = smoother

	// Move in a spiral using MoveMouseWithCurve
	for step := 0; step < turns*stepsPerTurn; step++ {
		angle := 2 * math.Pi * float64(step) / float64(stepsPerTurn)
		radius := maxRadius * step / (turns * stepsPerTurn)
		x := centerX + int(float64(radius)*math.Cos(angle))
		y := centerY + int(float64(radius)*math.Sin(angle))
		err := makcuConn.MoveMouseWithCurve(x, y, smoothness)
		if err != nil {
			fmt.Printf("Error moving mouse: %v\n", err)
			break
		}
		time.Sleep(8 * time.Millisecond)
	}

	time.Sleep(1 * time.Second)
	fmt.Printf("\033[2J\033[HScrolling mouse precisely...\n")

	// Scroll up and down with precise timing
	for i := 1; i <= 10; i++ {
		if err := makcuConn.ScrollMouse(i); err != nil {
			fmt.Printf("Error scrolling: %v\n", err)
		}
		time.Sleep(30 * time.Millisecond)
	}
	for i := 10; i >= 1; i-- {
		if err := makcuConn.ScrollMouse(-i); err != nil {
			fmt.Printf("Error scrolling: %v\n", err)
		}
		time.Sleep(30 * time.Millisecond)
	}

	fmt.Println("Done. Closing connection in 10 seconds...")
	time.Sleep(10 * time.Second)
}
