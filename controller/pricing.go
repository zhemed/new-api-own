package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

func filterPricingByUsableGroups(pricing []model.Pricing, usableGroup map[string]string) []model.Pricing {
	if len(pricing) == 0 {
		return pricing
	}
	if len(usableGroup) == 0 {
		return []model.Pricing{}
	}

	filtered := make([]model.Pricing, 0, len(pricing))
	for _, item := range pricing {
		if common.StringsContains(item.EnableGroup, "all") {
			filtered = append(filtered, item)
			continue
		}
		for _, group := range item.EnableGroup {
			if _, ok := usableGroup[group]; ok {
				filtered = append(filtered, item)
				break
			}
		}
	}
	return filtered
}

func GetPricing(c *gin.Context) {
	pricing := model.GetPricing()
	userId, exists := c.Get("id")
	usableGroup := map[string]string{}
	groupRatio := map[string]float64{}
	for s, f := range ratio_setting.GetGroupRatioCopy() {
		groupRatio[s] = f
	}
	var group string
	if exists {
		user, err := model.GetUserCache(userId.(int))
		if err == nil {
			group = user.Group
			for g := range groupRatio {
				ratio, ok := ratio_setting.GetGroupGroupRatio(group, g)
				if ok {
					groupRatio[g] = ratio
				}
			}
		}
	}

	usableGroup = service.GetUserUsableGroups(group)
	pricing = filterPricingByUsableGroups(pricing, usableGroup)
	// check groupRatio contains usableGroup
	for group := range ratio_setting.GetGroupRatioCopy() {
		if _, ok := usableGroup[group]; !ok {
			delete(groupRatio, group)
		}
	}

	c.JSON(200, gin.H{
		"success":            true,
		"data":               pricing,
		"vendors":            model.GetVendors(),
		"group_ratio":        groupRatio,
		"usable_group":       usableGroup,
		"supported_endpoint": model.GetSupportedEndpointMap(),
		"auto_groups":        service.GetUserAutoGroup(group),
		"pricing_version":    "a42d372ccf0b5dd13ecf71203521f9d2",
	})
}

// ResetModelRatio 把**整份定价**恢复为代码默认：倍率表、价格表、计费模式与计费表达式。
//
// 为什么不止重置 ModelRatio：面板按钮的语义是「模型价格重置」，而定价由多张表共同决定
// （倍率 + 输出倍率 + 缓存倍率 + 价格 + tiered 计费模式/表达式）。只重置倍率会让实例停在
// 「按 Token/仅基础」而默认模板本应是表达式计费，两个默认模型看起来"不一致"。
// 分组类键（GroupRatio / TopupGroupRatio / GroupGroupRatio / UserUsableGroups / AutoGroups）
// 与 ExposeRatioEnabled 等开关**不在**本按钮语义内，故一律不动。
func ResetModelRatio(c *gin.Context) {
	// key -> 代码默认 JSON。默认值全部取自各包的导出 getter，此处不复制任何字面量。
	pricingDefaults := map[string]string{
		"ModelRatio":           ratio_setting.DefaultModelRatio2JSONString(),
		"CompletionRatio":      ratio_setting.DefaultCompletionRatio2JSONString(),
		"CacheRatio":           ratio_setting.DefaultCacheRatio2JSONString(),
		"CreateCacheRatio":     ratio_setting.DefaultCreateCacheRatio2JSONString(),
		"ModelPrice":           ratio_setting.DefaultModelPrice2JSONString(),
		"ImageRatio":           ratio_setting.DefaultImageRatio2JSONString(),
		"AudioRatio":           ratio_setting.DefaultAudioRatio2JSONString(),
		"AudioCompletionRatio": ratio_setting.DefaultAudioCompletionRatio2JSONString(),
		"billing_setting." + billing_setting.BillingModeField: billing_setting.DefaultBillingMode2JSONString(),
		"billing_setting." + billing_setting.BillingExprField: billing_setting.DefaultBillingExpr2JSONString(),
	}

	// 逐项恢复：DB（含 OptionMap）与内存表都要落到默认值。任一步失败立即如实返回错误，
	// 不吞掉失败继续写后面的键（否则会出现"响应成功、实例半新半旧"）。
	for key, defaultStr := range pricingDefaults {
		if err := model.UpdateOption(key, defaultStr); err != nil {
			c.JSON(200, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	}

	// 成功文案保持与修复前完全一致：前端只在失败分支展示 message（成功提示走自己的 i18n key），
	// 但后端文案属对外响应的一部分，本次不改以免影响任何按文本匹配的调用方。
	c.JSON(200, gin.H{
		"success": true,
		"message": "重置模型倍率成功",
	})
}
