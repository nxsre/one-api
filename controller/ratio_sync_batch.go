package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/dto"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/setting/billing_setting"
)

var upstreamSyncBatchLocks sync.Map

func runUpstreamSyncBatch(batchID int64, req dto.UpstreamRequest) {
	lockIface, _ := upstreamSyncBatchLocks.LoadOrStore(batchID, &sync.Mutex{})
	lock := lockIface.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()

	batch, err := model.GetUpstreamSyncBatch(batchID)
	if err != nil {
		return
	}
	if batch.Status != model.UpstreamSyncStatusRunning {
		return
	}

	testResults, successfulChannels, err := executeUpstreamRatioFetch(context.Background(), req)
	if err != nil && len(successfulChannels) == 0 {
		_ = model.FinishUpstreamSyncBatch(batchID, model.UpstreamSyncStatusFailed, testResults, nil, nil, 0, err.Error())
		return
	}
	localData := getLocalPricingSyncData()
	differences := buildDifferences(localData, successfulChannels)
	diffCount := countDifferenceRows(differences)
	snapshots := make([]model.UpstreamSnapshot, 0, len(successfulChannels))
	for _, ch := range successfulChannels {
		snapshots = append(snapshots, model.UpstreamSnapshot{Name: ch.name, Data: ch.data})
	}
	if err := model.ReplaceUpstreamSyncDiffs(batchID, differences); err != nil {
		logger.SysError("save upstream sync diffs: " + err.Error())
		_ = model.FinishUpstreamSyncBatch(batchID, model.UpstreamSyncStatusFailed, testResults, localData, nil, 0, err.Error())
		return
	}
	upRaw := map[string]any{"snapshots": snapshots}
	status := model.UpstreamSyncStatusCompleted
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	if err := model.FinishUpstreamSyncBatch(batchID, status, testResults, localData, upRaw, diffCount, errMsg); err != nil {
		logger.SysError("finish upstream sync batch: " + err.Error())
	}
}

func countDifferenceRows(differences map[string]map[string]dto.DifferenceItem) int {
	n := 0
	for _, fields := range differences {
		n += len(fields)
	}
	return n
}

func batchToResponse(batch *model.UpstreamSyncBatch) gin.H {
	if batch == nil {
		return gin.H{}
	}
	var channelIDs []int64
	_ = json.Unmarshal([]byte(batch.ChannelIds), &channelIDs)
	return gin.H{
		"id":              batch.Id,
		"status":          batch.Status,
		"channel_ids":     channelIDs,
		"test_results":    model.ParseBatchTestResults(batch),
		"diff_count":      batch.DiffCount,
		"selection_count": batch.SelectionCount,
		"error_message":   batch.ErrorMessage,
		"created_time":    batch.CreatedTime,
		"completed_time":  batch.CompletedTime,
	}
}

func ListUpstreamSyncBatches(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	batches, err := model.ListUpstreamSyncBatches(limit)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	items := make([]gin.H, 0, len(batches))
	for i := range batches {
		items = append(items, batchToResponse(&batches[i]))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}

func GetLatestUpstreamSyncBatch(c *gin.Context) {
	batch, err := model.GetLatestUpstreamSyncBatch()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "暂无已完成批次"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": batchToResponse(batch)})
}

func GetUpstreamSyncBatch(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效批次 ID"})
		return
	}
	batch, err := model.GetUpstreamSyncBatch(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "批次不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": batchToResponse(batch)})
}

func ListUpstreamSyncBatchDiffs(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效批次 ID"})
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	params := model.UpstreamSyncDiffListParams{
		BatchId:  id,
		Page:     page,
		PageSize: pageSize,
		Model:    strings.TrimSpace(c.Query("model")),
	}
	if sel := strings.TrimSpace(c.Query("selected")); sel != "" {
		v := sel == "1" || strings.EqualFold(sel, "true")
		params.Selected = &v
	}
	result, err := model.ListUpstreamSyncDiffs(params)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

func SaveUpstreamSyncBatchSelections(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效批次 ID"})
		return
	}
	if _, err := model.GetUpstreamSyncBatch(id); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "批次不存在"})
		return
	}
	var req dto.UpstreamSyncSaveSelectionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求参数格式错误"})
		return
	}
	count, err := model.SaveUpstreamSyncSelections(id, req.Items)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "已暂存到草稿表", "data": gin.H{"selected_count": count}})
}

func SelectAllUpstreamSyncBatch(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效批次 ID"})
		return
	}
	var req dto.UpstreamSyncSelectAllRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求参数格式错误"})
		return
	}
	count, err := model.SelectAllUpstreamSyncDiffs(id, req.UpstreamName, req.Selected)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "已更新全选", "data": gin.H{"selected_count": count}})
}

func ApplyUpstreamSyncBatch(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效批次 ID"})
		return
	}
	batch, err := model.GetUpstreamSyncBatch(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "批次不存在"})
		return
	}
	if batch.Status != model.UpstreamSyncStatusCompleted {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "批次尚未完成"})
		return
	}
	var req dto.UpstreamSyncApplyBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求参数格式错误"})
		return
	}
	sels, err := model.ListUpstreamSyncSelections(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if len(sels) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请先勾选差异并点击「暂存勾选」"})
		return
	}
	items := make([]dto.PricingSyncApplyItem, 0, len(sels))
	for _, sel := range sels {
		var diff model.UpstreamSyncDiff
		if err := model.DB.Where("batch_id = ? AND model = ? AND field = ?", id, sel.Model, sel.Field).First(&diff).Error; err != nil {
			continue
		}
		row := model.DecodeUpstreamSyncDiffRow(diff, &sel)
		val, ok := row.Upstreams[sel.UpstreamName]
		if !ok || val == nil || val == "same" {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("%s / %s 无有效上游值", sel.Model, sel.Field)})
			return
		}
		items = append(items, dto.PricingSyncApplyItem{Model: sel.Model, Field: sel.Field, Value: val})
	}
	if len(items) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无有效差异项，请确认已保存勾选且批次中有差异"})
		return
	}
	applyReq := dto.PricingSyncApplyRequest{
		Items:    items,
		Activate: req.Activate,
		Label:    req.Label,
		Source:   req.Source,
		Note:     req.Note,
	}
	applyPricingSyncItems(c, applyReq)
}

func CompareUpstreamSync(c *gin.Context) {
	var req dto.UpstreamSyncCompareRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求参数格式错误"})
		return
	}
	baseline, channels, err := resolveComparePair(req.Left, req.Right)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	differences := buildDifferences(baseline, channels)
	rows := flattenDifferencesForCompare(differences)
	if m := strings.TrimSpace(req.Model); m != "" {
		filtered := rows[:0]
		for _, row := range rows {
			if strings.Contains(row.Model, m) {
				filtered = append(filtered, row)
			}
		}
		rows = filtered
	}
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.Size
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}
	total := len(rows)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":       rows[start:end],
			"total":       total,
			"page":        page,
			"page_size":   pageSize,
			"total_pages": totalPages,
			"left":        req.Left,
			"right":       req.Right,
		},
	})
}

type compareDiffRow struct {
	Key        string                 `json:"key"`
	Model      string                 `json:"model"`
	Field      string                 `json:"field"`
	Current    interface{}            `json:"current"`
	Upstreams  map[string]interface{} `json:"upstreams"`
	Confidence map[string]bool        `json:"confidence"`
}

func flattenDifferencesForCompare(differences map[string]map[string]dto.DifferenceItem) []compareDiffRow {
	rows := make([]compareDiffRow, 0, 128)
	for modelName, fields := range differences {
		for field, item := range fields {
			hasDiff := false
			for _, val := range item.Upstreams {
				if val != nil && val != "same" {
					hasDiff = true
					break
				}
			}
			if !hasDiff {
				continue
			}
			rows = append(rows, compareDiffRow{
				Key:        modelName + "::" + field,
				Model:      modelName,
				Field:      field,
				Current:    item.Current,
				Upstreams:  item.Upstreams,
				Confidence: item.Confidence,
			})
		}
	}
	sortCompareRows(rows)
	return rows
}

func sortCompareRows(rows []compareDiffRow) {
	for i := 0; i < len(rows); i++ {
		for j := i + 1; j < len(rows); j++ {
			if rows[j].Model < rows[i].Model || (rows[j].Model == rows[i].Model && rows[j].Field < rows[i].Field) {
				rows[i], rows[j] = rows[j], rows[i]
			}
		}
	}
}

func resolveComparePair(leftSpec, rightSpec string) (map[string]any, []upstreamChannelData, error) {
	leftSpec = strings.TrimSpace(leftSpec)
	rightSpec = strings.TrimSpace(rightSpec)
	if leftSpec == "" {
		leftSpec = "current"
	}
	if rightSpec == "" {
		rightSpec = "current"
	}
	baseline, err := resolveBaselineSide(leftSpec)
	if err != nil {
		return nil, nil, err
	}
	channels, err := resolveTargetSide(rightSpec)
	if err != nil {
		return nil, nil, err
	}
	return baseline, channels, nil
}

func resolveBaselineSide(spec string) (map[string]any, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" || spec == "current" {
		return getLocalPricingSyncData(), nil
	}
	if strings.HasPrefix(spec, "batch:") {
		id, err := strconv.ParseInt(strings.TrimPrefix(spec, "batch:"), 10, 64)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("无效批次 %s", spec)
		}
		batch, err := model.GetUpstreamSyncBatch(id)
		if err != nil {
			return nil, fmt.Errorf("批次不存在")
		}
		local, err := model.LoadLocalSnapshot(batch)
		if err != nil {
			return getLocalPricingSyncData(), nil
		}
		return local, nil
	}
	if strings.HasPrefix(spec, "version:") {
		parts := strings.Split(strings.TrimPrefix(spec, "version:"), ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("版本格式应为 version:block_id:version_id")
		}
		return loadPricingSyncDataFromBlockVersion(strings.TrimSpace(parts[0]), mustAtoi(parts[1]))
	}
	return nil, fmt.Errorf("未知对比端 %s", spec)
}

func resolveTargetSide(spec string) ([]upstreamChannelData, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" || spec == "current" {
		return []upstreamChannelData{{name: "当前生效", data: getLocalPricingSyncData()}}, nil
	}
	if strings.HasPrefix(spec, "batch:") {
		id, err := strconv.ParseInt(strings.TrimPrefix(spec, "batch:"), 10, 64)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("无效批次 %s", spec)
		}
		batch, err := model.GetUpstreamSyncBatch(id)
		if err != nil {
			return nil, fmt.Errorf("批次不存在")
		}
		snaps, err := model.LoadUpstreamSnapshots(batch)
		if err != nil {
			return nil, err
		}
		channels := make([]upstreamChannelData, 0, len(snaps))
		for _, s := range snaps {
			channels = append(channels, upstreamChannelData{name: s.Name, data: s.Data})
		}
		return channels, nil
	}
	if strings.HasPrefix(spec, "version:") {
		parts := strings.Split(strings.TrimPrefix(spec, "version:"), ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("版本格式应为 version:block_id:version_id")
		}
		data, err := loadPricingSyncDataFromBlockVersion(strings.TrimSpace(parts[0]), mustAtoi(parts[1]))
		if err != nil {
			return nil, err
		}
		return []upstreamChannelData{{name: compareSideLabel(spec), data: data}}, nil
	}
	return nil, fmt.Errorf("未知对比端 %s", spec)
}

func mustAtoi(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

func compareSideLabel(spec string) string {
	if strings.TrimSpace(spec) == "current" {
		return "当前生效"
	}
	return spec
}

func loadPricingSyncDataFromBlockVersion(blockID string, versionID int) (map[string]any, error) {
	def, ok := billing_setting.LookupBlockByID(blockID)
	if !ok || def.SyncFieldKey == "" {
		return nil, fmt.Errorf("配置块 %s 不参与上游对比", blockID)
	}
	data := getLocalPricingSyncData()
	fieldMap, err := model.LoadVersionEntriesAsFieldMap(blockID, versionID)
	if err != nil {
		return nil, err
	}
	if len(fieldMap) > 0 {
		data[def.SyncFieldKey] = fieldMap
	}
	return data, nil
}

func applyPricingSyncItems(c *gin.Context, req dto.PricingSyncApplyRequest) {
	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请至少勾选一项差异"})
		return
	}
	blockPatches := make(map[string]map[string]any)
	for _, item := range req.Items {
		modelName := strings.TrimSpace(item.Model)
		field := strings.TrimSpace(item.Field)
		if modelName == "" || field == "" {
			continue
		}
		def, ok := billing_setting.LookupBlockBySyncField(field)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": fmt.Sprintf("未知字段 %s", field)})
			return
		}
		if blockPatches[def.BlockID] == nil {
			blockPatches[def.BlockID] = make(map[string]any)
		}
		blockPatches[def.BlockID][modelName] = item.Value
	}
	if len(blockPatches) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无有效差异项"})
		return
	}
	label := strings.TrimSpace(req.Label)
	if label == "" {
		label = "上游同步"
	}
	created := make(map[string]int)
	for blockID, patches := range blockPatches {
		if err := model.ApplyPricingPatches(blockID, patches); err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
			return
		}
		versionID, err := billing_setting.CreatePricingVersionEntryBased(blockID, label, strings.TrimSpace(req.Source), strings.TrimSpace(req.Note), req.Activate)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
			return
		}
		created[blockID] = versionID
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "已写入本地价目版本",
		"data": gin.H{
			"created_versions": created,
			"activated":        req.Activate,
		},
	})
}
