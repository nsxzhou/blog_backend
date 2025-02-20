package utils

import (
	"fmt"
	"time"

	"blog/global"

	sf "github.com/bwmarrin/snowflake"
	"go.uber.org/zap"
)

// SnowflakeNode 封装雪花算法节点
type SnowflakeNode struct {
	node *sf.Node
}

// 全局节点实例
var snowflake *SnowflakeNode

// Init 初始化雪花算法节点
// startTime: 起始时间，格式："2006-01-02"
// machineID: 机器ID (0-1023)
func Init(startTime string, machineID int64) {
	st, err := time.Parse("2006-01-02", startTime)
	if err != nil {
		global.Log.Fatal("time.Parse() failed", zap.String("error", err.Error()))
	}

	sf.Epoch = st.UnixNano() / 1000000

	node, err := sf.NewNode(machineID)
	if err != nil {
		global.Log.Fatal("sf.NewNode() failed", zap.String("error", err.Error()))
	}

	snowflake = &SnowflakeNode{node: node}
}

// GenerateID 生成唯一ID
func GenerateID() (int64, error) {
	if snowflake == nil || snowflake.node == nil {
		return 0, fmt.Errorf("雪花节点未初始化")
	}
	return snowflake.node.Generate().Int64(), nil
}
