package main

import (
	"fmt"

	"github.com/awnumar/memguard"
)

func main() {
	// Tell memguard to listen out for interrupts, and cleanup in case of one.
	memguard.CatchInterrupt(func() {
		fmt.Println("Interrupt signal received. Exiting...")
	})
	// Make sure to destroy all LockedBuffers when returning.
	defer memguard.DestroyAll()

	// TODO: continue code from here
}
