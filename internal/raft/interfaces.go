package raft

import (
	"log"
	"raft/internal/comun/rpctimeout"
	"sync"
)

// =============================================================================
// Estado de un nodo
// =============================================================================
type State struct {
	CurrentTerm int
	VotedFor    int
	Log         []AplicaOperacion
	CommitIndex int
	LastApplied int
	NextIndex   []int
	MatchIndex  []int
}

// =============================================================================
// Nodo Raft
// =============================================================================
type NodoRaft struct {
	Mux             sync.Mutex
	Nodos           []rpctimeout.HostPort
	Yo              int
	IdLider         int
	Logger          *log.Logger
	State           State
	Role            string
	Almacen         map[string]string
	VotesCh         chan bool
	HeartbeatCh     chan bool
	Success         chan bool
	AplicaOperacion chan int
}

// =============================================================================
// Interfaces de la RPC PedirVoto
// =============================================================================
type ArgsPeticionVoto struct {
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

type RespuestaPeticionVoto struct {
	Term        int
	VoteGranted bool
}

// =============================================================================
// Interfaces de la RPC AppendEntries
// =============================================================================
type ArgAppendEntries struct {
	Term         int
	LeaderId     int
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []AplicaOperacion
	LeaderCommit int
}

type Results struct {
	Term       int
	Success    bool
	MatchIndex int
}

// =============================================================================
// Interfaces de la RPC SometerOperacionRaft
// =============================================================================

type TipoOperacion struct {
	Operacion string
	Clave     string
	Valor     string
}

type ResultadoRemoto struct {
	ValorADevolver string
	IndiceRegistro int
	EstadoParcial
}

// =============================================================================
// Interfaces para la RPC ObtenerEstadoNodo
// =============================================================================

type EstadoParcial struct {
	Mandato int
	EsLider bool
	IdLider int
}

type EstadoRemoto struct {
	IdNodo int
	EstadoParcial
}

// =============================================================================
// Interfaces para la RPC ObtenerEstadoAlmacen
// =============================================================================

type EstadoAlmacen struct {
	IdNodo  int
	Log     []AplicaOperacion
	Almacen map[string]string
}

// =============================================================================
// Interfaces auxiliares
// =============================================================================

type Vacio struct{}

type AplicaOperacion struct {
	Indice    int
	Term      int
	Operacion TipoOperacion
}
