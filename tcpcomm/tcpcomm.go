package tcpcomm

import (
	"bytes"
	"encoding/gob"
	"io"
	"log"
	"net"
	"time"
)

var commands map[string]func(string,net.Conn)

// Create your custom data struct
type Message struct {
	Cmd   string
	Data string
}

func logerr(err error) bool {
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			log.Println("read timeout:", err)
		} else if err == io.EOF {
		} else {
			log.Println("read error:", err)
		}
		return true
	}
	return false
}

func read(conn net.Conn) {
	// create a temp buffer
	tmp := make([]byte, 500)
	buf := make([]byte, 0, 4096) 

    timeoutDuration := 5 * time.Second
 
	// loop through the connection to read incoming connections. If you're doing by
	// directional, you might want to make this into a seperate go routine
	  for {
    	  
	
	    conn.SetReadDeadline(time.Now().Add(timeoutDuration))  
    	  
		n, err := conn.Read(tmp)
		if logerr(err) {
			break
		}

		// convert bytes into Buffer (which implements io.Reader/io.Writer)
		
         buf = append(buf, tmp[:n]...)
		
	   }
	   
	    tmpbuff := bytes.NewBuffer(buf)
	   
	    msgenv := new(Message)

		// creates a decoder object
		gobobj := gob.NewDecoder(tmpbuff)
		// decodes buffer and unmarshals it into a Message struct
		gobobj.Decode(msgenv)
		
		log.Printf("Command: %s",msgenv.Cmd)
		
		if fn,ok := commands[msgenv.Cmd]; ok {
    		
    		
    		fn(msgenv.Data,conn)
    		
    		
		}

	
	
}

func Resp(conn net.Conn,cmd string,data string) {
	msg := Message{Cmd: cmd, Data: data}
	bin_buf := new(bytes.Buffer)

	// create a encoder object
	gobobje := gob.NewEncoder(bin_buf)
	// encode buffer and marshal it into a gob object
	gobobje.Encode(msg)

	conn.Write(bin_buf.Bytes())
	
}

func handle(conn net.Conn) {
    

	
	//remoteAddr := conn.RemoteAddr().String()
	
	//log.Println("Client connected from " + remoteAddr)

	read(conn)
	
	//resp(conn)
}

func recv(conn net.Conn) {
	// create a temp buffer
	tmp := make([]byte, 5000)
	conn.Read(tmp)

	// convert bytes into Buffer (which implements io.Reader/io.Writer)
	tmpbuff := bytes.NewBuffer(tmp)
	msgenv := new(Message)

	// creates a decoder object
	gobobjdec := gob.NewDecoder(tmpbuff)
	// decodes buffer and unmarshals it into a Message struct
	gobobjdec.Decode(msgenv)

    if fn,ok := commands[msgenv.Cmd]; ok {
    		
    		
    		fn(msgenv.Data,conn)
    		
    		
	}
}


func SendMessage(target string,cmd string,data string) error{
    
    	    
    conn, err := net.Dial("tcp", target)
    
    if(err!=nil){
        
        log.Println(err)
        return err
    }

    msg := Message{Cmd: cmd, Data: data}
	
	bin_buf := new(bytes.Buffer)

	// create a encoder object
	gobobje := gob.NewEncoder(bin_buf)
	// encode buffer and marshal it into a gob object
	gobobje.Encode(msg)

	conn.Write(bin_buf.Bytes())

	recv(conn)
	
	conn.Close()

    return nil
}

func Register(cmd string,fn func(string,net.Conn)){
    
    commands[cmd] = fn
    
}

func init(){
    
    
    commands = make(map[string]func(string,net.Conn))
    
}

func StartServer(comport string) {
    log.Printf("Connection listening on %s",comport)
	server, _ := net.Listen("tcp", ":"+comport)
	for {
		conn, err := server.Accept()
		if err != nil {
			log.Println("Connection error: ", err)
			return
		}
		go handle(conn)
	}
}