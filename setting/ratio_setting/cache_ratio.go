package ratio_setting

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
)

// 官方缓存命中价 ÷ 未命中价 = 0.02 / 1.00 = 0.02（低谷），与 defaultModelRatio 的默认模板同族。
var defaultCacheRatio = map[string]float64{
	"deepseek-flash":      0.02,
	"deepseek-v4.1-flash": 0.02,
}

// 目录裁剪后没有需要单独计缓存写入价的模型，缓存写入倍率表为空。
var defaultCreateCacheRatio = map[string]float64{}

var cacheRatioMap = types.NewRWMap[string, float64]()
var createCacheRatioMap = types.NewRWMap[string, float64]()

// GetCacheRatioMap returns a copy of the cache ratio map
func GetCacheRatioMap() map[string]float64 {
	return cacheRatioMap.ReadAll()
}

// DefaultCacheRatio2JSONString 返回缓存倍率表的**代码默认** JSON。
func DefaultCacheRatio2JSONString() string {
	jsonBytes, err := common.Marshal(defaultCacheRatio)
	if err != nil {
		common.SysError("error marshalling cache ratio: " + err.Error())
	}
	return string(jsonBytes)
}

// GetDefaultCacheRatioMap 返回缓存倍率表的代码默认（与 GetDefaultModelRatioMap 同风格）。
func GetDefaultCacheRatioMap() map[string]float64 {
	return defaultCacheRatio
}

// DefaultCreateCacheRatio2JSONString 返回缓存写入倍率表的代码默认 JSON（当前是空表）。
func DefaultCreateCacheRatio2JSONString() string {
	jsonBytes, err := common.Marshal(defaultCreateCacheRatio)
	if err != nil {
		common.SysError("error marshalling create cache ratio: " + err.Error())
	}
	return string(jsonBytes)
}

// CacheRatio2JSONString converts the cache ratio map to a JSON string
func CacheRatio2JSONString() string {
	return cacheRatioMap.MarshalJSONString()
}

// CreateCacheRatio2JSONString converts the create cache ratio map to a JSON string
func CreateCacheRatio2JSONString() string {
	return createCacheRatioMap.MarshalJSONString()
}

// UpdateCacheRatioByJSONString updates the cache ratio map from a JSON string
func UpdateCacheRatioByJSONString(jsonStr string) error {
	return types.LoadFromJsonStringWithCallback(cacheRatioMap, jsonStr, InvalidateExposedDataCache)
}

// UpdateCreateCacheRatioByJSONString updates the create cache ratio map from a JSON string
func UpdateCreateCacheRatioByJSONString(jsonStr string) error {
	return types.LoadFromJsonStringWithCallback(createCacheRatioMap, jsonStr, InvalidateExposedDataCache)
}

// GetCacheRatio returns the cache ratio for a model
func GetCacheRatio(name string) (float64, bool) {
	ratio, ok := cacheRatioMap.Get(name)
	if !ok {
		return 1, false // Default to 1 if not found
	}
	return ratio, true
}

func GetCreateCacheRatio(name string) (float64, bool) {
	ratio, ok := createCacheRatioMap.Get(name)
	if !ok {
		return 1.25, false // Default to 1.25 if not found
	}
	return ratio, true
}

func GetCacheRatioCopy() map[string]float64 {
	return cacheRatioMap.ReadAll()
}

func GetCreateCacheRatioCopy() map[string]float64 {
	return createCacheRatioMap.ReadAll()
}
