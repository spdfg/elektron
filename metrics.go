package main

type Metric struct{
	Name string `json:"name"`
	CPU float64 `json:"cpu"`
	RAM float64 `json:"ram"`
	Watts float64 `json:"watts"`
}
