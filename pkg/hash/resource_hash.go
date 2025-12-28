package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

// CalculateResourceHash 计算资源对象的哈希值
// 只包含业务字段，排除元数据字段
func CalculateResourceHash(obj interface{}) (string, error) {
	// 将对象转换为 JSON
	data, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}

	// 解析 JSON 并移除元数据字段
	var objMap map[string]interface{}
	if err := json.Unmarshal(data, &objMap); err != nil {
		return "", err
	}

	// 移除元数据字段
	excludeFields := []string{
		"id",
		"create_time",
		"createAt",
		"createdAt",
		"update_time",
		"updateAt",
		"updatedAt",
		"resource_hash",
		"last_sync_time",
		"creator",
		"modifier",
	}

	for _, field := range excludeFields {
		delete(objMap, field)
	}

	// 创建一个有序的 map（按 key 排序）
	keys := make([]string, 0, len(objMap))
	for k := range objMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 按顺序构建有序的 map
	orderedMap := make(map[string]interface{})
	for _, k := range keys {
		orderedMap[k] = objMap[k]
	}

	// 重新序列化（确保字段顺序一致）
	cleanData, err := json.Marshal(orderedMap)
	if err != nil {
		return "", err
	}

	// 计算 SHA256 哈希
	hash := sha256.Sum256(cleanData)
	return hex.EncodeToString(hash[:]), nil
}
