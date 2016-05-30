package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"

	"github.com/bind-disney/filefetcher-core/cli"
	fileRpc "github.com/bind-disney/filefetcher-core/rpc"
)

const (
	defaultPort      = 27518
	defaultDirectory = "uploads"
	logPrefix        = "Filefetcher Server: "
)

var (
	portOption      uint
	directoryOption string
	helpOption      bool
	logger          *log.Logger
)

func init() {
	initFlags()
	initLogger()
}

func initFlags() {
	flag.UintVar(&portOption, "P", defaultPort, "Server port")
	flag.StringVar(&directoryOption, "D", defaultDirectory, "Files directory")
	flag.BoolVar(&helpOption, "h", false, "Display this help")
	flag.Usage = cli.ShowUsage

	flag.Parse()
}

func initLogger() {
	logger = log.New(os.Stderr, logPrefix, log.LstdFlags)
	log.SetOutput(os.Stderr)
	log.SetPrefix(logPrefix)
}

func main() {
	if flag.NFlag() == 1 && helpOption {
		flag.Usage()
		os.Exit(0)
	}

	address := fmt.Sprintf(":%d", portOption)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		cli.FatalError("Socket", err)
	}
	defer listener.Close()
	log.Printf("Listening on %s\n", address)

	server, err := fileRpc.NewServer(directoryOption, logger)
	if err != nil {
		cli.FatalError("RPC", err)
	}

	// Personally I like the more explicit way of describing things, that's why I don't use
	// rpc.Register(server) in order to have full control over registered RPC service and
	// to know exactly, what's the name it's accessible by.
	err = rpc.RegisterName("Server", server)
	if err != nil {
		cli.FatalError("RPC", err)
	}

	// We could simply do rpc.Accept(listener), but additional logging will be useful
	for {
		connection, err := listener.Accept()
		if err != nil {
			cli.LogError("Connection", err)
			continue
		}
		if _, err = server.Clients.Add(&connection); err != nil {
			cli.LogError("RPC Registry", err)
			continue
		}

		go func(connection *net.Conn) {
			clientConnection := *connection
			remoteAddress := clientConnection.RemoteAddr().String()

			log.Printf("Opened connection for %s\n", remoteAddress)

			rpc.ServeConn(clientConnection)
			server.Clients.Remove(remoteAddress)

			log.Printf("Closed connection for %s\n", remoteAddress)
		}(&connection)
	}
}
