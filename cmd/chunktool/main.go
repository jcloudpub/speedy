package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
)

const (
	opcodeWrite = 0
	opcodeRead  = 1

	opcodeCheckDisk = 11

	opcodeSetStatus = 20
	opcodeGetStatus = 21

	opcodeKillPdWr      = 30
	opcodeQueryIoStatus = 31
	opcodeQeuryDetails  = 32

	opcodeDumpChunk     = 40
)

/*
const (
	statusRW            =  0
	statusRO            =  1
	statusPreRO         =  2
)
*/

const (
	QueryDetailHdrSize = 68
)

type SpyClient struct {
	Conn net.Conn
	rb   *bufio.Reader
}

func NewSpyClient(addr string) (*SpyClient, error) {
	var err error

	c := new(SpyClient)

	if c.Conn, err = net.Dial("tcp", addr); err != nil {
		fmt.Printf("connect server failed\n")
		return nil, err
	}

	c.rb = bufio.NewReaderSize(c.Conn, 1024)

	return c, nil
}

func (c *SpyClient) upload(sid uint, fid uint64, data string) {
	output := new(bytes.Buffer)
	header := make([]byte, 6)

	binary.Write(output, binary.BigEndian, uint8(opcodeWrite))
	binary.Write(output, binary.BigEndian, uint32(len(data)+2+8))
	binary.Write(output, binary.BigEndian, uint16(sid))
	binary.Write(output, binary.BigEndian, uint64(fid))

	output.WriteString(data)

	_, err := c.Conn.Write(output.Bytes())

	if err != nil {
		fmt.Printf("write socket error %s\n", err.Error())
		return
	}

	if _, err := io.ReadFull(c.rb, header); err != nil {
		fmt.Printf("read socket error %s\n", err.Error())
		return
	}

	if header[0] == opcodeWrite && header[1] == 0 {
		fmt.Printf("upload file ok\n")
	} else {
		fmt.Printf("upload file failed, code = %d\n", header[1])
	}
}

func (c *SpyClient) download(sid uint, fid uint64, outFile string) {
	output := new(bytes.Buffer)
	header := make([]byte, 6)

	binary.Write(output, binary.BigEndian, uint8(opcodeRead))
	binary.Write(output, binary.BigEndian, uint32(2+8))
	binary.Write(output, binary.BigEndian, uint16(sid))
	binary.Write(output, binary.BigEndian, uint64(fid))

	_, err := c.Conn.Write(output.Bytes())

	if err != nil {
		fmt.Printf("write socket error %s\n", err.Error())
		return
	}

	if _, err := io.ReadFull(c.rb, header); err != nil {
		fmt.Printf("read socket error %s\n", err.Error())
		return
	}

	if header[0] != opcodeRead || header[1] != 0 {
		fmt.Printf("download file failed, code = %d\n", header[1])
		return
	}

	bodyLen := binary.BigEndian.Uint32(header[2:])
	data := make([]byte, bodyLen)

	if _, err := io.ReadFull(c.rb, data); err != nil {
		fmt.Printf("read socket error %s\n", err.Error())
		return
	}

	if len(outFile) > 0 {
		f, err := os.Create(outFile)
		if err != nil {
			fmt.Println(err)
			return
		}

		f.Write(data)
	} else {
		fmt.Printf("download file content: %s, len: %d\n", string(data), len(data))
		//		fmt.Printf("download file content: %x, len: %d\n", data, len(data))
	}
}

func (c *SpyClient) getStatus(s uint) {
	output := new(bytes.Buffer)

	fmt.Println("server id:", s)

	binary.Write(output, binary.BigEndian, uint8(opcodeGetStatus))
	binary.Write(output, binary.BigEndian, uint32(0))

	_, err := c.Conn.Write(output.Bytes())
	if err != nil {
		fmt.Println("write socket error:", err)
		return
	}

	header := make([]byte, 6)
	if _, err := io.ReadFull(c.rb, header); err != nil {
		fmt.Println("read socket header error:", err)
		return
	}

	if header[0] != opcodeGetStatus || header[1] != 0 {
		fmt.Println("get status failed:", header[1])
		return
	}

	bodyLen := binary.BigEndian.Uint32(header[2:])
	if bodyLen != 4 {
		fmt.Println("bodyLen is not 4")
		return
	}

	data := make([]byte, bodyLen)
	if _, err := io.ReadFull(c.rb, data); err != nil {
		fmt.Println("read socker error:", err)
		return
	}

	status := binary.BigEndian.Uint32(data)
	fmt.Println("get status:", status)
}

func (c *SpyClient) setStatus(s uint, status uint32) {
	output := new(bytes.Buffer)

	binary.Write(output, binary.BigEndian, uint8(opcodeSetStatus))
	binary.Write(output, binary.BigEndian, uint32(4))
	binary.Write(output, binary.BigEndian, uint32(status))

	_, err := c.Conn.Write(output.Bytes())
	if err != nil {
		fmt.Println("write socket error:", err)
		return
	}

	header := make([]byte, 6)
	if _, err := io.ReadFull(c.rb, header); err != nil {
		fmt.Println("read socket header error:", err)
		return
	}

	if header[0] != opcodeSetStatus || header[1] != 0 {
		fmt.Println("set status failed:", header[1])
		return
	}

	bodyLen := binary.BigEndian.Uint32(header[2:])
	if bodyLen != 0 {
		fmt.Println("bodyLen is not 0")
		return
	}

	fmt.Println("set status OK")
}

func (c *SpyClient) queryIoStatus(s uint) {
	output := new(bytes.Buffer)

	binary.Write(output, binary.BigEndian, uint8(opcodeQueryIoStatus))
	binary.Write(output, binary.BigEndian, uint32(0))

	_, err := c.Conn.Write(output.Bytes())
	if err != nil {
		fmt.Println("write socket error:", err)
		return
	}

	header := make([]byte, 6)
	if _, err := io.ReadFull(c.rb, header); err != nil {
		fmt.Println("read socket header error:", err)
		return
	}

	if header[0] != opcodeQueryIoStatus || header[1] != 0 {
		fmt.Println("get status failed:", header[1])
		return
	}

	bodyLen := binary.BigEndian.Uint32(header[2:])
	if bodyLen != 12 {
		fmt.Println("bodyLen is not 12")
		return
	}

	body := make([]byte, bodyLen)
	if _, err := io.ReadFull(c.rb, body); err != nil {
		fmt.Println("read socket header error:", err)
		return
	}

	pendingWrites := binary.BigEndian.Uint32(body)
	writtingCount := binary.BigEndian.Uint32(body[4:8])
	readingCount := binary.BigEndian.Uint32(body[8:12])

	fmt.Println("pendingWrites:", pendingWrites, "writingCount:", writtingCount, "readingCount:", readingCount)

	fmt.Println("query io status OK")
}

func (c *SpyClient) killPendingWrites(s uint) {
	output := new(bytes.Buffer)

	fmt.Println("server id:", s)

	binary.Write(output, binary.BigEndian, uint8(opcodeKillPdWr))
	binary.Write(output, binary.BigEndian, uint32(0))

	_, err := c.Conn.Write(output.Bytes())
	if err != nil {
		fmt.Println("write socket error:", err)
		return
	}

	header := make([]byte, 6)
	if _, err := io.ReadFull(c.rb, header); err != nil {
		fmt.Println("read socket header error:", err)
		return
	}

	if header[0] != opcodeKillPdWr || header[1] != 0 {
		fmt.Println("kill pending failed:", header[1])
		return
	}

	bodyLen := binary.BigEndian.Uint32(header[2:])
	if bodyLen != 0 {
		fmt.Println("bodyLen is not 0")
		return
	}

	fmt.Println("kill pending OK:")
}

func (c *SpyClient) dumpChunk(name string, target string) {
	if len(name) == 0 || len(target) == 0 {
		fmt.Println("chunk name and target name must be given")
		return
	}

	output := new(bytes.Buffer)
	
	binary.Write(output, binary.BigEndian, uint8(opcodeDumpChunk))
	binary.Write(output, binary.BigEndian, uint32(len(name)))

	output.WriteString(name)

	_, err := c.Conn.Write(output.Bytes())
	if err != nil {
		fmt.Println("write socket error:", err)
		return
	}

	header := make([]byte, 6)
	if _, err := io.ReadFull(c.rb, header); err != nil {
		fmt.Println("read socket header error:", err)
		return
	}

	if header[0] != opcodeDumpChunk || header[1] != 0 {
		fmt.Println("dump chunk failed:", header[0])
		return
	}

	bodyLen := binary.BigEndian.Uint32(header[2:])
	if bodyLen <= 0 {
		fmt.Println("dump chunk resp invalid, bodyLen = ", bodyLen)
		return
	}

	data := make([]byte, 4096)
	f, err := os.Create(target)

	if err != nil {
		fmt.Println("Error creating file\n")
		return
	}

	defer f.Close()

	for {
		n, err := c.Conn.Read(data)
		if err != nil {
			if err != io.EOF {
				fmt.Println("socket read error\n")
			}
			break;
		}

		_, err = f.Write(data[:n])

		if err != nil {
			fmt.Println("Error writing file\n")
			break;
		}
		
		bodyLen -= uint32(n)
		if bodyLen == 0 {
			break;
		}		
	}
}

func (c *SpyClient) checkDisk(s uint) {
	output := new(bytes.Buffer)

	fmt.Println("server id:", s)

	binary.Write(output, binary.BigEndian, uint8(opcodeCheckDisk))
	binary.Write(output, binary.BigEndian, uint32(0))

	_, err := c.Conn.Write(output.Bytes())
	if err != nil {
		fmt.Println("write socket error:", err)
		return
	}

	header := make([]byte, 6)
	if _, err := io.ReadFull(c.rb, header); err != nil {
		fmt.Println("read socket header error:", err)
		return
	}

	if header[0] != opcodeCheckDisk || header[1] != 0 {
		fmt.Println("check disk failed:", header[1])
		return
	}

	bodyLen := binary.BigEndian.Uint32(header[2:])
	if bodyLen != 0 {
		fmt.Println("bodyLen is not 0")
		return
	}

	fmt.Println("check disk OK")
}

func (c *SpyClient) queryDetails(s uint) {
	output := new(bytes.Buffer)

	//	fmt.Println("server id:", s)

	binary.Write(output, binary.BigEndian, uint8(opcodeQeuryDetails))
	binary.Write(output, binary.BigEndian, uint32(0))

	_, err := c.Conn.Write(output.Bytes())
	if err != nil {
		fmt.Println("write socket error:", err)
		return
	}

	header := make([]byte, 6)
	if _, err := io.ReadFull(c.rb, header); err != nil {
		fmt.Println("read socket header error:", err)
		return
	}

	if header[0] != opcodeQeuryDetails || header[1] != 0 {
		fmt.Println("query details failed:", header[1])
		return
	}

	bodyLen := binary.BigEndian.Uint32(header[2:])
	if bodyLen < QueryDetailHdrSize {
		fmt.Println("invalid bodylen:", bodyLen)
		return
	}

	infos := make([]byte, bodyLen)

	if _, err := io.ReadFull(c.rb, infos); err != nil {
		fmt.Printf("read socket error %s\n", err.Error())
		return
	}

	conn_count := binary.BigEndian.Uint32(infos[:4])
	reading_count := binary.BigEndian.Uint32(infos[4:8])
	writing_count := binary.BigEndian.Uint32(infos[8:12])
	pending_writes := binary.BigEndian.Uint32(infos[12:16])
	read_count := binary.BigEndian.Uint64(infos[16:24])
	write_count := binary.BigEndian.Uint64(infos[24:32])
	read_error := binary.BigEndian.Uint64(infos[32:40])
	write_error := binary.BigEndian.Uint64(infos[40:48])
	read_bytes := binary.BigEndian.Uint64(infos[48:56])
	write_bytes := binary.BigEndian.Uint64(infos[56:64])
	n_chunks := binary.BigEndian.Uint32(infos[64:68])

	var i uint32
	chunk_infos := make([]uint64, n_chunks)
	for i = 0; i < n_chunks; i++ {
		index := binary.BigEndian.Uint64(infos[QueryDetailHdrSize+i*16 : QueryDetailHdrSize+i*16+8])
		chunk_infos[index-1] = binary.BigEndian.Uint64(infos[QueryDetailHdrSize+i*16+8 : QueryDetailHdrSize+i*16+16])
	}

	files_count := binary.BigEndian.Uint64(infos[QueryDetailHdrSize+16*n_chunks:])

	fmt.Println("connection count:\t", conn_count, "\treading count:\t", reading_count,
		"\twriting count:\t", writing_count, "\tpending writes:\t", pending_writes)
	fmt.Println("total read count:\t", read_count, "\ttotal write count:\t", write_count,
		"\ttotal read error:\t", read_error, "\ttotal write error:\t", write_error)
	fmt.Println("total read bytes:\t", read_bytes, "\ttotal write bytes:\t", write_bytes)
	fmt.Println("total chunks:\t", n_chunks, "\ttotal files:\t", files_count)
	fmt.Println("=========================chunk avail space==============================")

	var total_space uint64
	total_space = 0
	for i, c := range chunk_infos {
		//		fmt.Println("chunk", i+1, "avail space:", c)

		fmt.Println("chunk", i+1, "avail space:\t", c/(1<<30), "G ", c%(1<<30)/(1<<20), "M ",
			c%(1<<20)/(1<<10), "K ", c%(1<<10), "bytes")

		total_space += c
	}

	fmt.Println("total avail space:", total_space/(1<<30), "G ", total_space%(1<<30)/(1<<20), "M ",
		total_space%(1<<20)/(1<<10), "K ", total_space%(1<<10), "bytes")

	fmt.Println("========================================================================")

	fmt.Println("query details OK")
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	var addr = flag.String("h", "127.0.0.1", "server addr")
	var port = flag.String("p", "9999", "server port")
	var sid = flag.Uint("s", 1, "server id")
	var fid = flag.Uint64("f", 1, "fid")
	var data = flag.String("d", "test_data", "file data")
	var cmd = flag.String("c", "upload", "command")
	var outFile = flag.String("o", "", "outFile")
	var chunkName = flag.String("n", "", "chunk name")
	var targetName = flag.String("t", "", "target name")

	flag.Parse()

	var serverAddr = *addr + ":" + *port

	client, err := NewSpyClient(serverAddr)
	if err != nil {
		fmt.Printf("client create failed\n")
		return
	}

	if *cmd == "upload" {
		client.upload(*sid, *fid, *data)
	} else if *cmd == "download" {
		client.download(*sid, *fid, *outFile)
		//	} else if *cmd == "getstatus" {
		//		fmt.Println("run get status")
		//		client.getStatus(*sid)
		//	} else if *cmd == "setstatus" {
		//		client.setStatus(*sid, uint32(*status))
	} else if *cmd == "queryiostatus" {
		client.queryIoStatus(*sid)
	} else if *cmd == "killpending" {
		client.killPendingWrites(*sid)
	} else if *cmd == "checkdisk" {
		client.checkDisk(*sid)
	} else if *cmd == "querydetail" {
		client.queryDetails(*sid)
	} else if *cmd == "dumpchunk" {
		client.dumpChunk(*chunkName, *targetName)
	}

	client.Conn.Close()
}
