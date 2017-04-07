package conn

// Data represents data that is retrieved from a websocket connection.
// If the connection closed, EOS field fill with true.
type Data struct {
	Message Message
	EOS     bool
}
