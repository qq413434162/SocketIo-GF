package main

import (
	"context"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gcfg"
	"github.com/gogf/gf/v2/os/gcron"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	socketio "github.com/googollee/go-socket.io"
	"github.com/gorilla/websocket"
	"net/http"
	"socketio-gf/internal/core"
	"socketio-gf/utility/response"
	"time"
)

const (
	RedisPrefix     = "socketio-gf:" //nolint:gofmt
	RedisCursor     = "0"
	RedisScanLimit  = 1000
	EveryMinuteCron  = "0 * * * * *" //nolint:gofmt

	// AckEventName ack的自定义事件名称
	AckEventName    = "ack" //nolint:gofmt

	// BeginAckContent 服务端往客户端的第一次ack内容
	BeginAckContent = "1"
)

var (
	// IsBeginRemoveAll 启动前先清理所有的redis
	IsBeginRemoveAll = gcfg.Instance().MustGet(context.TODO(), "websocket.IsBeginRemoveAll", false).Bool()

	// CheckAliveSecond 只往超过n秒的客户端发送ack
	CheckAliveSecond = gcfg.Instance().MustGet(context.TODO(), "websocket.CheckAliveSecond", 120).Duration() * time.Second

	// CleanupSecond 清理超过n秒没更新的的客户端
	CleanupSecond = gcfg.Instance().MustGet(context.TODO(), "websocket.CleanupSecond", 600).Duration() * time.Second
)

var upGrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func cors(r *ghttp.Request) {
	r.Response.CORSDefault()
	r.Middleware.Next()
}

func removeAll(ctx context.Context) (affect int) {
	rds := g.Redis()
	cursor := RedisCursor
	for {
		keySliceInterface := rds.MustDo(context.TODO(), "SCAN", cursor, "match", RedisPrefix+"*", "count", RedisScanLimit)
		if keySliceInterface.IsNil() {
			break
		}
		cursor = gconv.String(keySliceInterface.Interfaces()[0])
		keySlice := gconv.Strings(keySliceInterface.Interfaces()[1])
		if len(keySlice) > 0 {
			for _, key := range keySlice {
				memberInterface := rds.MustDo(ctx, "DEL", key)
				if memberInterface.Int() == 0 {
					continue
				}
				affect += 1
			}
		}
		if cursor == RedisCursor {
			break
		}
	}
	return
}

func checkIsAlive(ctx context.Context, server *socketio.Server) (broadcastToCount int) {
	rds := g.Redis()
	cursor := RedisCursor
	deadline := gtime.Now().Add(-CheckAliveSecond).Timestamp()
	for {
		keySliceInterface := rds.MustDo(ctx, "SCAN", cursor, "match", RedisPrefix+"*", "count", RedisScanLimit)
		if keySliceInterface.IsNil() {
			break
		}
		cursor = gconv.String(keySliceInterface.Interfaces()[0])
		keySlice := gconv.Strings(keySliceInterface.Interfaces()[1])
		if len(keySlice) > 0 {
			for _, key := range keySlice {
				userId := gconv.Int(gstr.TrimLeftStr(key, RedisPrefix))

				memberSliceInterface := rds.MustDo(ctx, "ZRANGEBYSCORE", key, 0, deadline)
				if memberSliceInterface.IsNil() {
					continue
				}
				memberSlice := memberSliceInterface.Strings()
				if len(memberSlice) < 1 {
					continue
				}
				for _, sid := range memberSlice {
					server.BroadcastToRoom("", sid, AckEventName, BeginAckContent)
					broadcastToCount += 1
					g.Log().Debugf(ctx, "BroadcastToRoom uid %d timestamp %d sid %s", userId, deadline, sid)
				}
			}
		}
		if cursor == RedisCursor {
			break
		}
	}
	return
}

func cleanup(ctx context.Context) (affect int) {
	rds := g.Redis()
	cursor := RedisCursor
	deadline := gtime.Now().Add(-CleanupSecond).Timestamp()
	for {
		keySliceInterface := rds.MustDo(ctx, "SCAN", cursor, "match", RedisPrefix+"*", "count", RedisScanLimit)
		if keySliceInterface.IsNil() {
			break
		}
		cursor = gconv.String(keySliceInterface.Interfaces()[0])
		keySlice := gconv.Strings(keySliceInterface.Interfaces()[1])
		if len(keySlice) > 0 {
			for _, key := range keySlice {
				removeAffect := rds.MustDo(ctx, "ZREMRANGEBYSCORE", key, 0, deadline)
				if removeAffect.IsNil() {
					continue
				}
				currentAffect := removeAffect.Int()
				affect += currentAffect
				g.Log().Debugf(ctx, "ZREMRANGEBYSCORE key %s timestamp %d affect %d", key, deadline, currentAffect)
			}
		}
		if cursor == RedisCursor {
			break
		}
	}
	return
}

func getUserId(ctx context.Context, s socketio.Conn) (userId int) {
	url := s.URL()
	tokenSlice := url.Query()["token"]
	if len(tokenSlice) < 1 {
		g.Log().Warning(ctx, "token is required")
		return
	}
	g.Log().Debugf(ctx, "token:%s", tokenSlice[0])
	// @todo 根据token获取用户id
	userId = gconv.Int(1)
	if userId < 1 {
		g.Log().Warningf(ctx, "uid!:%d", userId)
	}
	return
}

func main() {
	s := g.Server()
	rds := g.Redis()
	cfg := gcfg.Instance()

	// 是否启动时清理现有的redis内容
	if IsBeginRemoveAll {
		key := RedisPrefix+"removeAll"
		if !rds.MustDo(context.TODO(), "GET", key).IsNil() {
			return
		}
		rds.MustDo(context.TODO(), "SETEX", key, 30, 1)
		affect := removeAll(context.TODO())
		g.Log().Infof(context.TODO(), "remove redis key %d", affect)
	}

	server := socketio.NewServer(nil)

	// 多实例部署需要
	_, err := server.Adapter(&socketio.RedisAdapterOptions{
		Addr: cfg.MustGet(context.TODO(), "redis.default.address", "").String(),
		Prefix: "socket.io",
		DB: cfg.MustGet(context.TODO(), "redis.default.db", "").Int(),
	})
	if err != nil {
		g.Log().Error(context.TODO(), err)
		return
	}

	s.BindMiddlewareDefault(cors)
	s.BindHandler("/socket.io/", func(r *ghttp.Request) {
		g.Log().Info(r.GetCtx(), "begin")
		server.ServeHTTP(r.Response.Writer, r.Request)
		_, err = upGrader.Upgrade(r.Response.ResponseWriter, r.Request, r.Response.Header())
		if err != nil {
			return
		}
	})

	server.OnConnect("/", func(s socketio.Conn) (err error) {
		ctx := context.TODO()
		sid := s.ID()
		s.SetContext("")
		now := gtime.Timestamp()
		g.Log().Infof(ctx, "connected:%s", sid)

		uid := getUserId(ctx, s)
		if uid < 1 {
			return
		}

		g.Log().Debugf(ctx, "ZADD uid %d %d %s", uid, now, sid)
		rds.MustDo(ctx, "ZADD", RedisPrefix+gconv.String(uid), now, sid)

		return nil
	})

	server.OnError("/", func(s socketio.Conn, e error) {
		detailError, ok := e.(*websocket.CloseError)
		if ok {
			switch detailError.Code {
			// 正常错误
			case websocket.CloseGoingAway:
				return
			// nginx的proxy_timeout
			case websocket.CloseAbnormalClosure:
				return
			}
		}
		g.Log().Info(context.TODO(), "meet error:", s.ID(), e)
	})

	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		ctx := context.TODO()
		sid := s.ID()
		g.Log().Info(ctx, "closed", reason)

		uid := getUserId(ctx, s)
		if uid < 1 {
			return
		}

		g.Log().Debugf(ctx, "ZREM uid %d %s", uid, sid)
		rds.MustDo(ctx, "ZREM", RedisPrefix+gconv.String(uid), sid)
	})

	// 客户端收到广播的回应事件
	server.OnEvent("", AckEventName, func(s socketio.Conn, msg string) (result string) {
		ctx := context.TODO()
		sid := s.ID()
		now := gtime.Timestamp()
		g.Log().Debugf(context.TODO(), "accept %s client ACK with data: %s", sid, msg)
		result = gconv.String(gconv.Int(msg) + 1)

		uid := getUserId(ctx, s)
		if uid < 1 {
			return
		}

		g.Log().Debugf(ctx, "ZADD %d uid %s %d %s", uid, AckEventName, now, sid)
		rds.MustDo(ctx, "ZADD", RedisPrefix+gconv.String(uid), now, sid)

		return
	})

	server.OnEvent("/", "notice", func(s socketio.Conn, msg string) {
		g.Log().Info(context.TODO(), "notice:", msg)
		s.Emit("reply", "have "+msg)
	})

	server.OnEvent("/chat", "msg", func(s socketio.Conn, msg string) string {
		s.SetContext(msg)
		return "recv " + msg
	})

	server.OnEvent("/", "bye", func(s socketio.Conn) string {
		last := s.Context().(string)
		s.Emit("bye", last)
		s.Close()
		return last
	})

	go func() {
		if err = server.Serve(); err != nil {
			g.Log().Errorf(context.TODO(), "socketio listen error: %s\n", err)
		}
	}()
	defer server.Close()

	// 定时广播
	gcron.AddSingleton(
		context.TODO(),
		gcfg.Instance().MustGet(context.TODO(), "websocket.CheckAliveCron", EveryMinuteCron).String(),
		func(ctx context.Context) {
			key := RedisPrefix+"checkIsAlive"
			if !rds.MustDo(ctx, "GET", key).IsNil() {
				return
			}
			rds.MustDo(ctx, "SETEX", key, 30, 1)
			g.Log().Debug(ctx, "start check client isAlive")
			affect := checkIsAlive(ctx, server)
			g.Log().Debugf(ctx, "end check client isAlive send sid num %d", affect)
	}, "CheckIsAlive")

	// 定时清理无效的数据
	gcron.AddSingleton(
		context.TODO(),
		gcfg.Instance().MustGet(context.TODO(), "websocket.CleanupCron", EveryMinuteCron).String(),
		func(ctx context.Context) {
			key := RedisPrefix+"cleanup"
			if !rds.MustDo(ctx, "GET", key).IsNil() {
				return
			}
			rds.MustDo(ctx, "SETEX", key, 30, 1)

			g.Log().Debug(ctx, "start cleanup unused redis")
			affect := cleanup(ctx)
			g.Log().Debugf(ctx, "end cleanup unused redis affect members %d", affect)
	}, "CleanUpRedis")

	// 通知广播接口
	s.BindHandler(
		"POST:/push/",
		func(r *ghttp.Request) {
			ctx := r.GetCtx()
			data, e := r.GetJson()
			if e != nil {
				g.Log().Error(ctx, e)
				return
			}
			userId := data.Get("user_id", []int{}).Ints()
			event := data.Get("event", "").String()
			message := gjson.New(`{"data":` + data.GetJson("data", "{}").String() + `}`)

			g.Log().Debugf(ctx, "push uid %v", userId)
			for _, i := range userId {
				sidSlice := rds.MustDo(ctx, "ZRANGE", RedisPrefix+gconv.String(i), 0, -1)
				if sidSlice.IsNil() {
					continue
				}

				g.Log().Debugf(ctx, "push uid %d sid %v", i, sidSlice)
				for _, sid := range sidSlice.Slice() {
					server.BroadcastToRoom("", gconv.String(sid), event, message)
				}
			}

			r.Response.WriteJson(response.JsonResponse{
				Result:  200,
				Message: "success",
			})
		},
	)

	// ping接口
	s.BindHandler(
		"/ping",
		func(r *ghttp.Request) {
			r.Response.Writeln("pong")
		},
	)

	ctx := context.TODO()

	// 链路追踪初始化
	if core.Config.Tracer.Enable {
		flush, e := core.InitJaeger(
			core.Config.AppName,
			core.Config.Tracer.Address,
		)
		if e != nil {
			g.Log().Fatal(ctx, e)
		}
		defer func() {
			err = flush.Shutdown(ctx)
		}()
	}

	s.AddStaticPath("/public", "resource/public/html")
	s.Group("/", func(group *ghttp.RouterGroup) {
		// 格式返回统一处理
		group.Middleware(core.Middleware.HandlerResponse)
	})
	s.Run()
}
