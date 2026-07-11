// Command scripted-agent writes the scripted agent's canonical round envelopes
// to stdout as JSON lines. Pipe it into the stdio shell to run the walking
// skeleton end to end:
//
//	go run ./agent/scripted/main | go run ./shell/stdio
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/RiccardoCereghino/xenomancer/agent/scripted"
)

func main() {
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()
	enc := json.NewEncoder(out)
	for _, sub := range scripted.Script() {
		if err := enc.Encode(sub); err != nil {
			fmt.Fprintf(os.Stderr, "scripted-agent: encode: %v\n", err)
			os.Exit(1)
		}
	}
}
