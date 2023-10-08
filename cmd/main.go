package main

import (
	"crypto/sha256"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)


var (
	interactive = flag.Bool("i", false, "Interactive mode, prompt for inputs.")
	output = flag.String("o", "", "where to write converted file(s)")
	verbose     = flag.Bool("v", false, "Verbose.")
	helpFlag       = flag.Bool("help", false, "show detailed help message")
	version = "devel"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: file2fuzz [-o output] [input...]\nconverts files to Go fuzzer corpus format\n")
	fmt.Fprintf(os.Stderr, "\tinput: files to convert\n")
	fmt.Fprintf(os.Stderr, "\t-o: where to write converted file(s)\n")
	os.Exit(2)
			fmt.Fprintf(os.Stderr, "usage: %s old.txt new.txt\n\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprint(os.Stderr, usageFooter)
		os.Exit(2)
}

func verbosef(format string, v ...interface{}) {
	if !*verbose {
		return
	}

	fmt.Printf(format+"\n", v...)
}

func main() {
	if err := doMain(); err != nil {
		fmt.Fprintf(os.Stderr, "eg: %s\n", err)
		os.Exit(1)
	}
}


func doMain() error {
	log.SetFlags(0)
	log.SetPrefix("bibsin: ")

	flag.Usage = usage
	flag.Parse()
	if *helpFlag {
		help := eg.Help // hide %s from vet
		fmt.Fprint(os.Stderr, help)
		os.Exit(2)
	}
	// if *goVersion != "" && !strings.HasPrefix(*goVersion, "go") {
	// 	*goVersion = "go" + *goVersion
	// }

	// ctx := context.Background()

	verbosef("version " + version)

	if err := process(flag.Args(), *output); err != nil {
		log.Fatal(err)
	}
}	


func process(inputArgs []string, outputArg string) error {
	var input []io.Reader
	if args := inputArgs; len(args) == 0 {
		input = []io.Reader{os.Stdin}
	} else {
		for _, a := range args {
			f, err := os.Open(a)
			if err != nil {
				return fmt.Errorf("unable to open %q: %s", a, err)
			}
			defer f.Close()
			if fi, err := f.Stat(); err != nil {
				return fmt.Errorf("unable to open %q: %s", a, err)
			} else if fi.IsDir() {
				return fmt.Errorf("%q is a directory, not a file", a)
			}
			input = append(input, f)
		}
	}

	var output func([]byte) error
	if outputArg == "" {
		if len(inputArgs) > 1 {
			return errors.New("-o required with multiple input files")
		}
		output = func(b []byte) error {
			_, err := os.Stdout.Write(b)
			return err
		}
	} else {
		if len(inputArgs) > 1 {
			output = dirWriter(outputArg)
		} else {
			if fi, err := os.Stat(outputArg); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("unable to open %q for writing: %s", outputArg, err)
			} else if err == nil && fi.IsDir() {
				output = dirWriter(outputArg)
			} else {
				output = func(b []byte) error {
					return os.WriteFile(outputArg, b, 0666)
				}
			}
		}
	}

	for _, f := range input {
		b, err := io.ReadAll(f)
		if err != nil {
			return fmt.Errorf("unable to read input: %s", err)
		}
		if err := output(encodeByteSlice(b)); err != nil {
			return fmt.Errorf("unable to write output: %s", err)
		}
	}

	return nil
}


func dirWriter(dir string) func([]byte) error {
	return func(b []byte) error {
		sum := fmt.Sprintf("%x", sha256.Sum256(b))
		name := filepath.Join(dir, sum)
		if err := os.MkdirAll(dir, 0777); err != nil {
			return err
		}
		if err := os.WriteFile(name, b, 0666); err != nil {
			os.Remove(name)
			return err
		}
		return nil
	}
}

func encodeByteSlice(b []byte) []byte {
	return []byte(fmt.Sprintf("%s\n[]byte(%q)", encVersion1, b))
}
