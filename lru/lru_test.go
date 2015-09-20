package lru

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"testing"

	"bitbucket.org/creachadair/cache/value"
)

type evalue string

func (evalue) Size() int { return 1 }

func TestCapacity(t *testing.T) {
	var victim string
	c := New(3, OnEvict(func(v value.Interface) { // # entries
		victim = string(v.(evalue))
	}))
	tests := []struct {
		op, id, value string
		victim        string
	}{
		{"+", "x", "abc", ""},                 // add x
		{"+", "y", "defghij", ""},             // add y
		{"?", "x", "abc", ""},                 // hit
		{"+", "z", "123456", ""},              // add z
		{"+", "x", "ABC", "abc"},              // replace x
		{"?", "y", "defghij", ""},             // hit
		{"?", "x", "ABC", ""},                 // hit
		{"?", "z", "123456", ""},              // hit
		{"+", "e", "qqq", "defghij"},          // evict y
		{"?", "y", "", ""},                    // miss
		{"?", "x", "ABC", ""},                 // hit
		{"+", "m", "123456789", "123456"},     // evict z
		{"?", "z", "", ""},                    // miss
		{"?", "x", "ABC", ""},                 // hit
		{"?", "e", "qqq", ""},                 // hit
		{"?", "m", "123456789", ""},           // hit
		{"?", "q", "", ""},                    // miss
		{"?", "e", "qqq", ""},                 // hit
		{"+", "k", "0123456789abcdef", "ABC"}, // evict x
		{"?", "k", "0123456789abcdef", ""},    // hit
		{"?", "m", "123456789", ""},           // hit
		{"?", "x", "", ""},                    // miss
		{"?", "e", "qqq", ""},                 // hit
	}
	for _, test := range tests {
		victim = ""
		t.Logf("before %s %q: %s", test.op, test.id, c.seq)
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
		t.Logf(" after %s %q: %s; victim=%q", test.op, test.id, c.seq, victim)
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

func (e *entry) String() string {
	var buf bytes.Buffer
	for cur := e.next; ; cur = cur.next {
		v := cur.value
		if v == nil {
			v = evalue("")
		}
		fmt.Fprintf(&buf, "%q [%s] ", cur.id, string(v.(evalue)))
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
