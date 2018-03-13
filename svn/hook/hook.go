package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/css"
	"github.com/tdewolff/minify/js"
)

type config struct {
	SVNLook string `json:"svnlook"`
	Path    string `json:"path"`
	Debug   string `json:"debug"`
	UserID  int    `json:"user_id"`
	GroupID int    `json:"group_id"`
	Repos   []repo `json:"repos"`
	Yate    string `json:"yate"`
	Go      string `json:"go"`
}

type repo struct {
	Name         string   `json:"name"`
	Yate         []string `json:"yate"`
	YatePath     string   `json:"yate_path"`
	YateCompress bool     `json:"yate_compress"`
	Go           []string `json:"go"`
	GoPath       string   `json:"go_path"`
	JSCompress   bool     `json:"js_compress"`
	CSSCompress  bool     `json:"css_compress"`
}

const confPath = "/www/svn/core/conf.json"

var (
	conf    config
	args    []string
	nameReg *regexp.Regexp
	lookReg *regexp.Regexp
	pathReg *regexp.Regexp
	yateReg *regexp.Regexp
	goReg   *regexp.Regexp
	jsReg   *regexp.Regexp
	cssReg  *regexp.Regexp

	movedFiles []string

	usedRepo repo
)

func init() {
	// Получаем аргументы
	args = os.Args
	if len(args) < 3 {
		log.Fatalln("no args", args)
	}

	// Читаем конфиг
	b, err := ioutil.ReadFile(confPath)
	if err != nil {
		log.Fatalln("[fatal]", err)
	}

	err = json.Unmarshal(b, &conf)
	if err != nil {
		log.Fatalln("[fatal]", err)
	}

	if conf.Debug != "" {
		// Добавляем дату, время и строку в какой идет запись в лог
		log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
		// Перенаправляем вывод в файл
		f, err := os.OpenFile(conf.Debug, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0640)
		if err != nil {
			log.Fatalf("[fatal] error opening file: %v\n", err)
			os.Exit(2)
		}
		log.SetOutput(f)
	}

	nameReg = regexp.MustCompile("/([^/]+)$")
	lookReg = regexp.MustCompile("^([ADU]{1})\\s+(\\S+.+)$")
	pathReg = regexp.MustCompile("/$")
	yateReg = regexp.MustCompile(".yate$")
	goReg = regexp.MustCompile(".go$")
	jsReg = regexp.MustCompile(".js$")
	cssReg = regexp.MustCompile(".css$")

	movedFiles = []string{}
}

func main() {
	name := nameReg.FindStringSubmatch(args[1])

	for _, r := range conf.Repos {
		if r.Name == name[1] {
			usedRepo = r
			break
		}
	}

	if usedRepo.Name == "" {
		createRepo(name[1])
	}

	changes()
}

func createRepo(name string) {
	usedRepo.Name = name

	path := usedRepo.getPath("")
	_, err := os.Stat(path)
	if err != nil {
		log.Println("[error]", err)
		exit(2)
	}

	// Пробуем прочитать конфиг репы
	// Читаем конфиг
	b, err := ioutil.ReadFile(path + "svn.json")
	if err != nil {
		return
	}

	log.Println("config read")

	err = json.Unmarshal(b, &usedRepo)
	if err != nil {
		log.Fatalln("[fatal]", err)
	}

}

// Проверяем изменения
func changes() {

	// args[2] - номер транзакции, args[1] - репозитарий
	cmd := exec.Command(conf.SVNLook, "changed", "-t", args[2], args[1])
	b, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalln("[error]", err)
	}

	var isYate, isGo bool
	for _, s := range strings.Split(string(b), "\n") {
		if s == "" {
			continue
		}

		// l[1] - действие, l[2] - файл или папка
		l := lookReg.FindStringSubmatch(s)
		if len(l) == 0 {
			log.Println("bad string", s)
			exit(2)
		}

		var isPath bool
		if pathReg.MatchString(l[2]) {
			isPath = true
		} else if !isYate && yateReg.MatchString(l[2]) {
			isYate = true
		} else if !isGo && goReg.MatchString(l[2]) {
			isGo = true
		}

		switch l[1] {
		case "A", "U":
			add(l[2], isPath)
		case "D":
			del(l[2])
		default:
			log.Println("unknown action", l[1])
			exit(2)
		}
	}

	// Если надо пересобрать шаблон
	if isYate && usedRepo.YatePath != "" {
		fpath := usedRepo.getPath(usedRepo.YatePath)
		cpath := fpath + "compile/"
		_, err := os.Stat(cpath)
		if err != nil {
			log.Println("[error]", err)
			exit(2)
		}

		for _, yap := range usedRepo.Yate {
			jstmpl := cpath + yap + ".js"

			cmd := exec.Command(conf.Yate, "--output", jstmpl, fpath+yap+".yate")
			b, err := cmd.CombinedOutput()
			if err != nil {
				log.Println(string(b))
				log.Println("[error]", err)
				exit(2)
			}

			// Если надо сжать
			if usedRepo.YateCompress {
				compressJS(jstmpl, "")
			}
		}
	}

	// Если были go файлы
	if isGo {
		fpath := usedRepo.getPath(usedRepo.GoPath)
		for _, gp := range usedRepo.Go {
			cmd := exec.Command(conf.Go, "build", "-i", "-o", fpath+gp, fpath+gp+".go")
			b, err := cmd.CombinedOutput()
			if err != nil {
				log.Println(string(b))
				log.Println("[error]", err)
				exit(2)
			}

			err = os.Chmod(fpath+gp, 0750)
			if err != nil {
				log.Println("[error]", err)
				exit(2)
			}
		}
	}

	cleanTmp()

}

func add(path string, isPath bool) {

	fullpath := usedRepo.getPath(path)

	// Если это папка
	if isPath {
		_, err := os.Stat(fullpath)
		if err != nil {
			if strings.Contains(err.Error(), "no such file or directory") {
				os.Mkdir(fullpath, 0750)
				chown(fullpath)
			} else {
				log.Println("[error]", err)
				exit(2)
			}
		}
		return
	}

	// Читаем файл, args[2] - номер транзакции, args[1] - репозитарий
	cmd := exec.Command(conf.SVNLook, "cat", "-t", args[2], args[1], path)
	b, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("[error]", err)
		exit(2)
	}

	err = os.Rename(fullpath, fullpath+".svntmp")
	if err != nil && !strings.Contains(err.Error(), "no such file or directory") {
		log.Println("[error]", err)
		exit(2)
	}
	if err == nil {
		movedFiles = append(movedFiles, fullpath)
	}

	err = ioutil.WriteFile(fullpath, b, 0640)
	if err != nil {
		log.Println("[error]", err)
		exit(2)
	}

	chown(fullpath)

	modify(fullpath, string(b))
}

func modify(fullpath, data string) {
	if usedRepo.JSCompress && jsReg.MatchString(fullpath) && !strings.Contains(data, "/* NO COMPRESS */") {
		compressJS(fullpath, data)
	} else if usedRepo.CSSCompress && cssReg.MatchString(fullpath) && !strings.Contains(data, "/* NO COMPRESS */") {
		compressCSS(fullpath, data)
	}
}

func del(path string) {
	fullpath := usedRepo.getPath(path)

	log.Println(fullpath)

	err := os.RemoveAll(fullpath)
	if err != nil {
		log.Println("[error]", err)
		exit(2)
	}
}

func compressJS(fullpath, data string) {
	m := minify.New()
	m.AddFunc("text/javascript", js.Minify)

	if data == "" {
		b, err := ioutil.ReadFile(fullpath)
		if err != nil {
			log.Println("[error]", err)
			exit(2)
		}

		data = string(b)
	}

	data, err := m.String("text/javascript", data)
	if err != nil {
		log.Println("[error]", err)
		exit(2)
	}

	err = ioutil.WriteFile(fullpath, []byte(data), 0640)
	if err != nil {
		log.Println("[error]", err)
		exit(2)
	}

	chown(fullpath)
}

func compressCSS(fullpath string, data string) {
	m := minify.New()
	m.AddFunc("text/css", css.Minify)

	if data == "" {
		b, err := ioutil.ReadFile(fullpath)
		if err != nil {
			log.Println("[error]", err)
			exit(2)
		}

		data = string(b)
	}

	data, err := m.String("text/css", data)
	if err != nil {
		log.Println("[error]", err)
		exit(2)
	}

	err = ioutil.WriteFile(fullpath, []byte(data), 0640)
	if err != nil {
		log.Println("[error]", err)
		exit(2)
	}

	chown(fullpath)
}

func chown(path string) {
	err := os.Chown(path, conf.UserID, conf.GroupID)
	if err != nil {
		log.Println("[error]", err)
		exit(2)
	}
}

func exit(code int) {
	if len(movedFiles) > 0 {
		retoreFile()
	}

	os.Exit(code)
}

func retoreFile() {
	for _, fullpath := range movedFiles {
		err := os.Rename(fullpath+".svntmp", fullpath)
		if err != nil {
			log.Println("[error]", err)
		}
	}
}

func cleanTmp() {
	for _, fullpath := range movedFiles {
		err := os.Remove(fullpath + ".svntmp")
		if err != nil {
			log.Println("[error]", err)
		}
	}
}

func (r *repo) getPath(path string) string {
	return fmt.Sprintf("%s%s/%s", conf.Path, r.Name, path)
}
