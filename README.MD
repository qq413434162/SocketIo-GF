# SocketIo-GF

# Description
Scaffolding system combined with SocketIO and GoFrame framework

> SocketIO:github.com/googollee/go-socket.io <br/> GoFrame:github.com/gogf/gf

# Feature
- Using redis as the storage layer for user records
- The server will actively broadcast the client
    - in order to update the user's last action time
    - clean up client data that has not been used for a long time