package main

import "log"

// Nerr check for critical errors
func Nerr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
