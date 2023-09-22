package tcpserver

import (
	"context"
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
	URL       string
}

type Server struct {
	connections map[net.Conn]struct{}
	workerPool  *pond.WorkerPool
	config      Config
	tomb        tomb.Tomb
	logger      zerolog.Logger

	connMutex sync.RWMutex
}

func New(config Config, wp *pond.WorkerPool, logger zerolog.Logger) Server {
	return Server{
		connections: make(map[net.Conn]struct{}),
		workerPool:  wp,
		config:      config,
		logger:      logger,
		connMutex:   sync.RWMutex{},
	}
}

func (s *Server) addClient(conn net.Conn) {
	s.logger.Debug().Msg("Add client")

	s.connMutex.Lock()
	defer s.connMutex.Unlock()

	s.connections[conn] = struct{}{}
}

func (s *Server) removeClient(conn net.Conn) {
	s.logger.Debug().Msg("Remove client")

	s.connMutex.Lock()
	defer s.connMutex.Unlock()

	delete(s.connections, conn)
	conn.Close()
}

func (s *Server) handleConnection(conn net.Conn) {
	s.tomb.Go(func() error {
		s.addClient(conn)
		defer s.removeClient(conn)

		for {
			select {
			case <-s.tomb.Dying():
				s.logger.Info().Msg("handle connection stop..")
				return nil

			default:
				var msg = make([]byte, 1024)
				n, err := conn.Read(msg)
				if err != nil {
					s.logger.Error().Err(err).Msg("Got reading from connection error")
					return err
				}

				if n != 0 {
					if err := s.sendMessageToAll(msg); err != nil {
						s.logger.Error().Err(err).Msg("Failed to send message to all clients")
						return err
					}
				}
			}
		}
	})
}

func (s *Server) Start() {
	notifyCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	listener, err := net.Listen(s.config.Transport, s.config.URL)
	if err != nil {
		s.logger.Fatal().Err(err).Msg("Can not listen")
	}
	defer listener.Close()

	s.tomb.Go(func() error {
		for {
			conn, err := listener.Accept()
			if err != nil {
				s.logger.Error().Err(err).Msg("Got Accept error")
				return err
			}

			s.handleConnection(conn)
		}
	})

	<-notifyCtx.Done()

	s.logger.Info().Msg("Graceful Shutdown due signal..")
}

func (s *Server) Stop() {
	s.connMutex.Lock()
	defer s.connMutex.Unlock()

	s.workerPool.StopAndWait()
	s.tomb.Kill(nil)

	for conn := range s.connections {
		conn.Close()
	}
}

func (s *Server) sendMessageToAll(msg []byte) error {
	s.connMutex.Lock()
	defer s.connMutex.Unlock()

	errGroup, _ := s.workerPool.GroupContext(context.Background())

	for conn := range s.connections {
		conn := conn
		errGroup.Submit(func() error {
			if _, err := conn.Write(msg); err != nil {
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
