package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	// Ensure at least one argument is provided (the subcommand)
	if len(os.Args) < 2 {
		fmt.Println("expected 'w' (writer) or 'r' (reader) subcommand")
		os.Exit(1)
	}

	// Switch on the first argument (the subcommand)
	switch os.Args[1] {
	case "w":
		writerCmd := flag.NewFlagSet("w", flag.ExitOnError)
		input := writerCmd.String("i", "", "the input single file or directory")
		writerCmd.Parse(os.Args[2:])

		// Check that the required input flag is provided
		if *input == "" {
			writerCmd.PrintDefaults()
			os.Exit(1)
		}

		// Call your process function for the writer
		processWriter(*input)

	case "r":
		readerCmd := flag.NewFlagSet("r", flag.ExitOnError)
		readerCmd.Parse(os.Args[2:])

		// Call your process function for the reader
		processReader()

	default:
		fmt.Println("expected 'w' (writer) or 'r' (reader) subcommand")
		os.Exit(1)
	}
}

func processWriter(input string) {
	// Implementation for writer: transfer file to shared memory, etc.
	fmt.Println("Writer processing file/directory:", input)
}

func processReader() {
	// Implementation for reader: transfer from shared memory to file, etc.
	fmt.Println("Reader processing from shared memory")
}
