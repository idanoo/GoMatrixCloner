package main

import (
	f "gomatrixcloner/internal/gomatrixcloner"
	"os"
)

func main() {
	// Set Matrix Credentials
	f.MatrixHost = os.Getenv("MATRIX_HOST")
	f.MatrixUsername = os.Getenv("MATRIX_USERNAME")
	f.MatrixPassword = os.Getenv("MATRIX_PASSWORD")

	// Start application
	f.Run()
}
