package raft

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"raft/internal/comun/rpctimeout"
	"raft/internal/comun/utils"
	"time"
)

// =============================================================================
// Constantes
// =============================================================================

const (
	// --------------- LOGS -----------------
	// true para activar los logs de debug
	kEnableDebugLogs = true
	// true para imprimir los logs en stdout
	kLogToStdout = false
	// Directorio donde se guardan los logs
	kLogOutputDir = "./logs_raft/"
	// ---------- ESTADOS DEL NODO ----------
	Stopped   = "stopped"
	Follower  = "follower"
	Candidate = "candidate"
	Leader    = "leader"
)

/**
 * @brief Inicializa el logger para el nodo.
 * @param nodos Lista de nodos.
 */
func NuevoLogger(nodos []rpctimeout.HostPort, yo int) *log.Logger {
	var logger *log.Logger
	if kEnableDebugLogs {
		nombreNodo := nodos[yo].Host() + "_" + nodos[yo].Port()
		logPrefix := fmt.Sprintf("%s", nombreNodo)

		fmt.Println("LogPrefix: ", logPrefix)

		if kLogToStdout {
			logger = log.New(os.Stdout, nombreNodo+" -->> ",
				log.Lmicroseconds|log.Lshortfile)
		} else {
			err := os.MkdirAll(kLogOutputDir, os.ModePerm)
			if err != nil {
				panic(err.Error())
			}
			logOutputFile, err := os.OpenFile(fmt.Sprintf("%s/%s.txt",
				kLogOutputDir, logPrefix), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
			if err != nil {
				panic(err.Error())
			}
			logger = log.New(logOutputFile,
				logPrefix+" -> ", log.Lmicroseconds|log.Lshortfile)
		}
		logger.Println("logger initialized")
	} else {
		logger = log.New(ioutil.Discard, "", 0)
	}
	return logger
}

// =============================================================================
// Inicializadores
// =============================================================================

/**
 * @brief Inicializa el estado del nodo.
 */
func (nr *NodoRaft) initialiseState() {
	nr.Role = Follower
	nr.IdLider = -1
	// ----- State -----
	nr.State.CurrentTerm = 0
	nr.State.VotedFor = -1
	nr.State.Log = nil
	nr.State.CommitIndex = -1
	nr.State.LastApplied = -1
	nr.State.NextIndex = utils.Make(0, len(nr.Nodos))
	nr.State.MatchIndex = utils.Make(-1, len(nr.Nodos))
	// ----- Channels -----
	nr.VotesCh = make(chan bool)
	nr.HeartbeatCh = make(chan bool)
	nr.Success = make(chan bool)
	nr.AplicaOperacion = make(chan int)
	// ----- Machine -----
	nr.Almacen = make(map[string]string)
}

/**
 * @brief Inicializa el nodo.
 * @param nodos Lista de nodos.
 * @param yo Índice del nodo.
 * @return ptr NodoRaft.
 */
func NuevoNodo(nodos []rpctimeout.HostPort, yo int) *NodoRaft {
	nr := &NodoRaft{}
	nr.Nodos = nodos
	nr.Yo = yo
	nr.IdLider = -1
	nr.Logger = NuevoLogger(nodos, yo)
	// Inicializar NodoRaft
	nr.initialiseState()
	// Iniciar Raft
	go nr.loop()
	// Iniciar el sistema de almacenamiento local
	go nr.almacenamiento()

	return nr
}

// =============================================================================
// Funciones auxiliares para llamadas RPC
// =============================================================================

/**
 * @brief Detiene el nodo.
 */
func (nr *NodoRaft) para() {
	go func() { time.Sleep(5 * time.Millisecond); os.Exit(0) }()
}

/**
 * @brief Devuelve el estado del nodo.
 * @return[0] idNodo
 * @return[1] term
 * @return[2] ¿es lider?
 * @return[3] idLider
 */
func (nr *NodoRaft) obtenerEstado() (int, int, bool, int) {
	nr.Mux.Lock()
	yo := nr.Yo
	mandato := nr.State.CurrentTerm
	esLider := nr.Role == Leader
	idLider := -1
	if esLider {
		idLider = nr.Yo
	} else {
		idLider = nr.IdLider
	}
	nr.Mux.Unlock()

	return yo, mandato, esLider, idLider
}

/**
 * @brief Devuelve el estado del almacenamiento.
 * @return[0] idNodo
 * @return[1] Log
 * @return[2] Almacen
 */
func (nr *NodoRaft) obtenerEstadoAlmacen() (int, []AplicaOperacion, map[string]string) {
	nr.Mux.Lock()
	yo := nr.Yo
	log := nr.State.Log
	almacen := nr.Almacen
	nr.Mux.Unlock()

	return yo, log, almacen
}

/**
 * @brief Devuelve la información del nodo donde debe ser ejecutada la operación.
 * @param operation Operación a someter.
 * @return[0] idNodo
 * @return[1] term
 * @return[2] ¿es lider?
 * @return[3] idLider
 * @return[4] resultado de la operación
 */
func (nr *NodoRaft) someterOperacion(operacion TipoOperacion) (int, int,
	bool, int, string) {
	indice := -1
	mandato := -1
	EsLider := false
	idLider := -1
	valorADevolver := ""

	if nr.Role != Leader {
		return indice, mandato, EsLider, idLider, valorADevolver
	}

	indice = len(nr.State.Log)
	mandato = nr.State.CurrentTerm
	EsLider = true
	idLider = nr.Yo
	valorADevolver = operacion.Valor
	nr.State.Log = append(nr.State.Log, AplicaOperacion{indice, mandato, operacion})
	nr.Logger.Printf("Operacion añadida al log: %v", operacion)

	return indice, mandato, EsLider, idLider, valorADevolver
}

/**
 * @brief
 * @param logLength
 * @param leaderCommit
 * @param entries
 */
func (nr *NodoRaft) updateLog(logLength int, leaderCommit int,
	entries []AplicaOperacion) {
	if len(entries) > 0 && len(nr.State.Log) > logLength {
		if logLength == entries[0].Indice &&
			nr.State.Log[logLength].Term == entries[0].Term {
			nr.Logger.Println("T1 - Eliminamos si hay conflicto en el log")
			nr.State.Log = nr.State.Log[:logLength]
		} else if nr.State.Log[logLength].Term != entries[0].Term {
			nr.Logger.Println("T2 - Eliminamos si hay conflicto en el log")
			nr.State.Log = nr.State.Log[:logLength]
		}
	}

	if (logLength + len(entries)) > len(nr.State.Log) {
		nr.Logger.Println("Añadimos las entradas al log")
		nr.State.Log = append(nr.State.Log, entries...)
	}

	if leaderCommit > nr.State.CommitIndex {
		nr.Logger.Println("Aplicamos a la máquina de estado aquellas entradas " +
			"que no hayan sido aplicados")
		next := nr.State.CommitIndex + 1
		nr.State.CommitIndex = leaderCommit

		for i := next; i <= nr.State.CommitIndex; i++ {
			nr.AplicaOperacion <- i
		}
	}
}

// =============================================================================
// LLamadas RPC
// =============================================================================

/**
 * @brief RPC para detener el nodo.
 * @param args Vacio.
 * @param *reply Vacio.
 * @return error.
 */
func (nr *NodoRaft) ParaNodo(args Vacio, reply *Vacio) error {
	nr.Logger.Printf("NODO:\n"+
		"CommitIndex: %d\n"+"Log: %v\n"+"Almacen: %v\n",
		nr.State.CommitIndex, nr.State.Log, nr.Almacen)
	nr.Logger.Println("STATE: Stopped")
	defer nr.para()
	return nil
}

/**
 * @brief RPC para obtener el estado del nodo.
 * @param args Vacio.
 * @param *reply EstadoRemoto.
 * @return error.
 */
func (nr *NodoRaft) ObtenerEstadoNodo(args Vacio, reply *EstadoRemoto) error {
	reply.IdNodo, reply.Mandato, reply.EsLider, reply.IdLider = nr.obtenerEstado()
	return nil
}

/**
 * @brief RPC para obtener el estado del almacenamiento.
 * @param args Vacio.
 * @param *reply EstadoAlmacen.
 * @return error.
 */
func (nr *NodoRaft) ObtenerEstadoAlmacen(args Vacio, reply *EstadoAlmacen) error {
	reply.IdNodo, reply.Log, reply.Almacen = nr.obtenerEstadoAlmacen()
	return nil
}

/**
 * @brief RPC para someter una operación.
 * @param operacion TipoOperacion.
 * @param *reply ResultadoRemoto.
 */
func (nr *NodoRaft) SometerOperacionRaft(
	operacion TipoOperacion, reply *ResultadoRemoto) error {
	reply.IndiceRegistro, reply.Mandato, reply.EsLider, reply.IdLider,
		reply.ValorADevolver = nr.someterOperacion(operacion)
	return nil
}

/**
 * @brief RPC para solicitar voto.
 * @param *args ArgsPeticionVoto.
 * @param *reply RespuestaPeticionVoto.
 * @return error.
 */
func (nr *NodoRaft) PedirVoto(
	peticion *ArgsPeticionVoto, reply *RespuestaPeticionVoto) error {
	nr.Mux.Lock()
	myLogTerm := 0
	LogLength := len(nr.State.Log)
	if LogLength > 0 {
		myLogTerm = nr.State.Log[LogLength-1].Term
	}
	logOk := peticion.LastLogTerm > myLogTerm ||
		(peticion.LastLogTerm == myLogTerm && peticion.LastLogIndex >= LogLength-1)
	termOk := peticion.Term > nr.State.CurrentTerm ||
		(peticion.Term == nr.State.CurrentTerm &&
			(nr.State.VotedFor == -1 || nr.State.VotedFor == peticion.CandidateId))

	reply.Term = nr.State.CurrentTerm
	if logOk && termOk {
		nr.Logger.Printf("RECV RPC.PedirVoto: Voto concedido a %d\n",
			peticion.CandidateId)
		nr.State.CurrentTerm = peticion.Term
		nr.State.VotedFor = peticion.CandidateId
		nr.Role = Follower
		nr.HeartbeatCh <- true
		reply.VoteGranted = true
		reply.Term = nr.State.CurrentTerm
	} else {
		reply.VoteGranted = false
	}
	nr.Mux.Unlock()

	return nil
}

/**
 * @brief RPC para notificar que el lider sigue vivo.
 *  		  Si Entries es vacío, es un heartbeat.
 *        Si no, es un appendEntries.
 * @param *args ArgAppendEntries.
 * @param *reply Results.
 * @return error.
 */
func (nr *NodoRaft) AppendEntries(
	args *ArgAppendEntries, results *Results) error {
	nr.Logger.Printf("AppendEntries recibido de %d", args.LeaderId)
	if args.Term > nr.State.CurrentTerm {
		nr.Logger.Println("RECV RPC.AppendEntries: Voy rezagado")
		nr.Logger.Println()
		nr.State.CurrentTerm = args.Term
		nr.State.VotedFor = -1
	}

	logOk := len(nr.State.Log)-1 >= args.PrevLogIndex
	if logOk && args.PrevLogIndex > -1 {
		logOk = args.PrevLogTerm == nr.State.Log[args.PrevLogIndex].Term
	}

	if args.Term == nr.State.CurrentTerm && logOk {
		nr.Logger.Printf("RPC.AppendEntries: %v\n", args.Entries)
		nr.Role = Follower
		nr.IdLider = args.LeaderId
		nr.updateLog(args.PrevLogIndex+1, args.LeaderCommit, args.Entries)
		nr.Logger.Printf("Log: %v", nr.State.Log)
		results.Term = nr.State.CurrentTerm
		results.Success = true
		results.MatchIndex = args.PrevLogIndex + len(args.Entries)
		nr.HeartbeatCh <- true
	} else {
		nr.Logger.Printf("AppendEntries fallido: %v", args)
		results.Term = nr.State.CurrentTerm
		results.Success = false
		results.MatchIndex = -1
	}
	return nil
}

// =============================================================================
// Funciones auxiliares para el manejo de LLamadas RPC
// =============================================================================

/**
 * @brief Función auxiliar para solicitar voto.
 * @param nodo id del nodo remoto.
 * @param *args ArgsPeticionVoto.
 * @param *reply RespuestaPeticionVoto.
 * @return ¿error?
 */
func (nr *NodoRaft) enviarPeticionVoto(nodo int,
	args *ArgsPeticionVoto, reply *RespuestaPeticionVoto) bool {

	err := nr.Nodos[nodo].CallTimeout("NodoRaft.PedirVoto",
		args, reply, 25*time.Millisecond)
	nr.Logger.Printf("RPC.PedirVoto: REQ[%d]{%v} -> RES[%d]{%v}",
		nr.Yo, args, nodo, reply)

	return err == nil
}

/**
 * @brief Función auxiliar para enviar un heartbeat | appendEntries.
 * @param nodo id del nodo remoto.
 * @param *args ArgsPeticionVoto.
 * @param *reply RespuestaPeticionVoto.
 * @return ¿error?
 */
func (nr *NodoRaft) enviarAppendEntries(nodo int,
	args *ArgAppendEntries, reply *Results) bool {

	err := nr.Nodos[nodo].CallTimeout("NodoRaft.AppendEntries", args,
		reply, 25*time.Millisecond)
	nr.Logger.Printf("RPC.AppendEntries: REQ[%d]{%v} -> RES[%d]{%v}",
		nr.Yo, args, nodo, reply)

	return err == nil
}

// =============================================================================
// Funciones de tratamiento de respuestas de los nodos
// =============================================================================

/**
 * @brief Crea una nueva petición de voto para los nodo.
 */
func (nr *NodoRaft) nuevaEleccion() {
	nr.Logger.Printf("Nueva elección | %d\n", nr.State.CurrentTerm+1)

	nr.State.CurrentTerm++
	nr.State.VotedFor = nr.Yo
	args := ArgsPeticionVoto{nr.State.CurrentTerm, nr.Yo, -1, 0}
	if len(nr.State.Log) > 0 {
		args.LastLogIndex = len(nr.State.Log) - 1
		args.LastLogTerm = nr.State.Log[args.LastLogIndex].Term
	}

	for nodo := range nr.Nodos {
		if nodo != nr.Yo {
			var reply RespuestaPeticionVoto
			go nr.peticionVoto(nodo, &args, &reply)
		}
	}
}

/**
 * @brief Trata la respuesta de un nodo a una petición de voto.
 * @param nodo id del nodo remoto.
 * @param *args ArgsPeticionVoto.
 * @param *reply RespuestaPeticionVoto.
 */
func (nr *NodoRaft) peticionVoto(nodo int,
	args *ArgsPeticionVoto, reply *RespuestaPeticionVoto) {
	if nr.enviarPeticionVoto(nodo, args, reply) {
		nr.Mux.Lock()
		if nr.Role == Candidate &&
			reply.Term == nr.State.CurrentTerm && reply.VoteGranted {
			nr.Logger.Printf("RECV RPC.PedirVoto: Voto recibido de %d\n", nodo)
			nr.VotesCh <- true
		} else if reply.Term > nr.State.CurrentTerm {
			nr.Logger.Println("RECV RPC.PedirVoto: Voy rezagado")
			nr.Role = Follower
			nr.State.CurrentTerm = reply.Term
			nr.State.VotedFor = -1
		}
		nr.Mux.Unlock()
	} else {
		nr.Logger.Println("RECV RPC.PedirVoto: ERROR al recibir mensaje")
	}
}

/**
 * @brief Trata la respuesta de un AppendEntries.
 * @param nodo id del nodo remoto.
 * @param *args ArgAppendEntries.
 * @param *reply Results.
 */
func (nr *NodoRaft) peticionLatido(nodo int,
	args *ArgAppendEntries, reply *Results) {
	if nr.enviarAppendEntries(nodo, args, reply) {
		if reply.Term == nr.State.CurrentTerm && nr.Role == Leader {
			if reply.Success {
				if len(args.Entries) > 0 {
					nr.Logger.Printf("RECV RPC.AppendEntries: OK -> avanzamos "+
						"NI:%d -> NI:%d y MI:%d -> MI:%d\n",
						nr.State.NextIndex[nodo], reply.MatchIndex+1,
						nr.State.MatchIndex[nodo], reply.MatchIndex)
				} else {
					nr.Logger.Println("RECV RPC.AppendEntries: OK")
				}

				nr.State.NextIndex[nodo] = reply.MatchIndex + 1
				nr.State.MatchIndex[nodo] = reply.MatchIndex
				nr.CommitLogEntries()
			} else if nr.State.NextIndex[nodo] > 0 {
				nr.Logger.Printf("RECV RPC.AppendEntries: NOT OK -> retrocedemos "+
					"NI:%d -> NI:%d", nr.State.NextIndex[nodo], nr.State.NextIndex[nodo]-1)

				nr.State.NextIndex[nodo]--
			}
		} else if reply.Term > nr.State.CurrentTerm {
			nr.Logger.Println("RECV RPC.AppendEntries: Voy rezagado")
			nr.Role = Follower
			nr.State.CurrentTerm = reply.Term
			nr.State.VotedFor = -1
		}
	} else {
		nr.Logger.Println("RECV RPC.AppendEntries: ERROR al recibir mensaje")
	}
}

// =============================================================================
// Funciones para la creación de peticiones para los nodos
// =============================================================================

/**
 * @brief Crea una petición de voto para un nodo.
 */
func (nr *NodoRaft) nuevoLatido() {
	nr.Logger.Println("Nuevo latido")
	args := ArgAppendEntries{
		Term:         nr.State.CurrentTerm,
		LeaderId:     nr.Yo,
		PrevLogIndex: -1,
		PrevLogTerm:  0,
		Entries:      nil,
		LeaderCommit: nr.State.CommitIndex,
	}

	for nodo := range nr.Nodos {
		if nodo != nr.Yo {
			var reply Results
			args.PrevLogIndex = nr.State.NextIndex[nodo] - 1
			if args.PrevLogIndex > -1 {
				args.PrevLogTerm = nr.State.Log[args.PrevLogIndex].Term
			}
			if nr.State.NextIndex[nodo] < len(nr.State.Log) {
				args.Entries = nr.State.Log[nr.State.NextIndex[nodo]:]
			}
			go nr.peticionLatido(nodo, &args, &reply)
		}
	}
}

// =============================================================================
// Funciones para la gestión del almacenamiento en la máquina de estados
// =============================================================================

/**
 * @brief Calcula el número de nodos que tienen un índice de log mayor o igual
 * 				a un índice dado.
 * @param index Índice de log a comprobar.
 * @return Número de nodos que tienen un índice de log mayor o igual al dado.
 */
func (nr *NodoRaft) acks(index int) int {
	acks := 1
	for nodo := range nr.Nodos {
		if nodo != nr.Yo && nr.State.MatchIndex[nodo] >= index {
			acks++
		}
	}
	return acks
}

/**
 * @brief Comprueba para cada entrada sin comitear si tiene suficientes acks
 * 				para ser comiteada.
 */
func (nr *NodoRaft) CommitLogEntries() {
	minAcks := len(nr.Nodos) / 2
	for i := nr.State.CommitIndex + 1; i < len(nr.State.Log); i++ {
		acks := nr.acks(i)
		if acks > minAcks && nr.State.Log[i].Term <= nr.State.CurrentTerm {
			nr.State.CommitIndex = i
			nr.AplicaOperacion <- i
		}
	}
}

/**
 * @brief Función que se ejecuta en un hilo para atender las peticiones del
 * 				lider cuando este requiere aplicar una operación a la maquina de
 * 				estados.
 */
func (nr *NodoRaft) almacenamiento() {
	for {
		i := <-nr.AplicaOperacion
		nr.Almacen[nr.State.Log[i].Operacion.Clave] = nr.State.Log[i].Operacion.Valor
		nr.State.LastApplied++
		nr.Logger.Println("Aplicada operación", nr.State.Log[i].Operacion)
		nr.Logger.Printf("ESTADO NODO: init: CI:%d Log: %v Almacen: %v\n",
			nr.State.CommitIndex, nr.State.Log, nr.Almacen)
	}
}

// =============================================================================
// Maquina de estados
// =============================================================================
//													  timeout
//							              ______
//							             v      |
//	 --------    timeout    -----------  recv majority votes   -----------
//	|Follower| ----------> | Candidate |--------------------> |  Leader   |
//   --------               -----------                        -----------
//		  ^          higher term/ |                         higher term |
//			|            new leader |                                     |
//			|_______________________|____________________________________ |
// =============================================================================

func (nr *NodoRaft) loop() {
	time.Sleep(2000 * time.Millisecond)
	for {
		switch nr.Role {
		case Follower:
			nr.Logger.Printf("STATE: Follower | %d\n", nr.State.CurrentTerm)
			nr.followerLoop()
		case Candidate:
			nr.Logger.Printf("STATE: Candidate | %d\n", nr.State.CurrentTerm+1)
			nr.candidateLoop()
		case Leader:
			nr.Logger.Printf("STATE: Leader | %d\n", nr.State.CurrentTerm)
			nr.leaderLoop()
		}
	}
}

/**
 * @brief Caso de que el nodo sea un follower.
 */
func (nr *NodoRaft) followerLoop() {
	timer := time.NewTimer(utils.ElectionTimeout())
	defer timer.Stop()

	for nr.Role == Follower {
		select {
		case <-nr.HeartbeatCh:
			timer.Reset(utils.ElectionTimeout())
		case <-timer.C:
			nr.Role = Candidate
		}
	}
}

/**
 * @brief Caso de que el nodo sea un candidato.
 */
func (nr *NodoRaft) candidateLoop() {
	ticker := time.NewTicker(utils.ElectionTimeout())
	defer ticker.Stop()

	VotesReceived := 1
	nr.nuevaEleccion()
	for nr.Role == Candidate {
		select {
		case <-nr.VotesCh:
			VotesReceived++
			if VotesReceived > len(nr.Nodos)/2 {
				nr.Role = Leader
				nr.IdLider = nr.Yo
			}
		case <-ticker.C:
			VotesReceived = 1
			nr.nuevaEleccion()
		}
	}
}

/**
 * @brief Caso de que el nodo sea un líder.
 */
func (nr *NodoRaft) leaderLoop() {
	nr.Logger.Printf("Leader init: CI:%d Log: %v Almacen: %v\n",
		nr.State.CommitIndex, nr.State.Log, nr.Almacen)

	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()

	nr.IdLider = nr.Yo
	nr.State.NextIndex = utils.Make(len(nr.State.Log), len(nr.Nodos))
	nr.State.MatchIndex = utils.Make(-1, len(nr.Nodos))

	nr.nuevoLatido()

	for nr.Role == Leader {
		select {
		case <-ticker.C:
			nr.nuevoLatido()
		}
	}
}
