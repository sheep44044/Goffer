package mongodb

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Message struct {
	Role    string `bson:"role" json:"role"`       // "user" 或 "assistant"
	Content string `bson:"content" json:"content"` // 聊天内容
	Time    int64  `bson:"time" json:"time"`       // 时间戳
}

// ChatHistoryDoc 定义 MongoDB 里的历史会话文档
type ChatHistoryDoc struct {
	SessionId string    `bson:"session_id"`
	UserId    int64     `bson:"user_id"`
	ResumeId  int64     `bson:"resume_id"`
	StartTime int64     `bson:"start_time"`
	Messages  []Message `bson:"messages"`
}

// GetRecentChatHistory 获取最近指定轮数的历史对话
func (m *MongoManager) GetRecentChatHistory(ctx context.Context, sessionID string, rounds int) ([]Message, error) {
	// 获取 Collection
	coll := m.DB.Collection("chat_history")

	messageCount := rounds * 2

	filter := bson.M{"session_id": sessionID}

	// 使用 $slice 投影操作符
	// -messageCount 表示从数组末尾向前截取
	projection := bson.M{
		"messages": bson.M{"$slice": -messageCount},
	}

	opts := options.FindOne().SetProjection(projection)

	var doc ChatHistoryDoc
	err := coll.FindOne(ctx, filter, opts).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// 如果没查到该 session 的记录，返回空数组而不是报错
			return []Message{}, nil
		}
		return nil, fmt.Errorf("failed to fetch chat history from MongoDB: %w", err)
	}

	return doc.Messages, nil
}

// AppendMessage 追加一条新消息到指定的会话中
func (m *MongoManager) AppendMessage(ctx context.Context, sessionID string, msg Message) error {
	coll := m.DB.Collection("chat_history")

	filter := bson.M{"session_id": sessionID}

	// 使用 $push 操作符，将新消息原子性地追加到 messages 数组末尾
	update := bson.M{
		"$push": bson.M{
			"messages": msg,
		},
	}

	_, err := coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to append message to MongoDB: %w", err)
	}

	return nil
}
