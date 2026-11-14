package main

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"strings"
)

func sherdFile(path string) error {
	// check the file without a symbolic link
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("Info Error :-> ")
	}

	// only allow regular files
	if !info.Mode().IsRegular() {
		return fmt.Errorf("Not a regular file")
	}
	// open existing file for writing
	file, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("File Opening Error :-> %w", err)
	}
	// Get the orginal file size
	size, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("Getting file size error :-> %w", err)
	}
	// Perform three overwrite pass
	for pass := 0; pass < 3; pass++ {
		// Move file offset back to the beginning.
		_, err := file.Seek(0, io.SeekStart)
		if err != nil {
			file.Close()
			return fmt.Errorf("Cursor moving to start / Seeking to Begning pass %d Error %w:-> ", pass, err)
		}
		// Write exactly 'size' random bytes.
		writen, err := io.CopyN(file, rand.Reader, size)
		if err != nil {
			file.Close()
			return fmt.Errorf("Writing file with random bit pass %d error %w \n While Writing %d :-> ", pass, err, writen)
		}
		// Request that the operating system flush file changes.
		if err := file.Sync(); err != nil {
			file.Close()
			return fmt.Errorf("Flushing file \nPass :-> %d\nError :-> %w", pass, err)
		}
	}
	// Close the file before deleting it.
	if err := file.Close(); err != nil {
		return fmt.Errorf("File closing\nError :-> %w", err)
	}
	// Remove the file from the filesystem.
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("Deleting file\nError :-> %w", err)
	}
	fmt.Println("File removed successufully .")
	return nil
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter file path: ")
	path, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Input error : ", err)
		return
	}
	fmt.Println(path)
	path = strings.TrimSpace(path)
	fmt.Println(path)
	if path == "" {
		fmt.Println("File path cann't be empty")
		return
	}
	fmt.Printf("This will overwrite and delete:\n%s\n", path)
	fmt.Print("Type SHRED to continue: ")
	confirmation, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Input error : ", err)
		return
	}
	if strings.TrimSpace(confirmation) != "SHRED" {
		fmt.Println("Operation cancelled.")
		return
	}
	if err := sherdFile(path); err != nil {
		fmt.Println("Error:", err)
		return
	}

}
