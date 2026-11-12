package pg_test

import (
	"context"
	"net"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kataras/pg"
)

// fakePostgres is a PostgreSQL server that lives inside the test process.
// A connection to it is one end of a net.Pipe, so nothing listens on a port
// and no database is needed. It speaks just enough of the wire protocol for
// the tests: the startup handshake and the simple query protocol.
//
// It keeps every SQL statement it receives, with the connection it arrived
// on. The transaction commands are answered by the fake itself; any other
// statement is answered by the respond func of the test, or with an empty
// result when there is none.
type fakePostgres struct {
	mu      sync.Mutex
	conns   int
	queries []statement
	respond func(sql string) []pgproto3.BackendMessage
}

type statement struct {
	conn int
	sql  string
}

// newFakeDB returns a *pg.DB whose connection pool talks to a fakePostgres.
func newFakeDB(t *testing.T, schema *pg.Schema) (*pg.DB, *fakePostgres) {
	t.Helper()

	fake := new(fakePostgres)

	cfg, err := pgxpool.ParseConfig("host=127.0.0.1 port=5432 user=test password=test dbname=test sslmode=disable search_path=public")
	if err != nil {
		t.Fatal(err)
	}

	cfg.MaxConns = 4
	// The address is never dialed: every connection is an in-memory pipe.
	cfg.ConnConfig.DialFunc = fake.dial
	// The simple protocol sends each statement as one text message, with its
	// arguments already in place, which is all the fake has to understand.
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	return pg.OpenPool(schema, pool), fake
}

func (f *fakePostgres) dial(context.Context, string, string) (net.Conn, error) {
	f.mu.Lock()
	f.conns++
	id := f.conns
	f.mu.Unlock()

	client, server := net.Pipe()
	go f.serve(id, server)
	return client, nil
}

// statements returns the SQL received so far, without the pool's liveness pings.
func (f *fakePostgres) statements() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	list := make([]string, 0, len(f.queries))
	for _, q := range f.queries {
		if strings.HasPrefix(q.sql, "--") {
			continue
		}
		list = append(list, q.sql)
	}

	return list
}

// connections returns how many connections the statements received so far
// arrived on. Everything a transaction does arrives on one.
func (f *fakePostgres) connections() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	seen := make(map[int]struct{})
	for _, q := range f.queries {
		if strings.HasPrefix(q.sql, "--") {
			continue
		}
		seen[q.conn] = struct{}{}
	}

	return len(seen)
}

func (f *fakePostgres) reset() {
	f.mu.Lock()
	f.queries = nil
	f.mu.Unlock()
}

func (f *fakePostgres) record(conn int, sql string) func(string) []pgproto3.BackendMessage {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.queries = append(f.queries, statement{conn: conn, sql: sql})
	return f.respond
}

func (f *fakePostgres) setRespond(respond func(sql string) []pgproto3.BackendMessage) {
	f.mu.Lock()
	f.respond = respond
	f.mu.Unlock()
}

const (
	txIdle    = 'I'
	txActive  = 'T'
	txAborted = 'E'
)

func (f *fakePostgres) serve(id int, conn net.Conn) {
	defer conn.Close()

	backend := pgproto3.NewBackend(conn, conn)
	if _, err := backend.ReceiveStartupMessage(); err != nil {
		return
	}

	backend.Send(&pgproto3.AuthenticationOk{})
	backend.Send(&pgproto3.ParameterStatus{Name: "client_encoding", Value: "UTF8"})
	backend.Send(&pgproto3.ParameterStatus{Name: "standard_conforming_strings", Value: "on"})
	backend.Send(&pgproto3.ParameterStatus{Name: "server_version", Value: "16.0"})
	backend.Send(&pgproto3.BackendKeyData{ProcessID: 1, SecretKey: []byte{0, 0, 0, 1}})
	backend.Send(&pgproto3.ReadyForQuery{TxStatus: txIdle})
	if err := backend.Flush(); err != nil {
		return
	}

	status := byte(txIdle)

	for {
		msg, err := backend.Receive()
		if err != nil {
			return
		}

		switch m := msg.(type) {
		case *pgproto3.Terminate:
			return
		case *pgproto3.Query:
			for _, answer := range f.answer(id, m.String, &status) {
				backend.Send(answer)
			}
		default:
			backend.Send(pgError("0A000", "the fake server supports the simple query protocol only"))
		}

		backend.Send(&pgproto3.ReadyForQuery{TxStatus: status})
		if err := backend.Flush(); err != nil {
			return
		}
	}
}

func (f *fakePostgres) answer(conn int, sql string, status *byte) []pgproto3.BackendMessage {
	respond := f.record(conn, sql)

	switch strings.ToLower(strings.TrimSuffix(strings.TrimSpace(sql), ";")) {
	case "begin":
		if respond != nil {
			if answers := respond(sql); answers != nil {
				return answers
			}
		}

		*status = txActive
		return commandTag("BEGIN")
	case "commit":
		if respond != nil {
			if answers := respond(sql); answers != nil {
				*status = txIdle
				return answers
			}
		}

		// PostgreSQL answers a COMMIT of an aborted transaction with ROLLBACK.
		tag := "COMMIT"
		if *status == txAborted {
			tag = "ROLLBACK"
		}

		*status = txIdle
		return commandTag(tag)
	case "rollback":
		*status = txIdle
		return commandTag("ROLLBACK")
	}

	if strings.HasPrefix(sql, "--") || strings.TrimSpace(sql) == "" {
		return []pgproto3.BackendMessage{&pgproto3.EmptyQueryResponse{}}
	}

	var answers []pgproto3.BackendMessage
	if respond != nil {
		answers = respond(sql)
	}

	if answers == nil {
		answers = commandTag("SELECT 0")
	}

	for _, answer := range answers {
		if _, failed := answer.(*pgproto3.ErrorResponse); failed && *status == txActive {
			// A failed statement aborts the transaction it runs in.
			*status = txAborted
		}
	}

	return answers
}

func commandTag(tag string) []pgproto3.BackendMessage {
	return []pgproto3.BackendMessage{&pgproto3.CommandComplete{CommandTag: []byte(tag)}}
}

func pgError(code, message string) *pgproto3.ErrorResponse {
	return &pgproto3.ErrorResponse{Severity: "ERROR", Code: code, Message: message}
}

// The PostgreSQL object identifiers of the column types the tests return.
const (
	oidText = 25
	oidUUID = 2950
)

type column struct {
	name string
	oid  uint32
}

// rows answers a statement with the given columns and rows, in text format.
func rows(columns []column, values ...[]string) []pgproto3.BackendMessage {
	fields := make([]pgproto3.FieldDescription, 0, len(columns))
	for _, c := range columns {
		fields = append(fields, pgproto3.FieldDescription{
			Name:         []byte(c.name),
			DataTypeOID:  c.oid,
			DataTypeSize: -1,
			TypeModifier: -1,
		})
	}

	answers := []pgproto3.BackendMessage{&pgproto3.RowDescription{Fields: fields}}
	for _, row := range values {
		data := make([][]byte, 0, len(row))
		for _, v := range row {
			data = append(data, []byte(v))
		}
		answers = append(answers, &pgproto3.DataRow{Values: data})
	}

	count := []byte("SELECT 0")
	count[len(count)-1] = byte('0' + len(values))

	return append(answers, &pgproto3.CommandComplete{CommandTag: count})
}
