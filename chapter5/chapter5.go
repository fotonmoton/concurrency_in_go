package chapter5

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"runtime/debug"
	"sync"
	"time"
)

type MyError struct {
	Inner   error
	Message string
	Stack   string
	Misc    map[string]interface{}
}

type LowLevelError struct {
	error
}

type IntermediateError struct {
	error
}

func (e MyError) Error() string {
	return e.Message
}

func errorPropagation() {

	handleError := func(id int, msg string, err error) {
		log.SetPrefix(fmt.Sprintf("[%v]:", id))
		log.Printf("%#v", err)
		fmt.Printf("%s", msg)
	}

	wrapError := func(err error, messagef string, messagArgs ...interface{}) error {
		return MyError{
			err,
			fmt.Sprintf(messagef, messagArgs...),
			string(debug.Stack()),
			make(map[string]interface{}),
		}
	}

	isGloballyExec := func(name string) (bool, error) {
		info, err := os.Stat(name)

		if err != nil {
			return false, LowLevelError{wrapError(err, err.Error())}
		}

		return info.Mode().Perm()&0100 == 0100, nil
	}

	runJob := func(id string, binary string) error {
		isExecutable, err := isGloballyExec(binary)

		if err != nil {
			return IntermediateError{
				wrapError(err, "[%v] binary '%v' not avaliable", id, binary),
			}
		}

		if !isExecutable {
			return wrapError(err, "[%v] binary '%v' is not an executable")
		}

		return exec.Command(binary, "--id="+id).Run()
	}

	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ltime | log.LUTC)

	err := runJob("1", "/wrong/bin")
	if err != nil {
		msg := "unknown error, file a bug"
		if _, ok := err.(IntermediateError); ok {
			msg = err.Error()
		}
		handleError(1, msg, err)
	}
}

func heartbeat() {
	doWork := func(done <-chan interface{}, rate time.Duration) (<-chan interface{}, <-chan time.Time) {
		heartbeat := make(chan interface{})
		resultStream := make(chan time.Time)

		go func() {
			// do not close any of channels after 2 iterations
			// defer close(heartbeat)
			// defer close(resultStream)

			pulse := time.Tick(rate)
			work := time.Tick(rate * 3)

			makeBeat := func() {
				select {
				case heartbeat <- struct{}{}:
				default:
				}
			}

			sendWork := func(w time.Time) {
				for {
					select {
					case <-done:
						return
					case <-pulse:
						makeBeat()
					case resultStream <- w:
						return
					}
				}
			}

			// simulating panic
			for i := 0; i < 2; i++ {
				select {
				case <-done:
					return
				case <-pulse:
					makeBeat()
				case w := <-work:
					sendWork(w)
				}
			}

		}()

		return heartbeat, resultStream
	}

	timeout := time.Second
	done := make(chan interface{})

	time.AfterFunc(time.Second*10, func() { close(done) })

	heartbeat, results := doWork(done, timeout/2)

	for {
		select {
		case <-time.After(timeout):
			fmt.Printf("unhealthy delay!")
			return
		case _, ok := <-heartbeat:
			if !ok {
				return
			}
			fmt.Println("beat")
		case result, ok := <-results:
			if !ok {
				return
			}
			fmt.Printf("%v\n", result.Second())
		}
	}
}

func replicatedRequests() {
	doWork := func(done <-chan interface{}, id int, wg *sync.WaitGroup, result chan<- int) {
		defer wg.Done()

		start := time.Now()
		loadTime := time.Duration((1 + rand.Intn(5)) * int(time.Second))
		load := time.After(loadTime)

		select {
		case <-done:
		case <-load:
		}

		select {
		case <-done:
		case result <- id:
		}

		took := time.Since(start)

		if took < loadTime {
			took = loadTime
		}

		fmt.Printf("[%v]: %v\n", id, took)
	}

	done := make(chan interface{})
	result := make(chan int)
	wg := sync.WaitGroup{}

	wg.Add(10)
	for i := 0; i < 10; i++ {
		go doWork(done, i, &wg, result)
	}
	first := <-result
	close(done)
	wg.Wait()
	fmt.Printf("First to finish: %v", first)
}

func rateLimiting() {
	defer fmt.Println("done")

	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ltime | log.LUTC)

	api := Open()
	wg := sync.WaitGroup{}

	wg.Add(20)

	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			if err := api.ReadFile(context.Background()); err != nil {
				log.Fatalln("can't read file")
			}
			log.Println("reading file")
		}()
	}

	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			if err := api.ReadFile(context.Background()); err != nil {
				log.Fatalln("can't resolve domain")
			}
			log.Println("resolving domain")
		}()
	}

	wg.Wait()
}

func Chapter5() {
	// errorPropagation()
	// heartbeat()
	// replicatedRequests()
	rateLimiting()
}
