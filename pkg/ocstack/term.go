package ocstack

import "fmt"

var Reset = "\033[0m"
var Red = "\033[31m"
var Green = "\033[32m"
var Yellow = "\033[33m"
var Blue = "\033[34m"
var Magenta = "\033[35m"
var Cyan = "\033[36m"
var Gray = "\033[37m"
var White = "\033[97m"

// Helper - TODO: we might want to load it with the list of registered commands
func TermHelper(cmd string) {
	if cmd == "" {
		fmt.Println("----")
		fmt.Println("Available Commands :> ")
		fmt.Println("1. /quit ")
		fmt.Println("2. /template ")
		fmt.Println("----")
	} else {
		fmt.Println("----")
		help(cmd)
	}
}

func TermHeader(profile string) {
	fmt.Println("----")
	fmt.Println("Hello, ocstack!")
	fmt.Printf("Agent profile: %s\n", profile)
	fmt.Println("----")
	fmt.Println("I :> Run /help to get a list of available commands")
}

func ShowWarn(s string) {
	fmt.Printf("%s%s%s\n", Red, s, Reset)
}

func help(cmd string) {
	switch {
	case cmd == "template":
		fmt.Println("Usage: /template <profile>")
		// TODO: Print available profiles
		break;
	default:
	}
}

