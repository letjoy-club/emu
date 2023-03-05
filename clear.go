package main

import (
	"fmt"
	"os"
)

func clearHistory() error {
	files, err := os.ReadDir("binary")
	if err != nil {
		return err
	}
	for _, file := range files {
		fmt.Println(file.Name())
	}
	return nil
}
