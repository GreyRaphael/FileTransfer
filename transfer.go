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

const (
	shmName     = "my_shm"
	headerSize  = 8
	outDir      = "output"
	tmpFileName = "tmp.tar"
	outFileName = "new_tmp.tar"
)

func main() {
	// Ensure at least one argument is provided (the subcommand)
	if len(os.Args) < 2 {
		log.Fatal("Usage: <w|r> [-i <input_file_or_directory>]")
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
		log.Fatal("Invalid subcommand. Use 'w' (writer) or 'r' (reader).")
	}
}

func processWriter(path string) {
	// Implementation for writer: transfer file to shared memory, etc.
	info, err := os.Stat(path)
	if err != nil {
		log.Fatalf("Failed to access input path: %v", err)
	}
	log.Printf("Processing %s: %s\n", map[bool]string{true: "directory", false: "file"}[info.IsDir()], path)

	// archive file or directory to tmp.tar
	if err := archiver.Archive([]string{path}, tmpFileName); err != nil {
		log.Fatal("Archive error:", err)
	}
	// tmp.tar to shm
	data, _ := os.ReadFile(tmpFileName)
	dataSize := uint64(len(data))
	totalSize := dataSize + headerSize // extra 8 bytes to store file size header

	hMap, addr, err := createSharedMemory(totalSize)
	if err != nil {
		log.Fatalf("Shared memory error: %v", err)
	}
	defer cleanupSharedMemory(hMap, addr)

	mem := unsafe.Slice((*byte)(unsafe.Pointer(addr)), totalSize)
	binary.LittleEndian.PutUint64(mem[:headerSize], dataSize)
	copy(mem[headerSize:], data)

	log.Printf("Data copied to shared memory: %s\n", shmName)
	log.Println("Keep this program running until the reader has finished.")
	log.Println("Press Enter to exit...")
	fmt.Scanln()

	os.Remove(tmpFileName)
}

func createSharedMemory(size uint64) (windows.Handle, uintptr, error) {
	hMap, err := windows.CreateFileMapping(windows.InvalidHandle, nil, windows.PAGE_READWRITE,
		uint32(size>>32), uint32(size&0xffffffff), windows.StringToUTF16Ptr(shmName))
	if err != nil {
		return 0, 0, fmt.Errorf("CreateFileMapping failed: %w", err)
	}

	addr, err := windows.MapViewOfFile(hMap, windows.FILE_MAP_WRITE, 0, 0, uintptr(size))
	if err != nil {
		windows.CloseHandle(hMap)
		return 0, 0, fmt.Errorf("MapViewOfFile failed: %w", err)
	}

	return hMap, addr, nil
}

func cleanupSharedMemory(hMap windows.Handle, addr uintptr) {
	windows.UnmapViewOfFile(addr)
	windows.CloseHandle(hMap)
}

func processReader() {
	log.Printf("Reading from shared memory: %s\n", shmName)

	hMap, addr, fileSize, err := openSharedMemory()
	if err != nil {
		log.Fatalf("Failed to read shared memory: %v", err)
	}
	defer cleanupSharedMemory(hMap, addr)

	fileData := unsafe.Slice((*byte)(unsafe.Pointer(addr+headerSize)), fileSize)
	if err := os.WriteFile(outFileName, fileData, 0644); err != nil {
		log.Fatalf("Failed to write extracted archive: %v", err)
	}

	unzipArchiver := archiver.Tar{OverwriteExisting: true}
	if err := unzipArchiver.Unarchive(outFileName, outDir); err != nil {
		log.Fatalf("Error extracting archive: %v", err)
	}

	os.Remove(outFileName)
	log.Printf("Successfully extracted to '%s'\n", outDir)
}

// OpenFileMapping wraps the Windows API OpenFileMappingW call.
func OpenFileMapping(name string) (windows.Handle, error) {
	proc := windows.NewLazySystemDLL("kernel32.dll").NewProc("OpenFileMappingW")
	r, _, err := proc.Call(windows.FILE_MAP_READ, 0, uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(name))))
	if r == 0 {
		return 0, fmt.Errorf("OpenFileMappingW failed: %w", err)
	}
	return windows.Handle(r), nil
}

func openSharedMemory() (windows.Handle, uintptr, uint64, error) {
	hMap, err := OpenFileMapping(shmName)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("OpenFileMapping failed: %w", err)
	}

	addr, err := windows.MapViewOfFile(hMap, windows.FILE_MAP_READ, 0, 0, headerSize)
	if err != nil {
		windows.CloseHandle(hMap)
		return 0, 0, 0, fmt.Errorf("MapViewOfFile (header) failed: %w", err)
	}

	fileSize := binary.LittleEndian.Uint64(unsafe.Slice((*byte)(unsafe.Pointer(addr)), headerSize))
	windows.UnmapViewOfFile(addr)

	addr, err = windows.MapViewOfFile(hMap, windows.FILE_MAP_READ, 0, 0, uintptr(fileSize+headerSize))
	if err != nil {
		windows.CloseHandle(hMap)
		return 0, 0, 0, fmt.Errorf("MapViewOfFile (full) failed: %w", err)
	}

	return hMap, addr, fileSize, nil
}
