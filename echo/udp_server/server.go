package udpserver

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/alitto/pond"
	"github.com/rs/zerolog"
	tomb "gopkg.in/tomb.v2"
)

type Config struct {
	Transport string
	Port      string
}

type Server struct {
	connections map[string]*net.UDPAddr
	workerPool  *pond.WorkerPool
	config      Config
	tomb        tomb.Tomb
	udpConn     net.UDPConn
	logger      zerolog.Logger

	connMutex sync.RWMutex
}

func New(config Config, wp *pond.WorkerPool, logger zerolog.Logger) Server {
	return Server{
		connections: make(map[string]*net.UDPAddr),
		workerPool:  wp,
		config:      config,
		logger:      logger,
		connMutex:   sync.RWMutex{},
	}
}

func (s *Server) addClientIfNotExist(addr *net.UDPAddr) {
	s.logger.Info().Msg("Add client")

	s.connMutex.Lock()
	defer s.connMutex.Unlock()
	if _, ok := s.connections[addr.String()]; !ok {
		s.connections[addr.String()] = addr
	}
}

func (s *Server) removeClient(addr *net.UDPAddr) {
	s.logger.Info().Msg("Remove client")

	s.connMutex.Lock()
	defer s.connMutex.Unlock()
	delete(s.connections, addr.String())
}

func (s *Server) handleConnections() error {
	for {
		select {
		case <-s.tomb.Dying():
			s.logger.Info().Msg("handle connections shutdown gracefully")
			return nil

		default:
			var msg = make([]byte, 1024)
			n, addr, err := s.udpConn.ReadFromUDP(msg)
			if err != nil {
				s.logger.Error().Err(err).Msg("Error reading from udp")
				return err
			}

			if n != 0 {
				s.addClientIfNotExist(addr)

				if err := s.sendMessageToAll(msg); err != nil {
					s.logger.Error().Err(err).Msg("Failed to send message to all clients")
					return err
				}
			}
		}
	}
}

func (s *Server) Start() {
	notifyCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	lAdd, err := net.ResolveUDPAddr(s.config.Transport, s.config.Port)
	if err != nil {
		log.Fatal(err)
	}

	udpConn, err := net.ListenUDP(s.config.Transport, lAdd)
	if err != nil {
		s.logger.Fatal().Err(err).Msg("Can not listen")
	}
	s.udpConn = *udpConn

	s.tomb.Go(s.handleConnections)

	<-notifyCtx.Done()

	s.logger.Info().Msg("Graceful Shutdown due signal..")
}

func (s *Server) Stop() {
	s.connMutex.Lock()
	defer s.connMutex.Unlock()

	s.workerPool.StopAndWait()
	s.tomb.Kill(nil)
	for connStr := range s.connections {
		delete(s.connections, connStr)
	}
	s.udpConn.Close()
}

func (s *Server) sendMessageToAll(msg []byte) error {
	s.connMutex.Lock()
	defer s.connMutex.Unlock()

	errGroup, _ := s.workerPool.GroupContext(context.Background())

	for _, addr := range s.connections {
		addr := addr
		errGroup.Submit(func() error {
			if _, err := s.udpConn.WriteTo(msg, addr); err != nil {
				return err
			}
			return nil

		})
	}

	err := errGroup.Wait()
	if err != nil {
		return err
	}

	return nil
}
