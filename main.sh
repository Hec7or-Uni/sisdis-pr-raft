#!/bin/sh

# Nodos replicas
REPLICA1="127.0.0.1:29081"
REPLICA2="127.0.0.1:29082"
REPLICA3="127.0.0.1:29083"

# Paquete main de ejecutables relativos a PATH previo
EXECREPLICA="cmd/srvraft/main.go"

# Ejecutar replicas
go run ${EXECREPLICA} 0 ${REPLICA1} ${REPLICA2} ${REPLICA3} &
go run ${EXECREPLICA} 1 ${REPLICA1} ${REPLICA2} ${REPLICA3} &
go run ${EXECREPLICA} 2 ${REPLICA1} ${REPLICA2} ${REPLICA3} &

# Fin tras N segundos
sleep 3
exit 1
