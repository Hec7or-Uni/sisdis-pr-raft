package testintegracionraft1

import (
	"fmt"
	"raft/internal/comun/check"
	"raft/internal/comun/rpctimeout"
	"raft/internal/comun/utils"
	"raft/internal/despliegue"
	"raft/internal/raft"
	"strconv"
	"testing"
	"time"
)

const (
	// Hosts
	MAQUINA1 = "127.0.0.1"
	MAQUINA2 = "127.0.0.1"
	MAQUINA3 = "127.0.0.1"

	// Puertos
	PUERTOREPLICA1 = "29081"
	PUERTOREPLICA2 = "29082"
	PUERTOREPLICA3 = "29083"

	// Nodos replicas
	REPLICA1 = MAQUINA1 + ":" + PUERTOREPLICA1
	REPLICA2 = MAQUINA2 + ":" + PUERTOREPLICA2
	REPLICA3 = MAQUINA3 + ":" + PUERTOREPLICA3

	// Paquete main de ejecutables relativos a PATH previo
	EXECREPLICA = "cmd/srvraft/main.go"

	// Fichero de la clave privada
	PRIVKEYFILE = "id_ed25519"
)

// PATH de los ejecutables de modulo golang de servicio Raft
// var PATH string = filepath.Join(os.Getenv("HOME"), "tmp", "p5", "raft")
// var PATH string = "/mnt/c/Users/Hector/Documents/798095/sis_dis/sisdis-pr-4-pseudocodigoman"
var PATH string = "/home/hector/Desktop/sisdis-pr-4-aux"

// go run cmd/srvraft/main.go 0 127.0.0.1:29001 127.0.0.1:29002 127.0.0.1:29003
var EXECREPLICACMD string = "cd " + PATH + "; go run " + EXECREPLICA

// TEST primer rango
func TestPrimerasPruebas(t *testing.T) { // (m *testing.M) {
	// Crear canal de resultados de ejecuciones ssh en maquinas remotas
	cfg := makeCfgDespliegue(t,
		3,
		[]string{REPLICA1, REPLICA2, REPLICA3},
		[]bool{true, true, true})

	// tear down code
	// eliminar procesos en máquinas remotas
	defer cfg.stop()

	// Run test sequence
	// Test1 : No debería haber ningun primario, si SV no ha recibido aún latidos
	t.Run("T1:soloArranqueYparada",
		func(t *testing.T) { cfg.soloArranqueYparadaTest1(t) })
	// Test2 : No debería haber ningun primario, si SV no ha recibido aún latidos
	t.Run("T2:ElegirPrimerLider",
		func(t *testing.T) { cfg.elegirPrimerLiderTest2(t) })
	// Test3: tenemos el primer primario correcto
	t.Run("T3:FalloAnteriorElegirNuevoLider",
		func(t *testing.T) { cfg.falloAnteriorElegirNuevoLiderTest3(t) })
	// Test4: Tres operaciones comprometidas en configuración estable
	t.Run("T4:tresOperacionesComprometidasEstable",
		func(t *testing.T) { cfg.tresOperacionesComprometidasEstable(t) })
}

// TEST primer rango
func TestAcuerdosConFallos(t *testing.T) { // (m *testing.M) {
	// Crear canal de resultados de ejecuciones ssh en maquinas remotas
	cfg := makeCfgDespliegue(t,
		3,
		[]string{REPLICA1, REPLICA2, REPLICA3},
		[]bool{true, true, true})

	// tear down code
	// eliminar procesos en máquinas remotas
	defer cfg.stop()

	// Test5: Se consigue acuerdo a pesar de desconexiones de seguidor
	t.Run("T5:AcuerdoAPesarDeDesconexionesDeSeguidor ",
		func(t *testing.T) { cfg.AcuerdoApesarDeSeguidor(t) })

	t.Run("T5:SinAcuerdoPorFallos ",
		func(t *testing.T) { cfg.SinAcuerdoPorFallos(t) })

	t.Run("T5:SometerConcurrentementeOperaciones ",
		func(t *testing.T) { cfg.SometerConcurrentementeOperaciones(t) })

}

// ---------------------------------------------------------------------
// Canal de resultados de ejecución de comandos ssh remotos
type canalResultados chan string

func (cr canalResultados) stop() {
	close(cr)

	// Leer las salidas obtenidos de los comandos ssh ejecutados
	for s := range cr {
		fmt.Println(s)
	}
}

// ---------------------------------------------------------------------
// Operativa en configuracion de despliegue y pruebas asociadas
type configDespliegue struct {
	t           *testing.T
	conectados  []bool
	numReplicas int
	nodosRaft   []rpctimeout.HostPort
	cr          canalResultados
}

// Crear una configuracion de despliegue
func makeCfgDespliegue(t *testing.T, n int, nodosraft []string,
	conectados []bool) *configDespliegue {
	cfg := &configDespliegue{}
	cfg.t = t
	cfg.conectados = conectados
	cfg.numReplicas = n
	cfg.nodosRaft = rpctimeout.StringArrayToHostPortArray(nodosraft)
	cfg.cr = make(canalResultados, 2000)

	return cfg
}

func (cfg *configDespliegue) stop() {
	//cfg.stopDistributedProcesses()
	time.Sleep(50 * time.Millisecond)
	cfg.cr.stop()
}

// --------------------------------------------------------------------------
// FUNCIONES DE SUBTESTS

// Se pone en marcha una replica ?? - 3 NODOS RAFT
func (cfg *configDespliegue) soloArranqueYparadaTest1(t *testing.T) {
	// t.Skip("SKIPPED soloArranqueYparadaTest1")
	fmt.Println(t.Name(), ".....................")
	cfg.t = t // Actualizar la estructura de datos de tests para errores
	cfg.startDistributedProcesses()

	// Comprobar estado replica 0
	cfg.comprobarEstadoRemoto(0, 0, false, -1)
	// Comprobar estado replica 1
	cfg.comprobarEstadoRemoto(1, 0, false, -1)
	// Comprobar estado replica 2
	cfg.comprobarEstadoRemoto(2, 0, false, -1)

	// Parar réplicas almacenamiento en remoto
	cfg.stopDistributedProcesses()
	fmt.Println(".............", t.Name(), "Superado")
}

// Primer lider en marcha - 3 NODOS RAFT
func (cfg *configDespliegue) elegirPrimerLiderTest2(t *testing.T) {
	// t.Skip("SKIPPED ElegirPrimerLiderTest2")
	fmt.Println(t.Name(), ".....................")
	cfg.startDistributedProcesses()

	// Se ha elegido lider ?
	fmt.Printf("Probando lider en curso\n")
	cfg.pruebaUnLider(3)

	// Parar réplicas alamcenamiento en remoto
	cfg.stopDistributedProcesses() // Parametros
	fmt.Println(".............", t.Name(), "Superado")
}

// Fallo de un primer lider y reeleccion de uno nuevo - 3 NODOS RAFT
func (cfg *configDespliegue) falloAnteriorElegirNuevoLiderTest3(t *testing.T) {
	// t.Skip("SKIPPED FalloAnteriorElegirNuevoLiderTest3")
	fmt.Println(t.Name(), ".....................")
	cfg.startDistributedProcesses()

	cfg.pruebaReeleccionLider()

	// Parar réplicas almacenamiento en remoto
	cfg.stopDistributedProcesses() //parametros
	fmt.Println(".............", t.Name(), "Superado")
}

// 3 operaciones comprometidas con situacion estable y sin fallos - 3 NODOS RAFT
func (cfg *configDespliegue) tresOperacionesComprometidasEstable(t *testing.T) {
	// t.Skip("SKIPPED tresOperacionesComprometidasEstable")
	fmt.Println(t.Name(), ".....................")
	cfg.startDistributedProcesses()

	cfg.pruebaSometer3()

	// Parar réplicas almacenamiento en remoto
	cfg.stopDistributedProcesses() //parametros
	fmt.Println(".............", t.Name(), "Superado")
}

// Se consigue acuerdo a pesar de desconexiones de seguidor -- 3 NODOS RAFT
func (cfg *configDespliegue) AcuerdoApesarDeSeguidor(t *testing.T) {
	// t.Skip("SKIPPED AcuerdoApesarDeSeguidor")
	fmt.Println(t.Name(), ".....................")
	cfg.startDistributedProcesses()

	cfg.pruebaAcuerdoApesarDeSeguidor()

	cfg.stopDistributedProcesses() //parametros
	fmt.Println(".............", t.Name(), "Superado")
}

// NO se consigue acuerdo al desconectarse mayoría de seguidores -- 3 NODOS RAFT
func (cfg *configDespliegue) SinAcuerdoPorFallos(t *testing.T) {
	// t.Skip("SKIPPED SinAcuerdoPorFallos")
	fmt.Println(t.Name(), ".....................")
	cfg.startDistributedProcesses()

	cfg.pruebaSinAcuerdoPorFallos()

	cfg.stopDistributedProcesses() //parametros
	fmt.Println(".............", t.Name(), "Superado")
}

// Se somete 5 operaciones de forma concurrente -- 3 NODOS RAFT
func (cfg *configDespliegue) SometerConcurrentementeOperaciones(t *testing.T) {
	// t.Skip("SKIPPED SometerConcurrentementeOperaciones")
	fmt.Println(t.Name(), ".....................")
	cfg.startDistributedProcesses()

	cfg.pruebaSometerConcurrentementeOperaciones(5)

	cfg.stopDistributedProcesses() //parametros
	fmt.Println(".............", t.Name(), "Superado")
}

// --------------------------------------------------------------------------
// FUNCIONES DE APOYO
// Comprobar que hay un solo lider
// probar varias veces si se necesitan reelecciones
func (cfg *configDespliegue) pruebaUnLider(numreplicas int) int {
	for iters := 0; iters < 10; iters++ {
		time.Sleep(500 * time.Millisecond)
		mapaLideres := make(map[int][]int)
		for i := 0; i < numreplicas; i++ {
			if cfg.conectados[i] {
				if _, mandato, eslider, _ := cfg.obtenerEstadoRemoto(i); eslider {
					mapaLideres[mandato] = append(mapaLideres[mandato], i)
				}
			}
		}

		ultimoMandatoConLider := -1
		for mandato, lideres := range mapaLideres {
			if len(lideres) > 1 {
				cfg.t.Fatalf("mandato %d tiene %d (>1) lideres",
					mandato, len(lideres))
			}
			if mandato > ultimoMandatoConLider {
				ultimoMandatoConLider = mandato
			}
		}

		if len(mapaLideres) != 0 {
			return mapaLideres[ultimoMandatoConLider][0] // Termina
		}
	}

	cfg.t.Fatalf("un lider esperado, ninguno obtenido")

	return -1 // Termina
}

func (cfg *configDespliegue) obtenerEstadoRemoto(
	indiceNodo int) (int, int, bool, int) {
	var reply raft.EstadoRemoto
	err := cfg.nodosRaft[indiceNodo].CallTimeout("NodoRaft.ObtenerEstadoNodo",
		raft.Vacio{}, &reply, 10*time.Millisecond)
	check.CheckError(err, "Error en llamada RPC ObtenerEstadoRemoto")

	return reply.IdNodo, reply.Mandato, reply.EsLider, reply.IdLider
}

// start  gestor de vistas; mapa de replicas y maquinas donde ubicarlos;
// y lista clientes (host:puerto)
func (cfg *configDespliegue) startDistributedProcesses() {
	//cfg.t.Log("Before starting following distributed processes: ", cfg.nodosRaft)

	for i, endPoint := range cfg.nodosRaft {
		despliegue.ExecMutipleHosts(EXECREPLICACMD+
			" "+strconv.Itoa(i)+" "+
			rpctimeout.HostPortArrayToString(cfg.nodosRaft),
			[]string{endPoint.Host()}, cfg.cr, PRIVKEYFILE)

		// dar tiempo para se establezcan las replicas
		//time.Sleep(500 * time.Millisecond)
	}

	// aproximadamente 500 ms para cada arranque por ssh en portatil
	time.Sleep(2500 * time.Millisecond)
}

func (cfg *configDespliegue) stopDistributedProcesses() {
	var reply raft.Vacio

	for _, endPoint := range cfg.nodosRaft {
		err := endPoint.CallTimeout("NodoRaft.ParaNodo",
			raft.Vacio{}, &reply, 10*time.Millisecond)
		check.CheckError(err, "Error en llamada RPC Para nodo")
	}
}

// Comprobar estado remoto de un nodo con respecto a un estado prefijado
func (cfg *configDespliegue) comprobarEstadoRemoto(idNodoDeseado int,
	mandatoDeseado int, esLiderDeseado bool, IdLiderDeseado int) {
	idNodo, mandato, esLider, idLider := cfg.obtenerEstadoRemoto(idNodoDeseado)

	// cfg.t.Log("Estado replica 0: ", idNodo, mandato, esLider, idLider, "\n")

	if idNodo != idNodoDeseado || mandato != mandatoDeseado ||
		esLider != esLiderDeseado || idLider != IdLiderDeseado {
		cfg.t.Fatalf("Estado incorrecto en replica %d en subtest %s",
			idNodoDeseado, cfg.t.Name())
	}
}

// =============================================================================
// Tests primeras pruebas
// =============================================================================

/**
 * @brief Un líder nuevo toma el relevo de uno caido.
 *  	1- Se obtiene un líder y se para.
 *  	2- Se comprueba que se elige un nuevo líder.
 */
func (cfg *configDespliegue) pruebaReeleccionLider() {
	var nodes []int = []int{1, 2, 0}
	idLider := cfg.consigueLider()
	cfg.desconectarNodos([]int{nodes[idLider]})

	for {
		_, _, _, idLider = cfg.obtenerEstadoRemoto(1)
		if idLider != -1 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
}

/**
 * @brief Se consigue comprometer 3 operaciones seguidas en 3 entradas, con un
 *			  líder estable y sin fallos en el sistema.
 *  	1- Se obtiene un líder y se comprometen 3 entrada
 *  	2- Se comprueba que se han comprometido las 3 entradas
 */
func (cfg *configDespliegue) pruebaSometer3() {
	idLider := cfg.consigueLider()
	cfg.someterOperaciones(idLider, 0, 3)
	cfg.comprobarEntradasComprometidas(idLider, 3) // CHECK
}

// =============================================================================
// Tests acuerdos con fallos
// =============================================================================

/**
 * @brief Se consigue acuerdos de varias entradas de registro a pesar de que una
 * 				réplica (de un grupo de 3 nodos) se desconecta del grupo.
 * 		1- Se obtiene un líder y se compromete una entrada
 * 		2- Se COMPRUEBA que se consigue un acuerdo con la entrada comprometida
 * 		3- Se desconecta una réplica
 * 		4- Se comprometen varias entradas
 * 		5- Se reconecta la réplica caída
 * 		6- Se COMPRUEBA que se consiguen varios acuerdos
 */
func (cfg *configDespliegue) pruebaAcuerdoApesarDeSeguidor() {
	var nodes []int = []int{1, 2, 0}

	idLider := cfg.consigueLider()
	cfg.someterOperaciones(idLider, 0, 1)
	cfg.comprobarEntradasComprometidas(idLider, 1) // CHECK

	cfg.desconectarNodos([]int{nodes[idLider]})
	cfg.someterOperaciones(idLider, 1, 3)
	cfg.reconectarNodos([]int{nodes[idLider]})
	time.Sleep(200 * time.Millisecond)
	cfg.comprobarEntradasComprometidas(idLider, 4) // CHECK
}

/**
 * @brief NO se consigue acuerdo de varias entradas al desconectarse 2
 * 				nodos Raft de 3
 * 		1- Se obtiene un líder y se compromete una entrada
 * 		2- Se desconecta las réplicas seguidoras
 * 		3- Se comprometen varias entradas
 * 		4- Se COMPRUEBA que NO se consiguen acuerdos
 * 		5- Se reconecta las réplicas seguidoras
 * 		6- Se COMPRUEBA que se consiguen varios acuerdos
 */
func (cfg *configDespliegue) pruebaSinAcuerdoPorFallos() {
	var nodes []int = []int{1, 2, 0}
	idLider := cfg.consigueLider()

	cfg.someterOperaciones(idLider, 0, 1)
	cfg.comprobarEntradasComprometidas(idLider, 1) // CHECK

	cfg.desconectarNodos([]int{nodes[idLider], nodes[nodes[idLider]]})
	cfg.someterOperaciones(idLider, 1, 3)
	cfg.comprobarEntradasComprometidas(idLider, 1, []int{nodes[idLider],
		nodes[nodes[idLider]]}) // CHECK

	cfg.reconectarNodos([]int{nodes[idLider], nodes[nodes[idLider]]})
	cfg.comprobarEntradasComprometidas(idLider, 4) // CHECK
}

/**
 * @brief Someter 5 operaciones cliente de forma concurrente y comprobar
 * 				avance de indice del registro.
 * 		1- Se obtiene un líder y se compromete una entrada
 * 		2- Se someten 5 operaciones cliente de forma concurrente
 * 		3- Se COMPRUEBA que se consiguen varios acuerdos y que el indice
 * 				del registro es el mismo en todos los nodos
 */
func (cfg *configDespliegue) pruebaSometerConcurrentementeOperaciones(ops int) {
	// Obtener un lider
	idLider := cfg.consigueLider()
	cfg.someterOperaciones(idLider, 0, 1)

	index := make(chan int, 5)
	// Someter 5  operaciones concurrentemente
	for i := 0; i < 5; i++ {
		go func(i int) {
			operacion := raft.TipoOperacion{
				Operacion: "escribir",
				Clave:     fmt.Sprintf("key: %d", i),
				Valor:     fmt.Sprintf("value: %d", i),
			}

			var reply raft.ResultadoRemoto
			err := cfg.nodosRaft[idLider].CallTimeout("NodoRaft.SometerOperacionRaft",
				operacion, &reply, 25*time.Millisecond)
			check.CheckError(err, "Error en llamada RPC SometerOperacionRaft")
			index <- reply.IndiceRegistro
		}(i)
	}
	time.Sleep(200 * time.Millisecond)
	var indexList map[int]int = make(map[int]int)
	for j := 0; j < 5; j++ {
		idx := <-index
		cfg.t.Log("Indice recibido: ", idx)
		indexList[idx] = idx
	}

	if len(indexList) != 5 {
		cfg.t.Fatalf("Estado incorrecto, no hay 5 indices distintos")
	}
}

// =============================================================================
// Funciones auxiliares
// =============================================================================

/**
 * @brief Devuelve el id del nodo lider
 */
func (cfg *configDespliegue) consigueLider() int {
	var idLider int
	for {
		time.Sleep(20 * time.Millisecond)
		_, _, _, idLider = cfg.obtenerEstadoRemoto(0)
		if idLider != -1 {
			break
		}
	}
	return idLider
}

/**
 * @brief Somete N operaciones
 * @param idLider id del nodo lider
 * @param start indice de la primera operacion
 * @param numOperaciones Numero de operaciones a someter
 */
func (cfg *configDespliegue) someterOperaciones(
	idLider, start, numOperaciones int) {
	for i := start; i < start+numOperaciones; i++ {
		operacion := raft.TipoOperacion{
			Operacion: "escribir",
			Clave:     fmt.Sprintf("key: %d", i),
			Valor:     fmt.Sprintf("value: %d", i),
		}

		// someter operación
		var reply raft.ResultadoRemoto
		err := cfg.nodosRaft[idLider].CallTimeout("NodoRaft.SometerOperacionRaft",
			operacion, &reply, 25*time.Millisecond)
		check.CheckError(err, "Error en llamada RPC SometerOperacionRaft")
	}
	time.Sleep(2000 * time.Millisecond)
}

/**
 * @brief Desconecta los nodos que esten dentro de la lista
 */
func (cfg *configDespliegue) desconectarNodos(idNodos []int) {
	for _, nodo := range idNodos {
		var res raft.Vacio
		err := cfg.nodosRaft[nodo].CallTimeout("NodoRaft.ParaNodo",
			raft.Vacio{}, &res, 25*time.Millisecond)
		check.CheckError(err, "Error en llamada RPC Para nodo")
	}
}

/**
 * @brief Conecta los nodos que esten dentro de la lista
 */
func (cfg *configDespliegue) reconectarNodos(idNodos []int) {
	for _, nodo := range idNodos {
		despliegue.ExecMutipleHosts(EXECREPLICACMD+" "+strconv.Itoa(nodo)+" "+
			rpctimeout.HostPortArrayToString(cfg.nodosRaft),
			[]string{cfg.nodosRaft[nodo].Host()}, cfg.cr, PRIVKEYFILE)
	}
	time.Sleep(2500 * time.Millisecond)
}

/**
 * @brief Comprueba que haya un numero <numAcuerdos> de acuerdos
 * @param idLider id del nodo lider
 * @param numAcuerdos numero de acuerdos esperados
 */
func (cfg *configDespliegue) comprobarEntradasComprometidas(idLider, numAcuerdos int, caidos ...[]int) {
	// Espero a que se consigan varios acuerdos
	time.Sleep(250 * time.Millisecond)

	// Obtener el estado de los almacenes de cada nodo
	var replies []raft.EstadoAlmacen = make([]raft.EstadoAlmacen, len(cfg.nodosRaft))
	for i, endPoint := range cfg.nodosRaft {
		if caidos == nil {
			err := endPoint.CallTimeout("NodoRaft.ObtenerEstadoAlmacen",
				raft.Vacio{}, &replies[i], 25*time.Millisecond)
			check.CheckError(err, "Error en llamada RPC ObtenerEstadoAlmacen")
		} else if caidos != nil && !utils.EstaEnLista(i, caidos[0]) {
			err := endPoint.CallTimeout("NodoRaft.ObtenerEstadoAlmacen",
				raft.Vacio{}, &replies[i], 25*time.Millisecond)
			check.CheckError(err, "Error en llamada RPC ObtenerEstadoAlmacen")
		}
	}

	// Busco el indice del registro en el lider
	var idxLider int
	for _, reply := range replies {
		if reply.IdNodo == idLider {
			idxLider = reply.IdNodo
			break
		}
	}

	// Comprobar que el numero de acuerdos es el esperado
	if len(replies[idxLider].Almacen) != numAcuerdos {
		cfg.t.Fatalf("No hay %d acuerdos", numAcuerdos)
	}

	// Comprobar que los acuerdos son iguales
	for i := 0; i < len(cfg.nodosRaft)-1; i++ {
		if len(replies[i].Almacen) != len(replies[i+1].Almacen) ||
			len(replies[i].Log) != len(replies[i+1].Log) {
			cfg.t.Fatalf("No se han completado todos los acuerdos")
		}
	}
}
