package perfomancetests

import (
	"net"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/alitto/pond"
	webpractice "github.com/almostinf/web_practice"
	tcpserver "github.com/almostinf/web_practice/echo/tcp_server"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type ConnectionTestCase struct {
	NumOfConnections int
	Message          string
}

func min(arr []float64) float64 {
	min := arr[0]
	for _, el := range arr {
		if el < min {
			min = el
		}
	}
	return min
}

func max(arr []float64) float64 {
	max := arr[0]
	for _, el := range arr {
		if el > max {
			max = el
		}
	}
	return max
}

func average(arr []float64) float64 {
	sum := 0.
	for _, el := range arr {
		sum += el
	}
	return sum / float64(len(arr))
}

func runTest(b *testing.B, config tcpserver.Config, testname string, testcase ConnectionTestCase) {
	connections := make(map[net.Conn]struct{}, testcase.NumOfConnections)
	durations := make([]float64, 0, testcase.NumOfConnections)

	for i := 0; i < testcase.NumOfConnections; i++ {
		conn, err := net.Dial(config.Transport, config.URL)
		if err != nil {
			b.Error("failed to dial with tcp server: ", err)
		}

		connections[conn] = struct{}{}

		start := time.Now()
		_, err = conn.Write([]byte(testcase.Message))
		if err != nil {
			b.Error("failed to write to tcp server: ", err)
		}

		var msg = make([]byte, 1024)
		var n int
		for n != 0 {
			n, err = conn.Read(msg)
			if err != nil {
				b.Error("failed to read from tcp server: ", err)
			}
		}

		end := time.Now()
		durations = append(durations, end.Sub(start).Seconds())
	}

	for conn := range connections {
		conn.Close()
		delete(connections, conn)
	}

	log.Info().
		Str("name", testname).
		Float64("min", min(durations)).
		Float64("max", max(durations)).
		Float64("average", average(durations)).
		Msg("Successful test")
}

func BenchmarkTCPServer(b *testing.B) {
	b.SetParallelism(1)

	wp := pond.New(runtime.NumCPU(), 120)
	config := tcpserver.Config{
		Transport: "tcp",
		URL:       "localhost:4000",
	}

	server := tcpserver.New(config, wp, webpractice.GetLogger())
	zerolog.SetGlobalLevel(zerolog.FatalLevel)

	go server.Start()
	defer server.Stop()

	time.Sleep(time.Second) // Give server time to start

	testcases := map[string]ConnectionTestCase{
		"low clients and small messages": {
			NumOfConnections: 100,
			Message:          strings.Repeat("test", 20),
		},
		"medium clients and small messages": {
			NumOfConnections: 1000,
			Message:          strings.Repeat("test", 20),
		},
		"many clients and small messages": {
			NumOfConnections: 10000,
			Message:          strings.Repeat("test", 20),
		},
		"low clients and big messages": {
			NumOfConnections: 100,
			Message:          strings.Repeat("test", 200),
		},
		"medium clients and big messages": {
			NumOfConnections: 1000,
			Message:          strings.Repeat("test", 200),
		},
		"many clients and big messages": {
			NumOfConnections: 10000,
			Message:          strings.Repeat("test", 200),
		},
	}

	for name, testcase := range testcases {
		b.Run(name, func(b *testing.B) {
			runTest(b, config, name, testcase)
		})
	}
}
