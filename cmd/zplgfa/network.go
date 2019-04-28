package main

import (
	"bufio"
	"fmt"
	"net"
	"time"
)

func sendDataToZebra(ip, port, str string) error {
	tcpAddr, err := net.ResolveTCPAddr("tcp", ip+":"+port)
	if err != nil {
		return err
	}
	conn, err := net.DialTCP("tcp4", nil, tcpAddr)
	if err == nil {
		defer conn.Close()

		payloadBytes := []byte(fmt.Sprintf("%s\r\n\r\n", str))
		_, err = conn.Write(payloadBytes)
		return err
	}
	return err
}

func sendFeedCmdToZebra(ip, port string) error {
	return sendDataToZebra(ip, port, "^xa^aa^fd ^fs^xz")
}

func sendCalibCmdToZebra(ip, port string) error {
	return sendDataToZebra(ip, port, "~jc^xa^jus^xz")
}

func sendCancelCmdToZebra(ip, port string) error {
	return sendDataToZebra(ip, port, "~ja")
}

func getInfoFromZebra(ip, port string) (string, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", ip+":"+port)
	if err != nil {
		return "", err
	}
	conn, err := net.DialTCP("tcp4", nil, tcpAddr)
	if err == nil {
		defer conn.Close()

		reader := bufio.NewReader(conn)

		conn.Write([]byte(fmt.Sprintf("%s\r\n\r\n", "~HI")))

		message0, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}

		conn.Write([]byte(fmt.Sprintf("%s\r\n\r\n", "~HS")))

		message1, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		message2, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		message3, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		return fmt.Sprint(message0, message1, message2, message3), err
	}
	return "", err
}

func getTerminalOutputFromZebra(ip, port, cmd string) (string, error) {
	var config string
	var lastInput time.Time
	tcpAddr, err := net.ResolveTCPAddr("tcp", ip+":"+port)
	if err != nil {
		return "", err
	}
	conn, err := net.DialTCP("tcp4", nil, tcpAddr)
	if err == nil {
		defer conn.Close()

		conn.Write([]byte(fmt.Sprintf("%s\r\n\r\n", cmd)))
		scanner := bufio.NewScanner(conn)
		ticker := time.NewTicker(300 * time.Millisecond)
		input := make(chan string)
		go func(scanner *bufio.Scanner, input chan string) {
			for scanner.Scan() {
				input <- scanner.Text()
			}
		}(scanner, input)

		for {
			select {
			case i := <-input:
				config += fmt.Sprintln(i)
				lastInput = time.Now()
			case <-ticker.C:
				if time.Since(lastInput) > time.Duration(50*time.Millisecond) {
					return config, nil
				}
			}
		}
	}
	return "", err
}

func getConfigFromZebra(ip, port string) (string, error) {
	return getTerminalOutputFromZebra(ip, port, "^XA^HH^XZ")
}

func getDiagFromZebra(ip, port string) (string, error) {
	return getTerminalOutputFromZebra(ip, port, "~HD")
}
