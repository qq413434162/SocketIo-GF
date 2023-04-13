# SocketIo-GF

# Description
Scaffolding system combined with SocketIO and GoFrame framework

> SocketIO:github.com/googollee/go-socket.io <br/> GoFrame:github.com/gogf/gf

# Feature
- Using redis as the storage layer for user records
- The server will actively broadcast the client
    - in order to update the user's last action time
    - clean up client data that has not been used for a long time
  
# Nginx
```text
location /websocket/ {
    proxy_pass {{path}};

    proxy_redirect    off;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;

    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection $connection_upgrade;
}
```