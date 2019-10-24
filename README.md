# ssh-webclient

client.go has the ssh client implementation using the go ssh library. It exposes connect method.
server.go has the websocket implementaition using the go gorilla websocket library. It utilizes the connect method to create
ssh client on-demand and stream the piped input and output through the websocket
