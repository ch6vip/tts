/**
 * VoiceSelector 组件
 * 负责语音选择和搜索
 */

import { $, toggleClass } from '../utils/dom.js';
import { debounce } from '../utils/dom.js';
import { loadFormData } from '../utils/storage.js';

class VoiceSelector {
  constructor(api) {
    this.api = api;
    this.voices = [];
    this.filteredVoices = [];
    this.elements = {};

    this.init();
  }

  /**
   * 初始化
   */
  init() {
    this.elements = {
      voiceSelect: $('#voice'),
      voiceSearch: $('#voiceSearch'),
      styleSelect: $('#style'),
      previewBtn: $('#previewVoice'),
      styleDescription: $('#styleDescription')
    };

    this.attachEvents();
    this.loadVoices();
  }

  /**
   * 绑定事件
   */
  attachEvents() {
    // 语音搜索
    if (this.elements.voiceSearch) {
      const debouncedSearch = debounce((e) => {
        this.filterVoices(e.target.value);
      }, 300);

      this.elements.voiceSearch.addEventListener('input', debouncedSearch);
    }

    // 语音选择
    this.elements.voiceSelect?.addEventListener('change', () => {
      this.onVoiceChange();
    });

    // 情感风格选择
    this.elements.styleSelect?.addEventListener('change', () => {
      this.onStyleChange();
    });

    // 预览语音
    this.elements.previewBtn?.addEventListener('click', () => {
      this.previewVoice();
    });
  }

  /**
   * 加载语音列表
   */
  async loadVoices() {
    try {
      const data = await this.api.getVoices();
      this.voices = data.voices || [];
      this.filteredVoices = [...this.voices];
      this.renderVoices();
    } catch (error) {
      console.error('加载语音列表失败:', error);
      this.showError('加载语音列表失败');
    }
  }

  /**
   * 渲染语音列表
   */
  renderVoices() {
    if (!this.elements.voiceSelect) return;

    this.elements.voiceSelect.innerHTML = '';

    if (this.filteredVoices.length === 0) {
      const option = document.createElement('option');
      option.value = '';
      option.textContent = '没有找到匹配的语音';
      this.elements.voiceSelect.appendChild(option);
      return;
    }

    this.filteredVoices.forEach(voice => {
      const option = document.createElement('option');
      option.value = voice.ShortName;
      // 优先使用 LocalName（中文名称），如果没有则使用 DisplayName
      const displayName = voice.LocalName || voice.DisplayName;
      // 性别翻译为中文
      const gender = voice.Gender === 'Female' ? '女' : voice.Gender === 'Male' ? '男' : voice.Gender;
      option.textContent = `${displayName} (${gender})`;
      this.elements.voiceSelect.appendChild(option);
    });

    // 设置默认选中：优先使用 localStorage 保存的选择，否则使用后端注入的 defaultVoice
    const saved = loadFormData();
    const candidates = [saved?.voice, window.config?.defaultVoice].filter(Boolean);
    for (const candidate of candidates) {
      if (this.elements.voiceSelect.querySelector(`option[value="${CSS.escape(candidate)}"]`)) {
        this.elements.voiceSelect.value = candidate;
        break;
      }
    }

    // 初始化对应的风格列表
    this.onVoiceChange();

    // 启用预览按钮
    if (this.elements.previewBtn) {
      this.elements.previewBtn.disabled = false;
    }
  }

  /**
   * 过滤语音
   * @param {string} query - 搜索关键词
   */
  filterVoices(query) {
    if (!query.trim()) {
      this.filteredVoices = [...this.voices];
    } else {
      const lowerQuery = query.toLowerCase();
      this.filteredVoices = this.voices.filter(voice => {
        return (
          voice.DisplayName.toLowerCase().includes(lowerQuery) ||
          voice.LocalName.toLowerCase().includes(lowerQuery) ||
          voice.ShortName.toLowerCase().includes(lowerQuery)
        );
      });
    }

    this.renderVoices();
  }

  /**
   * 语音变化事件
   */
  onVoiceChange() {
    const selectedVoice = this.getSelectedVoice();
    if (selectedVoice) {
      this.updateStyles(selectedVoice);
    }
  }

  /**
   * 风格变化事件
   */
  onStyleChange() {
    const style = this.elements.styleSelect?.value;
    if (style && this.elements.styleDescription) {
      toggleClass(this.elements.styleDescription, 'hidden', false);
    }
  }

  /**
   * 获取选中的语音
   * @returns {Object|null}
   */
  getSelectedVoice() {
    const voiceName = this.elements.voiceSelect?.value;
    if (!voiceName) return null;

    return this.voices.find(v => v.ShortName === voiceName);
  }

  /**
   * 更新可用的情感风格
   * @param {Object} voice - 语音对象
   */
  updateStyles(voice) {
    if (!this.elements.styleSelect) return;

    this.elements.styleSelect.innerHTML = '<option value="general">普通</option>';

    if (voice.StyleList && voice.StyleList.length > 0) {
      voice.StyleList.forEach(style => {
        const option = document.createElement('option');
        option.value = style;
        option.textContent = this.getStyleDisplayName(style);
        this.elements.styleSelect.appendChild(option);
      });
    }
  }

  /**
   * 获取风格显示名称
   * @param {string} style - 风格值
   * @returns {string}
   */
  getStyleDisplayName(style) {
    const styleNames = {
      'general': '普通',
      'cheerful': '开朗',
      'sad': '悲伤',
      'angry': '愤怒',
      'fearful': '恐惧',
      'calm': '平静',
      'assistant': '助手',
      'newscast': '新闻播报',
      'customerservice': '客服'
    };

    return styleNames[style] || style;
  }

  /**
   * 预览语音
   */
  async previewVoice() {
    const voice = this.getSelectedVoice();
    if (!voice) return;

    // TODO: 实现语音预览功能
    console.log('预览语音:', voice);
  }

  /**
   * 显示错误
   * @param {string} message - 错误消息
   */
  showError(message) {
    if (this.elements.voiceSelect) {
      this.elements.voiceSelect.innerHTML = `<option value="">${message}</option>`;
    }
  }
}

export default VoiceSelector;
