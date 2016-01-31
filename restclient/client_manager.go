package restclient

type ClientManager struct {
	client Client
}

var clientManager *ClientManager

func init() {
	clientManager = &ClientManager{}
}

func RegisterClient(client Client) {
	clientManager.client = client
}

func GetClient() Client {
	return clientManager.client
}
