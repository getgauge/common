package common

import (
	"fmt"
	"github.com/daviddengcn/go-colortext"
)

func PrintSuccess(text ...string) {
	ct.ChangeColor(ct.Green, false, ct.None, false)
	for _, value := range text {
		fmt.Println(value)
	}
	ct.ResetColor()
}

func PrintFailure(text ...string) {
	ct.ChangeColor(ct.Red, false, ct.None, false)
	for _, value := range text {
		fmt.Println(value)
	}
	ct.ResetColor()
}
