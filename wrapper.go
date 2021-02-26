package sugar

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func Print(a ...interface{}) (int, error) {
	return fmt.Print(a...)
}
func Println(a ...interface{}) (int, error) {
	return fmt.Println(a...)
}
func Printf(format string, a ...interface{}) (int, error) {
	return fmt.Printf(format, a...)
}
func Sprintf(format string, a ...interface{}) string {
	return fmt.Sprintf(format, a...)
}

func SplitStr(s string, sep string) []string {
	return strings.Split(s, sep)
}

func StrSplit(s string, sep string) []string {
	return strings.Split(s, sep)
}

func SplitStrN(s string, sep string, n int) []string {
	return strings.SplitN(s, sep, n)
}

func StrSplitN(s string, sep string, n int) []string {
	return strings.SplitN(s, sep, n)
}

func StrFind(s string, f string) int {
	return strings.Index(s, f)
}

func FindStr(s string, f string) int {
	return strings.Index(s, f)
}

func ReplaceStr(s, old, new string) string {
	return strings.Replace(s, old, new, -1)
}

func StrReplace(s, old, new string) string {
	return strings.Replace(s, old, new, -1)
}

func TrimStr(s string) string {
	return strings.TrimSpace(s)
}

func StrTrim(s string) string {
	return strings.TrimSpace(s)
}

func StrContains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func ContainsStr(s, substr string) bool {
	return strings.Contains(s, substr)
}

func JoinStr(a []string, sep string) string {
	return strings.Join(a, sep)
}

func StrJoin(a []string, sep string) string {
	return strings.Join(a, sep)
}

func StrToLower(s string) string {
	return strings.ToLower(s)
}

func ToLowerStr(s string) string {
	return strings.ToLower(s)
}

func StrToUpper(s string) string {
	return strings.ToUpper(s)
}

func ToUpperStr(s string) string {
	return strings.ToUpper(s)
}

func StrTrimRight(s, cutset string) string {
	return strings.TrimRight(s, cutset)
}

func TrimRightStr(s, cutset string) string {
	return strings.TrimRight(s, cutset)
}

func PathBase(p string) string {
	return path.Base(p)
}

func PathDir(p string) string {
	return path.Dir(p)
}

func PathExt(p string) string {
	return path.Ext(p)
}

func PathClean(p string) string {
	return path.Clean(p)
}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

func NewDir(path string) error {
	return os.MkdirAll(path, 0777)
}

func ReadFile(path string) ([]byte, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func WriteFile(path string, data []byte) {
	dir := PathDir(path)
	if !PathExists(dir) {
		NewDir(dir)
	}
	ioutil.WriteFile(path, data, 0777)
}

func GetFiles(path string) []string {
	files := []string{}
	filepath.Walk(path, func(path string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}
		if f.IsDir() {
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files
}

func DelFile(path string) {
	os.Remove(path)
}

func DelDir(path string) {
	os.RemoveAll(path)
}
