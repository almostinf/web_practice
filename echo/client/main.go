package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/signal"

	webpractice "github.com/almostinf/web_practice"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/tomb.v2"
)

type Client struct {
	tomb   tomb.Tomb
	conn   net.Conn
	logger zerolog.Logger
	exit   chan os.Signal
}

func New(logger zerolog.Logger) Client {
	conn, err := net.Dial("tcp", "localhost:4000")
	if err != nil {
		logger.Fatal().Err(err)
	}
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt)

	return Client{
		conn:   conn,
		logger: logger,
		exit:   exit,
	}
}

func (c *Client) Start() {
	c.tomb.Go(c.handleInputMessages)
	c.tomb.Go(c.readMessagesFromServer)

	<-c.exit
	log.Info().Msg("Graceful shutdown..")
}

func (c *Client) Stop() {
	c.tomb.Kill(nil)
	close(c.exit)
	c.conn.Close()
}

func (c *Client) handleInputMessages() error {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		scanner.Scan()

		if err := scanner.Err(); err != nil {
			c.logger.Error().Err(err).Msg("Scan error")
			return err
		}

		text := scanner.Text()
		if text == "" {
			c.logger.Info().Msg("Successfully exit")
			c.exit <- os.Interrupt
			return nil
		}

		_, err := c.conn.Write([]byte(text))
		if err != nil {
			c.logger.Error().Err(err).Msg("Write message err")
			return err
		}
	}
}

func (c *Client) readMessagesFromServer() error {
	for {
		select {
		case <-c.tomb.Dying():
			c.logger.Info().Msg("Read messages from server stop..")
			return nil

		default:
			var msg = make([]byte, 1024)
			n, err := c.conn.Read(msg)
			if err != nil {
				c.logger.Error().Err(err).Msg("Failed to read from conn")
				return err
			}
			if n != 0 {
				fmt.Println("Other client message: ", string(msg))
			}
		}
	}
}

func main() {
	fmt.Println("Write your message (Enter to exit)")
	client := New(webpractice.GetLogger())

	client.Start()
	defer client.Stop()
}
