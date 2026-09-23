package robot
type MessageHandler interface {
	HandleMessage(platform, userID, text string) string
}
