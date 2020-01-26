//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package main

import (
	"fmt"

	"github.com/ezrec/uv3dp"
	_ "github.com/ezrec/uv3dp/cbddlp"
	_ "github.com/ezrec/uv3dp/sl1"

	"github.com/spf13/pflag"
)

const (
	defaultCachedLayers = 64
)

type Verbosity int

const (
	VerbosityWarning = Verbosity(iota)
	VerbosityNotice
	VerbosityInfo
	VerbosityDebug
)

var param struct {
	Verbose int // Verbose counts the number of '-v' flags
}

func TraceVerbosef(level Verbosity, format string, args ...interface{}) {
	if param.Verbose >= int(level) {
		fmt.Printf("<%v>", level)
		fmt.Printf(format+"\n", args...)
	}
}

type Commander interface {
	Parse(args []string) error
	Args() []string
	NArg() int
	PrintDefaults()
	Filter(input uv3dp.Printable) (output uv3dp.Printable, err error)
}

var commandMap = map[string]struct {
	NewCommander func() (cmd Commander)
	Description  string
}{
	"info": {
		NewCommander: func() Commander { return NewInfoCommand() },
		Description:  "Dumps information about the printable",
	},
	"decimate": {
		NewCommander: func() Commander { return NewDecimateCommand() },
		Description:  "Remove outmost pixels of all islands in each layer (reduces over-curing on edges)",
	},
	"exposure": {
		NewCommander: func() Commander { return NewExposureCommand() },
		Description:  "Alters exposure times",
	},
}

func Usage() {
	fmt.Println("Usage:")
	fmt.Println()
	fmt.Println("  uv3dp [options] INFILE [command [options] | OUTFILE]...")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println()
	pflag.PrintDefaults()
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println()
	fmt.Printf("  %-20s %s\n", "(none)", "Translates input file to output file")

	for key, item := range commandMap {
		fmt.Printf("  %-20s %s\n", key, item.Description)
	}

	for key, item := range commandMap {
		fmt.Println()
		fmt.Printf("Options for '%s':\n", key)
		fmt.Println()
		item.NewCommander().PrintDefaults()
	}

	uv3dp.FormatterUsage()
}

func init() {
	pflag.CountVarP(&param.Verbose, "verbose", "v", "Verbosity")
	pflag.SetInterspersed(false)
}

func evaluate(args []string) (err error) {
	if len(args) == 0 {
		Usage()
		return
	}

	var input uv3dp.Printable
	var format *uv3dp.Format

	for len(args) > 0 {
		if args[0] == "help" {
			Usage()
			return
		}

		item, found := commandMap[args[0]]
		if !found {
			format, err = uv3dp.NewFormat(args[0], args[1:])
			if err != nil {
				return err
			}
			err = format.Parse(args[1:])
			if err != nil {
				return err
			}
			TraceVerbosef(VerbosityNotice, "%v", args)
			args = format.Args()

			if input == nil {
				// If we have no input, get it from this file
				input, err = format.Printable()
				TraceVerbosef(VerbosityDebug, "%v: Input (err: %v)", format.Filename, err)
				if err != nil {
					return
				}

				// Cache layer decoding
				input = uv3dp.NewCachedPrintable(input, defaultCachedLayers)
			} else {
				// Otherwise save the file
				err = format.SetPrintable(input)
				TraceVerbosef(VerbosityDebug, "%v: Output (err: %v)", format.Filename, err)
				if err != nil {
					return
				}
			}
		} else {
			cmd := item.NewCommander()
			err = cmd.Parse(args[1:])
			if err != nil {
				return
			}
			TraceVerbosef(VerbosityNotice, "%v", args)
			args = cmd.Args()

			input, err = cmd.Filter(input)
			if err != nil {
				return
			}
		}
	}

	return
}

func main() {
	pflag.Parse()

	err := evaluate(pflag.Args())
	if err != nil {
		panic(err)
	}
}