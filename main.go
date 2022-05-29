package main

import (
	"flag"
	"fmt"
	"github.com/golang/glog"
	"gopkg.in/yaml.v2"
	"net"
	"os"
	"strings"
	"sync"
)

type ProxyConfig struct {
	Proxy [] ProxySt `yaml:"proxy"`
}


type ProxySt struct {
	Host string `yaml:"host"`
	Port []string `yaml:"port"`
}


func ReadYamlConfig(path string) (*ProxyConfig,error){
	conf := &ProxyConfig{}
	f, err := os.Open(path)

	if err != nil {
		return nil,err
	}
	defer f.Close()

	err = yaml.NewDecoder(f).Decode(conf)
	if err != nil {
		return nil,err
	}
	return conf,nil

}

func proxy( a net.Conn , b net.Conn , closed chan bool) {
	buff := make( []byte , 2048)
	for{

		n , err := a.Read( buff  )
		if err != nil {
			closed <- true
			glog.Errorf("recv data from %v failed,err: %v",a , err  )

			return
		}
		if n == 0 {
			closed <- true
			glog.Errorf("recv data from %v return 0 , remove closed ",a )
			return
		}

		n1, err := b.Write(buff[ :n])
		if err != nil {
			closed <- true
			glog.Errorf("recv data from %v failed,err: %v",a , err  )
			return
		}
		if n != n1 {
			closed <- true
			glog.Errorf("send  data   %v != %d 0 , remove closed ", n1, 1 )
			return
		}

	}


}

func processClientConnect(client net.Conn , ipport string  )    {
	defer client.Close()



	remove, err := net.Dial("tcp", ipport)
	if err != nil {
		glog.Errorf(" Conenct remove %s err %v " , ipport, err )
		return   ;
	}
	defer remove.Close()

	var chans = make(chan bool, 2)
	go proxy(  client, remove , chans)
	go proxy(  remove , client,  chans)
	<- chans
	glog.Infof(" Proxy closed ")
}

func OpenProxy( wg *sync.WaitGroup , localPort string, host string , remotePort string ) {
	// 监听TCP 服务端口

	wg.Add(1)
	defer wg.Done()

	local := fmt.Sprintf("0.0.0.0:%s", localPort)

	listener,err := net.Listen("tcp",local)
	if err != nil {
		glog.Errorf("Listen tcp server failed,err: %d",err)
		return
	}

	ipport := fmt.Sprintf("%s:%s", host, remotePort)

	for{
		// 建立socket连接
		conn,err := listener.Accept()
		if err != nil {
			glog.Errorf("Listen.Accept failed,err:",err)
			continue
		}

		// 业务处理逻辑
		go processClientConnect(conn , ipport )
	}
}

var (

	configFile        string

)

func init() {


	flag.StringVar(&configFile, "f","./config.yaml", "config file")

}




func main() {
	flag.Parse()

	fmt.Printf("Load config %s\n" , configFile)

	// 监听TCP 服务端口
	config , err :=  ReadYamlConfig( configFile)
	if err!=nil {
		fmt.Printf(" Read yaml file error %v\n", err );
//		glog.Errorf(" Read yaml file error %v", err );
		return ;
	}

	fmt.Printf("Configs : %v \n" , config )

	var wg sync.WaitGroup
	for _ , proxy := range config.Proxy {
		host := proxy.Host
		for _ , port := range proxy.Port {
			two := strings.Split( port , ":")
			ls := len( two )
			var src, target string
			if ls == 1 {
				src = port
				target = port
			}else if ls == 2{
				src = two[0]
				target = two[1]
			} else {
				fmt.Printf("Invalid port %v, ignore\n" , port )
			}

			fmt.Printf("Listen %v -> %v : %\n" , src , host , target)
			go OpenProxy( &wg, src , host , target)
		}
	}
	fmt.Printf("Waiting for proxy exit\n");
	wg.Wait()

}