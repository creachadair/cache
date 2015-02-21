package lru

import (
	"bytes"
	"fmt"
	"sync"
	"testing"
)

func TestCapacity(t *testing.T) {
	c := New(16)
	tests := []struct {
		op, id, value string
		size          int
	}{
		{"+", "x", "abc", 3},
		{"+", "y", "defghij", 10},
		{"?", "x", "abc", 10}, // hit
		{"+", "z", "123456", 16},
		{"+", "x", "ABC", 16},
		{"?", "y", "defghij", 16},          // hit
		{"?", "x", "ABC", 16},              // hit
		{"?", "z", "123456", 16},           // hit
		{"+", "e", "qqq", 12},              // evict y
		{"?", "y", "", 12},                 // miss
		{"?", "x", "ABC", 12},              // hit
		{"+", "m", "123456789", 15},        // evict z
		{"?", "z", "", 15},                 // miss
		{"?", "x", "ABC", 15},              // hit
		{"?", "e", "qqq", 15},              // hit
		{"?", "m", "123456789", 15},        // hit
		{"?", "q", "", 15},                 // miss
		{"?", "e", "qqq", 15},              // hit
		{"+", "k", "0123456789abcdef", 16}, // evict x, m, e
		{"?", "k", "0123456789abcdef", 16}, // hit
		{"?", "e", "", 16},                 // miss
	}
	for _, test := range tests {
		t.Logf("before %s %q: %s", test.op, test.id, c.seq)
		switch test.op {
		case "+":
			c.Put(test.id, []byte(test.value))
		case "?":
			got := string(c.Get(test.id))
			if got != test.value {
				t.Errorf("Get %q: got %q, want %q", test.id, got, test.value)
			}
		default:
			t.Fatalf("Invalid test: %+v", test)
		}
		if c.size != test.size {
			t.Errorf("Size after %s %q: got %d, want %d", test.op, test.id, c.size, test.size)
		}
		t.Logf(" after %s %q: %s", test.op, test.id, c.seq)
	}
}

func TestConcurrency(t *testing.T) {
	const numWorkers = 16

	c := New(1000)
	ch := make(chan string)
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		v := bytes.Repeat([]byte{'A' + byte(i)}, 274)
		wg.Add(1)
		go func() {
			defer wg.Done()
			for key := range ch {
				switch key[0] {
				case '+':
					c.Put(key[1:], v)
				case '?':
					c.Get(key[1:])
				case '*':
					c.Reset()
				}
				if n := c.Size(); n < 0 || n > c.cap {
					t.Errorf("Size %d out of range [0..%d]", n, c.cap)
				}
			}
		}()
	}

	keys := []string{"alpha", "bravo", "charlie", "delta", "echo", "foxtrot", "golf", "hotel", "india"}
	for i := 0; i < 1000; i++ {
		key := keys[i%len(keys)]
		var op string
		switch v := i % 100; {
		case v == 99:
			op = "*"
		case v < 50:
			op = "+"
		default:
			op = "?"
		}
		ch <- op + key
	}
	close(ch)
	wg.Wait()
}

func (e *entry) String() string {
	var buf bytes.Buffer
	for cur := e.next; ; cur = cur.next {
		fmt.Fprintf(&buf, "%q [%q] ", cur.id, string(cur.data))
		if cur.prev.next == cur {
			fmt.Fprint(&buf, "✓ ")
		} else {
			fmt.Fprint(&buf, "✗ ")
		}
		if cur == e {
			break
		}
	}
	return buf.String()
}
