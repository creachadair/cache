package lfu

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/creachadair/cache"
)

type evalue string

func (evalue) Size() int { return 1 }

func TestCapacity(t *testing.T) {
	var victim string
	c := New(3, OnEvict(func(v cache.Value) { // # entries
		victim = string(v.(evalue))
	}))
	tests := []struct {
		op, id, value string
		victim        string
	}{
		{"+", "x", "abc", ""},                       // add x
		{"+", "y", "defghij", ""},                   // add y
		{"?", "x", "abc", ""},                       // hit
		{"+", "z", "123456", ""},                    // add z
		{"+", "x", "ABC", "abc"},                    // replace x
		{"?", "y", "defghij", ""},                   // hit
		{"?", "x", "ABC", ""},                       // hit
		{"+", "e", "qqq", "123456"},                 // evict z
		{"?", "z", "", ""},                          // miss
		{"?", "x", "ABC", ""},                       // hit
		{"+", "m", "123456789", "qqq"},              // evict e
		{"?", "e", "", ""},                          // miss
		{"?", "x", "ABC", ""},                       // hit
		{"?", "y", "defghij", ""},                   // hit
		{"?", "m", "123456789", ""},                 // hit
		{"?", "q", "", ""},                          // miss
		{"+", "k", "0123456789abcdef", "123456789"}, // evict m
		{"?", "k", "0123456789abcdef", ""},          // hit
		{"?", "m", "", ""},                          // miss
		{"?", "k", "0123456789abcdef", ""},          // hit
		{"?", "y", "defghij", ""},                   // hit
		{"?", "x", "ABC", ""},                       // hit
	}
	for _, test := range tests {
		victim = ""
		t.Logf("before %s %q: %s", test.op, test.id, eheap(c.heap))
		switch test.op {
		case "+":
			c.Put(test.id, evalue(test.value))
		case "?":
			got := c.Get(test.id)
			if got == nil {
				got = evalue("")
			}
			if got != evalue(test.value) {
				t.Errorf("Get %q: got %q, want %q", test.id, got, test.value)
			}
		default:
			t.Fatalf("Invalid test: %+v", test)
		}
		if test.victim != "" && victim != test.victim {
			t.Errorf("Victim after %s %q: got %q, want %q", test.op, test.id, victim, test.victim)
		}
		t.Logf(" after %s %q: %s; victim=%q", test.op, test.id, eheap(c.heap), victim)
	}
}

func TestConcurrency(t *testing.T) {
	const numWorkers = 16

	c := New(1000)
	ch := make(chan string)
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		v := strings.Repeat(string('A'+byte(i)), 274)
		wg.Add(1)
		go func() {
			defer wg.Done()
			for key := range ch {
				switch key[0] {
				case '+':
					c.Put(key[1:], evalue(v))
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

func TestEmpties(t *testing.T) {
	for _, c := range []*Cache{nil, New(0)} {
		if size := c.Size(); size != 0 {
			t.Errorf("Size(nil): got %d, want 0", size)
		}
		if cap := c.Cap(); cap != 0 {
			t.Errorf("Cap(nil): got %d, want 0", cap)
		}
		c.Put("foo", evalue("bar")) // shouldn't crash...
		// ...but also shouldn't store anything
		if v := c.Get("foo"); v != nil {
			t.Errorf("Get(foo): got %q, want nil", v)
		}
		c.Reset() // shouldn't crash
	}
}

type eheap []*entry

func (e eheap) String() string {
	if len(e) == 0 {
		return "<empty>"
	}
	var buf bytes.Buffer
	for _, elt := range e {
		v := elt.value
		if v == nil {
			v = evalue("")
		}
		fmt.Fprintf(&buf, "%q#%d [%s] ", elt.id, elt.uses, string(v.(evalue)))
	}
	return buf.String()
}

func ExampleNew() {
	c := New(200)
	c.Put("x", cache.Nil)
	c.Put("y", cache.Nil)
	if v := c.Get("x"); v != nil {
		fmt.Println("x is present")
	} else {
		fmt.Println("x is absent")
	}
	// Output: x is present
}
