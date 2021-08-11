package main


import (
        "github.com/bradfitz/gomemcache/memcache"
        "github.com/remeh/sizedwaitgroup"
        "fmt"
        "flag"
        "time"
        "log"
)

var target string
var synchost string
var checks chan Pair
var mcs *memcache.Client
var mct *memcache.Client
var keys int 
var label string

type Pair struct{
    Key string
    Interval string
    Timestamp time.Time
}

func (p Pair) String() string{
    return fmt.Sprintf("%s - %s",p.Key,p.Interval)
}

func main(){


  flag.StringVar(&label,"label","keyprefix","Memcache Key Prefix")
  flag.StringVar(&target,"target","","Target Machine host:port")
  flag.StringVar(&synchost,"synchost","","Machine to read from host:port")
  flag.IntVar(&keys,"keys",100,"Number of keys to check")
  flag.Parse()
  
  checks = make(chan Pair)

  mct = memcache.New(target)
  
  mcs = memcache.New(synchost)
  
  go watchPairs()
  
  swg := sizedwaitgroup.New(25)
  
  
  for j:=0;j<keys;j++ {
    
    swg.Add()
    
    go func(i int){  
      
    defer swg.Done()  
    //log.Printf("Setting Key: %s%d\n",label,i)  
    
    pr := Pair{Key: fmt.Sprintf("%s%d",label,i), Interval: "",Timestamp: time.Now()}  
    err := mct.Set(&memcache.Item{Key: pr.Key, Expiration: 60, Value: []byte("test set")})    
    
    if(err!=nil){
        
        log.Println(err)
        return
    }
    
    pr.Timestamp = time.Now() //just in case there is a delay in setting
    checks <- pr   
    
    }(j)
      
  }
  
  select{}
  

}


func watchPairs(){
    
  for elem := range checks {
      
        go func(it Pair){
            
             //log.Printf("Reading Key: %s\n",it.Key)  
            
            _,err := mcs.Get(it.Key)
            
            i := 0
            for err!=nil {
                
                
                i++
                
                if i > 500 {
                    log.Printf("Not Found %s\n",it.Key)
                    log.Println("Time-out reached!")
                    return
                }
                
                _,err = mcs.Get(it.Key)
                time.Sleep(100 * time.Millisecond)
            }
            
            currentTime := time.Now()
            
            diff := currentTime.Sub(it.Timestamp)
    
            
            it.Interval = fmt.Sprintf("%f",diff.Seconds())
            
            
            log.Println(it)
            
            
        }(elem)
  }
    
}