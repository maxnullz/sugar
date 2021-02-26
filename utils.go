package sugar

import (
	"bytes"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

func Atoi(str string) int {
	i, err := strconv.Atoi(str)
	if err != nil {
		return 0
	}
	return i
}

func Itoa(num interface{}) string {
	switch n := num.(type) {
	case int8:
		return strconv.FormatInt(int64(n), 10)
	case int16:
		return strconv.FormatInt(int64(n), 10)
	case int32:
		return strconv.FormatInt(int64(n), 10)
	case int:
		return strconv.FormatInt(int64(n), 10)
	case int64:
		return strconv.FormatInt(int64(n), 10)
	case uint8:
		return strconv.FormatUint(uint64(n), 10)
	case uint16:
		return strconv.FormatUint(uint64(n), 10)
	case uint32:
		return strconv.FormatUint(uint64(n), 10)
	case uint:
		return strconv.FormatUint(uint64(n), 10)
	case uint64:
		return strconv.FormatUint(uint64(n), 10)
	}
	return ""
}

func ParseBaseKind(kind reflect.Kind, data string) (interface{}, error) {
	switch kind {
	case reflect.String:
		return data, nil
	case reflect.Bool:
		v := data == "1" || data == "true"
		return v, nil
	case reflect.Int:
		x, err := strconv.ParseInt(data, 0, 64)
		return int(x), err
	case reflect.Int8:
		x, err := strconv.ParseInt(data, 0, 8)
		return int8(x), err
	case reflect.Int16:
		x, err := strconv.ParseInt(data, 0, 16)
		return int16(x), err
	case reflect.Int32:
		x, err := strconv.ParseInt(data, 0, 32)
		return int32(x), err
	case reflect.Int64:
		x, err := strconv.ParseInt(data, 0, 64)
		return int64(x), err
	case reflect.Float32:
		x, err := strconv.ParseFloat(data, 32)
		return float32(x), err
	case reflect.Float64:
		x, err := strconv.ParseFloat(data, 64)
		return float64(x), err
	case reflect.Uint:
		x, err := strconv.ParseUint(data, 10, 64)
		return uint(x), err
	case reflect.Uint8:
		x, err := strconv.ParseUint(data, 10, 8)
		return uint8(x), err
	case reflect.Uint16:
		x, err := strconv.ParseUint(data, 10, 16)
		return uint16(x), err
	case reflect.Uint32:
		x, err := strconv.ParseUint(data, 10, 32)
		return uint32(x), err
	case reflect.Uint64:
		x, err := strconv.ParseUint(data, 10, 64)
		return uint64(x), err
	default:
		return nil, errors.Errorf("type not found type:%v data:%v", kind, data)
	}
}

func HTTPGetWithBasicAuth(url, name, passwd string) (string, *http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", nil, err
	}
	req.SetBasicAuth(name, passwd)
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}
	resp.Body.Close()
	return string(body), resp, nil
}

func HTTPGet(url string) (string, *http.Response, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", resp, err
	}
	resp.Body.Close()
	return string(body), resp, nil
}

func HTTPPost(url, form string) (string, *http.Response, error) {
	resp, err := http.Post(url, "application/x-www-form-urlencoded", strings.NewReader(form))
	if err != nil {
		return "", nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", resp, err
	}
	resp.Body.Close()
	return string(body), resp, nil
}

func HTTPUpload(url, field, file string) (*http.Response, error) {
	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)
	formFile, err := writer.CreateFormFile(field, file)
	if err != nil {
		return nil, err
	}

	srcFile, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer srcFile.Close()
	_, err = io.Copy(formFile, srcFile)
	if err != nil {
		return nil, err
	}

	contentType := writer.FormDataContentType()
	writer.Close()
	resp, err := http.Post(url, contentType, buf)
	if err != nil {
		return nil, err
	}

	return resp, err
}

func SendMail(user, password, host, to, subject, body, mailtype string) error {
	hp := strings.Split(host, ":")
	auth := smtp.PlainAuth("", user, password, hp[0])
	var contentType string
	if mailtype == "html" {
		contentType = "Content-Type: text/" + mailtype + "; charset=UTF-8"
	} else {
		contentType = "Content-Type: text/plain" + "; charset=UTF-8"
	}

	msg := []byte("To: " + to + "\r\nFrom: " + user + ">\r\nSubject: " + "\r\n" + contentType + "\r\n\r\n" + body)
	sendTO := strings.Split(to, ";")
	err := smtp.SendMail(host, auth, user, sendTO, msg)
	return err
}

var allIP []string

func GetSelfIP(ifNames ...string) []string {
	if allIP != nil {
		return allIP
	}
	inters, _ := net.Interfaces()
	if len(ifNames) == 0 {
		ifNames = []string{"eth", "lo", "wireless network", "local network"}
	}

	filterFunc := func(name string) bool {
		for _, v := range ifNames {
			if strings.Index(name, v) != -1 {
				return true
			}
		}
		return false
	}

	for _, inter := range inters {
		if !filterFunc(inter.Name) {
			continue
		}
		address, _ := inter.Addrs()
		for _, a := range address {
			if ipNet, ok := a.(*net.IPNet); ok {
				if ipNet.IP.To4() != nil {
					allIP = append(allIP, ipNet.IP.String())
				}
			}
		}
	}
	return allIP
}

func GetSelfIntraIP(ifNames ...string) (ips []string) {
	all := GetSelfIP(ifNames...)
	for _, v := range all {
		ipA := strings.Split(v, ".")[0]
		if ipA == "10" || ipA == "172" || ipA == "192" || v == "127.0.0.1" {
			ips = append(ips, v)
		}
	}

	return
}

func GetSelfExtraIP(ifNames ...string) (ips []string) {
	all := GetSelfIP(ifNames...)
	for _, v := range all {
		ipA := strings.Split(v, ".")[0]
		if ipA == "10" || ipA == "172" || ipA == "192" || v == "127.0.0.1" {
			continue
		}
		ips = append(ips, v)
	}

	return
}
