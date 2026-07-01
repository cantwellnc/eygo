package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/cantwellnc/eygo/builtin"
	"github.com/cantwellnc/eygo/eval"
	"github.com/cantwellnc/eygo/ir"
	"github.com/cantwellnc/eygo/runtime"
)

func main() {
	inspect := flag.Bool("inspect", false, "print effect labels in the program and exit")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: eygo [flags] <file.eyg.json>\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	data, err := os.ReadFile(flag.Arg(0))
	if err != nil {
		fatalf("read: %v", err)
	}

	node, err := ir.Decode(data)
	if err != nil {
		fatalf("decode: %v", err)
	}

	if *inspect {
		labels := runtime.Inspect(node)
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(map[string]any{"effects": labels})
		return
	}

	interp := eval.New()
	builtin.Register(interp)
	builtin.RegisterFix(interp, interp.Apply)

	rt := runtime.New(interp, runtime.StdHandlers()...)

	val, err := rt.Run(node)
	if err != nil {
		fatalf("run: %v", err)
	}

	fmt.Println(eval.Sprint(val))
}

func fatalf(f string, a ...any) {
	fmt.Fprintf(os.Stderr, "eygo: "+f+"\n", a...)
	os.Exit(1)
}
