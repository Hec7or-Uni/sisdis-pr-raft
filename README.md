# sisdis-pr-RAFT

## 1. Introducción

Se va a realizar un sistema de almacenamiento clave/valor basado en RAFT y su posterior despliegue en un cluster de Kubernetes.

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

#### 3.1.1 Añádir la clave pública al fichero authorized_keys

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

Desde el directorio raíz del proyecto ejecutar el siguiente comando:

```bash
go test -v ./...
```

> **Warning**: Si se ejecuta el comando `go test -v ./...` y no funciona ejecutar el comando siguiente

```bash
go test -v ./internal/testintegracionraft1/testintegracionraft1_test.go
```

## 4. Kubernetes

### 4.1. Compilar ejecutables
```bash
CGO_ENABLED=0 go build -o cmd/srvraft/servidor cmd/srvraft/main.go
CGO_ENABLED=0 go build -o cmd/cltraft/cliente cmd/cltraft/main.go
```

### 4.2. Construir las imágenes de docker

```bash
docker build cmd/cltraft/ -t localhost:5001/cliente:latest
docker build cmd/srvraft/ -t localhost:5001/servidor:latest
```

### 4.3. Subir las imágenes al repositorio local
```bash
docker push localhost:5001/cliente:latest
docker push localhost:5001/servidor:latest
```

### 4.4. Desplegar el cluster

```bash
./deployCluster.sh 
```

> **Warning**: Se ha tenido en cuenta que ya existe un cluster de kubernetes en la máquina local, sino ejecutar el siguiente comando

```bash
./kind-with-registry.sh
```

### Comandos útiles

```bash
kubectl exec c1 -ti -- sh
kubectl get all
kubectl get all -o wide
kubectl get pods
kubectl get pods -o wide
kubectl describe svc ss-service
```