# Neuron

一款强大的HTTP / Websocket服务器，可用于开发微服务框架，IoT，分布式爬网程序。

## 特点

* 「仅使用Go标准库」实现HTTP服务器及HTTP静态文件服务器，启动后自动创建static目录及index.html

* 生命周期明晰，微服务架构，可在配置文件中开关服务，启动后自动创建config.json

  ```go
  "AutorunConfig": {
      # 端口转发服务
      "ADProxy": true,
      # RPC & MQ 服务端
      "ADCommander": true,
      # RPC & MQ 客户端
      "ADReceiver": true,
      # Example -> 分布式爬虫服务端
      "SDExamplePublish": true,
      # Example -> 分布式爬虫客户端
      "SDExampleSubscribe": true
  }
  ```

* 自带HTTPS实现

  ```go
  "HTTPS": {
      # 开关
      "Open": false,
      # 服务端口号（通常为443）
      "TLSPort": 8443,
      # 对应根目录下证书文件 -> ./tls/tls.crt & ./tls/tls.key
      "TLSCertPath": "/tls/tls" 
  }
  ```

* 自带代理及端口转发（支持TCP2TCP & TCP2UDP & UDP2UDP & UDP2TCP & UART2UDP）

```go
{
    "ProxyHub": {
        "TCP": {
            # 主机192.168.1.100的6666端口转发到本机的6666端口
            "0.0.0.0:6666":"192.168.1.100:6666",
            # 主机192.168.1.100的7777端口转发到本机的7777端口
            "0.0.0.0:7777":"192.168.1.100:7777"
        },
        "UDP": {},
        "TCP2UDP": {},
        "UDP2TCP": {},
        "UART2UDP": {
            # 主机RS232或者RS485串口转发到本机UDP协议55555端口
            "0.0.0.0:55555": {
                "PortName": "/dev/ttyAMA0",
                "BaudRate": 115200,
                "DataBits": 8,
                "StopBits": 1,
                "MinimumReadSize": 4
            }
        }
    }
}
```
