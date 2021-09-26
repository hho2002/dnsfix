package main

import (
	"errors"
	"fmt"
	"golang.org/x/text/encoding/simplifiedchinese"
	"os/exec"
	"strings"
)

type Charset string

const (
	UTF8    = Charset("UTF-8")
	GB18030 = Charset("GB18030")
)

func ReplaceHosts(old string, content string) string {
	var startText = "### start dnsfix"
	var endText = "### end dnsfix"
	var start = strings.Index(old, startText)
	var end = strings.Index(old, endText)
	if start <= 0 || end <= 0 {
		return fmt.Sprintf("%s\n%s\n%s\n%s\n", old, startText, content, endText)
	}
	var oldContent = old[start:end]
	new := strings.ReplaceAll(old, oldContent, startText+"\n"+content)
	fmt.Println(old)
	return new
}

func flushDns() error {
	var cmd *exec.Cmd
	switch goos {
	case "windows":
		cmd = exec.Command("ipconfig", "/flushdns")
	case "linux":
		cmd = exec.Command("service", "network", "restart")
	case "darwin":
		cmd = exec.Command("killall", "-HUP", "mDNSResponder")
	default:
		return errors.New("unknown os, ignore dns flush")
	}

	buf, err := cmd.CombinedOutput()

	byte2String := ConvertByte2String([]byte(buf), "GB18030")
	fmt.Printf("%s \n", byte2String)
	return err
}

func ConvertByte2String(byte []byte, charset Charset) string {
	var str string
	switch charset {
	case GB18030:
		var decodeBytes, _ = simplifiedchinese.GB18030.NewDecoder().Bytes(byte)
		str = string(decodeBytes)
	case UTF8:
		fallthrough
	default:
		str = string(byte)
	}
	return str
}
