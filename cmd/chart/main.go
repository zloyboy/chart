package main

import (
	"fmt"
	"os"

	"github.com/zloyboy/chart"
)

func main() {
	values := []chart.DoubleValue{
		{Label: "ill", Lab: [2]string{"M\n55.00 %", "F\n25.00 %"}, Val: [2]float64{0.55, 0.25}},
		{Label: "vac", Lab: [2]string{"M\n45.00 %", "F\n65.00 %"}, Val: [2]float64{0.45, 0.65}},
	}

	bc := chart.BarChart2{
		Title: "example",
		Background: chart.Style{
			Padding: chart.Box{
				Top:    40,
				Bottom: 40,
				Left:   10,
				Right:  20,
			},
		},
		Width:      500,
		Height:     200,
		BarWidth:   200,
		BarSpacing: 10,

		LabelFirst:  false,
		LabelTop:    10,
		SubLabelTop: 30,

		YAxis: chart.YAxis{
			ValueFormatter: chart.PercentValueFormatter,
			Range: &chart.ContinuousRange{
				Min: 0,
				Max: 1,
			},
		},
		Bars: values,
	}

	//buffer := &bytes.Buffer{}
	//bc.Render(chart.PNG, buffer)

	f, _ := os.Create("output.png")
	defer f.Close()
	bc.Render(chart.PNG, f)
	fmt.Println("bar chart")
}
