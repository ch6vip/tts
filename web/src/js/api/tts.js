/**
 * TTS API 客户端
 * 负责所有与后端 API 的通信
 */

class TTSApi {
  constructor(basePath = '') {
    this.basePath = basePath;
  }

  /**
   * 获取可用的语音列表
   * @returns {Promise<Array>} 语音列表
   */
  async getVoices() {
    try {
      const response = await fetch(`${this.basePath}/api/voices`);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      return await response.json();
    } catch (error) {
      console.error('Failed to fetch voices:', error);
      throw error;
    }
  }

  /**
   * 合成语音
   * @param {Object} params - 合成参数
   * @param {string} params.text - 要合成的文本
   * @param {string} params.voice - 语音名称
   * @param {string} params.style - 情感风格
   * @param {number} params.rate - 语速 (-100 到 100)
   * @param {number} params.pitch - 音调 (-100 到 100)
   * @param {string} params.format - 输出格式
   * @returns {Promise<string>} 音频 URL
   */
  async synthesize(params) {
    try {
      // 转换 rate 和 pitch 为字符串以匹配后端期望的类型
      const apiParams = {
        ...params,
        rate: String(params.rate),
        pitch: String(params.pitch)
      };

      const response = await fetch(`${this.basePath}/api/tts`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(apiParams)
      });

      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || `HTTP error! status: ${response.status}`);
      }

      const blob = await response.blob();
      return URL.createObjectURL(blob);
    } catch (error) {
      console.error('Failed to synthesize:', error);
      throw error;
    }
  }

  /**
   * 获取系统监控指标
   * @returns {Promise<Object>} 指标数据
   */
  async getMetrics() {
    try {
      const response = await fetch(`${this.basePath}/metrics`);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const text = await response.text();
      return this.parseMetrics(text);
    } catch (error) {
      console.error('Failed to fetch metrics:', error);
      throw error;
    }
  }

  /**
   * 解析 Prometheus 格式的指标
   * @param {string} text - 指标文本
   * @returns {Object} 解析后的指标对象
   */
  parseMetrics(text) {
    const metrics = {};
    const lines = text.split('\n');

    for (const line of lines) {
      if (line.startsWith('#') || !line.trim()) continue;

      const match = line.match(/^(\w+)(?:\{.*?\})?\s+(.+)$/);
      if (match) {
        const [, name, value] = match;
        metrics[name] = parseFloat(value);
      }
    }

    return metrics;
  }
}

export default TTSApi;
