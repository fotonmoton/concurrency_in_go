package chapter3

import (
	"sync"
	"testing"
)

func BenchmarkContextSwitch(b *testing.B) {
	var begin = make(chan struct{})
	var c = make(chan struct{})
	var token struct{}
	var wg sync.WaitGroup
	var sender = func() {
		defer wg.Done()
		<-begin
		for i := 0; i < b.N; i++ {
			c <- token
		}
	}
	var receiver = func() {
		defer wg.Done()
		<-begin
		for i := 0; i < b.N; i++ {
			<-c
		}
	}

	wg.Add(2)
	go sender()
	go receiver()
	b.StartTimer()
	close(begin)
	wg.Wait()
}
