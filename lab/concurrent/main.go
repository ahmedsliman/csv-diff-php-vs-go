package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

var (
	mu      sync.Mutex
	rwMutex sync.RWMutex
)
var TotalNumbersProcessed = 1_000_000

// https://dev.to/kittipat1413/concurrency-in-go-a-practical-guide-with-hands-on-examples-37od
// https://www.freecodecamp.org/news/concurrent-programming-in-go/

func readCounter(id int, counter int, wg *sync.WaitGroup) {
	defer wg.Done()
	rwMutex.RLock()
	defer rwMutex.RUnlock()
	fmt.Printf("Goroutine %d read counter: %d\n", id, counter)
	time.Sleep(time.Millisecond * time.Duration(rand.Intn(100)))
}

func writeCounter(id int, counter int, wg *sync.WaitGroup) {
	defer wg.Done()
	rwMutex.Lock()
	defer rwMutex.Unlock()
	counter++
	fmt.Printf("Goroutine %d incremented counter to: %d\n", id, counter)
	time.Sleep(time.Millisecond * time.Duration(rand.Intn(100)))
}

func main_rwlock() {
	rand.Seed(time.Now().UnixNano())
	var wg sync.WaitGroup
	var counter = 0

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go writeCounter(i, counter, &wg)
	}

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go readCounter(i, counter, &wg)
	}

	wg.Wait()
	fmt.Println("Final counter value:", counter)
}

// multi
func multi_main_safe() {
	start := time.Now()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	numbers := make([]int, TotalNumbersProcessed)
	for i := 0; i < TotalNumbersProcessed; i++ {
		numbers[i] = r.Intn(1000)
	}

	numCh := make(chan int, 1000)
	doneCh := make(chan bool, 4)
	var counter int
	var mu sync.Mutex

	worker := func(id int, numCh <-chan int, doneCh chan<- bool, mu *sync.Mutex, counter *int) {
		for num := range numCh {
			time.Sleep(time.Duration(num))
			mu.Lock()
			*counter += 1
			mu.Unlock()
		}
		doneCh <- true
	}

	for i := 0; i < 4; i++ {
		go worker(i, numCh, doneCh, &mu, &counter)
	}

	for _, num := range numbers {
		numCh <- num
	}
	close(numCh)

	for i := 0; i < 4; i++ {
		<-doneCh
	}

	fmt.Println("Total counter:", counter)
	fmt.Printf("Execution time: %s\n", time.Since(start))
}

// multi
func multi_main_not_safe() {
	start := time.Now()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	numbers := make([]int, TotalNumbersProcessed)
	for i := 0; i < TotalNumbersProcessed; i++ {
		numbers[i] = r.Intn(1000)
	}

	numCh := make(chan int, 1000)
	doneCh := make(chan bool, 4)
	var counter int

	for i := 0; i < 4; i++ {
		go func(id int) {
			for num := range numCh {
				time.Sleep(time.Duration(num))
				counter += 1
			}
			doneCh <- true
		}(i)
	}

	for _, num := range numbers {
		numCh <- num
	}
	close(numCh)

	for i := 0; i < 4; i++ {
		<-doneCh
	}

	fmt.Println("Total counter:", counter)
	fmt.Printf("Execution time: %s\n", time.Since(start))
}

// seq
func main_seq() {
	start := time.Now()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	numbers := make([]int, TotalNumbersProcessed)
	for i := 0; i < TotalNumbersProcessed; i++ {
		numbers[i] = r.Intn(1000)
	}

	counter := 0
	for _, num := range numbers {
		counter++
		time.Sleep(time.Duration(num))
	}
	fmt.Println("counter:", counter)
	fmt.Printf("Execution time: %s\n", time.Since(start))
}

func main() {
	// main_seq()
	// multi_main_not_safe()
	// multi_main_safe()
	main_rwlock()
}
