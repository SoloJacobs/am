package main

import (
	"fmt"

	"github.com/SoloJacobs/am/orchestrate"
)

func main() {
	_, err := orchestrate.NewBuild()
	println(fmt.Sprintf("%v", err))
}
