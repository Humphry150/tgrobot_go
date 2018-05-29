package bot

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

var BeiJing = time.FixedZone("BeiJing", 8*3600)

func todayStart(now time.Time) time.Time {
	now = now.In(BeiJing)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, BeiJing)
}

func todayPast(now time.Time) time.Duration {
	return now.Sub(todayStart(now))
}

func writeFile(path, name string, data interface{}) {
	file, err := os.OpenFile(path+name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		Errorf("[%s] open file failed. error: %v", name, err)
		return
	}

	defer file.Close()
	e := json.NewEncoder(file)
	e.SetIndent("", "	")
	err = e.Encode(data)
	if err != nil {
		Errorf("[%s] write amount failed. error: %v", name, err)
	}
}

func readFile(path, name string, dst interface{}) {
	file, err := os.OpenFile(path+name, os.O_RDONLY|os.O_CREATE, os.ModePerm)
	if err != nil {
		Errorf("read file failed. error: %v", err)
		return
	}
	defer file.Close()
	d := json.NewDecoder(file)
	for {
		err = d.Decode(dst)
		if err == io.EOF {
			break
		}
		if err != nil {
			Errorf("[%s] decode failed. error: %v", name, err)
			return
		}
	}
	return
}

func readDirLastFile(path string, dst interface{}) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		Errorf("read path failed. error: %v", err)
		return
	}

	l := len(files)
	if l == 0 {
		Warnf("no file found")
		return
	}

	file, err := os.OpenFile(path+files[l-1].Name(), os.O_RDONLY|os.O_CREATE, os.ModePerm)
	if err != nil {
		Errorf("read file failed. error: %v", err)
		return
	}
	defer file.Close()
	d := json.NewDecoder(file)
	for {
		err = d.Decode(dst)
		if err == io.EOF {
			break
		}
		if err != nil {
			Errorf("[%s] decode failed. error: %v", file.Name(), err)
			return
		}
	}

	return
}

func removeFile(path, name string) {
	os.Remove(path + name)
}
func removeFiles(path string) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		Warnf("read dir failed. error: %v", err)
		return
	}
	for _, file := range files {
		removeFile(path, file.Name())
	}
}

func equalString(a, b string) bool {
	return strings.ToLower(a) == strings.ToLower(b)
}

func toString(v interface{}) string {
	switch a := v.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", a)
	case float32, float64:
		return fmt.Sprintf("%f", a)
	default:
		return fmt.Sprintf("%#v", a)
	}
}
