package utils

import (
	"os"
	"fmt"
	"errors"
)

func CheckoutDir(dir string) error {
	if _, err := os.Stat(dir); err != nil {
		fmt.Println("Dir not exists, try to create...", dir)
		err := os.MkdirAll(dir, 0711)
		if err != nil {
			fmt.Println("Error creating directory", dir)
			fmt.Println("err:", err)
			return errors.New("ERROR CREATING DIRECTORY")
		}
	}
	return nil
}

func CheckoutIfFileExists(filePath string) bool {
	if _, err := os.Stat(filePath); err != nil {
		return false
	}
	return true
}