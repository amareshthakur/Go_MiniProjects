# CLI File Shredder (`FileShadder`)

A secure, multi-pass file shredder built in Go using the standard `os`, `io`, and `crypto/rand` packages.

---

## 1. Overview & Purpose

When you delete a file using traditional means (such as moving it to the Trash/Recycle Bin or using `os.Remove`), the operating system **does not erase the actual data bytes** from the storage medium. Instead, it merely removes the filesystem's pointer (directory entry/inode) to that data and marks those disk sectors as "available for reuse." Until new data overwrites those sectors, data recovery tools can easily reconstruct the original content.

`FileShadder` solves this problem by performing **cryptographic multi-pass in-place overwrites**:
1. It measures the exact byte length of the target file.
2. It overwrites every byte of the file 3 times with cryptographically secure pseudo-random bytes.
3. It forces the operating system to flush internal memory buffers to physical storage after each pass.
4. Only after the content has been completely randomized does it remove the file entry from the filesystem.

---

## 2. Deep Dive: Core Concepts & Implementation

### 2.1 File Inspection & Checking for Regular Files
```go
info, err := os.Lstat(path)
if !info.Mode().IsRegular() {
    return fmt.Errorf("not a regular file")
}
```

* **`os.Lstat(path)` vs `os.Stat(path)`**:
  `os.Lstat` retrieves file metadata without following symbolic links. If the target path is a symlink, `os.Lstat` returns information about the symlink itself rather than the file it points to.
* **What is a Regular File?**
  A **regular file** contains contiguous user data on a storage volume (e.g., text documents, images, PDFs, compiled binaries, `.dat`, or `.dll` files).
* **What is NOT a Regular File?**
  In operating systems (especially Unix-like environments), many abstractions are represented as files:
  - **Directories**: Contain listings of filenames, not raw data streams.
  - **Symbolic Links (Symlinks)**: Pointers to another file location.
  - **Named Pipes (FIFOs) & Sockets**: Inter-process communication channels.
  - **Device Nodes (`/dev/*`)**: Block/character devices mapping directly to hardware or disk partitions.
* **Why this check is mandatory**: Attempting to shred a directory, device, or pipe could cause system lockups, partition corruption, or unintended file destruction. `info.Mode().IsRegular()` ensures we only operate on standard, standalone data files.

---

### 2.2 Opening the File for In-Place Overwriting
```go
file, err := os.OpenFile(path, os.O_WRONLY, 0)
```

* **`os.O_WRONLY`**: Opens the file with write permissions only.
* **Why NOT `os.O_TRUNC`?**
  `os.O_TRUNC` immediately truncates the file size to 0 bytes upon opening, destroying the original size reference and preventing in-place sector overwriting.
* **Why NOT `os.O_APPEND`?**
  `os.O_APPEND` forces every write to be added to the end of the file rather than replacing existing bytes at specific offsets.
* **Why Permission Mode `0`?**
  The third argument of `os.OpenFile` defines the file mode/permissions (e.g., `0644`) when creating a file with `os.O_CREATE`. Because we are opening an **existing** file and not creating a new one, the permission mask is ignored, so passing `0` is standard practice.

---

### 2.3 Measuring File Size & Repositioning with `Seek`
```go
// Measure file size
size, err := file.Seek(0, io.SeekEnd)

// Rewind cursor before each pass
_, err := file.Seek(0, io.SeekStart)
```

* **`file.Seek(offset, whence)`**: Moves the read/write cursor of the file:
  - `io.SeekEnd` with offset `0`: Moves the cursor to the very end of the file. The returned offset from the beginning corresponds to the total file size in bytes.
  - `io.SeekStart` with offset `0`: Rewinds the cursor back to byte `0` so the subsequent write pass overwrites the file from the start.

---

### 2.4 Generating Cryptographic Randomness (`crypto/rand`)
```go
writen, err := io.CopyN(file, rand.Reader, size)
```

* **`rand.Reader`**: A global, cryptographically secure pseudo-random number generator (CSPRNG) backed by operating system entropy sources (`/dev/urandom` on Unix/Linux/macOS, `CryptGenRandom` / `BCryptGenRandom` on Windows).
* **`io.CopyN(dst, src, n)`**: Copies exactly `size` bytes from `rand.Reader` into the target file, replacing the previous data byte-for-byte with random noise.

---

### 2.5 Flushing Caches to Disk (`file.Sync()`)
```go
if err := file.Sync(); err != nil { ... }
```

* When write operations take place, modern operating systems do not immediately write every byte to physical media; they buffer writes in RAM (the OS page cache) for performance.
* `file.Sync()` commits the file's in-memory copy to storage (equivalent to the `fsync` system call). This guarantees that each pass of random data is physically written to disk before the next pass or deletion starts.

---

### 2.6 File Deletion (`os.Remove`)
```go
if err := file.Close(); err != nil { ... }
if err := os.Remove(path); err != nil { ... }
```

* The file handle is cleanly closed before invoking `os.Remove`.
* On Windows, an open file handle prevents deletion (`sharing violation`), making `file.Close()` necessary before `os.Remove`.
* `os.Remove` unlinks the filename from the directory structure. Because all underlying bytes have already been randomized across 3 passes, the remaining raw sectors contain only useless entropy.

---

## 3. Interactive CLI Flow & User Input Safety

The program implements an interactive terminal confirmation flow:

1. **Buffered Input (`bufio.NewReader`)**:
   Reads user input line-by-line from standard input (`os.Stdin`) up to the newline character (`'\n'`).
2. **Whitespace Trimming (`strings.TrimSpace`)**:
   Strips extraneous spaces and carriage return (`\r`) / newline (`\n`) characters added across Windows and Unix terminals.
3. **Explicit Confirmation**:
   The user must type the exact word `SHRED` in capital letters before any irreversible destructive operations begin.

---

## 4. Modern Storage Realities & Limitations

While multi-pass overwriting works reliably on traditional mechanical hard drives (HDDs):

1. **Solid State Drives (SSDs) & NVMe**:
   Flash storage devices use **Wear Leveling** and **Flash Translation Layers (FTL)**. Writing to the same logical block address (LBA) might write to a different physical flash cell to distribute wear evenly.
2. **Copy-on-Write (CoW) Filesystems**:
   Filesystems like Btrfs, ZFS, and APFS do not overwrite data in-place; they write modifications to new blocks and update tree pointers.
3. **Recommendation**:
   For maximum security on modern SSDs, full-disk encryption (BitLocker, LUKS, FileVault) or firmware-level ATA/NVMe Secure Erase should be used in conjunction with file shredding.

---

## 5. How to Run

### Run directly:
```bash
go run main.go
```

### Build as a standalone binary:
```bash
go build -o fileshredder.exe main.go
./fileshredder.exe
```
