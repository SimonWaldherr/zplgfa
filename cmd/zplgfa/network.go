package main

import (
	"fmt"
	"net"
)

func sendDataToZebra(ip, port, str string) error {
	tcpAddr, err := net.ResolveTCPAddr("tcp", ip+":"+port)
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
