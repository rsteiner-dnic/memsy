package main 

import(
"log"
"flag"
"github.com/rs/xid"
"fmt"
"time"
"strings"
memcache "github.com/bradfitz/gomemcache/memcache"
)

var target string
var keys []string

func main(){
    
    
    flag.StringVar(&target, "target","127.0.0.1:11211", "TCP port number to listen on")

    flag.Parse()
    
    mc := memcache.New(target)
    
    log.Println("Adding items to memcache")
    
    for i:=0; i<100;i++ {
        
        guid := xid.New()
        kk := guid.String() 
        
        keys = append(keys,kk)
        
        log.Printf("Key: %s\n",kk)
        mc.Set(&memcache.Item{Key: kk, Value: []byte(fmt.Sprintf("Test Value: %d",i)),Expiration: 3600})
        
        
    }
    
    return
    time.Sleep(2 * time.Second)
    
    for _,v := range keys{
       log.Printf("Fetch: %s\n",v)
       item,_ := mc.Get(strings.TrimSpace(v))
       log.Printf("%#v\n",item) 
       
    }
    
    
    
    
}