package snowflake

import (
	"hash/fnv"
	"os"
	"sync"

	"github.com/bwmarrin/snowflake"
)

var (
	node *snowflake.Node
	once sync.Once
)

// Init 显式初始化雪花算法节点 (通常在微服务启动的 main.go 中调用)
// 如果不传 machineID，则默认使用主机名 Hash 生成
func Init(machineID ...int64) error {
	var err error
	once.Do(func() {
		var id int64
		if len(machineID) > 0 {
			// 手动指定了机器 ID
			id = machineID[0] & 0x3FF
		} else {
			// 默认：利用主机名生成哈希值作为机器 ID (适合 Docker/K8s 部署)
			host, _ := os.Hostname()
			h := fnv.New32a()
			_, _ = h.Write([]byte(host))
			id = int64(h.Sum32()) & 0x3FF // 取低 10 位，最大值 1023
		}

		node, err = snowflake.NewNode(id)
		if err != nil {
			// Fallback: 如果生成失败，默认使用节点 1
			node, _ = snowflake.NewNode(1)
		}
	})
	return err
}

// GenInt64 生成 int64 类型的全局唯一 ID (适合存入 MySQL 优化索引)
func GenInt64() int64 {
	if node == nil {
		_ = Init() // 懒加载防错
	}
	return node.Generate().Int64()
}

// GenString 生成 string 类型的全局唯一 ID (👈 强烈推荐：适合直接返回给前端和存入 MongoDB)
func GenString() string {
	if node == nil {
		_ = Init() // 懒加载防错
	}
	return node.Generate().String()
}
