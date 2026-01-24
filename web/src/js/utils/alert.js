/**
 * 自定义提示组件
 */

import { createElement } from './dom.js';

/**
 * 显示提示消息
 * @param {string} message - 消息内容
 * @param {string} type - 消息类型 (success, error, warning, info)
 * @param {number} duration - 显示时长(ms)
 */
export function showAlert(message, type = 'info', duration = 3000) {
  const container = document.getElementById('custom-alert-container');
  if (!container) return;

  const alert = createElement('div', {
    className: `custom-alert custom-alert-${type}`,
    style: {
      opacity: '0',
      transform: 'translateY(-20px)'
    }
  });

  const icon = getAlertIcon(type);
  const iconEl = createElement('span', { className: 'custom-alert-icon' });
  iconEl.innerHTML = icon;

  const messageEl = createElement('span', { className: 'custom-alert-message' }, message);

  alert.appendChild(iconEl);
  alert.appendChild(messageEl);
  container.appendChild(alert);

  // 动画显示
  requestAnimationFrame(() => {
    alert.style.opacity = '1';
    alert.style.transform = 'translateY(0)';
  });

  // 自动移除
  setTimeout(() => {
    alert.style.opacity = '0';
    alert.style.transform = 'translateY(-20px)';
    setTimeout(() => {
      container.removeChild(alert);
    }, 300);
  }, duration);
}

/**
 * 获取提示图标
 * @param {string} type - 消息类型
 * @returns {string} SVG 图标
 */
function getAlertIcon(type) {
  const icons = {
    success: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" width="20" height="20"><path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/></svg>',
    error: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" width="20" height="20"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/></svg>',
    warning: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" width="20" height="20"><path d="M1 21h22L12 2 1 21zm12-3h-2v-2h2v2zm0-4h-2v-4h2v4z"/></svg>',
    info: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" width="20" height="20"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-6h2v6zm0-8h-2V7h2v2z"/></svg>'
  };
  return icons[type] || icons.info;
}

/**
 * 快捷方法
 */
export const alert = {
  success: (message, duration) => showAlert(message, 'success', duration),
  error: (message, duration) => showAlert(message, 'error', duration),
  warning: (message, duration) => showAlert(message, 'warning', duration),
  info: (message, duration) => showAlert(message, 'info', duration)
};
