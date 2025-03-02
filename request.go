package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

const MERCHANT_URL string = "http://127.0.0.1:5000/merchants"

type HealthCheck struct {
	Status string `json:list`
}

func main() {
	// Make an HTTP GET request to the merchant URL
	res, err := http.Get(MERCHANT_URL)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close() // Ensure the response body is closed

	// Read the response body
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	// Parse the JSON response (which is an array of strings)
	var merchants []string
	if err := json.Unmarshal(body, &merchants); err != nil {
		log.Fatal(err)
	}

	// Print the list of merchants
	fmt.Println("Merchants:", merchants)

	if res.StatusCode == 200 {
		fmt.Println("SUCCESS")
	} else if res.StatusCode > 299 {
		log.Fatalf("Response failed with status code: %d and\nbody: %s\n", res.StatusCode, body)
	}
	if err != nil {
		log.Fatal(err)
	}

	var compactJSON bytes.Buffer
	if err := json.Compact(&compactJSON, body); err != nil {
		log.Fatal(err)
	}

	f, err := os.OpenFile("workfile.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := f.Write([]byte(compactJSON.String() + "\n")); err != nil {
		f.Close() // ignore error; Write error takes precedence
		log.Fatal(err)
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", compactJSON.String())
}
