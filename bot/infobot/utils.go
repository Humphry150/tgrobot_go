package bot

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

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
