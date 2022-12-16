package chapter3

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

func wrongLoop() {
	var wg sync.WaitGroup

	for _, salutation := range []string{"hello", "good day", "salute!"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fmt.Println(salutation)
		}()
	}

	wg.Wait()
}

func correctLoop() {
	var wg sync.WaitGroup

	for _, salutation := range []string{"hello", "hi", "darova"} {
		wg.Add(1)
		go func(s string) {
			fmt.Println(s)
			wg.Done()
		}(salutation)
	}

	wg.Wait()
}

func goroutineMemory() {
	var c <-chan interface{}
	var wg sync.WaitGroup
	var numGoroutines = 100e4
	noop := func() {
		wg.Done()
		<-c
	}
	memConsumed := func() uint64 {
		runtime.GC()
		var s runtime.MemStats
		runtime.ReadMemStats(&s)
		return s.Sys
	}

	wg.Add(int(numGoroutines))
	before := memConsumed()
	for i := numGoroutines; i > 0; i-- {
		go noop()
	}
	wg.Wait()
	after := memConsumed()
	fmt.Printf("%.3fkb", float64(after-before)/numGoroutines/1000)
}

func mutexes() {
	var count int
	var mutex sync.Mutex
	var wg sync.WaitGroup
	var increment = func() {
		mutex.Lock()
		defer mutex.Unlock()
		count++
		fmt.Printf("Increment: %d\n", count)
	}
	var decrement = func() {
		mutex.Lock()
		defer mutex.Unlock()
		panic("asd")
		count--
		fmt.Printf("Decrement: %d\n", count)
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			increment()
		}()
	}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					fmt.Println("Recovered in f", r)
				}
			}()
			decrement()
		}()
	}
	wg.Wait()
}

func condQueue() {
	queue := make([]struct{}, 0, 10)
	cond := sync.NewCond(&sync.Mutex{})

	removeFromQueue := func(delay time.Duration) {
		time.Sleep(delay)
		cond.L.Lock()
		queue = queue[1:]
		fmt.Println("Removed from queue")
		cond.L.Unlock()
		cond.Signal()
	}

	for i := 0; i < 10; i++ {
		cond.L.Lock()
		for len(queue) == 2 {
			cond.Wait()
		}
		queue = append(queue, struct{}{})
		fmt.Println("Added to queue")
		go removeFromQueue(time.Second)
		cond.L.Unlock()
	}
}

func pool() {
	created := 0
	pool := &sync.Pool{
		New: func() interface{} {
			created++
			mem := make([]byte, 1024)

			return &mem
		},
	}

	// 4kb memory pool
	pool.Put(pool.New())
	pool.Put(pool.New())
	pool.Put(pool.New())
	pool.Put(pool.New())

	workers := 1024 * 1024
	wg := sync.WaitGroup{}
	wg.Add(workers)

	for i := workers; i > 0; i-- {
		go func() {
			defer wg.Done()
			mem := pool.Get().(*[]byte)
			defer pool.Put(mem)

			// slow operation like printf
			// keep pool resource occupied
			// and force poll to create new ones
			// fmt.Printf("%x\n", (*mem)[0])
			(*mem)[0] = byte(1)
		}()
	}

	wg.Wait()
	fmt.Printf("memory created times: %v", created)
}

func simpleChannel() {
	messageStream := make(chan string)

	go func() {
		messageStream <- "Hello, world"
	}()

	fmt.Println(<-messageStream)
}

func rangeOverChannel() {
	intStream := make(chan int)

	go func() {
		defer close(intStream)
		for i := 0; i < 10; i++ {
			intStream <- i
			time.Sleep(time.Second)
		}
	}()

	for num := range intStream {
		fmt.Printf("%v ", num)
	}
}

func synchronizeGoroutines() {
	wg := sync.WaitGroup{}
	begin := make(chan interface{})

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			<-begin
			fmt.Printf("%v begin\n", n)
		}(i)
	}

	fmt.Println("Start all goroutines...")
	close(begin)
	wg.Wait()
}

func simpleSelect() {
	start := time.Now()
	ch := make(chan int)

	go func() {
		defer close(ch)
		time.Sleep(time.Second * 2)
	}()

	fmt.Println("Blocking...")
	select {
	case <-ch:
		fmt.Printf("Unblocked after: %v", time.Since(start))
	}

}

func simultaneousSelect() {
	c1 := make(chan int)
	close(c1)
	c2 := make(chan int)
	close(c2)

	c1Num := 0
	c2Num := 0

	for i := 0; i < 1000; i++ {
		select {
		case <-c1:
			c1Num++
		case <-c2:
			c2Num++
		}
	}

	fmt.Printf("c1: %v\nc2: %v", c1Num, c2Num)
}

func selectDefault() {
	done := make(chan struct{})
	workDone := 0

	go func() {
		defer close(done)
		time.Sleep(time.Second * 5)
	}()

loop:
	for {
		select {
		case <-done:
			break loop
		default:
			{
				fmt.Println("Working...")
				workDone++
				time.Sleep(time.Second)
			}
		}
	}
	fmt.Printf("Work done %v times", workDone)
}

func Chapter3() {
	// wrongLoop()
	// correctLoop()
	// goroutineMemory()
	// mutexes()
	// condQueue()
	// pool()
	// simpleChannel()
	// rangeOverChannel()
	// synchronizeGoroutines()
	// simpleSelect()
	// simultaneousSelect()
	selectDefault()
}
