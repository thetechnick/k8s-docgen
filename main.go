package main

import (
	"context"
	"flag"
	"os"
)

func main() {
	templateFile := flag.String("template", "", "Go template for the documentation.")
	flag.Parse()

	var opts []Option
	if templateFile != nil && *templateFile != "" {
		opts = append(opts, TemplateFile(*templateFile))
	}
	d := NewDocgen(opts...)

	ctx := context.Background()
	if err := d.Parse(ctx, flag.Arg(0), os.Stdout); err != nil {
		panic(err)
	}
}
