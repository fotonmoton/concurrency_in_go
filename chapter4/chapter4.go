package chapter4

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"sync"
	"time"
)

func channelConfinement() {
	producer := func() <-chan int {
		ch := make(chan int, 5)
		defer close(ch)

		for i := 0; i < 5; i++ {
			ch <- i
		}

		return ch
	}

	consumer := func(ch <-chan int) {
		for val := range ch {
			fmt.Printf("%v ", val)
		}
	}

	consumer(producer())
}

func variableConfinement() {
	printData := func(wg *sync.WaitGroup, data []byte) {
		defer wg.Done()

		buf := bytes.Buffer{}
		for _, char := range data {
			fmt.Fprintf(&buf, "%c", char)
		}

		fmt.Println(buf.String())
	}

	wg := sync.WaitGroup{}
	wg.Add(2)

	data := []byte("golang")

	go printData(&wg, data[3:])
	go printData(&wg, data[:3])

	wg.Wait()
}

func forSelect() {
	done := make(chan interface{})

	go func() {
		time.Sleep(time.Second * 5)
		close(done)
	}()

	for {
		select {
		case <-done:
			return
		default:
			fmt.Printf("%v\n", time.Now())
			time.Sleep(time.Millisecond * 100)
		}
	}
}

func readerCancelation() {
	doWork := func(done <-chan interface{}, strings <-chan string) <-chan interface{} {
		terminated := make(chan interface{})
		go func() {
			defer close(terminated)
			defer fmt.Println("Exit doWork.")
			for {
				select {
				case s := <-strings:
					fmt.Println(s)
				case <-done:
					return
				}

			}
		}()
		return terminated
	}

	canceller := func(done chan<- interface{}) {
		time.Sleep(time.Second * 5)
		fmt.Println("Canceling...")
		close(done)
	}

	fmt.Println("Started...")
	done := make(chan interface{})
	terminated := doWork(done, nil)
	go canceller(done)
	<-terminated
	fmt.Println("Done.")
}

func writerCancellation() {
	intGenerator := func(done chan interface{}) <-chan int {
		intStream := make(chan int)

		go func() {
			defer close(intStream)
			defer fmt.Println("Done generating.")
			for {
				select {
				case <-done:
					return
				case intStream <- rand.Int():
				}
			}
		}()

		return intStream
	}

	done := make(chan interface{})
	intStream := intGenerator(done)

	fmt.Println("Random ints:")

	for i := 0; i < 3; i++ {
		time.Sleep(time.Millisecond * 100)
		fmt.Printf("%v: %v\n", i, <-intStream)
	}

	close(done)

	time.Sleep(time.Second)

	fmt.Println("Done.")
}

func orChannel() {
	var or func(channels ...<-chan interface{}) <-chan interface{}
	or = func(channels ...<-chan interface{}) <-chan interface{} {
		switch len(channels) {
		case 0:
			return nil
		case 1:
			return channels[0]
		}

		orDone := make(chan interface{})

		go func() {
			fmt.Println("goroutine created")
			defer close(orDone)

			switch len(channels) {
			case 2:
				select {
				case <-channels[0]:
				case <-channels[1]:
				}
			default:
				select {
				case <-channels[0]:
				case <-channels[1]:
				case <-channels[2]:
				case <-or(append(channels[3:], orDone)...):
				}
			}
		}()

		return orDone
	}

	sig := func(delay time.Duration) <-chan interface{} {
		ch := make(chan interface{})
		go func() {
			time.Sleep(delay)
			close(ch)
		}()
		return ch
	}

	fmt.Println("Start")
	start := time.Now()
	<-or(
		sig(time.Second*2),
		sig(time.Second*3),
		sig(time.Second*5),
		sig(time.Second*10),
	)
	fmt.Printf("Finish after %v", time.Since(start))
}

func wrongCheckStatus() {
	checkStatus := func(done <-chan interface{}, urls []string) <-chan *http.Response {
		responses := make(chan *http.Response)

		go func() {
			defer close(responses)
			defer fmt.Println("done...")
			for _, url := range urls {
				res, err := http.Get(url)
				if err != nil {
					fmt.Println(err)
					continue
				}

				select {
				case <-done:
					return
				case responses <- res:
				}
			}
		}()
		return responses
	}

	done := make(chan interface{})
	urls := []string{
		"https://google.com",
		"https://youtube.com",
		"https://badhost",
		"https://golang.com",
	}
	for response := range checkStatus(done, urls) {
		fmt.Println(response.Status)
	}
	close(done)
}

type Result struct {
	Error    error
	Response *http.Response
}

func checkStatusWithErrorHandling() {
	checkStatus := func(done <-chan interface{}, urls []string) <-chan Result {
		responses := make(chan Result)

		go func() {
			defer close(responses)
			defer fmt.Println("done...")
			for _, url := range urls {
				res, err := http.Get(url)

				result := Result{
					Error:    err,
					Response: res,
				}

				select {
				case <-done:
					return
				case responses <- result:
				}
			}
		}()
		return responses
	}

	done := make(chan interface{})
	urls := []string{
		"https://google.com",
		"https://google.com",
		"https://youtube.com",
		"https://google.com",
		"https://badhost",
		"https://wikipedia.org",
		"https://google.com",
		"https://badhost",
		"https://youtube.com",
		"https://badhost",
		"https://golang.com",
	}
	for result := range checkStatus(done, urls) {
		if result.Error != nil {
			fmt.Println(result.Error)
			continue
		}
		fmt.Println(result.Response.Status)
	}
	close(done)
}

func concurrentCheckStatus() {
	checkStatus := func(done <-chan interface{}, urls []string) <-chan Result {
		responses := make(chan Result)
		wg := sync.WaitGroup{}
		wg.Add(len(urls))

		go func() {
			wg.Wait()
			fmt.Println("finishing...")
			close(responses)
		}()

		for _, url := range urls {
			go func(url string) {
				defer wg.Done()

				select {
				case <-done:
					return
				default:
					res, err := http.Get(url)
					result := Result{
						Error:    err,
						Response: res,
					}
					responses <- result
				}
			}(url)
		}

		return responses
	}

	done := make(chan interface{})
	urls := []string{
		"https://google.com",
		"https://google.com",
		"https://youtube.com",
		"https://google.com",
		"https://badhost",
		"https://wikipedia.org",
		"https://google.com",
		"https://badhost",
		"https://youtube.com",
		"https://badhost",
		"https://golang.com",
	}
	for result := range checkStatus(done, urls) {
		if result.Error != nil {
			fmt.Println(result.Error)
			continue
		}
		fmt.Println(result.Response.Status)
	}
	close(done)
}

func concurrentCheckStatusWithoutWaitGroup() {
	checkStatus := func(done <-chan interface{}, urls []string) <-chan Result {
		responses := make(chan Result)
		finished := make(chan struct{})

		go func() {
			for i := 0; i < len(urls); i++ {
				<-finished
			}
			fmt.Println("finishing...")
			close(responses)
		}()

		for _, url := range urls {
			go func(url string) {
				select {
				case <-done:
					return
				default:
					res, err := http.Get(url)
					result := Result{
						Error:    err,
						Response: res,
					}
					responses <- result
					finished <- struct{}{}
				}
			}(url)
		}

		return responses
	}

	done := make(chan interface{})
	urls := []string{
		"https://google.com",
		"https://google.com",
		"https://youtube.com",
		"https://google.com",
		"https://badhost",
		"https://wikipedia.org",
		"https://google.com",
		"https://badhost",
		"https://youtube.com",
		"https://badhost",
		"https://golang.com",
	}
	for result := range checkStatus(done, urls) {
		if result.Error != nil {
			fmt.Println(result.Error)
			continue
		}
		fmt.Println(result.Response.Status)
	}
	close(done)
}

func repeat(done <-chan interface{}, value interface{}) <-chan interface{} {
	valStream := make(chan interface{})

	go func() {
		defer close(valStream)
		for {
			select {
			case <-done:
				return
			default:
				valStream <- value
			}
		}
	}()

	return valStream
}

func repeatFn(done <-chan interface{}, fn func() interface{}) <-chan interface{} {
	valStream := make(chan interface{})

	go func() {
		defer close(valStream)
		for {
			select {
			case <-done:
				return
			default:
				valStream <- fn()
			}
		}
	}()

	return valStream
}

func take(done <-chan interface{}, input <-chan interface{}, num int) <-chan interface{} {
	output := make(chan interface{})

	go func() {
		defer close(output)
		for i := 0; i < num; i++ {
			select {
			case <-done:
				return
			default:
				output <- <-input
			}

		}
	}()

	return output
}

func repeatTake() {
	done := make(chan interface{})
	defer close(done)

	for val := range take(done, repeat(done, "hello"), 5) {
		fmt.Println(val)
	}
}

func fanIn(done <-chan interface{}, channels ...<-chan interface{}) <-chan interface{} {
	downStram := make(chan interface{})
	wg := sync.WaitGroup{}

	multiplex := func(ch <-chan interface{}) {
		defer wg.Done()
		for val := range ch {
			select {
			case <-done:
				return
			case downStram <- val:
			}
		}
	}

	wg.Add(len(channels))
	for _, ch := range channels {
		go multiplex(ch)
	}

	go func() {
		wg.Wait()
		close(downStram)
	}()

	return downStram
}

func primeNumber(done <-chan interface{}, intStream <-chan interface{}) <-chan interface{} {
	primeStream := make(chan interface{})

	go func() {
		defer close(primeStream)
		for num := range intStream {
			select {
			case <-done:
				return
			default:
				var n = num.(int)
				var i int
				for i = 2; i < n; i++ {
					if n%i == 0 {
						break
					}
				}
				if i == n {
					primeStream <- num
				}
			}
		}
	}()

	return primeStream
}

func primeFinderSlow() {
	randInt := func() interface{} {
		return rand.Intn(100000000)
	}

	done := make(chan interface{})
	fmt.Println("Primes:")
	start := time.Now()
	for prime := range take(done, primeNumber(done, repeatFn(done, randInt)), 10) {
		fmt.Println(prime)
	}
	fmt.Printf("Elapsed: %v", time.Since(start))
}

func primeFinderFaster() {
	randInt := func() interface{} {
		return rand.Intn(100000000)
	}

	generateFinders := func(
		done <-chan interface{},
		intStream <-chan interface{},
		numFinders int,
	) []<-chan interface{} {

		finders := make([]<-chan interface{}, numFinders)
		for i := 0; i < numFinders; i++ {
			finders[i] = primeNumber(done, intStream)
		}

		return finders
	}

	numFinders := runtime.NumCPU()
	done := make(chan interface{})
	fmt.Printf("Number of finders: %v\n", numFinders)
	fmt.Println("Primes:")
	start := time.Now()
	stream := take(done, fanIn(done, generateFinders(done, repeatFn(done, randInt), numFinders)...), 10)
	for prime := range stream {
		fmt.Println(prime)
	}
	fmt.Printf("Elapsed: %v", time.Since(start))
}

func orDone(done <-chan interface{}, input <-chan interface{}) <-chan interface{} {
	output := make(chan interface{})

	go func() {
		defer close(output)
		for {
			select {
			case <-done:
				return
			default:
				res, err := <-input
				if err == false {
					return
				}
				select {
				case <-done:
				case output <- res:
				}
			}
		}
	}()

	return output
}

func testOrDone() {

	generateNum := func(upTo int) <-chan interface{} {
		numStream := make(chan interface{})

		go func() {
			defer close(numStream)
			for i := 0; i < upTo; i++ {
				numStream <- i
				time.Sleep(time.Millisecond * 100)
			}
		}()
		return numStream
	}

	done := make(chan interface{})

	time.AfterFunc(time.Millisecond*300, func() { close(done) })

	fmt.Println("Numbers:")
	for val := range orDone(done, generateNum(10)) {
		fmt.Println(val)
	}
}

func tee(
	done <-chan interface{},
	in <-chan interface{},
) (
	<-chan interface{},
	<-chan interface{},
) {
	out1 := make(chan interface{})
	out2 := make(chan interface{})

	go func() {
		defer close(out1)
		defer close(out2)

		for val := range orDone(done, in) {
			o1, o2 := out1, out2

			for i := 0; i < 2; i++ {
				select {
				case <-done:
					return
				case o1 <- val:
					o1 = nil
				case o2 <- val:
					o2 = nil
				}
			}
		}
	}()

	return out1, out2
}

func testTee() {

	done := make(chan interface{})

	stream1, stream2 := tee(done, take(done, repeat(done, 5), 10))

	for val := range stream1 {
		fmt.Printf("stream1: %v, stream2: %v\n", val, <-stream2)
	}
}

func bridge(done <-chan interface{}, chanStream <-chan <-chan interface{}) <-chan interface{} {
	valueStream := make(chan interface{})

	go func() {
		defer close(valueStream)

		for {
			select {
			case <-done:
				return
			case stream, ok := <-chanStream:
				if !ok {
					return
				}
				for val := range orDone(done, stream) {
					select {
					case valueStream <- val:
					case <-done:
					}
				}
			}
		}
	}()

	return valueStream
}

func testBridge() {

	genVals := func() <-chan <-chan interface{} {
		chanStream := make(chan (<-chan interface{}))

		go func() {
			defer close(chanStream)

			for i := 0; i < 10; i++ {
				time.Sleep(time.Millisecond * 100)
				ch := make(chan interface{}, 1)
				ch <- i
				chanStream <- ch
				close(ch)
			}
		}()

		return chanStream

	}

	for val := range bridge(nil, genVals()) {
		fmt.Printf("%v ", val)
	}

}

func testContext() {
	wg := sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	locale := func(c context.Context) (string, error) {
		if deadline, ok := c.Deadline(); ok {
			if deadline.Sub(time.Now().Add(time.Minute)) <= 0 {
				return "", context.DeadlineExceeded
			}
		}

		select {
		case <-c.Done():
			return "", c.Err()
		case <-time.After(time.Minute):
		}

		return "EN/US", nil
	}

	genGreeting := func(c context.Context) (string, error) {
		c, cancel := context.WithTimeout(c, time.Second)
		defer cancel()

		switch locale, err := locale(c); {
		case err != nil:
			return "", err
		case locale == "EN/US":
			return "World", nil
		}
		return "", fmt.Errorf("unsupported locale %v", locale)
	}

	genFarewell := func(c context.Context) (string, error) {
		switch locale, err := locale(c); {
		case err != nil:
			return "", err
		case locale == "EN/US":
			return "goodbye!", nil
		}
		return "", fmt.Errorf("unsupported locale %v", locale)
	}

	printGreeting := func(c context.Context) error {
		greeting, err := genGreeting(c)
		if err != nil {
			return err
		}
		fmt.Printf("Hello, %v!", greeting)
		return nil
	}

	printFarewell := func(c context.Context) error {
		farewell, err := genFarewell(c)
		if err != nil {
			return err
		}
		fmt.Printf("%v, World!", farewell)
		return nil
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := printGreeting(ctx); err != nil {
			fmt.Printf("cannot print greeting: %v\n", err)
			cancel()
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := printFarewell(ctx); err != nil {
			fmt.Printf("cannot print farewell: %v\n", err)
		}
	}()

	wg.Wait()
}

type ctxKey int

const (
	nameKey = iota
	tokenKey
)

func contextValue() {

	userName := func(ctx context.Context) string {
		return ctx.Value(nameKey).(string)
	}

	userToken := func(ctx context.Context) string {
		return ctx.Value(tokenKey).(string)
	}

	handleRequest := func(ctx context.Context) {
		name := userName(ctx)
		token := userToken(ctx)

		fmt.Printf("Request by: %v, (%v)\n", name, token)
	}

	processRequest := func(name string, token string) {
		ctx := context.WithValue(context.Background(), nameKey, name)
		ctx = context.WithValue(ctx, tokenKey, token)

		handleRequest(ctx)
	}

	processRequest("Gregory", "authToken")

}

func Chapter4() {
	// channelConfinement()
	// variableConfinement()
	// forSelect()
	// readerCancelation()
	// writerCancellation()
	// orChannel()
	// wrongCheckStatus()
	// checkStatusWithErrorHandling()
	// concurrentCheckStatus()
	// concurrentCheckStatusWithoutWaitGroup()
	// repeatTake()
	// primeFinderSlow()
	// primeFinderFaster()
	// testOrDone()
	// testTee()
	// testBridge()
	// testContext()
	contextValue()
}
