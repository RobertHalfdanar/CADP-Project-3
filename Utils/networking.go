package Utils

import (
	"CADP-Project-3/Logger"
	"CADP-Project-3/Raft"
	"fmt"
	"google.golang.org/protobuf/proto"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// GetPortFromListener gets the port from the listener
// Way to get what port we are listening to because
// we let the OS find an available port
func GetPortFromListener(conn net.Listener) int32 {
	port := conn.Addr().(*net.TCPAddr).Port

	return int32(port)
}

// GetRemotePortAndIPFromConn gets the IP address and port of a
// specified connection that we are connected to
func GetRemotePortAndIPFromConn(conn net.Conn) (net.IP, int32) {
	port := conn.RemoteAddr().(*net.UDPAddr).Port
	ip := conn.RemoteAddr().(*net.UDPAddr).IP

	return ip, int32(port)
}

// GetRemoteHostFromConn gets the IP address and port that we are connected to
// and returns it as a string like so: IP:PORT
func GetRemoteHostFromConn(conn net.Conn) string {
	ip, port := GetRemotePortAndIPFromConn(conn)

	return ToAddress(ip.String(), port)
}

// GetLocalPortAndIPFromConn gets our IP address and port that we are using in a connection
func GetLocalPortAndIPFromConn(conn net.Conn) (net.IP, int) {
	port := conn.LocalAddr().(*net.TCPAddr).Port
	ip := conn.LocalAddr().(*net.TCPAddr).IP

	return ip, port
}

// CreatingTCPListener establishes a TCP listener with the next available port
// 0 in port parameter means the OS finds the next available.
func CreatingTCPListener(address string) net.Listener {
	Logger.Log(Logger.INFO, "Creating a TCP listener...")

	listenerConn, err := net.Listen("tcp", address)

	if err != nil {
		Logger.Log(Logger.ERROR, "Failed to created a TCP listener")
		os.Exit(-1)
	}

	Logger.Log(Logger.INFO, "TCP listener created on port: "+strconv.Itoa(int(GetPortFromListener(listenerConn))))

	return listenerConn
}

func CreateUDPListener(address string) (*net.UDPConn, error) {
	Logger.Log(Logger.INFO, "Creating a UDP listener...")

	listenerConn, err := net.ListenUDP("udp4", CreateUDPAddr(address))

	if err != nil {
		return nil, err
	}

	// Logger.Log(Logger.INFO, "TCP listener created on port: "+strconv.Itoa(int(GetPortFromListener(listenerConn))))

	return listenerConn, nil

}

func ReadFromUDPConn(conn *net.UDPConn, msg *Raft.Raft) (*net.UDPAddr, error) {
	buffer := make([]byte, 65535)
	length, address, err := conn.ReadFromUDP(buffer)

	if err != nil {
		return nil, err
	}

	err = proto.Unmarshal(buffer[:length], msg)

	if err != nil {
		return nil, err
	}

	return address, nil
}

func WriteToUDPConn(conn *net.UDPConn, addr *net.UDPAddr, msg *Raft.Raft) error {
	Logger.Log(Logger.INFO, "Sending message to: "+addr.String())

	msgByte, err := proto.Marshal(msg)
	if err != nil {
		Logger.Log(Logger.ERROR, "Failed to marshal: "+err.Error())
		return err
	}

	_, err = conn.WriteToUDP(msgByte, addr)

	if err != nil {
		Logger.Log(Logger.ERROR, "Failed to send to conn: "+err.Error())
		return err
	}

	// Logger.LogWithHost(Logger.INFO, host, "Message sent")

	return nil

}

func CreateUDPAddr(address string) *net.UDPAddr {

	ip, port := ToIPAndPort(address)

	udpAddress := new(net.UDPAddr)

	udpAddress.IP = net.ParseIP(ip)
	udpAddress.Port = int(port)
	return udpAddress
}

// ToAddress converts IP and Port into a string represented like so IP:PORT
func ToAddress(ip string, port int32) string {
	return fmt.Sprintf(ip+":%d", port)
}

// ToIPAndPort gets a string that is represented like IP:PORT and returns the IP and port separately
func ToIPAndPort(address string) (string, int32) {
	ipAndPort := strings.Split(address, ":")

	if len(ipAndPort) != 2 {
		Logger.Log(Logger.ERROR, "Invalid parameters to convert to IP and Port!")
		os.Exit(-1)
	}

	port, err := strconv.Atoi(ipAndPort[1])

	if err != nil {
		Logger.Log(Logger.ERROR, "Invalid parameters to convert Port!")
		os.Exit(-1)
	}

	return ipAndPort[0], int32(port)

}

// ReadFromConn listens for a message to a specified conn and unmarshal it into the msg pointer
// User then uses the msg pointer to read what was sent
func ReadFromConn(conn net.Conn, msg *Raft.Raft) error {
	host := GetRemoteHostFromConn(conn)
	Logger.LogWithHost(Logger.INFO, host, "Listening...")

	data := make([]byte, 65535)

	if data == nil {
		Logger.Log(Logger.ERROR, "Failed to allocate memory for data")
		return fmt.Errorf("failed to allocate memory for data")
	}

	length, err := conn.Read(data)
	if err != nil {
		Logger.Log(Logger.ERROR, "Failed to receive from "+conn.RemoteAddr().String()+": "+err.Error())
		return err
	}

	Logger.LogWithHost(Logger.INFO, host, "Received "+strconv.Itoa(length)+" bytes from "+conn.RemoteAddr().String())

	err = proto.Unmarshal(data[:length], msg)
	if err != nil {
		Logger.Log(Logger.ERROR, "Failed to unmarshal from "+conn.RemoteAddr().String()+": "+err.Error())
		return err
	}

	Logger.LogWithHost(Logger.INFO, host, "Received")
	return nil
}

// WriteToConn sends a specified message to a specified connection.
func WriteToConn(conn net.Conn, message *Raft.Raft) error {
	// Getting the host just for our logger
	host := GetRemoteHostFromConn(conn)

	Logger.LogWithHost(Logger.INFO, host, "Sending message...")
	msgByte, err := proto.Marshal(message)
	if err != nil {
		Logger.Log(Logger.ERROR, "Failed to marshal: "+err.Error())
		return err
	}

	_, err = conn.Write(msgByte)

	if err != nil {
		Logger.Log(Logger.ERROR, "Failed to send to conn: "+err.Error())
		return err
	}

	Logger.LogWithHost(Logger.INFO, host, "Message sent")

	return nil
}

// EstablishConnection Establishes a TCP connection to a specified address
// The address is represented in a format like so IP:PORT
func EstablishConnection(address string) net.Conn {
	Logger.Log(Logger.INFO, "Establishing a connection with node...")
	defer Logger.Log(Logger.INFO, "Connection established with node")

	conn, err := net.DialTimeout("tcp", address, time.Second*6)

	if err != nil {
		Logger.Log(Logger.ERROR, err.Error())
		os.Exit(-1)
	}

	return conn
}
