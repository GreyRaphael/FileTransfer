package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"os"
	"unsafe"

	"github.com/mholt/archiver/v3"
	"golang.org/x/sys/windows"
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

const shmName = "my_shm"

func processWriter(path string) {
	// Implementation for writer: transfer file to shared memory, etc.
	info, _ := os.Stat(path)
	if info.IsDir() {
		log.Println("Writer processing directory: ", path)
	} else {
		log.Println("Writer processing single file: ", path)
	}

	// archive file or directory to tmp.tar
	tmpFile := "tmp.tar"
	if err := archiver.Archive([]string{path}, tmpFile); err != nil {
		log.Fatal("Archive error:", err)
	}
	// tmp.tar to shm
	data, _ := os.ReadFile(tmpFile)
	dataSize := uint64(len(data))
	totalSize := dataSize + 8 // extra 8 bytes to store file size header

	// Create the file mapping.
	hMap, err := windows.CreateFileMapping(
		windows.InvalidHandle, // use system paging file
		nil,
		windows.PAGE_READWRITE,
		uint32(totalSize>>32),
		uint32(totalSize&0xffffffff),
		windows.StringToUTF16Ptr(shmName),
	)
	if err != nil {
		log.Fatal("CreateFileMapping failed:", err)
	}
	defer windows.CloseHandle(hMap)

	// Map the view of the file mapping.
	addr, err := windows.MapViewOfFile(hMap, windows.FILE_MAP_WRITE, 0, 0, uintptr(totalSize))
	if err != nil {
		log.Fatal("MapViewOfFile failed:", err)
	}
	defer windows.UnmapViewOfFile(addr)

	// Create a byte slice backed by the shared memory.
	mem := unsafe.Slice((*byte)(unsafe.Pointer(addr)), totalSize)

	// Write the file size into the first 8 bytes (little-endian).
	binary.LittleEndian.PutUint64(mem[:8], dataSize)

	// Copy the file data into shared memory after the header.
	copy(mem[8:], data)

	log.Println("Data copied to shared memory: ", shmName)
	log.Println("Keep this program running until another side has finished reading the data.")
	log.Println("Press Enter to exit...")
	fmt.Scanln() // wait for user input before exiting
	os.Remove(tmpFile)

}

func processReader() {
	// Implementation for reader: transfer from shared memory to file, etc.
	log.Println("Reader read from shared memory: ", shmName)
	// open shared memory
	hMap, err := OpenFileMapping(shmName)
	if err != nil {
		log.Fatal("OpenFileMapping failed:", err)
	}
	defer windows.CloseHandle(hMap)

	// Map header to read file size
	const headerSize = 8
	addr, err := windows.MapViewOfFile(hMap, windows.FILE_MAP_READ, 0, 0, headerSize)
	if err != nil {
		log.Fatal("MapViewOfFile (header) failed:", err)
	}
	fileSize := *(*uint64)(unsafe.Pointer(addr))
	windows.UnmapViewOfFile(addr)

	// Map entire region including data
	totalSize := uintptr(fileSize) + headerSize
	addr, err = windows.MapViewOfFile(hMap, windows.FILE_MAP_READ, 0, 0, totalSize)
	if err != nil {
		log.Fatal("MapViewOfFile (full) failed:", err)
	}
	defer windows.UnmapViewOfFile(addr)

	// Access file data directly after header
	fileData := unsafe.Slice((*byte)(unsafe.Pointer(addr+headerSize)), fileSize)

	// Write to output file
	outFile := "new_tmp.tar"
	if err := os.WriteFile(outFile, fileData, 0644); err != nil {
		log.Fatal("Failed to write new_tmp.tar:", err)
	}
	// Unarchive outFile into the "output" directory
	unzipArchiver := archiver.Tar{OverwriteExisting: true}
	err = unzipArchiver.Unarchive(outFile, "output")
	if err != nil {
		log.Fatal("Error unzipping file:", err)
	}
	os.Remove(outFile)
	log.Println("Successfully shm & unzipped into ouput")
}

// OpenFileMapping wraps the Windows API OpenFileMappingW call.
func OpenFileMapping(name string) (windows.Handle, error) {
	proc := windows.NewLazySystemDLL("kernel32.dll").NewProc("OpenFileMappingW")

	ptr := windows.StringToUTF16Ptr(name)

	r, _, err := proc.Call(windows.FILE_MAP_READ, 0, uintptr(unsafe.Pointer(ptr)))
	if r == 0 {
		return 0, fmt.Errorf("OpenFileMappingW failed: %w", err)
	}
	return windows.Handle(r), nil
}
