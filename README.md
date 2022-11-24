# sisdis-pr-RAFT

## 1. Introduction

## 2. Uso

```bash
./main.sh || pkill main
```

## 3. Test

### 3.1. Clave ssh
```bash
cd ~/.ssh
ssh-keygen -t ed25519
```

#### 3.1.1 Añádir la clave publica al fichero authorized_keys

Desde la máquina local
```bash
cat id_ed25519.pub >> authorized_keys
```

Desde la máquina remota
```bash
cat id_ed25519.pub >> authorized_keys
```

#### 3.1.2 probar la conexión ssh

```bash
ssh user@host ls
```
si se ve algo en pantalla es que la conexión ssh funciona

### 3.2. Modificación del fichero de test

Dentro del fichero de tests hay que modificar las siguientes variables:

- `MAQUINAX`: IP de la máquina X
- `PUERTOREPLICAX`: Puerto de la máquina X
- `PRIVKEYFILE`: Fichero de la clave privada creado anteriormente
- `PATH`: Ruta del directorio raiz del proyecto

### 3.3. Ejecución de los tests

Desde el directorio raiz del proyecto ejecutar el siguiente comando:

```bash
go test -v ./...
```

> **Warning**: Si se ejecuta el comando `go test -v ./...` y no funciona ejecutar el comando siguiente

```bash
go test -v ./internal/testintegracionraft1/testintegracionraft1_test.go
```
