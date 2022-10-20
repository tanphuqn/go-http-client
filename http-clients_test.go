package main

import (
	"bufio"
	"fmt"
	"net/http"
	"testing"
)

func CallNbconvert() {
	resp, err := http.Get("http://localhost:8080/nbconvert")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Println("Response status:", resp.Status)

	scanner := bufio.NewScanner(resp.Body)
	for i := 0; scanner.Scan() && i < 5; i++ {
		fmt.Println(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}
}

func TestCallNbconvert(t *testing.T) {
	for i := 0; i < 100; i++ {
		CallNbconvert()
	}
}
