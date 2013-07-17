package main

// This is a quick attempt to send pixels via SPI to LED strips
// using the LPD8806 chipset, like the ones available from Adafruit.

// This has not been tested yet.

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Set one of the on-board LEDs on the Beaglebone.
//    ledNum: between 0 and 3 inclusive
//    val: 0 or 1.
func setOnboardLED(ledNum int, val int) {
	ledFn := fmt.Sprintf("/sys/class/leds/beaglebone:green:usr%d/brightness", ledNum)
	fmt.Println(ledFn)

	// open output file
	ledFile, err := os.Create(ledFn)
	if err != nil {
		panic(err)
	}
	// close ledFile on exit and check for its returned error
	defer func() {
		if err := ledFile.Close(); err != nil {
			panic(err)
		}
	}()

	if _, err := ledFile.WriteString(strconv.Itoa(val)); err != nil {
		panic(err)
	}
}

// Write the byte slice to the open file descriptor
func sendBytes(fd *os.File, bytes []byte) {
	fmt.Println("[sendBytes]", bytes)
	if _, err := fd.Write(bytes); err != nil {
		panic(err)
	}
}

// Recieve byte slices over the pixelsToSend channel.
// When we get one, write it to the SPI file descriptor and toggle one
//  of the Beaglebone's onboard LEDs.
// After sending the frame, send 1 over the sendingIsDone channel.
// The byte slice should hold values from 0 to 255 in [r g b  r g b  r g b  ... ] order.
func spiThread(pixelsToSend chan []byte, sendingIsDone chan int) {

	spiFn := "/dev/spidev1.0"

	// open output file and keep the file descriptor around
	spiFile, err := os.Create(spiFn)
	if err != nil {
		panic(err)
	}
	// close spiFile on exit and check for its returned error
	defer func() {
		if err := spiFile.Close(); err != nil {
			panic(err)
		}
	}()

	flipper := 0
	// as we get byte slices over the channel...
	for pixels := range pixelsToSend {
		fmt.Println("[send] starting to send", len(pixels), "values")

		// toggle onboard LED
		setOnboardLED(0, flipper)
		flipper = 1 - flipper

		// build a new slice of bytes in the format the LED strand wants
		bytes := make([]byte, 0)

		// leading zeros to begin a new frame of pixels
		numZeroes := (len(pixels) / 32) + 2
		for ii := 0; ii < numZeroes*5; ii++ {
			bytes = append(bytes, 0)
		}

		// pixels
		for _, v := range pixels {
			// high bit must be always on, remaining seven bits are data
			v2 := 128 | (v >> 1)
			bytes = append(bytes, v2)
		}

		// final zero to latch the last pixel
		bytes = append(bytes, 0)
		sendBytes(spiFile, bytes)

		sendingIsDone <- 1
	}
}

func main() {
	fmt.Println("--------------------------------------------------------------------------------\\")

	pixelsToSend := make(chan []byte, 0)
	sendingIsDone := make(chan int, 0)

	go spiThread(pixelsToSend, sendingIsDone)

	// send some test data
	for ii := 0; true; ii = (ii + 1) % 256 {
		pixels := []byte{255, 0, 0, 0, 255, 0, 0, 0, 255, 0, 0, 0}
		pixels[9] = byte((ii * 9) % 256)
		pixels[10] = byte((ii * 9) % 256)
		pixels[11] = byte((ii * 9) % 256)
		fmt.Println("[main] pixels =", pixels)
		pixelsToSend <- pixels
		<-sendingIsDone
		time.Sleep(300 * time.Millisecond)
	}

	fmt.Println("--------------------------------------------------------------------------------/")
}
