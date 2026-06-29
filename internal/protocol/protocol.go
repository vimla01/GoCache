package protocol

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/vimla/gocache/internal/store"
)

// Command represents a parsed client command.
type Command struct {
	Name string   // e.g. "SET", "GET", "DELETE", "PING"
	Args []string // positional arguments (key, value, etc.)
}

// Parse takes a raw command line string and returns a parsed Command.
// Commands are case-insensitive. Quoted strings are treated as single arguments.
//
// Supported formats:
//
//	PING
//	GET key
//	SET key value [TTL_seconds]
//	DELETE key
func Parse(line string) (Command, error) {
	args, err := tokenize(line)
	if err != nil {
		return Command{}, err
	}
	if len(args) == 0 {
		return Command{}, fmt.Errorf("empty command")
	}

	cmd := Command{
		Name: strings.ToUpper(args[0]),
		Args: args[1:],
	}

	// Validate argument counts
	switch cmd.Name {
	case "PING":
		// no args required
	case "GET":
		if len(cmd.Args) < 1 {
			return Command{}, fmt.Errorf("wrong number of arguments for 'GET'")
		}
	case "SET":
		if len(cmd.Args) < 2 {
			return Command{}, fmt.Errorf("wrong number of arguments for 'SET'")
		}
		if len(cmd.Args) > 3 {
			return Command{}, fmt.Errorf("wrong number of arguments for 'SET'")
		}
	case "DELETE":
		if len(cmd.Args) < 1 {
			return Command{}, fmt.Errorf("wrong number of arguments for 'DELETE'")
		}
	default:
		return Command{}, fmt.Errorf("unknown command '%s'", cmd.Name)
	}

	return cmd, nil
}

// Execute runs a parsed command against the store and returns the response string.
func Execute(cmd Command, s *store.Store) string {
	switch cmd.Name {
	case "PING":
		return "PONG"

	case "SET":
		var ttl time.Duration
		if len(cmd.Args) == 3 {
			seconds, err := strconv.Atoi(cmd.Args[2])
			if err != nil {
				return fmt.Sprintf("ERROR invalid TTL: %s", cmd.Args[2])
			}
			if seconds <= 0 {
				return "ERROR TTL must be a positive integer"
			}
			ttl = time.Duration(seconds) * time.Second
		}
		s.Set(cmd.Args[0], cmd.Args[1], ttl)
		return "OK"

	case "GET":
		val, ok := s.Get(cmd.Args[0])
		if !ok {
			return "(nil)"
		}
		return fmt.Sprintf("\"%s\"", val)

	case "DELETE":
		if s.Delete(cmd.Args[0]) {
			return "(integer) 1"
		}
		return "(integer) 0"

	default:
		return fmt.Sprintf("ERROR unknown command '%s'", cmd.Name)
	}
}

// tokenize splits a command line into tokens, respecting double-quoted strings.
// e.g. `SET name "hello world"` → ["SET", "name", "hello world"]
func tokenize(line string) ([]string, error) {
	var tokens []string
	var current strings.Builder
	inQuotes := false

	for i := 0; i < len(line); i++ {
		ch := line[i]

		switch {
		case ch == '"':
			if inQuotes {
				// End of quoted string — emit token
				tokens = append(tokens, current.String())
				current.Reset()
				inQuotes = false
			} else {
				// Start of quoted string
				if current.Len() > 0 {
					// Flush any accumulated unquoted chars
					tokens = append(tokens, current.String())
					current.Reset()
				}
				inQuotes = true
			}

		case ch == ' ' || ch == '\t':
			if inQuotes {
				current.WriteByte(ch)
			} else if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}

		default:
			current.WriteByte(ch)
		}
	}

	if inQuotes {
		return nil, fmt.Errorf("unterminated quoted string")
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens, nil
}
