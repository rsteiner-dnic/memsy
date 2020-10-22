package main

import (
	"flag"
	"strings"
	"fmt"
	"time"
	memcached "github.com/mattrobenolt/go-memcached"
	"log"
	"runtime"
	"net"
    "github.com/DncDev/memsy/tcpcomm"

)


var listen string
var port int
var threads int 
var peers_raw string
var peers []string
var comport string
var cache *Cache
var cacheloc string
var syncinterval string



func main() {
    
    flag.StringVar(&cacheloc,"cachedir","/var/cache","Location directory for memsy db")
    flag.StringVar(&syncinterval,"syncinterval","30m","How often to sync all records to other nodes")
    flag.StringVar(&peers_raw,"peers","", "Comma separated list of servers to peer with")
    flag.StringVar(&listen,"listen","0.0.0.0", "Interface to listen on. Default to all addresses.")
    flag.IntVar(&port, "port",11211, "TCP port number to listen on")
    flag.StringVar(&comport, "comport","1180", "TCP port number for communication")
    flag.IntVar(&threads,"threads", runtime.NumCPU(),"Number of threads to use")
	flag.Parse()
	
	peers = strings.Split(peers_raw,",")
	
	runtime.GOMAXPROCS(threads)

	address := fmt.Sprintf("%s:%d", listen, port)
	
	//init the cache handler
	cache = NewCache(cacheloc)
	
	cache.Peers(peers)
	//Sync all peers when memcache set is made
	go cache.PeerDistribute()
	//restore from disk on startup, this is non blocking
	cache.Reload()
	
	defer cache.Storage.Close()
	
	server := memcached.NewServer(address, cache)
	
	tcpcomm.Register("KeySync",keySync);
	
	tcpcomm.Register("Ack",func(data string, conn net.Conn){ conn.Close() })
	
	go tcpcomm.StartServer(comport)
	//do sync to other nodes
    go startSyncPeers()
	
	go checkKeyCount()
		
	log.Fatal(server.ListenAndServe())
}


//Output the keys we have on this instance
func checkKeyCount(){
    
        ticker2 := time.NewTicker(10 * time.Second)
    	
    	for range ticker2.C {
    	
    	log.Printf("Keys Stored: %d",len(cache.Index.Keys()))
    	
    	}
    
}

func syncPeers(){
    
    
            for _,p := range peers{
            	
            	//send a sync request
         
                parts := strings.Split(p,":")   
                
            	log.Printf("Syncing from node %s",parts[0]+":"+comport)
              	
              	log.Printf("Current keys Stored: %d",len(cache.Index.Keys()))

            	tcpcomm.SendMessage(parts[0]+":"+comport,"KeySync",strings.Join(cache.Index.Keys(),","))
            	
            	time.Sleep(30 * time.Second)
            	//give things a chance to settle
        	
        	}
    
}


//Sync to peers and then sync every X duration
func startSyncPeers(){
    
   	 
       log.Println("Syncing to Peers...")
       
       syncPeers()
    	
       dur,perr := time.ParseDuration(syncinterval)	
    	
       if(perr!=nil){
           
            dur,_ = time.ParseDuration("30m")	
           
       }	
    	
       ticker := time.NewTicker(dur)

       for range ticker.C {
	   
	     syncPeers()
    
       }
    
}

//sync to peers 
func keySync (data string, conn net.Conn){
    	
      log.Println("KeySync Command Running...")
     
      defer conn.Close()
      
      diff:=difference(cache.Index.Keys(),strings.Split(data,","))
      
      log.Printf("Keys to sync: %d\n",len(diff))
      
      parts := strings.Split(conn.RemoteAddr().String(),":")

      log.Printf("Disk Sync to memcache @ %s",parts[0])
    
      go func(){
    
         
      for _,key := range diff{
            
      
            if tmp, ok := cache.Index.Get(key); ok {
	               	         
		         item := tmp.(*memcached.Item)
		         cache.SyncItem(item)
            }else{
                
                log.Println("Key unable to sync",key)
            }   
            

      }

      
      }()
      
      tcpcomm.Resp(conn,"Ack","OK")	
      
      
    	
    	
}

