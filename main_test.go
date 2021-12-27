package main

import (
	"net"
	"testing"
	"time"

	"github.com/StuartsHome/number-server/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ss *Server

func init() {
	ss = startServer()

	// For testing the logger can write to stderr.
	logger.InitLogger(false)
}

func startServer() *Server {
	// Start the new server
	tcpServer := NewServer()

	// Run the servers in goroutines to stop blocking.
	go func() {
		tcpServer.run(false) // We don't need the report to run.
	}()
	return tcpServer
}

func TestServer_Running(t *testing.T) {
	conn, err := net.Dial("tcp", ":4000")
	if err != nil {
		t.Error("could not connect to server: ", err)
	}
	defer conn.Close()
}

func TestHandleConnections_Success(t *testing.T) {
	payload := []byte("123456789\n")

	conn, err := net.Dial("tcp", ":4000")
	require.Nil(t, err)
	defer conn.Close()

	_, err = conn.Write(payload)
	require.Nil(t, err)

	out := make([]byte, 28)
	gotLen, err := conn.Read(out)
	require.Nil(t, err)

	got := string(out[:gotLen])

	assert.Equal(t, 28, gotLen)
	assert.Equal(t, "message received: 123456789\n", got)
}

// This test verifies the application catches duplicates
// and the counters increment correctly.
func TestHandleConnections_Duplicates_Counters(t *testing.T) {
	// Stop the current server to remove the current totals.

	ss.stop()

	// Start the new server.
	ss := startServer()

	tests := []struct {
		payload []byte
	}{
		{
			payload: []byte("123456789\n"),
		},
		{
			payload: []byte("123456789\n"),
		},
		{
			payload: []byte("123456789\n"),
		},
	}

	conn, err := net.Dial("tcp", ":4000")
	require.Nil(t, err)
	defer conn.Close()

	for _, tt := range tests {
		_, err = conn.Write(tt.payload)
		require.Nil(t, err)
	}

	time.Sleep(5 * time.Millisecond)
	assert.Equal(t, 1, len(ss.duplicateNums))

	// Total of unique numbers since the start of the application.
	expectedTotal := ss.total.get()
	// Current amount of new unique numbers in the 10 second reporting period.
	// Note: the 10 second report has been switched off for the test.
	expectedNew := ss.new.get()
	// Current number of new duplicate numbers in the 10 second reporting period.
	// Note the 10 second report has been switched off for the test.
	expectedDuplicates := ss.duplicates.get()

	assert.Equal(t, int32(1), expectedTotal)
	assert.Equal(t, int32(1), expectedNew)
	assert.Equal(t, int32(2), expectedDuplicates)
}

// This test verifies that when characters are input
// the server closes the current connection.
func TestHandleConnections_ErrorChars(t *testing.T) {
	payload := []byte("this is a byte")

	conn, err := net.Dial("tcp", ":4000")
	require.Nil(t, err)
	defer conn.Close()

	_, err = conn.Write(payload)
	require.Nil(t, err)

	out := make([]byte, 10)
	_, err = conn.Read(out)
	assert.Error(t, err)
}

// This test verifies entering 'terminate'
// closes the current connection and also shutdown
// the server.
func TestHandleConnections_Terminate(t *testing.T) {
	payload := []byte("terminate\n")

	conn, err := net.Dial("tcp", ":4000")
	require.Nil(t, err)
	defer conn.Close()

	_, err = conn.Write(payload)
	require.Nil(t, err)

	out := make([]byte, 25)
	_, err = conn.Read(out)
	assert.EqualError(t, err, "EOF")

	// Once connection terminated, user is then unable to dial again.
	_, err = net.Dial("tcp", ":4000")
	assert.EqualError(t, err, "dial tcp :4000: connect: connection refused")

	// Restart the server so other tests don't fail.
	startServer()
}

// This test verifies entering 'Terminate' in
// uppercase closes the current connection
// but doesn't shutdown the server.
func TestHandleConnections_Terminate_UpperCaseDoesntWork(t *testing.T) {
	payload := []byte("Terminate\n")
	conn, err := net.Dial("tcp", ":4000")
	require.Nil(t, err)
	defer conn.Close()

	_, err = conn.Write(payload)
	require.Nil(t, err)

	// 'Terminate' closes current connection, but enables user to dial again.
	conn, err = net.Dial("tcp", ":4000")
	require.Nil(t, err)

	// User is able to write again.
	_, err = conn.Write(payload)
	require.Nil(t, err)
}

func TestCheckForTerminate(t *testing.T) {
	buffer := []byte("terminate")
	got := checkForTerminate(buffer)
	expected := true
	assert.Equal(t, expected, got)
}
