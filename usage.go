package main

import "fmt"

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("------Control------")
	fmt.Println("   go run . start")
	fmt.Println("   go run . pause")
	fmt.Println("   go run . resume")
	fmt.Println("   go run . stop")
	fmt.Println("------Stats------")
	fmt.Println("   go run . stats <day|week|month|year|total>")
	fmt.Println("   go run . goal")
	fmt.Println("   go run . history")
	fmt.Println("------Edit------")
	fmt.Println("   go run . edit <session-id> <duration>")
	fmt.Println("   Use 'go run . focus history' to find a session ID")

}

func printStatsUsage() {
	fmt.Println("Usage:")
	fmt.Println("	go run . stats day")
	fmt.Println("	go run . stats week")
	fmt.Println("	go run . month")
	fmt.Println("	go run . year")
	fmt.Println("	go run . total")
}

func printEditUsage() {
	fmt.Printf("Usage: go run . edit <session-id> <duration>\n-------\nExample: go run . edit 20260525-143012 1h30m\n")

}
