package dispatcher

type publishMessage struct {
	Text  string `json:"text"`
	Icon  string `json:"icon,omitempty"`
	Color string `json:"color,omitempty"`
}
