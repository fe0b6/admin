package main

import (
	"io/ioutil"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/fe0b6/tools"
)

var (
	spaceReg *regexp.Regexp
	raidReg  *regexp.Regexp

	maxSize   int64
	checkDisk []string
)

func init() {
	spaceReg = regexp.MustCompile("\\s+")
	raidReg = regexp.MustCompile("\\[(\\d+)/(\\d+)\\]")

	maxSize = 80
	checkDisk = []string{"/"}
}

func main() {

	diskSpace()

	raidTest()
}

func raidTest() {
	b, err := ioutil.ReadFile("/proc/mdstat")
	if err != nil {
		log.Println("[error]", err)
		return
	}

	for _, str := range strings.Split(string(b), "\n") {
		str = spaceReg.ReplaceAllString(str, " ")
		if strings.Contains(str, "active") && strings.Contains(str, "(F)") {
			log.Println("raid faild")
		} else if strings.Contains(str, "blocks super") {
			arr := raidReg.FindStringSubmatch(str)
			if len(arr) > 0 && arr[1] != arr[2] {
				log.Println("raid faild", arr[1], arr[2])
			}
		}
	}
}

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
