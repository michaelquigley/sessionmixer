package sessionmixer

// Number is a constraint for numeric types that can be used with RingBuffer
type Number interface {
	~int | ~int64 | ~float32 | ~float64
}

// RingBuffer is a fixed-size circular buffer that supports average calculation
type RingBuffer[T Number] struct {
	data  []T
	head  int
	count int
	size  int
}

// NewRingBuffer creates a new ring buffer with the specified size
func NewRingBuffer[T Number](size int) *RingBuffer[T] {
	return &RingBuffer[T]{
		data: make([]T, size),
		size: size,
	}
}

// Push adds a value to the ring buffer, overwriting the oldest value if full
func (rb *RingBuffer[T]) Push(value T) {
	rb.data[rb.head] = value
	rb.head = (rb.head + 1) % rb.size
	if rb.count < rb.size {
		rb.count++
	}
}

// Average returns the simple mean of all values in the buffer
func (rb *RingBuffer[T]) Average() T {
	if rb.count == 0 {
		return 0
	}
	var sum T
	for i := 0; i < rb.count; i++ {
		sum += rb.data[i]
	}
	return sum / T(rb.count)
}

// IsEmpty returns true if the buffer contains no values
func (rb *RingBuffer[T]) IsEmpty() bool {
	return rb.count == 0
}
