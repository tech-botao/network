# network

> 网络相关的函数写到这里
> api，ws有很多用法，统一起来写一下
> 参考Goex的写法

## http

```golang

api := network.NewAPI()


// 扩充写法

api2 struct {
    *newwork.API
    handler func() []byte
}
```

## websocket

```golang

func ExampleHbg() {

	ctx, cancel := context.WithCancel(context.Background())
	client := NewBuilder().URL("wss://api.huobi.pro/ws"). // 链接
		Subs([]string{`{"id": "id1", "sub": "market.btcusdt.kline.1min"}`,}). // 启动时订阅的频道
        ReadDeadLineTime(t time.Duration). // 如果长时间没数据的时候， 自动重启
		Dump(). // 服务器信息显示
		AutoReconnect(). // 自动重连
		Build(ctx)

    // 接受到数据后的操作
	client.MessageFunc = func(msg []byte) error {
		if strings.Contains(string(msg), "ping") {
			pong := strings.Replace(string(msg),"ping", "pong", 1)
			pp.Println(pong)
			return client.WriteMessage(websocket.TextMessage, []byte(pong))
		}
		logger.Info("[ws] receive message", msg)
		return nil
	}

    // 建立链接
	err := client.Connect()
	if err != nil {
		pp.Println(err)
	}

    // 另一个线程中启动数据监听
	go client.ReceiveMessage()

    // 另一个线程杀掉这个程序
	go func() {
		time.Sleep(100 * time.Second)
		cancel()
	}()

    // 关闭链接:w

	select {
	case <-ctx.Done():
		client.Close()
	}

	// output:

}
```

## rpc <TODO>

## decode

