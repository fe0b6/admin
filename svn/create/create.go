package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
)

const (
	userName     = "fe0b6"
	svnAdminPath = "/usr/bin/svnadmin"
	resopDir     = "/www/svn/repos/"
	hookPath     = "/www/svn/core/hook"
	basePath     = "/www/sites/"
	svnUID       = 1000
	svnGID       = 1000
	wwwGid       = 33
)

var (
	name     string
	repoPath string
)

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

func main() {
	name = os.Args[1]
	if name == "" {
		log.Fatalln("miss repo name")
	}

	repoPath = resopDir + name

	cmd := exec.Command(svnAdminPath, "create", repoPath)
	b, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalln("[error]", string(b), err)
	}

	b, err = ioutil.ReadFile(repoPath + "/conf/svnserve.conf")
	if err != nil {
		log.Fatalln("[error]", err)
	}

	str := string(b)
	str = regexp.MustCompile("# anon-access[^\n]+\n").ReplaceAllString(str, "anon-access=none\n")
	str = regexp.MustCompile("# auth-access[^\n]+\n").ReplaceAllString(str, "auth-access=write\n")
	str = regexp.MustCompile("# realm[^\n]+\n").ReplaceAllString(str, "realm="+name+"\n")
	str = regexp.MustCompile("# use-sasl[^\n]+\n").ReplaceAllString(str, "use-sasl=true\n")
	str = regexp.MustCompile("# min-encryption[^\n]+\n").ReplaceAllString(str, "min-encryption=128\n")
	str = regexp.MustCompile("# max-encryption[^\n]+\n").ReplaceAllString(str, "max-encryption=512\n")

	err = ioutil.WriteFile(repoPath+"/conf/svnserve.conf", []byte(str), 0640)
	if err != nil {
		log.Fatalln("[error]", err)
	}

	err = os.Link(hookPath, repoPath+"/hooks/pre-commit")
	if err != nil {
		log.Fatalln("[error]", err)
	}

	err = os.Mkdir(basePath+name, 0750)
	if err != nil {
		log.Fatalln("[error]", err)
	}

	err = os.Chown(basePath+name, svnUID, wwwGid)
	if err != nil {
		log.Fatalln("[error]", err)
	}

	err = filepath.Walk(repoPath+"/db/", func(path string, info os.FileInfo, err error) error {
		err = os.Chown(path, svnUID, svnGID)
		if err != nil {
			log.Fatalln("[error]", err)
		}
		return err
	})
	if err != nil {
		log.Fatalln("[error]", err)
	}

	err = filepath.Walk(repoPath+"/locks/", func(path string, info os.FileInfo, err error) error {
		err = os.Chown(path, svnUID, svnGID)
		if err != nil {
			log.Fatalln("[error]", err)
		}
		return err
	})
	if err != nil {
		log.Fatalln("[error]", err)
	}

	log.Println("repo created")
	log.Printf("saslpasswd2 -c -f /www/svn/sasldb -u %s %s\n", name, userName)
}
