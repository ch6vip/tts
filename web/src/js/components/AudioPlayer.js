/**
 * AudioPlayer 组件
 * 负责音频播放控制
 */

import { $, formatTime, toggleClass } from '../utils/dom.js';

class AudioPlayer {
  constructor() {
    this.audio = null;
    this.isPlaying = false;
    this.elements = {};

    this.init();
  }

  /**
   * 初始化音频播放器
   */
  init() {
    this.elements = {
      audio: $('#audioPlayer'),
      playPauseBtn: $('#playPauseBtn'),
      playIcon: $('#playIcon'),
      pauseIcon: $('#pauseIcon'),
      progressBar: $('#progressBar'),
      progressBarFill: $('#progressBarFill'),
      progressHandle: $('#progressHandle'),
      currentTime: $('#currentTime'),
      duration: $('#duration'),
      volumeSlider: $('#volumeSlider'),
      volumeBar: $('#volumeBar'),
      volumeHandle: $('#volumeHandle'),
      playbackSpeed: $('#playbackSpeed'),
      resultSection: $('#resultSection')
    };

    if (this.elements.audio) {
      this.audio = this.elements.audio;
      this.attachEvents();
    }
  }

  /**
   * 绑定事件
   */
  attachEvents() {
    // 播放/暂停按钮
    this.elements.playPauseBtn?.addEventListener('click', () => this.togglePlay());

    // 音频事件
    this.audio.addEventListener('loadedmetadata', () => this.onLoadedMetadata());
    this.audio.addEventListener('timeupdate', () => this.onTimeUpdate());
    this.audio.addEventListener('ended', () => this.onEnded());

    // 进度条拖拽
    this.elements.progressBar?.addEventListener('click', (e) => this.onProgressBarClick(e));
    this.initProgressDrag();

    // 音量控制
    this.initVolumeControl();

    // 播放速度
    this.elements.playbackSpeed?.addEventListener('change', (e) => {
      this.audio.playbackRate = parseFloat(e.target.value);
    });
  }

  /**
   * 加载音频
   * @param {string} url - 音频 URL
   */
  load(url) {
    if (this.audio) {
      this.audio.src = url;
      this.audio.load();
      this.show();
    }
  }

  /**
   * 播放/暂停切换
   */
  togglePlay() {
    if (!this.audio) return;

    if (this.isPlaying) {
      this.pause();
    } else {
      this.play();
    }
  }

  /**
   * 播放
   */
  async play() {
    if (!this.audio) return;

    try {
      await this.audio.play();
      this.isPlaying = true;
      this.updatePlayPauseIcon();
    } catch (error) {
      console.error('播放失败:', error);
    }
  }

  /**
   * 暂停
   */
  pause() {
    if (!this.audio) return;

    this.audio.pause();
    this.isPlaying = false;
    this.updatePlayPauseIcon();
  }

  /**
   * 停止
   */
  stop() {
    if (!this.audio) return;

    this.audio.pause();
    this.audio.currentTime = 0;
    this.isPlaying = false;
    this.updatePlayPauseIcon();
  }

  /**
   * 设置音量
   * @param {number} value - 音量值 (0-1)
   */
  setVolume(value) {
    if (this.audio) {
      this.audio.volume = Math.max(0, Math.min(1, value));
    }
  }

  /**
   * 跳转到指定时间
   * @param {number} time - 时间(秒)
   */
  seek(time) {
    if (this.audio) {
      this.audio.currentTime = time;
    }
  }

  /**
   * 显示播放器
   */
  show() {
    toggleClass(this.elements.resultSection, 'hidden', false);
  }

  /**
   * 隐藏播放器
   */
  hide() {
    toggleClass(this.elements.resultSection, 'hidden', true);
  }

  /**
   * 元数据加载完成
   */
  onLoadedMetadata() {
    if (this.elements.duration) {
      this.elements.duration.textContent = formatTime(this.audio.duration);
    }
  }

  /**
   * 时间更新
   */
  onTimeUpdate() {
    if (!this.audio) return;

    const currentTime = this.audio.currentTime;
    const duration = this.audio.duration;

    // 更新当前时间显示
    if (this.elements.currentTime) {
      this.elements.currentTime.textContent = formatTime(currentTime);
    }

    // 更新进度条
    if (this.elements.progressBarFill && this.elements.progressHandle && duration > 0) {
      const progress = (currentTime / duration) * 100;
      this.elements.progressBarFill.style.width = `${progress}%`;
      this.elements.progressHandle.style.left = `${progress}%`;
    }
  }

  /**
   * 播放结束
   */
  onEnded() {
    this.isPlaying = false;
    this.updatePlayPauseIcon();
  }

  /**
   * 更新播放/暂停图标
   */
  updatePlayPauseIcon() {
    if (this.isPlaying) {
      toggleClass(this.elements.playIcon, 'hidden', true);
      toggleClass(this.elements.pauseIcon, 'hidden', false);
    } else {
      toggleClass(this.elements.playIcon, 'hidden', false);
      toggleClass(this.elements.pauseIcon, 'hidden', true);
    }
  }

  /**
   * 进度条点击
   */
  onProgressBarClick(e) {
    if (!this.audio || !this.elements.progressBar) return;

    const rect = this.elements.progressBar.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const width = rect.width;
    const progress = x / width;

    this.seek(this.audio.duration * progress);
  }

  /**
   * 初始化进度条拖拽
   */
  initProgressDrag() {
    if (!this.elements.progressHandle) return;

    let isDragging = false;

    const onMouseDown = () => {
      isDragging = true;
    };

    const onMouseMove = (e) => {
      if (!isDragging || !this.elements.progressBar) return;

      const rect = this.elements.progressBar.getBoundingClientRect();
      const x = Math.max(0, Math.min(e.clientX - rect.left, rect.width));
      const progress = x / rect.width;

      this.seek(this.audio.duration * progress);
    };

    const onMouseUp = () => {
      isDragging = false;
    };

    this.elements.progressHandle.addEventListener('mousedown', onMouseDown);
    document.addEventListener('mousemove', onMouseMove);
    document.addEventListener('mouseup', onMouseUp);
  }

  /**
   * 初始化音量控制
   */
  initVolumeControl() {
    if (!this.elements.volumeSlider || !this.elements.volumeHandle) return;

    let isDragging = false;

    const updateVolume = (e) => {
      const rect = this.elements.volumeSlider.getBoundingClientRect();
      const x = Math.max(0, Math.min(e.clientX - rect.left, rect.width));
      const volume = x / rect.width;

      this.setVolume(volume);
      this.elements.volumeBar.style.width = `${volume * 100}%`;
      this.elements.volumeHandle.style.left = `${volume * 100}%`;
    };

    this.elements.volumeSlider.addEventListener('click', updateVolume);

    this.elements.volumeHandle.addEventListener('mousedown', () => {
      isDragging = true;
    });

    document.addEventListener('mousemove', (e) => {
      if (isDragging) {
        updateVolume(e);
      }
    });

    document.addEventListener('mouseup', () => {
      isDragging = false;
    });

    // 初始化音量为 100%
    this.setVolume(1);
    this.elements.volumeBar.style.width = '100%';
    this.elements.volumeHandle.style.left = '100%';
  }

  /**
   * 销毁播放器
   */
  destroy() {
    this.stop();
    if (this.audio) {
      this.audio.src = '';
    }
  }
}

export default AudioPlayer;
