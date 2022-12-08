package main

import (
	"bufio"
	"fmt"
	"os"
	"raft/internal/comun/rpctimeout"
	"raft/internal/raft"
	"strconv"
	"strings"
	"time"
)

const (
	// Nodos replicas
	REPLICA1 = "ss-0.ss-service.default.svc.cluster.local:6000"
	REPLICA2 = "ss-1.ss-service.default.svc.cluster.local:6000"
	REPLICA3 = "ss-2.ss-service.default.svc.cluster.local:6000"

	PREFIX = "NodoRaft"
	// RPC Calls
	GET_STATE       = "ObtenerEstadoNodo"
	GET_VAULT_STATE = "ObtenerEstadoAlmacen"
	STOP_NODE       = "ParaNodo"
	SUBMIT_OP       = "SometerOperacionRaft"
)

var REPLICAS = rpctimeout.StringArrayToHostPortArray([]string{REPLICA1, REPLICA2, REPLICA3})

func main() {
	var rEstadoRemoto raft.EstadoRemoto
	var rResultadoRemoto raft.ResultadoRemoto
	var rEstadoAlmacen raft.EstadoAlmacen

	scanner := bufio.NewScanner(os.Stdin)
	menu()
	for scanner.Scan() {
		values := strings.Split(scanner.Text(), " ")
		opt, _ := strconv.Atoi(values[0])
		node, _ := strconv.Atoi(values[1])
		clave := values[2]
		valor := values[3]

		fmt.Println("----------------------------------------")
		fmt.Println("Opcion seleccionada: ", opt)
		fmt.Println("Nodo seleccionado: ", node)
		fmt.Println("----------------------------------------")

		switch opt {
		case 0:
			fmt.Printf("Hasta la proxima.")
			break
		case 1:
			replicaExec(node, GET_STATE, raft.Vacio{}, &rEstadoRemoto)
			printEstadoRemoto(rEstadoRemoto)
		case 2:
			replicaExec(node, GET_VAULT_STATE, raft.Vacio{}, &rEstadoAlmacen)
			printEstadoRemotoAlmacen(rEstadoAlmacen)
		case 3:
			replicaExec(node, STOP_NODE, raft.Vacio{}, &rEstadoRemoto)
			fmt.Printf("Nodo %d detenido.\n", node)
		case 4, 5:
			var operacion raft.TipoOperacion
			fmt.Printf("\nclave: ")
			operacion.Clave = clave
			if opt == 4 {
				operacion.Operacion = "lectura"
			} else {
				operacion.Operacion = "escritura"
				fmt.Printf("\nvalor: ")
				operacion.Valor = valor
			}
			replicaExec(node, SUBMIT_OP, operacion, &rResultadoRemoto)
			printOperacion(rResultadoRemoto)
		}
		menu()
	}
}

func menu() {
	fmt.Println("\n========================================")
	fmt.Println("	1. ObtenerEstadoNodo - [i]")
	fmt.Println("	2. ObtenerEstadoAlmacen - [i]")
	fmt.Println("	3. ParaNodo - [i]")
	fmt.Println("	4. SometerOperacionRaft - [i] -> lectura | clave")
	fmt.Println("	5. SometerOperacionRaft - [i] -> escritura | clave, valor")
	fmt.Println("	0. Exit")
	fmt.Println("========================================")
	fmt.Printf("Introduzca una opcion y un nodo (<opcion*> <nodo*> <clave> <valor>): ")
}

func replicaExec(node int, rpc string, args interface{}, reply interface{}) {
	err := REPLICAS[node].CallTimeout(PREFIX+"."+rpc,
		args, reply, 25*time.Millisecond)
	if err != nil {
		fmt.Println("Error en llamada RPC Para nodo")
		fmt.Println(err)
	}
}

func printEstadoRemoto(estadoRemoto raft.EstadoRemoto) {
	fmt.Println("----------------------------------------")
	fmt.Printf("IdNodo:  %v\n", estadoRemoto.IdNodo)
	fmt.Printf("Mandato: %v\n", estadoRemoto.Mandato)
	fmt.Printf("EsLider: %v\n", estadoRemoto.EsLider)
	fmt.Printf("IdLider: %v\n", estadoRemoto.IdLider)
	fmt.Println("========================================")
}

func printEstadoRemotoAlmacen(estadoRemoto raft.EstadoAlmacen) {
	fmt.Println("----------------------------------------")
	fmt.Printf("IdNodo:  %v\n", estadoRemoto.IdNodo)
	fmt.Printf("Log: 		 %v\n", estadoRemoto.Log)
	fmt.Printf("Almacen: %v\n", estadoRemoto.Almacen)
	fmt.Println("========================================")
}

func printOperacion(estadoRemoto raft.ResultadoRemoto) {
	fmt.Println("----------------------------------------")
	fmt.Printf("Mandato: %v\n", estadoRemoto.EstadoParcial.Mandato)
	fmt.Printf("EsLider: %v\n", estadoRemoto.EstadoParcial.EsLider)
	fmt.Printf("IdLider: %v\n", estadoRemoto.EstadoParcial.IdLider)
	fmt.Printf("ValorADevolver: %v\n", estadoRemoto.ValorADevolver)
	fmt.Printf("IndiceRegistro: %v\n", estadoRemoto.IndiceRegistro)
	fmt.Println("========================================")
}
