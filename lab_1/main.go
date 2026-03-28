package main

import (
  "bytes"
  "encoding/binary"
  "flag"
  "fmt"
  "net"
  "os"
  "syscall"
  "time"
)

type ICMPHeader struct {
  Type     uint8  // 1 байт 
  Code     uint8  // 1 байт 
  Checksum uint16 // 2 байт 
  ID       uint16 // 2 байт 
  Seq      uint16 // 2 байт 
}

func calculateChecksum(data []byte) uint16 {
 
  var sum uint32

  for i := 0; i < len(data)-1; i += 2 {

    sum += uint32(binary.BigEndian.Uint16(data[i : i+2]))

  }

  if len(data)%2 != 0 {
    
    sum += uint32(data[len(data)-1]) << 8

  }

  for sum > 0xffff {

    sum = (sum >> 16) + (sum & 0xffff)

  }

  return uint16(^sum)
  
}

func main() {

  dnsFlag := flag.Bool("dns", false, "")

  flag.Parse()

  args := flag.Args()

  if len(args) < 1 {

    return

  }

  target := args[0]

  addrs, err := net.LookupIP(target)

  if err != nil {

    return
    
  }

  destIP := addrs[0]

  line := "+-------+---------+---------+---------+-----------------+------------+------------+------------------+----------------------------------------------+"
  fmt.Println(line)
  fmt.Printf("| %-5s | %-7s | %-7s | %-7s | %-15s | %-10s | %-10s | %-16s | %-44s |\n", "TTL_0", "RTT_1", "RTT_2", "RTT_3", "Address", "Checksum", "Sequence", "Status", "Host")
  fmt.Println(line)

  startTrace(destIP, *dnsFlag, line)
}

func startTrace(destIP net.IP, useDNS bool, line string) {

    fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_ICMP)
    
    if err != nil {

        return

    }

    defer syscall.Close(fd)

    tv := syscall.Timeval{Sec: 1, Usec: 500000}

    syscall.SetsockoptTimeval(fd, syscall.SOL_SOCKET, syscall.SO_RCVTIMEO, &tv)

    myID := uint16(os.Getpid() & 0xffff)

    displayIndex := 1

    for ttl := 1; ttl <= 30; ttl++ {

        syscall.SetsockoptInt(fd, syscall.IPPROTO_IP, syscall.IP_TTL, ttl)
        
        var rtts         [3]string
        var lastIP       string
        var lastChecksum uint16
        var lastSeq      uint16
        var reached      bool

        for i := 0; i < 3; i++ {
            
            seq := uint16(ttl*100 + i)

            start := time.Now()

            ip, icmpType, checksum, err := sendEcho(fd, destIP, myID, seq)

            duration := time.Since(start)

            if err != nil {

                rtts[i] = "*"

            } else {

                rtts[i] = fmt.Sprintf("%dms", duration.Milliseconds())

                lastIP = ip

                lastChecksum = checksum

                lastSeq = seq

                if icmpType == 0 { reached = true }

            }

        }

        if ttl == 1 {

            continue 

        }

        status := "Time Exceeded"

        if reached { 

          status = "Echo Reply" 

        }

        if lastIP == "" { 
          
          status = "Timed Out" 
        
        }

        host := ""

        if useDNS && lastIP != "" {

            names, _ := net.LookupAddr(lastIP) // Pointer Record - запрос

            if len(names) > 0 { 
              
              host = names[0] 
            
            }

        }

        beSeq := lastSeq

        leSeq := ((lastSeq & 0xFF) << 8) | ((lastSeq & 0xFF00) >> 8)

        wiresharkSeq := fmt.Sprintf("%d/%d", beSeq, leSeq)

        fmt.Printf("| %-5d | %-7s | %-7s | %-7s | %-15s | 0x%04x     | %-10s | %-16s | %-44s |\n", displayIndex, rtts[0], rtts[1], rtts[2], lastIP, lastChecksum, wiresharkSeq, status, host)
        
        fmt.Println(line)

        displayIndex++

        if reached { break }

    }

}

func sendEcho(fd int, destIP net.IP, id, seq uint16) (string, uint8, uint16, error) {

  header := ICMPHeader{Type: 8, Code: 0, ID: id, Seq: seq}

  var buf bytes.Buffer

  binary.Write(&buf, binary.BigEndian, header)

  header.Checksum = calculateChecksum(buf.Bytes())

  finalChecksum := header.Checksum

  buf.Reset()

  binary.Write(&buf, binary.BigEndian, header)

  dst := &syscall.SockaddrInet4{Port: 0}

  copy(dst.Addr[:], destIP.To4()) 

  // Вызов Sendto отправляет байты через дескриптор сокета fd
  if err := syscall.Sendto(fd, buf.Bytes(), 0, dst); err != nil {

    return "", 0, 0, err

  }

  reply := make([]byte, 1500)

  // Ждем получения данных
  _, from, err := syscall.Recvfrom(fd, reply, 0)

  if err != nil {

    return "", 0, 0, err

  }

  // Достаем IP адрес отправителя из структуры 'from'.
  nodeIP := net.IP(from.(*syscall.SockaddrInet4).Addr[:]).String()

  return nodeIP, reply[20], finalChecksum, nil

}