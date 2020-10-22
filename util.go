package main 

import(

 "log"

)

func difference(a []string, b []string) []string {
    
    log.Println("Starting Calculate Difference")
    
    mb := make(map[string]struct{}, len(b))
    for _, x := range b {
        mb[x] = struct{}{}
    }
    var diff []string
    for _, x := range a {
        if _, found := mb[x]; !found {
            diff = append(diff, x)
        }
    }
    
    log.Println("Ending Calculate Difference")
      
    return diff
}


