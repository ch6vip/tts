/**
 * 轻量级状态管理
 * 使用发布订阅模式实现响应式状态
 */

class Store {
  constructor(initialState = {}) {
    this.state = initialState;
    this.listeners = new Map();
  }

  /**
   * 获取状态
   * @param {string} key - 状态键名
   * @returns {any}
   */
  getState(key) {
    return key ? this.state[key] : this.state;
  }

  /**
   * 设置状态
   * @param {Object} updates - 要更新的状态
   */
  setState(updates) {
    const changedKeys = [];

    Object.entries(updates).forEach(([key, value]) => {
      if (this.state[key] !== value) {
        this.state[key] = value;
        changedKeys.push(key);
      }
    });

    // 通知订阅者
    changedKeys.forEach(key => {
      this.notify(key, this.state[key]);
    });
  }

  /**
   * 订阅状态变化
   * @param {string} key - 状态键名
   * @param {Function} callback - 回调函数
   * @returns {Function} 取消订阅函数
   */
  subscribe(key, callback) {
    if (!this.listeners.has(key)) {
      this.listeners.set(key, new Set());
    }

    this.listeners.get(key).add(callback);

    // 返回取消订阅函数
    return () => {
      this.listeners.get(key)?.delete(callback);
    };
  }

  /**
   * 通知订阅者
   * @param {string} key - 状态键名
   * @param {any} value - 新值
   */
  notify(key, value) {
    const callbacks = this.listeners.get(key);
    if (callbacks) {
      callbacks.forEach(callback => callback(value));
    }
  }

  /**
   * 重置状态
   */
  reset() {
    this.state = {};
    this.listeners.clear();
  }
}

// 创建全局 store 实例
const store = new Store({
  // UI 状态
  isSSMLMode: false,
  isLoading: false,

  // 数据状态
  voices: [],
  currentVoice: null,
  currentStyle: 'general',

  // 音频状态
  currentAudio: null,
  isPlaying: false,

  // 历史记录
  history: [],

  // 表单数据
  formData: {
    text: '',
    ssml: '',
    voice: '',
    style: 'general',
    rate: 0,
    pitch: 0
  }
});

export default store;
