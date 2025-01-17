package main

import (
	"fmt"
	"github.com/Meshate/ebp"
	"time"
)

var hash = "0x21c3ac17a523528af506a37601fcb1c81d029f8b68dc63cd094f72767acdfd13"

func main() {
	p := ebp.NewParser(10 * time.Second)

	fmt.Println(p.GetCurrentBlock())
	fmt.Println(p.Subscribe(hash))
	fmt.Println(p.GetTransactions(hash))
}
