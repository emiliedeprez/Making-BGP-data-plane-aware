package main

import(
	"goRQM"
	"measurement"
	"flag"
)

func main(){
	action := flag.String("action", "server", "action to perform: measurement or server")
	configPath := flag.String("config", "config.yaml", "Path to the config gile")
	flag.Parse()

	if *action == "server"{
		goRQM.LaunchServer(*configPath)
	}
	
	if *action == "measurement"{
		measurement.LaunchMeasurement()
	}
	
}