package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"go.etcd.io/etcd/clientv3"
)

type EtcdMaster struct {
	Cluster string // 集群
	Path    string // 路径
	Nodes   map[string]*EtcdNode
	Client  *clientv3.Client
}

// Etcd注册的节点，一个节点代表一个client
type EtcdNode struct {
	State   bool
	Cluster string          // 集群
	Key     string          // key
	Info    EtcdServiceInfo // 节点信息
}

func NewMaster(host []string, cluster string, watchPath string) (*EtcdMaster, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   host,
		DialTimeout: time.Second,
	})

	if nil != err {
		fmt.Println(err.Error())
		return nil, err
	}

	master := &EtcdMaster{
		Cluster: cluster,
		Path:    watchPath,
		Nodes:   make(map[string]*EtcdNode),
		Client:  cli,
	}

	// 监听观察节点
	go master.WatchNodes()

	return master, err
}

func NewEtcdNode(ev *clientv3.Event) *EtcdServiceInfo {
	info := &EtcdServiceInfo{}
	err := json.Unmarshal([]byte(ev.Kv.Value), info)
	if nil != err {
		fmt.Println(err.Error())
	}
	return info
}

// 监听观察节点
func (m *EtcdMaster) WatchNodes() {
	// 查看之前存在的节点
	resp, err := m.Client.Get(context.Background(), m.Cluster+"/"+m.Path, clientv3.WithPrefix())
	if nil != err {
		fmt.Println(err.Error())
	} else {
		for _, ev := range resp.Kvs {
			fmt.Printf("add dir:%q, value:%q\n", ev.Key, ev.Value)
			info := &EtcdServiceInfo{}
			json.Unmarshal([]byte(ev.Value), info)
			m.addNode(string(ev.Key), info)
		}
	}

	rch := m.Client.Watch(context.Background(), m.Cluster+"/"+m.Path, clientv3.WithPrefix(), clientv3.WithPrevKV())
	for wresp := range rch {
		for _, ev := range wresp.Events {
			switch ev.Type {
			case clientv3.EventTypePut:
				var action string
				if ev.IsCreate() {
					action = "create"
				} else if ev.IsModify() {
					action = "update"
				}
				fmt.Printf("--- action is:%s\n", action)
				fmt.Printf("[%s] dir:%q, value:%q\n", ev.Type, ev.Kv.Key, ev.Kv.Value)
				info := NewEtcdNode(ev)
				m.addNode(string(ev.Kv.Key), info)
			case clientv3.EventTypeDelete:
				fmt.Printf("[%s] dir:%q, value:%q\n", ev.Type, ev.Kv.Key, ev.Kv.Value)
				k := ev.Kv.Key
				if len(ev.Kv.Key) > (len(m.Cluster) + 1) {
					k = ev.Kv.Key[len(m.Cluster)+1:]
				}
				delete(m.Nodes, string(k))
			default:
				fmt.Printf("[%s] dir:%q, value:%q\n", ev.Type, ev.Kv.Key, ev.Kv.Value)
			}
		}
	}
}

// 添加节点
func (m *EtcdMaster) addNode(key string, info *EtcdServiceInfo) {
	k := key
	if len(key) > (len(m.Cluster) + 1) {
		k = key[len(m.Cluster)+1:]
	}

	node := &EtcdNode{
		State:   true,
		Cluster: m.Cluster,
		Key:     k,
		Info:    *info,
	}

	m.Nodes[node.Key] = node
}

// 获取该集群下所有的节点
func (m *EtcdMaster) GetAllNodes() []EtcdNode {
	var temp []EtcdNode
	for _, v := range m.Nodes {
		if nil != v {
			temp = append(temp, *v)
		}
	}
	return temp
}

func (m *EtcdMaster) GetNodeRandom() (EtcdNode, bool) {
	count := len(m.Nodes)
	// 该集群不存在节点时，直接返回false
	if 0 == count {
		return EtcdNode{}, false
	}
	idx := rand.Intn(count)
	for _, v := range m.Nodes {
		if idx == 0 {
			return *v, true
		}
		idx = idx - 1
	}
	return EtcdNode{}, false
}
}
