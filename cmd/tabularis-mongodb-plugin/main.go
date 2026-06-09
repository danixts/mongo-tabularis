package main

import (
	"os"

	"github.com/danixts/mongo-tabularis/internal/mongodb"
	"github.com/danixts/mongo-tabularis/internal/rpc"
)

func main() {
	pool := mongodb.NewPool()
	defer pool.Close()

	server := rpc.NewServer(pool, os.Stdin, os.Stdout, os.Stderr)
	if err := server.Run(); err != nil {
		os.Stderr.WriteString("plugin terminated: " + err.Error() + "\n")
		os.Exit(1)
	}
}
