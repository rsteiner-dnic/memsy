package main 


import(
    "log"
    "strings"
    "time"
    "strconv"
    "encoding/gob"
    "bytes"
    memcache "github.com/bradfitz/gomemcache/memcache"
    "github.com/DncDev/memsy/dkv"
    cmap "github.com/orcaman/concurrent-map"
    memcached "github.com/mattrobenolt/go-memcached"
    "github.com/paulbellamy/ratecounter"
)

type Cache struct{
  Loaded bool
  Storage  *dkv.KVStore
  Index cmap.ConcurrentMap
  items chan *memcached.Item
  peeritems map[string]chan *memcached.Item
  peers []string
  counter *ratecounter.RateCounter
}

func (c *Cache) Get(key string) memcached.MemcachedResponse {
    
    
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
	}
	
	
    
	return nil
}

func (c *Cache) Peers(p []string){
    
    c.peers = p
    
    c.peeritems = make(map[string]chan  *memcached.Item)
    
    for _,pp := range c.peers {
        
          c.peeritems[pp] = make(chan *memcached.Item)
        
    }
    
}

func (c *Cache) Set(item *memcached.Item) memcached.MemcachedResponse {

    c.counter.Incr(1)
    
    dosync := true
    
    if(strings.Contains(item.Key,"memsysync_")){
        dosync = false
        item.Key=strings.Replace(item.Key,"memsysync_","",-1)
    }

    c.Index.Set(item.Key,item)
    
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

func (c *Cache) PeerDistribute(){ 
    
    var items []*memcached.Item
    
    for {
     
       override:=false

       select {
       
       case it := <-c.items:
    
           items = append(items,it)
           
       case <-time.After(5 * time.Second):
           
           override=true
       //allow flusing the current list 
        
       }
       
       if(len(items)>0){
        
       //log.Println("Items in distribution queue:",len(items))
        
       } 
        //batch N items to send
            
        crate := c.counter.Rate()
                
        rate := 1
      
        if(crate>750){
        
            rate = 1000
            
        }else if(crate>500){
            
            rate = 500
            
        }else if(crate>100){
            
            rate = 100
            
        }else if(crate>50){
            
            rate = 50
            
        }else if(crate>10){
            
            rate = 10
            
        }
        
        if(len(items)>0){
        
        //log.Println("Rate:",rate)
        
        }
        
        if(len(items)==rate || override==true){
            
            go func(ite []*memcached.Item){
                
             for _,p := range c.peers {
                
                if(len(ite)>0){
                    
                    pport := strconv.Itoa(port)    
                        
                    log.Printf("Sending to peer: %s ct: %d\n",p+":"+pport,len(ite))    
                    
                    mc := memcache.New(p+":"+pport)
                    
                    for _,item := range ite {
                        
                        exp := int32(item.Expires.Sub(time.Now()).Seconds())
                        
                    
                        mc.Set(&memcache.Item{Key: "memsysync_"+item.Key, Value: item.Value,Expiration: exp})  
                  
                    
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

func (c *Cache) Delete(key string) memcached.MemcachedResponse {
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

    cache.Index = cmap.New()
    
    cache.items = make(chan *memcached.Item)
    
    cache.counter = ratecounter.NewRateCounter(1 * time.Second)
        
    if(err!=nil){
      log.Fatal(err)
    }

       
    go func (){

      	
       dur,perr := time.ParseDuration(syncinterval)	
    	
       if(perr!=nil){
           
            dur,_ = time.ParseDuration("30m")	
           
       }	
    	

    
       ticker := time.NewTicker(dur)



       for range ticker.C {
           log.Println("Checking for expired keys...")
           cache.Storage.Objects(cache.CleanExpired) 
       }
        
        
    }()
    
    return cache   
}
