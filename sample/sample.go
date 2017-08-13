package main

import (
	"github.com/ziutek/blas"
	"math"
	d "./data"
	"fmt"
)

func main() {
	data := d.GetData()

	// Sum from 0 to 999.
	total := blas.Dasum(1000, data, 1)
	fmt.Println(fmt.Sprintf("Total = %f", total))

	ndigits := math.Ceil(math.Log10(total))
	fmt.Println(fmt.Sprintf("# of digits = %v", ndigits))
}


