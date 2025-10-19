# 缓存驱逐机制改进

## 概述

本次改进为 `caching.go` 中的缓存管理系统添加了智能驱逐功能，当缓存大小达到限制时，系统会自动删除旧的缓存项以腾出空间，而不是简单地跳过新项的缓存。

## 改进内容

### 1. 新增 `evictCacheItems` 方法

在 [`cachingService`](caching.go:27) 结构体中新增了 [`evictCacheItems`](caching.go:120) 方法，用于智能驱逐缓存项：

```go
func (s *cachingService) evictCacheItems(spaceNeeded int64) bool
```

**功能特点：**
- 按过期时间排序，优先删除最早过期的项
- 如果过期时间相同，优先删除体积较大的项
- 逐个删除缓存项，直到腾出足够空间
- 返回布尔值表示是否成功腾出足够空间
- 详细的日志记录每次驱逐操作

### 2. 改进 `SynthesizeSpeech` 方法

修改了 [`SynthesizeSpeech`](caching.go:73) 方法中的缓存大小管理逻辑：

**原有逻辑：**
- 当缓存大小超限时，直接跳过缓存写入
- 导致即使有旧的不常用缓存项，新项也无法被缓存

**新逻辑：**
- 当检测到缓存大小超限时，调用 `evictCacheItems` 尝试驱逐旧项
- 如果成功腾出足够空间，则继续缓存新项
- 如果无法腾出足够空间（如新项本身超过最大限制），则跳过缓存

## 驱逐策略

### 排序规则
1. **主排序：** 按过期时间升序排序（最早过期的在前面）
2. **次排序：** 过期时间相同时，按大小降序排序（大的在前面）

### 驱逐过程
1. 获取所有缓存项及其元数据（键、过期时间、大小）
2. 按上述规则排序
3. 逐个删除缓存项，直到释放足够空间
4. 记录详细的驱逐日志

## 示例场景

### 场景 1：缓存接近满载
- 当前缓存：4KB (最大限制 5KB)
- 新请求：2KB
- **处理：** 删除至少 1KB 的旧缓存项，然后缓存新项

### 场景 2：单个项目过大
- 当前缓存：3KB (最大限制 5KB)
- 新请求：6KB (超过最大限制)
- **处理：** 删除所有现有缓存项，但由于新项超过限制，仍不缓存该项

### 场景 3：正常缓存
- 当前缓存：2KB (最大限制 5KB)
- 新请求：1KB
- **处理：** 直接缓存，无需驱逐

## 日志输出

### 驱逐触发日志
```
Cache size limit reached, attempting to evict old items
- key: <cache_key>
- current_size: 4000
- response_size: 2000
- max_size: 5000
```

### 单项驱逐日志
```
Evicted cache item
- key: <cache_key>
- size: 1000
- freed_space: 1000
```

### 驱逐完成日志
```
Cache eviction completed
- evicted_count: 2
- space_needed: 1000
- space_freed: 2000
```

### 驱逐成功日志
```
Successfully evicted cache items
- space_freed: 1000
```

### 无法腾出空间日志
```
Unable to free enough space, skipping cache
- key: <cache_key>
```

## 性能影响

### 时间复杂度
- 获取所有项：O(n)
- 排序：O(n²) - 使用简单的冒泡排序
- 删除项：O(m) - m 是需要删除的项数

对于大多数实际应用场景，缓存项数量不会太大，性能影响可以忽略不计。

### 空间复杂度
- O(n) - 需要创建一个临时切片存储所有缓存项的元数据

## 测试覆盖

新增了两个测试函数：

1. **TestCacheEviction** - 测试基本的缓存驱逐功能
   - 验证缓存大小限制
   - 验证超大项的处理
   - 验证驱逐后的缓存状态

2. **TestCacheEvictionWithPartialSpace** - 测试部分空间驱逐
   - 验证部分驱逐策略
   - 确保新项能够成功缓存
   - 验证缓存总大小不超过限制

## 向后兼容性

- 完全向后兼容，不影响现有 API
- 如果未设置 `maxTotalSize`（默认为 0），行为与之前完全相同（无限制）
- 所有现有测试通过

## 未来改进建议

1. **使用更高效的排序算法**
   - 可以考虑使用 Go 标准库的 `sort.Slice`
   - 对于大量缓存项，可以使用堆排序或快速排序

2. **实现 LRU (Least Recently Used) 策略**
   - 当前基于过期时间，可以改为基于访问时间
   - 需要记录每个缓存项的最后访问时间

3. **添加缓存预热和分级驱逐**
   - 可以为不同类型的请求设置不同的优先级
   - 实现分级驱逐策略，保护高优先级缓存项

4. **支持更多驱逐策略**
   - LFU (Least Frequently Used)
   - FIFO (First In First Out)
   - 可配置的驱逐策略选择

## 相关文件

- [`internal/tts/caching.go`](caching.go) - 主要实现文件
- [`internal/tts/caching_eviction_test.go`](caching_eviction_test.go) - 驱逐功能测试
- [`internal/tts/caching_test.go`](caching_test.go) - 原有缓存测试