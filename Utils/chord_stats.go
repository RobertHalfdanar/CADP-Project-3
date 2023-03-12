package Utils

import "sync"

// Stats is a struct that holds all the information related to a messaging node
type Stats struct {
	theLock     sync.Mutex
	Sent        uint32
	Received    uint32
	Relayed     uint32
	SentSum     int64
	ReceivedSum int64
}

// IncrementSent locks the struct and increments the Sent variable
func (s *Stats) IncrementSent() {
	s.theLock.Lock()
	defer s.theLock.Unlock()
	s.Sent++
}

// IncrementReceived locks the struct and increments the Received variable
func (s *Stats) IncrementReceived() {
	s.theLock.Lock()
	defer s.theLock.Unlock()
	s.Received++
}

// IncrementRelayed locks the struct and increments the Relayed variable
func (s *Stats) IncrementRelayed() {
	s.theLock.Lock()
	defer s.theLock.Unlock()
	s.Relayed++
}

// AddSentSum locks the struct and increments the SentSum variable
func (s *Stats) AddSentSum(sum int64) {
	s.theLock.Lock()
	defer s.theLock.Unlock()
	s.SentSum += sum
}

// AddReceivedSum locks the struct and increments the ReceivedSum variable
func (s *Stats) AddReceivedSum(sum int64) {
	s.theLock.Lock()
	defer s.theLock.Unlock()
	s.ReceivedSum += sum
}

// Clear locks the struct and resets all the variables to 0
func (s *Stats) Clear() {
	s.theLock.Lock()
	defer s.theLock.Unlock()

	s.Sent = 0
	s.Received = 0
	s.Relayed = 0
	s.SentSum = 0
	s.ReceivedSum = 0
}
