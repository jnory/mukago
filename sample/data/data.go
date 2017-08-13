package data

func GetData() []float64{
	data := make([]float64, 1000)
	for i := 0; i < 1000; i++ {
		data[i] = float64(i)
	}
	return data
}
