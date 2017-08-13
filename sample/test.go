package main

import (
	"math"
	"fmt"

	"github.com/ziutek/blas"
)

func main() {

	data := make([]float64, 1000)
	for i := 0; i < 1000; i++ {
		data[i] = float64(i)
	}

	/*
	Sum from 0 to 999.
	 */
	total := blas.Dasum(1000, data, 1)
	fmt.Println(fmt.Sprintf("Total = %f", total))

	ndigits := math.Ceil(math.Log10(total))
	fmt.Println(fmt.Sprintf("# of digits = %v", ndigits))
}


