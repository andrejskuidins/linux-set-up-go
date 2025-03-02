package main

import "fmt"

func main() {
	// Scenario: Checking if the submarine is navigating within a safe depth range
	const criticalSafeDepth = 300 // critical depth limit in meters
	var submarineDepth = 350      // current submarine depth in meters

	// TODO: Create a new boolean variable "isTooDeepForSafety" and update the print statement accordingly
	isTooDeepForSafety := submarineDepth < criticalSafeDepth
	// If submarineDepth is 350, it will print: Is the submarine within the safe depth limit? false
	fmt.Println("Is the submarine within the safe depth limit? ", isTooDeepForSafety)
}
