/**
 * 输入验证工具
 */

/**
 * 验证文本长度
 * @param {string} text - 文本内容
 * @param {number} maxLength - 最大长度
 * @returns {Object} { valid: boolean, message: string }
 */
export function validateTextLength(text, maxLength = 5000) {
  if (!text || text.trim().length === 0) {
    return { valid: false, message: '请输入要转换的文本' };
  }

  if (text.length > maxLength) {
    return {
      valid: false,
      message: `文本长度超出限制，最多 ${maxLength} 个字符，当前 ${text.length} 个字符`
    };
  }

  return { valid: true, message: '' };
}

/**
 * 验证 SSML 格式
 * @param {string} ssml - SSML 内容
 * @returns {Object} { valid: boolean, message: string }
 */
export function validateSSML(ssml) {
  if (!ssml || ssml.trim().length === 0) {
    return { valid: false, message: '请输入 SSML 内容' };
  }

  // 基本的 SSML 格式验证
  if (!ssml.includes('<speak>') || !ssml.includes('</speak>')) {
    return {
      valid: false,
      message: 'SSML 格式错误：缺少 <speak> 标签'
    };
  }

  // 检查标签是否匹配
  const openTags = ssml.match(/<(\w+)[^>]*>/g) || [];
  const closeTags = ssml.match(/<\/(\w+)>/g) || [];

  if (openTags.length !== closeTags.length) {
    return {
      valid: false,
      message: 'SSML 格式错误：标签不匹配'
    };
  }

  return { valid: true, message: '' };
}

/**
 * 验证语速值
 * @param {number} rate - 语速值
 * @returns {boolean}
 */
export function validateRate(rate) {
  return typeof rate === 'number' && rate >= -100 && rate <= 100;
}

/**
 * 验证音调值
 * @param {number} pitch - 音调值
 * @returns {boolean}
 */
export function validatePitch(pitch) {
  return typeof pitch === 'number' && pitch >= -100 && pitch <= 100;
}

/**
 * 验证语音选择
 * @param {string} voice - 语音名称
 * @returns {Object} { valid: boolean, message: string }
 */
export function validateVoice(voice) {
  if (!voice || voice === 'loading') {
    return { valid: false, message: '请选择语音' };
  }
  return { valid: true, message: '' };
}

/**
 * 验证合成参数
 * @param {Object} params - 合成参数
 * @returns {Object} { valid: boolean, errors: Array }
 */
export function validateSynthesisParams(params) {
  const errors = [];

  // 验证文本或 SSML
  if (params.ssml) {
    const ssmlValidation = validateSSML(params.ssml);
    if (!ssmlValidation.valid) {
      errors.push(ssmlValidation.message);
    }
  } else {
    const textValidation = validateTextLength(params.text);
    if (!textValidation.valid) {
      errors.push(textValidation.message);
    }
  }

  // 验证语音
  const voiceValidation = validateVoice(params.voice);
  if (!voiceValidation.valid) {
    errors.push(voiceValidation.message);
  }

  // 验证语速和音调
  if (!validateRate(params.rate)) {
    errors.push('语速值无效');
  }

  if (!validatePitch(params.pitch)) {
    errors.push('音调值无效');
  }

  return {
    valid: errors.length === 0,
    errors
  };
}

/**
 * 清理文本（移除多余空白字符）
 * @param {string} text - 文本内容
 * @returns {string}
 */
export function sanitizeText(text) {
  return text.trim().replace(/\s+/g, ' ');
}
