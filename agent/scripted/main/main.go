// Command scripted-agent runs the scripted agent as a bidirectional subprocess
// under the cmd/run harness. It speaks the JSONL wire protocol on stdio: before
// each round it reads one observation packet from stdin (and discards it — the
// scripted agent is fully determined and ignores observations, GDD §3), then
// writes that round's canonical envelope to stdout.
//
// Being lockstep, it is driven by the harness, which sends the opening packet
// first:
//
//	go run ./cmd/run --agent "go run ./agent/scripted/main"
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/RiccardoCereghino/xenomancer/agent/scripted"
)

func main() {
	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()
	enc := json.NewEncoder(out)

	for _, sub := range scripted.Script() {
		// Wait for the harness's observation packet, then act. The packet is
		// discarded: the script is fixed. If stdin closes early, stop.
		if !in.Scan() {
			if err := in.Err(); err != nil {
				fmt.Fprintf(os.Stderr, "scripted-agent: read packet: %v\n", err)
				os.Exit(1)
			}
			return
		}
		if err := enc.Encode(sub); err != nil {
			fmt.Fprintf(os.Stderr, "scripted-agent: encode: %v\n", err)
			os.Exit(1)
		}
		if err := out.Flush(); err != nil {
			fmt.Fprintf(os.Stderr, "scripted-agent: flush: %v\n", err)
			os.Exit(1)
		}
	}
}
