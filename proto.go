package main

import (
	"bytes"
	"fmt"
	"github.com/tidwall/resp"
	"io"
	"log"
)

const (
	CommandSET = "SET"
)

type Command interface {
	SetCommand
}
type SetCommand struct {
	key, value string
}

func parseCommand(raw string) (Command, error) {
	rd := resp.NewReader(bytes.NewBufferString(raw))
	for {
		v, _, err := rd.ReadValue()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Read %s\n", v.Type())
		var cmd Command
		if v.Type() == resp.Array {
			for i, value := range v.Array() {
				switch value.String() {
				case CommandSET:
					if len(value.Array()) != 3 {
						return nil, fmt.Errorf("invalid number of variables for SET command")
					}
					cmd = SetCommand{
						key:   value.Array()[1].String(),
						value: value.Array()[2].String(),
					}
					return cmd, nil
				fmt.Printf("  #%d %s, value: '%s'\n", i, v.Type(), v)
			}
		}
	}
	return nil, nil
}
