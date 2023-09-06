package util

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	if err == nil {
		return true // 文件存在
	}
	if os.IsNotExist(err) {
		return false // 文件不存在
	}
	return false // 其他错误，检查文件对于当前用户是否有读取权限
}

func IsValidIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	return ip != nil
}

func PrintErr(e error) {
	if e != nil {
		log.Fatalln(e)
	}
}

func Strip(data string) string {
	//剔除字符串中的所有空白字符
	rep, _ := regexp.Compile(`\s`)
	result := rep.ReplaceAllString(data, "")
	return result
}

func Rstrip(data string, new string) string {
	//替换字符串中的空白字符为其他字符
	rep, _ := regexp.Compile(`\s`)
	result := rep.ReplaceAllString(data, new)
	return result
}

func Atoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return -1
	}
	return i
}

func CreateOutFile(filePath string, suffix string) error {
	// Check if the file already exists
	if _, err := os.Stat(filePath); err == nil {
		return nil
	}

	// Check if the file extension is "csv"
	if !strings.HasSuffix(filePath, suffix) {
		return fmt.Errorf("输出文件后缀应为%s", suffix)
	}

	// Create the file
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write the header to the file
	header := "server,command,date\n"
	_, err = file.WriteString(header)
	if err != nil {
		return err
	}

	return nil
}

func AppendFile(filePath, content string) error {
	// 打开文件以追加模式打开，如果文件不存在则创建
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// 将内容写入文件
	_, err = io.WriteString(file, content)
	if err != nil {
		return err
	}

	return nil
}

func ParamToA(p string) int {
	i, err := strconv.Atoi(p)
	if err != nil {
		return -1
	}
	return i
}

func Escape(s string) string {
	//转义字符串中的字符
	s = strings.ReplaceAll(s, "\n", "+")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\"", "'")
	return s
}

func SToParMap(input string) map[string]string {
	dictionary := make(map[string]string)

	pairs := strings.Split(input, " ")
	for _, pair := range pairs {
		keyValue := strings.Split(pair, "=")
		if len(keyValue) == 2 {
			key := strings.ToLower(keyValue[0])
			value := keyValue[1]
			dictionary[key] = value
		}
	}

	return dictionary
}
