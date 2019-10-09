package main

import (
    "bytes"
    "compress/zlib"
    "encoding/binary"
    "flag"
    "fmt"
    "io"
    "io/ioutil"
    "os"

    "../../dfversions"
)

func readUInt32(f *os.File) (ret uint32, err error) {
    err = binary.Read(f, binary.LittleEndian, &ret)
    return
}

func bytesRemaining (f *os.File) (ret uint64) {
    orig, _ := f.Seek(0, 1)
    end, _ := f.Seek(0, 2)
    ret = uint64(end - orig)
    f.Seek(orig, 0)
    return
}

func main() {
    var quiet bool
    flag.BoolVar(&quiet, "q", false, "Display only error messages")
    flag.Parse()
    if len(os.Args) == 1 {
        fmt.Printf("%s: No save file(s) given\n", os.Args[0])
        return
    }
    for _, file := range flag.Args() {
        f, err := os.Open(file)
        if err != nil {
            fmt.Printf("%s: Not found\n", file)
            continue
        }

        save_version, err := readUInt32(f)
        if err != nil {
            fmt.Println(err)
            continue
        }
        save_desc := dfversions.Describe(save_version)
        if !quiet {
            if dfversions.IsKnown(save_version) {
                fmt.Printf("%s: Save version %s (%d)\n", file, save_desc, save_version)
            } else {
                fmt.Printf("%s: Unknown save version: %s (%d)\n", file, save_desc, save_version)
            }
        }

        compressed, err := readUInt32(f)
        if err != nil {
            fmt.Println(err)
            continue
        }
        if compressed != 1 {
            fmt.Printf("%s: Not compressed\n", file)
            continue
        }

        // Attempt to decompress
        ok := false
        chunk := 1
        bytes_read := 8
        for ;; chunk++ {
            start := bytes_read
            length, err := readUInt32(f)
            bytes_read += 4
            if err == io.EOF {
                ok = true
                break
            }
            if err != nil {
                fmt.Printf("%s: chunk %d: %s\n", file, chunk, err)
                break
            }
            mbytes := length / (1024 * 1024)
            if mbytes > 10 {
                fmt.Printf("%s: chunk %d: Memory threshold exceeded: tried to read %d MB\n", file, chunk, mbytes)
                break
            }

            buf := make([]byte, length)
            n, err := f.Read(buf)
            bytes_read += n
            if err != nil {
                fmt.Printf("%s: chunk %d: %s\n", file, chunk, err)
                break
            }
            if n != int(length) {
                fmt.Printf("%s: chunk %d (start=%d): Expected %d bytes, got %d\n", file, chunk, start, length, n)
                break
            }

            reader, err := zlib.NewReader(bytes.NewReader(buf))
            if err != nil {
                fmt.Printf("%s: chunk %d (start=%d, length=%d): Read failed: %s\n", file, chunk, start, n, err)
                break
            }
            _, err = io.Copy(ioutil.Discard, reader)
            if err != nil {
                fmt.Printf("%s: chunk %d (start=%d, length=%d): Compression error: %s\n", file, chunk, start, n, err)
                break
            }
        }
        if ok {
            if !quiet {
                fmt.Printf("%s: No compression errors detected (%d chunks)\n", file, chunk)
            }
        } else {
            fmt.Printf("%s: %d bytes unread, %d bytes read\n", file, bytesRemaining(f), bytes_read)
        }
    }
}