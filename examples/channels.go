package main

import (
	"fmt"
	"time"
)

func main() {
	messageChan := make(chan string)
	doneChan := make(chan bool)
	numbersChan := make(chan int)

	go func() {
		messageChan <- "Hello"
		messageChan <- "World"
		doneChan <- true
	}()

	go func() {
		for i := 0; i < 5; i++ {
			numbersChan <- i
		}
		close(numbersChan)
	}()

	go func() {
		for {
			select {
			case msg := <-messageChan:
				fmt.Println("Received message:", msg)
			case <-doneChan:
				return
			}
		}
	}()

	for num := range numbersChan {
		fmt.Println("Received number:", num)
	}

	time.Sleep(time.Second)
} 