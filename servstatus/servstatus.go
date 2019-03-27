package main

import (
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/fe0b6/tools"
)

type smartParam struct {
	Name   string
	Value  int64
	Worst  int64
	Thresh int64
	Type   string
}

var (
	spaceReg *regexp.Regexp

	maxSize   int64
	checkDisk []string
)

func init() {
	spaceReg = regexp.MustCompile("\\s+")

	maxSize = 80
	checkDisk = []string{"/"}
}

func main() {
	diskSpace()
}

// Проверяем свободное место
func diskSpace() {
	cmd := exec.Command("df", "-h")
	b, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("[error]", err)
		return
	}

	for i, str := range strings.Split(string(b), "\n") {
		if i == 0 {
			continue
		}
		str = spaceReg.ReplaceAllString(str, " ")

		data := strings.Split(str, " ")
		if len(data) < 6 {
			continue
		}

		size, err := strconv.ParseInt(strings.TrimRight(data[4], "%"), 10, 64)
		if err != nil {
			log.Println("[error]", err, data[4])
			return
		}

		if tools.InArray(checkDisk, data[5]) && size >= maxSize {
			log.Printf("disk busy: %s %d\n", data[5], size)
		}
	}
}
