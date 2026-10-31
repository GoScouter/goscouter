package terminal

import "sync"

// keyBuffer holds keyboard input until the line editor reads it. Unlike a pipe,
// writing never blocks, so mouse events keep being handled while a command
// runs and nobody is reading keys.
type keyBuffer struct {
	mu   sync.Mutex
	cond *sync.Cond
	data []byte
	err  error
}

func newKeyBuffer() *keyBuffer {
	k := &keyBuffer{}
	k.cond = sync.NewCond(&k.mu)
	return k
}

func (k *keyBuffer) Read(p []byte) (int, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	for len(k.data) == 0 && k.err == nil {
		k.cond.Wait()
	}
	if len(k.data) == 0 {
		return 0, k.err
	}
	n := copy(p, k.data)
	k.data = k.data[n:]
	return n, nil
}

func (k *keyBuffer) write(s string) {
	k.mu.Lock()
	k.data = append(k.data, s...)
	k.mu.Unlock()
	k.cond.Signal()
}

func (k *keyBuffer) closeWithError(err error) {
	k.mu.Lock()
	k.err = err
	k.mu.Unlock()
	k.cond.Broadcast()
}
