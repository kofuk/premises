package monitor

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"chronoscoper.com/premises/config"
)

const (
	StatusPayloadVerifyVersion  = byte('v')
	StatusPayloadSubscribe      = byte('s')
	StatusPayloadResult         = byte('R')
	StatusPayloadStatus         = byte('S')
	StatusPayloadStop           = byte('t')
	StatusPayloadServerFinished = byte('F')
)

func readStatus(conn io.Reader) (bool, error) {
	var buf [4]byte
	_, err := io.ReadFull(conn, buf[:3])
	if err != nil {
		return false, err
	}
	if buf[0] != StatusPayloadResult {
		return false, errors.New("Not a result payload")
	}
	dataLength := binary.LittleEndian.Uint16(buf[1:])
	if dataLength != 1 {
		return false, errors.New("Invalid length")
	}

	_, err = io.ReadFull(conn, buf[:1])
	if err != nil {
		return false, err
	}

	return buf[0] == 1, nil
}

const (
	RequestStopServer = iota
)

func MonitorServer(config *config.Config, addr string, evCh chan string, reqCh chan int) error {
	rootCAs := x509.NewCertPool()
	certFile, err := os.ReadFile(filepath.Join(config.Prefix, "/opt/premises/server.crt"))
	if err != nil {
		return err
	}
	rootCAs.AppendCertsFromPEM(certFile)

	tlsConfig := &tls.Config{
		RootCAs: rootCAs,
	}

	go func() {
		lastConnected := int(time.Now().Unix())
		evCh <- "Connecting..."
		for {
			conn, err := tls.Dial("tcp", addr, tlsConfig)
			if err != nil {
				log.Println(err)
				evCh <- "Connection failed"

				if int(time.Now().Unix())-lastConnected > int(time.Minute*10) {
					// No connection for 10 minutes, server's died?
					close(evCh)
					break
				}

				time.Sleep(time.Second * 10)
				evCh <- "Connecting..."
				continue
			}
			defer conn.Close()
			lastConnected = int(time.Now().Unix())

			var buf [7]byte
			buf[0] = StatusPayloadVerifyVersion
			binary.LittleEndian.PutUint16(buf[1:], uint16(4))
			binary.LittleEndian.PutUint32(buf[3:], uint32(1))

			if _, err = conn.Write(buf[:]); err != nil {
				log.Println(err)
				continue
			}

			ok, err := readStatus(conn)
			if err != nil {
				log.Println(err)
				continue
			}
			if !ok {
				log.Println("Version verification failed")
				evCh <- "Can't monitor! Version is not match."
				break
			}
			log.Println("Version verified")

			buf[0] = StatusPayloadSubscribe
			binary.LittleEndian.PutUint16(buf[1:], uint16(len([]byte(config.MonitorKey))))
			if _, err := conn.Write(buf[0:3]); err != nil {
				log.Println(err)
				continue
			}

			if _, err := conn.Write([]byte(config.MonitorKey)); err != nil {
				log.Println(err)
				continue
			}
			ok, err = readStatus(conn)
			if err != nil {
				log.Println(err)
				continue
			}
			if !ok {
				log.Println("Event subscription failed")
				continue
			}
			log.Println("Event subscribed successfully")

			inEventChannel := make(chan string)
			go func() {
				for {
					var buf [3]byte
					if _, err := io.ReadFull(conn, buf[:]); err != nil && err == io.EOF {
						break
					} else if err != nil {
						log.Println(err)
						continue
					}

					dataLength := binary.LittleEndian.Uint16(buf[1:])

					if buf[0] == StatusPayloadServerFinished {
						log.Println("Server finished")
						close(evCh)
						close(inEventChannel)
						break
					} else if buf[0] == StatusPayloadStatus {
						data := make([]byte, dataLength)
						if _, err := io.ReadFull(conn, data); err != nil {
							log.Println(err)
							continue
						}

						inEventChannel <- string(data)
					} else {
						log.Println("Payload is not status")
						buf := make([]byte, dataLength)
						io.ReadFull(conn, buf)
						continue
					}
				}
			}()

			for {
				select {
				case req := <-reqCh:
					if req == RequestStopServer {
						var buf [3]byte
						buf[0] = StatusPayloadStop
						if _, err := conn.Write(buf[:]); err != nil {
							log.Println(err)
							continue
						}
					}
					break

				case ev, ok := <-inEventChannel:
					if !ok {
						goto finish
					}
					evCh <- ev
					break
				}
			}
		}
	finish:
	}()
	return nil
}
