/**
 * TTS 应用主入口文件
 */

// 导入样式文件
import '../css/main.css';

import TTSApi from './api/tts.js';
import AudioPlayer from './components/AudioPlayer.js';
import VoiceSelector from './components/VoiceSelector.js';
import store from './state/store.js';
import { $, toggleClass, copyToClipboard } from './utils/dom.js';
import { alert } from './utils/alert.js';
import { validateSynthesisParams } from './utils/validator.js';
import { saveFormData, loadFormData, addHistoryItem, loadHistory } from './utils/storage.js';

// 全局配置（从 Go 模板注入）
const config = window.config || {
  basePath: '',
  defaultVoice: '',
  defaultRate: 0,
  defaultPitch: 0,
  defaultFormat: ''
};

class TTSApp {
  constructor() {
    this.api = new TTSApi(config.basePath);
    this.audioPlayer = null;
    this.voiceSelector = null;
    this.elements = {};

    this.init();
  }

  /**
   * 初始化应用
   */
  init() {
    this.initElements();
    this.initComponents();
    this.initEventListeners();
    this.loadSavedFormData();
    this.initKeyboardShortcuts();
    this.startMetricsPolling();
  }

  /**
   * 初始化 DOM 元素
   */
  initElements() {
    this.elements = {
      textInput: $('#text'),
      ssmlInput: $('#ssml'),
      voiceSelect: $('#voice'),
      styleSelect: $('#style'),
      rateInput: $('#rate'),
      rateValue: $('#rateValue'),
      pitchInput: $('#pitch'),
      pitchValue: $('#pitchValue'),
      speakButton: $('#speak'),
      downloadButton: $('#download'),
      copyLinkButton: $('#copyLink'),
      copyHttpTtsLinkButton: $('#copyHttpTtsLink'),
      copyIfreetimeLinkButton: $('#copyIfreetimeLink'),
      charCount: $('#charCount'),
      toggleInputModeBtn: $('#toggleInputMode'),
      ssmlHelp: $('#ssmlHelp'),
      loadingOverlay: $('#loadingOverlay'),
      historyPanel: $('#historyPanel'),
      historyPanelClose: $('#historyPanelClose'),
      historyPanelContent: $('#historyPanelContent'),
      historyToggle: $('#historyToggle'),
      metricsContainer: $('#metrics-container')
    };
  }

  /**
   * 初始化组件
   */
  initComponents() {
    this.audioPlayer = new AudioPlayer();
    this.voiceSelector = new VoiceSelector(this.api);
  }

  /**
   * 初始化事件监听
   */
  initEventListeners() {
    // 合成按钮
    this.elements.speakButton?.addEventListener('click', () => this.synthesize());

    // 下载按钮
    this.elements.downloadButton?.addEventListener('click', () => this.downloadAudio());

    // 复制链接按钮
    this.elements.copyLinkButton?.addEventListener('click', () => this.copyLink());
    this.elements.copyHttpTtsLinkButton?.addEventListener('click', () => this.copyHttpTtsLink());
    this.elements.copyIfreetimeLinkButton?.addEventListener('click', () => this.copyIfreetimeLink());

    // 输入模式切换
    this.elements.toggleInputModeBtn?.addEventListener('click', () => this.toggleInputMode());

    // 字符计数
    this.elements.textInput?.addEventListener('input', () => this.updateCharCount());
    this.elements.ssmlInput?.addEventListener('input', () => this.updateCharCount());

    // 语速音调滑块
    this.elements.rateInput?.addEventListener('input', (e) => {
      if (this.elements.rateValue) {
        this.elements.rateValue.textContent = `${e.target.value}%`;
      }
    });

    this.elements.pitchInput?.addEventListener('input', (e) => {
      if (this.elements.pitchValue) {
        this.elements.pitchValue.textContent = `${e.target.value}%`;
      }
    });

    // 历史记录面板
    this.elements.historyToggle?.addEventListener('click', () => this.toggleHistoryPanel());
    this.elements.historyPanelClose?.addEventListener('click', () => this.closeHistoryPanel());

    // 表单数据自动保存
    ['textInput', 'ssmlInput', 'voiceSelect', 'styleSelect', 'rateInput', 'pitchInput'].forEach(key => {
      this.elements[key]?.addEventListener('change', () => this.saveFormData());
    });
  }

  /**
   * 初始化键盘快捷键
   */
  initKeyboardShortcuts() {
    document.addEventListener('keydown', (e) => {
      // Ctrl+Enter: 合成语音
      if (e.ctrlKey && e.key === 'Enter') {
        e.preventDefault();
        this.synthesize();
      }

      // Ctrl+H: 打开历史记录
      if (e.ctrlKey && e.key === 'h') {
        e.preventDefault();
        this.toggleHistoryPanel();
      }
    });
  }

  /**
   * 合成语音
   */
  async synthesize() {
    const params = this.getFormData();

    // 验证参数
    const validation = validateSynthesisParams(params);
    if (!validation.valid) {
      alert.error(validation.errors.join('\n'));
      return;
    }

    // 显示加载状态
    this.setLoading(true);

    try {
      const audioUrl = await this.api.synthesize(params);

      // 加载音频
      this.audioPlayer.load(audioUrl);

      // 保存到历史记录
      this.saveToHistory(params, audioUrl);

      // 更新状态
      store.setState({ currentAudio: audioUrl });

      alert.success('语音合成成功！');
    } catch (error) {
      alert.error(error.message || '语音合成失败，请稍后重试');
    } finally {
      this.setLoading(false);
    }
  }

  /**
   * 获取表单数据
   * @returns {Object}
   */
  getFormData() {
    const isSSML = store.getState('isSSMLMode');

    const rate = parseInt(this.elements.rateInput?.value ?? '0', 10);
    const pitch = parseInt(this.elements.pitchInput?.value ?? '0', 10);

    return {
      text: isSSML ? '' : this.elements.textInput?.value || '',
      ssml: isSSML ? this.elements.ssmlInput?.value || '' : '',
      voice: this.elements.voiceSelect?.value || '',
      style: this.elements.styleSelect?.value || 'general',
      rate: Number.isFinite(rate) ? rate : 0,
      pitch: Number.isFinite(pitch) ? pitch : 0,
      format: config.defaultFormat || ''
    };
  }

  /**
   * 保存表单数据
   */
  saveFormData() {
    const data = this.getFormData();
    saveFormData(data);
  }

  /**
   * 加载已保存的表单数据
   */
  loadSavedFormData() {
    const data = loadFormData();
    if (!data) return;

    if (this.elements.textInput && data.text) {
      this.elements.textInput.value = data.text;
    }
    if (this.elements.voiceSelect && data.voice) {
      this.elements.voiceSelect.value = data.voice;
    }
    if (this.elements.styleSelect && data.style) {
      this.elements.styleSelect.value = data.style;
    }
    if (this.elements.rateInput && data.rate !== undefined) {
      this.elements.rateInput.value = data.rate;
      if (this.elements.rateValue) {
        this.elements.rateValue.textContent = `${data.rate}%`;
      }
    }
    if (this.elements.pitchInput && data.pitch !== undefined) {
      this.elements.pitchInput.value = data.pitch;
      if (this.elements.pitchValue) {
        this.elements.pitchValue.textContent = `${data.pitch}%`;
      }
    }

    this.updateCharCount();
  }

  /**
   * 保存到历史记录
   * @param {Object} params - 合成参数
   * @param {string} audioUrl - 音频 URL
   */
  saveToHistory(params, audioUrl) {
    const item = {
      ...params,
      audioUrl,
      timestamp: Date.now()
    };
    addHistoryItem(item);
  }

  /**
   * 切换输入模式
   */
  toggleInputMode() {
    const isSSML = !store.getState('isSSMLMode');
    store.setState({ isSSMLMode: isSSML });

    toggleClass(this.elements.textInput, 'hidden', isSSML);
    toggleClass(this.elements.ssmlInput, 'hidden', !isSSML);
    toggleClass(this.elements.ssmlHelp, 'hidden', !isSSML);

    if (this.elements.toggleInputModeBtn) {
      this.elements.toggleInputModeBtn.textContent = isSSML ? 'SSML模式' : '文本模式';
    }

    this.updateCharCount();
  }

  /**
   * 更新字符计数
   */
  updateCharCount() {
    if (!this.elements.charCount) return;

    const isSSML = store.getState('isSSMLMode');
    const text = isSSML
      ? this.elements.ssmlInput?.value || ''
      : this.elements.textInput?.value || '';

    this.elements.charCount.textContent = text.length;

    // 警告提示
    if (text.length > 4500) {
      this.elements.charCount.classList.add('danger');
    } else if (text.length > 4000) {
      this.elements.charCount.classList.add('warning');
    } else {
      this.elements.charCount.classList.remove('danger', 'warning');
    }
  }

  /**
   * 下载音频
   */
  downloadAudio() {
    const audioUrl = store.getState('currentAudio');
    if (!audioUrl) return;

    const link = document.createElement('a');
    link.href = audioUrl;
    link.download = `tts_${Date.now()}.mp3`;
    link.click();
  }

  /**
   * 复制链接
   */
  async copyLink() {
    const audioUrl = store.getState('currentAudio');
    if (!audioUrl) return;

    const success = await copyToClipboard(audioUrl);
    if (success) {
      alert.success('链接已复制到剪贴板');
    } else {
      alert.error('复制失败');
    }
  }

  /**
   * 复制阅读APP链接
   */
  async copyHttpTtsLink() {
    const params = this.getFormData();
    const url = this.buildHttpTtsUrl(params);

    const success = await copyToClipboard(url);
    if (success) {
      alert.success('已复制到剪贴板，可导入「阅读」APP');
    } else {
      alert.error('复制失败');
    }
  }

  /**
   * 复制爱阅记APP链接
   */
  async copyIfreetimeLink() {
    const params = this.getFormData();
    const url = this.buildIfreetimeUrl(params);

    const success = await copyToClipboard(url);
    if (success) {
      alert.success('已复制到剪贴板，可导入「爱阅记」APP');
    } else {
      alert.error('复制失败');
    }
  }

  /**
   * 构建阅读APP链接
   */
  buildHttpTtsUrl(params) {
    const baseUrl = window.location.origin + config.basePath;
    const name = encodeURIComponent(params.voice || 'AzureTTS');
    const voice = encodeURIComponent(params.voice || '');
    const rate = encodeURIComponent(String(params.rate ?? ''));
    const pitch = encodeURIComponent(String(params.pitch ?? ''));
    const style = encodeURIComponent(params.style || '');
    const format = encodeURIComponent(params.format || '');

    const query = [
      `v=${voice}`,
      `r=${rate}`,
      `p=${pitch}`,
      style ? `s=${style}` : '',
      format ? `f=${format}` : '',
      `n=${name}`
    ].filter(Boolean).join('&');

    return `${baseUrl}/reader.json?${query}`;
  }

  /**
   * 构建爱阅记APP链接
   */
  buildIfreetimeUrl(params) {
    const baseUrl = window.location.origin + config.basePath;
    const name = encodeURIComponent(params.voice || 'AzureTTS');
    const voice = encodeURIComponent(params.voice || '');
    const rate = encodeURIComponent(String(params.rate ?? ''));
    const pitch = encodeURIComponent(String(params.pitch ?? ''));
    const style = encodeURIComponent(params.style || '');
    const format = encodeURIComponent(params.format || '');

    const query = [
      `v=${voice}`,
      `r=${rate}`,
      `p=${pitch}`,
      style ? `s=${style}` : '',
      format ? `f=${format}` : '',
      `n=${name}`
    ].filter(Boolean).join('&');

    return `${baseUrl}/ifreetime.json?${query}`;
  }

  /**
   * 切换历史记录面板
   */
  toggleHistoryPanel() {
    const isOpen = this.elements.historyPanel?.classList.contains('open');

    if (isOpen) {
      this.closeHistoryPanel();
    } else {
      this.openHistoryPanel();
    }
  }

  /**
   * 打开历史记录面板
   */
  openHistoryPanel() {
    toggleClass(this.elements.historyPanel, 'open', true);
    this.renderHistory();
  }

  /**
   * 关闭历史记录面板
   */
  closeHistoryPanel() {
    toggleClass(this.elements.historyPanel, 'open', false);
  }

  /**
   * 渲染历史记录
   */
  renderHistory() {
    const history = loadHistory();

    if (!this.elements.historyPanelContent) return;

    if (history.length === 0) {
      this.elements.historyPanelContent.innerHTML = '<div class="history-empty">暂无历史记录</div>';
      return;
    }

    const html = history.map(item => `
      <div class="history-item">
        <div class="history-item-text">${this.truncate(item.text || item.ssml, 50)}</div>
        <div class="history-item-meta">
          <span>${item.voice}</span>
          <span>${new Date(item.timestamp).toLocaleString()}</span>
        </div>
      </div>
    `).join('');

    this.elements.historyPanelContent.innerHTML = html;
  }

  /**
   * 截断文本
   */
  truncate(text, length) {
    return text.length > length ? text.substring(0, length) + '...' : text;
  }

  /**
   * 设置加载状态
   * @param {boolean} loading - 是否加载中
   */
  setLoading(loading) {
    store.setState({ isLoading: loading });
    toggleClass(this.elements.loadingOverlay, 'show', loading);

    if (this.elements.speakButton) {
      this.elements.speakButton.disabled = loading;
    }
  }

  /**
   * 开始监控指标轮询
   */
  startMetricsPolling() {
    this.updateMetrics();
    setInterval(() => this.updateMetrics(), 30000); // 每30秒更新一次
  }

  /**
   * 更新监控指标
   */
  async updateMetrics() {
    try {
      const metrics = await this.api.getMetrics();
      this.renderMetrics(metrics);
    } catch (error) {
      console.error('获取指标失败:', error);
    }
  }

  /**
   * 渲染监控指标
   * @param {Object} metrics - 指标数据
   */
  renderMetrics(metrics) {
    if (!this.elements.metricsContainer) return;

    const html = `
      <div class="metric-item">
        <span class="metric-label">请求总数:</span>
        <span class="metric-value">${metrics.tts_requests_total || 0}</span>
      </div>
      <div class="metric-item">
        <span class="metric-label">成功率:</span>
        <span class="metric-value">${this.calculateSuccessRate(metrics)}%</span>
      </div>
      <div class="metric-item">
        <span class="metric-label">平均响应时间:</span>
        <span class="metric-value">${(metrics.tts_request_duration_seconds || 0).toFixed(2)}s</span>
      </div>
    `;

    this.elements.metricsContainer.innerHTML = html;
  }

  /**
   * 计算成功率
   */
  calculateSuccessRate(metrics) {
    const total = metrics.tts_requests_total || 0;
    const errors = metrics.tts_errors_total || 0;

    if (total === 0) return 100;

    return ((total - errors) / total * 100).toFixed(1);
  }
}

// 应用初始化
document.addEventListener('DOMContentLoaded', () => {
  window.ttsApp = new TTSApp();
});
