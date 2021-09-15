package main 


import(
    "log"
    "strings"
    "time"
    "context"
    "strconv"
    "encoding/gob"
    "fmt"
    "bytes"
    memcache "github.com/bradfitz/gomemcache/memcache"
    "github.com/DncDev/memsy/dkv"
    cmap "github.com/orcaman/concurrent-map"
    memcached "github.com/ralfonso-directnic/go-memcached"
    "github.com/paulbellamy/ratecounter"
)

var peercon map[string]*memcache.Client

type Cache struct{
  Loaded bool
  Storage  *dkv.KVStore
  Index cmap.ConcurrentMap
  items chan *memcached.Item
  peeritems map[string]chan *memcached.Item
  peers []string
  counter *ratecounter.RateCounter
  StatsObj memcached.Stats
}

func (c *Cache) Stats(s memcached.Stats){
    
    c.StatsObj = s
}

func (c *Cache) GetWithContext(ctx *context.Context,key string) memcached.MemcachedResponse {
    
    
    //for some reason the key is passed with a preceding space, it's probably a bug in the lib
    key = strings.TrimSpace(key)
    
     if tmp, ok := c.Index.Get(key); ok {
		item := tmp.(*memcached.Item)
		
		if item.IsExpired() {
    		log.Println("Item Expired")
			c.Index.Remove(key)
		} else {
			return &memcached.ItemResponse{item}
		}	
     }else if(debug==true){ 
       go log.Printf("Key Missing: %s",key)
     }
	
	
    
	return nil
}

func (c *Cache) Peers(p []string){
    
    c.peers = p
    
    c.peeritems = make(map[string]chan  *memcached.Item)
    
    for _,pp := range c.peers {

    	if(len(pp)<1){
        	continue
    	}
        
          c.peeritems[pp] = make(chan *memcached.Item)
        
    }
    
}

func (c *Cache) SetWithContext(ctx *context.Context,item *memcached.Item) memcached.MemcachedResponse {

    go c.counter.Incr(1)
    
    go c.StatsObj["total_items"].(*memcached.CounterStat).Increment(1)
    
    dosync := true
    
    if(strings.Contains(item.Key,"memsysync_")){
        dosync = false
        item.Key=strings.Replace(item.Key,"memsysync_","",-1)
    }

    if(debug==true){

    go log.Printf("Set Key: %s",item.Key)

    }

    go c.Index.Set(item.Key,item)
    
    go c.DurableSave(item)
    
    if(dosync==true){
        c.items <- item
    }
	
	return nil 
}

func (c *Cache) SyncItem(item *memcached.Item){
    
    c.counter.Incr(1)
    
    c.items <- item
    
}

//syncs to all peers

func (c *Cache) PeerConnect(){
    
    pport := strconv.Itoa(port)  
    
    for _,p := range c.peers {
            
                  
                    mc := memcache.New(fmt.Sprintf("%s:%s",p,pport))
                    mc.MaxIdleConns = 100
                    peercon[p] = mc
                    
    }           

    
    
}

func (c *Cache) PeerDistribute(){ 
    
    
    //peer connection management
    
    peercon = make(map[string]*memcache.Client)
    
    go func(){
    
    c.PeerConnect()
    
    tick := time.NewTicker(5*time.Second)
        
    pport := strconv.Itoa(port)  
        
    for range tick.C {
        
         for p,cli := range peercon {
            
            
                 err := cli.Set(&memcache.Item{Key: "memcache_connection_check", Value: []byte("1"),Expiration: 3600})
                 //check the connection is up, if it's not create it
                 if(err!=nil){
                    log.Printf("Peercon ping failed %s, reconnecting to %s\n",err,p)
                    mc := memcache.New(fmt.Sprintf("%s:%s",p,pport))
                    mc.MaxIdleConns = 100
                    peercon[p] = mc
                 }   
                    
         }   
         
         if(len(c.peers)!=len(peercon)){
             log.Println("Peers mismatch, reconnect all peers")
             go c.PeerConnect() //this will ensure if a peer is down when started it gets added when it's back up
             
         }        

    }
    
    }()
    
    
    
    
    var items []*memcached.Item
    
    for {
     
       override:=false

       select {
       
       case it := <-c.items:
    
           items = append(items,it)
           
       case <-time.After(500 * time.Millisecond):
           
           override=true
       //allow flusing the current list 
        
       }
       
        //batch N items to send
            
        crate := c.counter.Rate()
                
        rate := 1

        if(crate>100){
            
            rate = 100
            
        }else if(crate>25){
            
            rate = 25
            
        }
        
        if(len(items)>0){
        
        //log.Println("Rate:",rate)
        
        }
        
        if(len(items)==rate || override==true){
            
            
            go func(ite []*memcached.Item){
            
             pport := strconv.Itoa(port)  
                    
             for p,mc := range peercon {
                
                if(len(ite)>0){
                    
                    if(len(p)<1){
                        continue
                    } 
                    
                    
                      
                    log.Printf("Sending to peer: %s ct: %d\n",p+":"+pport,len(ite))   
                                       
                    for _,item := range ite {
                        
                        exp := int32(item.Expires.Sub(time.Now()).Seconds())
                        
                    
                        mc.Set(&memcache.Item{Key: fmt.Sprintf("memsysync_%s",item.Key), Value: item.Value,Expiration: exp})
                        time.Sleep(5*time.Millisecond) 
                  
                    }
                    
                    
                
               }
        
             } 
                
            }(items)
            
            items=nil
            
        }
        
    }
    
}

func (c *Cache) DurableSave(item *memcached.Item){
    
     err := c.Storage.Put(item.Key,item)
     
     if(err!=nil){
           log.Println(err)
     }
    
    
     //send this to other boxes?
    
}

func (c *Cache) DeleteWithContext(ctx *context.Context,key string) memcached.MemcachedResponse {
	c.Index.Remove(key)
	//Remove from disk storage
	go c.Storage.Delete(key)
	return nil
}

/*
   Restore the cache from the disk 
*/

func (c *Cache) Restore(key string,value []byte){
    
    
    
    d := gob.NewDecoder(bytes.NewReader(value))
    var item *memcached.Item
    d.Decode(&item)

    if(!item.IsExpired()){
   // log.Println("Restoring key...",key)
    c.Index.Set(key,item)
       
    }
    
}

func (c *Cache) CleanExpired(key string,value []byte){
    
    
    d := gob.NewDecoder(bytes.NewReader(value))
    var item *memcached.Item
    d.Decode(&item)
    

    //cleanup old data
    if(item.IsExpired()){
    log.Println("Expired key....",key)  
    go c.Storage.Delete(key)
    c.Index.Remove(key)
     if(c.StatsObj!=nil){
      c.StatsObj["expired_unfetched"].(*memcached.CounterStat).Increment(1)
     }
    }
    
}

func (c *Cache) Reload(){
    
 
       log.Println("Restoring from disk")
       c.Storage.Objects(c.Restore) 
       log.Println("Completed restore from disk")
       c.Loaded = true
    
   
}

func NewCache(cacheloc string) *Cache { 
    
    var err error

    cache := &Cache{}
    
    cache.Storage,err = dkv.Open(cacheloc+"/memsy.db")

    cmap.SHARD_COUNT =  256

    cache.Index = cmap.New()
    
    cache.items = make(chan *memcached.Item)
    
    cache.counter = ratecounter.NewRateCounter(1 * time.Second)
        
    if(err!=nil){
      log.Fatal(err)
    }

       
    go func (){

      	
       dur,perr := time.ParseDuration(expireinterval)	
    	
       if(perr!=nil){
           
            dur,_ = time.ParseDuration("17m")	
           
       }	
    	

    
       ticker := time.NewTicker(dur)



       for range ticker.C {
           log.Println("Checking for expired keys...")
           cache.Storage.Objects(cache.CleanExpired) 
       }
        
        
    }()
    
    return cache   
}
