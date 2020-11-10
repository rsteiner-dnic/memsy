package main 

import(
"log"
"flag"
"github.com/rs/xid"
"time"
"strings"
"github.com/remeh/sizedwaitgroup"
memcache "github.com/bradfitz/gomemcache/memcache"
)

var target string
var target1 string
var target2 string
var keys []string
var val string

func main(){
    val = `<Results xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="http://dtd.overture.com/schema/affiliate/2.9.3/OvertureSearchResults.xsd">
<KeywordsRatingSet keywords="lawnmower deals">
<Market>us</Market>
</KeywordsRatingSet>
<SearchID>F818366092E84B7F</SearchID>
<ResultSet id="searchResults" numResults="2" adultRating="G" searchId="F818366092E84B7F" plaCount="0">
<Listing rank="1" title="Husqvarna® Official Site - Shop High-Quality Products" description="Get the Right Robotic Mower for the Job. Learn More About our Best Selling Models. Find a Dealer Near You &amp; Browse Husqvarna®&#39;s Range of Quality Automowers." siteHost="www.husqvarna.com" adultRating="G" ImpressionId="76484917017923" phoneNumber="">
<ClickUrl type="body">://searchfeed.adssquared.com/visit?c=F818366092E84B7F-8c52ce9ef6482bae82ad0062bb6532f8</ClickUrl>
<Extensions>
<PartnerOptOut>
<IsAllowed>1</IsAllowed>
</PartnerOptOut>
</Extensions>
</Listing>
<Listing rank="2" title="Lawn Mower Deals - Lawn Mower Deals" description="Find Lawn Mower Deals. Compare Results! Search for Lawn Mower Deals. Smart Results Today!" siteHost="www.topsearch.co/Lawn Mower Deals/Now" adultRating="G" ImpressionId="72499204291648" phoneNumber="">
<ClickUrl type="body">://searchfeed.adssquared.com/visit?c=F818366092E84B7F-465d0bc027cd52f5e5e7a4381a69fec8</ClickUrl>
<Extensions>
<PartnerOptOut>
<IsAllowed>1</IsAllowed>
</PartnerOptOut>
</Extensions>
</Listing>
<NextArgs>Keywords=lawnmower%20deals&xargs=12KPjg1oZpy5a3vOHvKvjFTvXBhg9O0JC35Is%5FWMQaRp8L%5FXB7EbEoO92ExpctD%2DFmzz7g&hData=12KPjg1o1g4M%5F%2Dx72scrzBTpaKxiwL4pDC%2DspmDJd%5FH9xchXppI%2DcJT5Px</NextArgs>
</ResultSet>
</Results>`
    
    flag.StringVar(&target, "target","doma-lander1.dnc.io:11211", "TCP port number to listen on")
    
    target1 = "doma-lander2.dnc.io:11211"
   // target2 = "doma-lander3.dnc.io:11211"
   
    flag.Parse()
    
  
     log.SetFlags(log.LstdFlags | log.Lshortfile)
    log.Println("Adding items to memcache")
    
    swg := sizedwaitgroup.New(100)

    for i:=0; i<501;i++ {

        swg.Add()
       
        go func(){          
       
          defer swg.Done()
          guid := xid.New()
          kk := guid.String() 
        
          keys = append(keys,kk)
        
        
          mc := memcache.New(target)

          err := mc.Set(&memcache.Item{Key: kk, Value: []byte(val),Expiration: 3600})
      
          if(err!=nil){
              
              log.Println(err)
          }else{
              
              log.Printf("Putting Key: %s\n",kk)
          }
      
        }()
        
    }
    
   // return
    time.Sleep(1 * time.Second)
    
  
    
    wg := sizedwaitgroup.New(20)
    mc1 := memcache.New(target)
    mc2 := memcache.New(target1) 
    
    for _,v := range keys{
        
       wg.Add() 
       
       go func(){ 
        
      
       defer wg.Done()
       
       log.Printf("Fetch Key: %s\n",v)
     
       item,e := mc1.Get(strings.TrimSpace(v))
      
       if(e==nil){
            log.Printf("Host 1: %#v\n",len(item.Value)) 
       }else{
           log.Println(e)
       }
       
       item2,e2 := mc2.Get(strings.TrimSpace(v))
       
       
       if(e2==nil){
           log.Printf("Host 2: %#v\n",len(item2.Value))            
       }else{
           log.Println(e2)   
       }
       
       
       }()
       

    }

    wg.Wait()
           
    
    
    
}
