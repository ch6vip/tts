/**
 * LocalStorage 封装工具
 */

const STORAGE_PREFIX = 'tts_';

/**
 * 保存数据到 LocalStorage
 * @param {string} key - 键名
 * @param {any} value - 值
 * @returns {boolean} 是否成功
 */
export function setStorage(key, value) {
  try {
    const serialized = JSON.stringify(value);
    localStorage.setItem(STORAGE_PREFIX + key, serialized);
    return true;
  } catch (error) {
    console.error('保存到 LocalStorage 失败:', error);
    return false;
  }
}

/**
 * 从 LocalStorage 获取数据
 * @param {string} key - 键名
 * @param {any} defaultValue - 默认值
 * @returns {any}
 */
export function getStorage(key, defaultValue = null) {
  try {
    const item = localStorage.getItem(STORAGE_PREFIX + key);
    return item ? JSON.parse(item) : defaultValue;
  } catch (error) {
    console.error('从 LocalStorage 读取失败:', error);
    return defaultValue;
  }
}

/**
 * 删除 LocalStorage 中的数据
 * @param {string} key - 键名
 */
export function removeStorage(key) {
  try {
    localStorage.removeItem(STORAGE_PREFIX + key);
  } catch (error) {
    console.error('删除 LocalStorage 数据失败:', error);
  }
}

/**
 * 清空所有 TTS 相关的 LocalStorage 数据
 */
export function clearStorage() {
  try {
    const keys = Object.keys(localStorage);
    keys.forEach(key => {
      if (key.startsWith(STORAGE_PREFIX)) {
        localStorage.removeItem(key);
      }
    });
  } catch (error) {
    console.error('清空 LocalStorage 失败:', error);
  }
}

/**
 * 保存表单数据
 * @param {Object} formData - 表单数据
 */
export function saveFormData(formData) {
  setStorage('formData', formData);
}

/**
 * 加载表单数据
 * @returns {Object|null}
 */
export function loadFormData() {
  return getStorage('formData');
}

/**
 * 保存历史记录
 * @param {Array} history - 历史记录数组
 */
export function saveHistory(history) {
  setStorage('history', history);
}

/**
 * 加载历史记录
 * @returns {Array}
 */
export function loadHistory() {
  return getStorage('history', []);
}

/**
 * 添加历史记录项
 * @param {Object} item - 历史记录项
 * @param {number} maxItems - 最大保存数量
 */
export function addHistoryItem(item, maxItems = 50) {
  const history = loadHistory();

  // 添加时间戳
  item.timestamp = Date.now();

  // 添加到开头
  history.unshift(item);

  // 限制数量
  if (history.length > maxItems) {
    history.length = maxItems;
  }

  saveHistory(history);
}

/**
 * 清空历史记录
 */
export function clearHistory() {
  saveHistory([]);
}
