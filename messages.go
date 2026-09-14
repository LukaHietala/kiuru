package main

type MessageLog struct {
	active  string
	history []string
}

func NewMessageLog() *MessageLog {
	return &MessageLog{}
}

func (m *MessageLog) Post(msg string) {
	m.active = msg
	m.history = append(m.history, msg)
}

func (m *MessageLog) Clear() {
	m.active = ""
}

func (m *MessageLog) Current() string {
	return m.active
}
