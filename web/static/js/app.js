// 全局状态
let isSSMLMode = false;
let voicesData = [];
let lastAudioUrl = '';

document.addEventListener('DOMContentLoaded', function () {
    // 获取所有需要的 DOM 元素
    const elements = {
        textInput: document.getElementById('text'),
        ssmlInput: document.getElementById('ssml'),
        voiceSelect: document.getElementById('voice'),
        styleSelect: document.getElementById('style'),
        rateInput: document.getElementById('rate'),
        rateValue: document.getElementById('rateValue'),
        pitchInput: document.getElementById('pitch'),
        pitchValue: document.getElementById('pitchValue'),
        apiKeyInput: document.getElementById('api-key'),
        apiKeyGroup: document.getElementById('api-key-group'),
        speakButton: document.getElementById('speak'),
        downloadButton: document.getElementById('download'),
        copyLinkButton: document.getElementById('copyLink'),
        copyHttpTtsLinkButton: document.getElementById('copyHttpTtsLink'),
        copyIfreetimeLinkButton: document.getElementById('copyIfreetimeLink'),
        audioPlayer: document.getElementById('audioPlayer'),
        resultSection: document.getElementById('resultSection'),
        charCount: document.getElementById('charCount'),
        togglePasswordButton: document.getElementById('toggle-password'),
        toggleInputModeBtn: document.getElementById('toggleInputMode'),
        ssmlHelp: document.getElementById('ssmlHelp'),
        saveApiKeyBtn: document.getElementById('save-api-key-btn'),
    };

    // 初始化
    initEventListeners(elements);
    initVoicesList(elements);
    loadApiKeyFromLocalStorage(elements);
    loadFormData(elements);
});

// 设置按钮加载状态
function setButtonLoading(elements, isLoading) {
    const { speakButton } = elements;
    if (!speakButton) return;

    if (isLoading) {
        speakButton.disabled = true;
        speakButton.innerHTML = `<span class="loader"></span><span>正在合成...</span>`;
    } else {
        speakButton.disabled = false;
        speakButton.innerHTML = `
            <svg class="w-5 h-5" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor"><path d="M7 17H5a2 2 0 0 1-2-2V9a2 2 0 0 1 2-2h2v10Zm2 2h2a1 1 0 0 0 1-1v-1a1 1 0 0 0-1-1h-2a1 1 0 0 0-1 1v1a1 1 0 0 0 1 1Zm10.298-9.942a1 1 0 0 0-1.09.218L15 13.11V4a1 1 0 0 0-2 0v2.11l-3.21-3.832a1 1 0 0 0-1.58.125l-2.4 4A1 1 0 0 0 7 8h2a1 1 0 0 0 .89-.55l1.6-2.667 3.01 3.613a1 1 0 0 0 .8.4H17a1 1 0 0 0 .89-1.447l-1.6-2.666 1.708-2.05a1 1 0 0 0-1.12-1.664L14 7.234V9a1 1 0 0 0 2 0v-2.11l1.298-1.557a1 1 0 0 0 .22-1.09l-.7-1.226a1 1 0 0 0-1.366-.363L12 5.11V4a1 1 0 1 0-2 0v1.11L6.79.278a1 1 0 0 0-1.09-.218l-2.4 1.2a1 1 0 0 0-.472 1.366L6.29 9.308a1 1 0 0 0 1.366.472l.7-.35-1.92 3.2a1 1 0 0 0 .89 1.447H9v3a1 1 0 0 0 2 0v-3.11l1.79 2.148a1 1 0 0 0 1.58-.125l2.4-4a1 1 0 0 0-.22-1.31zM19.5 21a1.5 1.5 0 1 0 0-3 1.5 1.5 0 0 0 0 3z"/></svg>
            <span>立即合成</span>`;
    }
}

// 更新字符计数
function updateCharCount(elements) {
    const { textInput, ssmlInput, charCount } = elements;
    if (!charCount) return;
    const count = isSSMLMode ? ssmlInput.value.length : textInput.value.length;
    charCount.textContent = count;
}

// 初始化事件监听器
function initEventListeners(elements) {
    const {
        speakButton, textInput, ssmlInput, rateInput, pitchInput, voiceSelect, styleSelect,
        toggleInputModeBtn, downloadButton, copyLinkButton, copyHttpTtsLinkButton,
        copyIfreetimeLinkButton, apiKeyInput, togglePasswordButton
    } = elements;

    if (speakButton) {
        speakButton.addEventListener('click', () => generateSpeech(elements));
    }

    if (toggleInputModeBtn) {
        toggleInputModeBtn.addEventListener('click', () => {
            isSSMLMode = !isSSMLMode;
            const { textInput, ssmlInput, ssmlHelp } = elements;
            if (isSSMLMode) {
                textInput.classList.add('hidden');
                ssmlInput.classList.remove('hidden');
                ssmlHelp.classList.remove('hidden');
                toggleInputModeBtn.textContent = 'SSML模式';
            } else {
                textInput.classList.remove('hidden');
                ssmlInput.classList.add('hidden');
                ssmlHelp.classList.add('hidden');
                toggleInputModeBtn.textContent = '文本模式';
            }
            updateCharCount(elements);
        });
    }

    const handleInput = () => {
        updateCharCount(elements);
        saveFormData(elements);
    };
    if (textInput) textInput.addEventListener('input', handleInput);
    if (ssmlInput) ssmlInput.addEventListener('input', handleInput);

    if (rateInput) {
        rateInput.addEventListener('input', function () {
            elements.rateValue.textContent = this.value + '%';
            saveFormData(elements);
        });
    }

    if (pitchInput) {
        pitchInput.addEventListener('input', function () {
            elements.pitchValue.textContent = this.value + '%';
            saveFormData(elements);
        });
    }

    if (voiceSelect) {
        voiceSelect.addEventListener('change', () => {
            updateStyleOptions(elements);
            saveFormData(elements);
        });
    }

    if (styleSelect) {
        styleSelect.addEventListener('change', () => saveFormData(elements));
    }

    if (downloadButton) {
        downloadButton.addEventListener('click', () => {
            if (lastAudioUrl) {
                const a = document.createElement('a');
                a.href = lastAudioUrl.startsWith('blob:') ? elements.audioPlayer.src : lastAudioUrl;
                a.download = 'speech.mp3';
                document.body.appendChild(a);
                a.click();
                document.body.removeChild(a);
            }
        });
    }

    if (copyLinkButton) {
        copyLinkButton.addEventListener('click', () => {
            if (lastAudioUrl) {
                const fullUrl = new URL(lastAudioUrl, window.location.origin).href;
                copyToClipboard(fullUrl);
            }
        });
    }
    
    if (copyHttpTtsLinkButton) {
        copyHttpTtsLinkButton.addEventListener('click', function () {
            const text = "{{java.encodeURI(speakText)}}";
            const voice = voiceSelect.value;
            const displayName = voiceSelect.options[voiceSelect.selectedIndex].text;
            const style = styleSelect.value;
            const rate = "{{speakSpeed*4}}"
            const pitch = pitchInput.value;
            const apiKey = apiKeyInput.value.trim();

            let httpTtsLink = `${window.location.origin}${config.basePath}/reader.json?&v=${voice}&r=${rate}&p=${pitch}&n=${displayName}`;
            if (style) httpTtsLink += `&s=${style}`;
            if (apiKey) httpTtsLink += `&api_key=${apiKey}`;
            window.open(httpTtsLink, '_blank');
        });
    }

    if (copyIfreetimeLinkButton) {
        copyIfreetimeLinkButton.addEventListener('click', function () {
            const voice = voiceSelect.value;
            const displayName = voiceSelect.options[voiceSelect.selectedIndex].text;
            const style = styleSelect.value;
            const rate = rateInput.value;
            const pitch = pitchInput.value;
            const apiKey = apiKeyInput.value.trim();

            let ifreetimeLink = `${window.location.origin}${config.basePath}/ifreetime.json?&v=${voice}&r=${rate}&p=${pitch}&n=${displayName}`;
            if (style) ifreetimeLink += `&s=${style}`;
            if (apiKey) ifreetimeLink += `&api_key=${apiKey}`;
            window.open(ifreetimeLink, '_blank');
        });
    }

    if (togglePasswordButton) {
        togglePasswordButton.addEventListener('click', function () {
            const type = apiKeyInput.getAttribute('type') === 'password' ? 'text' : 'password';
            apiKeyInput.setAttribute('type', type);
            this.innerHTML = type === 'password'
                ? `<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor"><path d="M10 12a2 2 0 100-4 2 2 0 000 4z" /><path fill-rule="evenodd" d="M.458 10C1.732 5.943 5.522 3 10 3s8.268 2.943 9.542 7c-1.274 4.057-5.064 7-9.542 7S1.732 14.057.458 10zM14 10a4 4 0 11-8 0 4 4 0 018 0z" clip-rule="evenodd" /></svg>`
                : `<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l18 18" /></svg>`;
        });
    }

    if (apiKeyInput) {
        apiKeyInput.addEventListener('keydown', (event) => {
            if (event.key === 'Enter') {
                event.preventDefault();
                saveApiKey();
            }
        });
    }

    if (elements.saveApiKeyBtn) {
        elements.saveApiKeyBtn.addEventListener('click', saveApiKey);
    }
}

// 生成语音核心函数
async function generateSpeech(elements) {
    const {
        textInput, ssmlInput, voiceSelect, styleSelect, rateInput, pitchInput,
        apiKeyInput, audioPlayer, resultSection
    } = elements;

    setButtonLoading(elements, true);
    saveFormData(elements);

    try {
        const inputText = isSSMLMode ? ssmlInput.value.trim() : textInput.value.trim();
        if (!inputText) {
            showCustomAlert(isSSMLMode ? '请输入 SSML 内容' : '请输入要转换的文本', 'warning');
            return;
        }

        const apiKey = apiKeyInput.value.trim() || localStorage.getItem('apiKey');
        const params = new URLSearchParams();

        if (isSSMLMode) {
            params.append('ssml', inputText);
        } else {
            params.append('t', inputText);
            params.append('v', voiceSelect.value);
            params.append('r', rateInput.value);
            params.append('p', pitchInput.value);
            params.append('s', styleSelect.value);
        }

        if (apiKey) {
            params.append('api_key', apiKey);
        }

        const url = `${config.basePath}/tts?${params.toString()}`;
        const response = await fetch(url);

        if (!response.ok) {
            const errorData = await response.json().catch(() => ({ error: `HTTP 错误: ${response.status}` }));
            showCustomAlert(errorData.error || '合成失败', 'error');
            if (response.status === 401) {
                elements.apiKeyGroup.classList.remove('hidden');
            }
            return;
        }

        const blob = await response.blob();
        const audioUrl = URL.createObjectURL(blob);

        audioPlayer.src = audioUrl;
        audioPlayer.play();
        resultSection.classList.remove('hidden');
        lastAudioUrl = url; // 保存用于复制链接的原始URL

    } catch (e) {
        console.error("生成失败:", e);
        showCustomAlert('发生网络错误，请稍后重试。', 'error');
    } finally {
        setButtonLoading(elements, false);
    }
}

// 获取可用语音列表
async function initVoicesList(elements) {
    const { voiceSelect } = elements;
    try {
        const response = await fetch(`${config.basePath}/voices`);
        if (!response.ok) throw new Error('获取语音列表失败');

        voicesData = await response.json();
        voiceSelect.innerHTML = '';

        if (voicesData.length === 0) {
            voiceSelect.innerHTML = '<option value="loading">加载中...</option>';
            return;
        }

        const voicesByLocale = voicesData.reduce((acc, voice) => {
            if (!acc[voice.locale]) acc[voice.locale] = [];
            acc[voice.locale].push(voice);
            return acc;
        }, {});

        for (const locale in voicesByLocale) {
            const optgroup = document.createElement('optgroup');
            optgroup.label = voicesByLocale[locale][0].locale_name;
            voicesByLocale[locale].forEach(voice => {
                const option = document.createElement('option');
                option.value = voice.short_name;
                option.textContent = `${voice.local_name || voice.display_name} (${voice.gender})`;
                if (voice.short_name === config.defaultVoice) {
                    option.selected = true;
                }
                optgroup.appendChild(option);
            });
            voiceSelect.appendChild(optgroup);
        }
        updateStyleOptions(elements);
        // 语音列表加载后，再加载表单数据以确保选中项正确
        loadFormData(elements);
    } catch (error) {
        console.error('获取语音列表失败:', error);
        voiceSelect.innerHTML = '<option value="" class="text-red-500">无法加载语音列表</option>';
    }
}

// 更新风格选项
function updateStyleOptions(elements) {
    const { voiceSelect, styleSelect } = elements;
    const selectedVoiceName = voiceSelect.value;
    const voiceData = voicesData.find(v => v.short_name === selectedVoiceName);

    styleSelect.innerHTML = ''; // 清空

    if (!voiceData || !voiceData.style_list || voiceData.style_list.length === 0) {
        styleSelect.innerHTML = '<option value="general">普通</option>';
        return;
    }

    styleSelect.innerHTML = '<option value="">-- 无风格 --</option>';
    voiceData.style_list.forEach(style => {
        const option = document.createElement('option');
        option.value = style;
        option.textContent = style;
        if (style === config.defaultStyle || (!config.defaultStyle && style === "general")) {
            option.selected = true;
        }
        styleSelect.appendChild(option);
    });
}

// 保存/加载 API Key
function saveApiKey() {
    const apiKeyInput = document.getElementById('api-key');
    const apiKey = apiKeyInput.value.trim();
    if (apiKey) {
        localStorage.setItem('apiKey', apiKey);
        showCustomAlert('API Key 已保存', 'success');
        document.getElementById('api-key-status').classList.remove('hidden');
    } else {
        localStorage.removeItem('apiKey');
        showCustomAlert('API Key 已清除', 'info');
        document.getElementById('api-key-status').classList.add('hidden');
    }
}

function loadApiKeyFromLocalStorage(elements) {
    const { apiKeyInput, apiKeyStatus } = elements;
    const apiKey = localStorage.getItem('apiKey');
    if (apiKey && apiKeyInput) {
        apiKeyInput.value = apiKey;
        if (apiKeyStatus) {
            apiKeyStatus.classList.remove('hidden');
        }
    }
}

// 保存/加载表单数据
function saveFormData(elements) {
    const { textInput, voiceSelect, styleSelect, rateInput, pitchInput } = elements;
    localStorage.setItem('ttsText', textInput.value);
    localStorage.setItem('ttsVoice', voiceSelect.value);
    localStorage.setItem('ttsStyle', styleSelect.value);
    localStorage.setItem('ttsRate', rateInput.value);
    localStorage.setItem('ttsPitch', pitchInput.value);
}

function loadFormData(elements) {
    const { textInput, voiceSelect, styleSelect, rateInput, rateValue, pitchInput, pitchValue, charCount } = elements;

    const savedText = localStorage.getItem('ttsText');
    if (savedText && textInput) {
        textInput.value = savedText;
        if (charCount) charCount.textContent = savedText.length;
    }

    const savedRate = localStorage.getItem('ttsRate');
    if (savedRate && rateInput) {
        rateInput.value = savedRate;
        if (rateValue) rateValue.textContent = savedRate + '%';
    }

    const savedPitch = localStorage.getItem('ttsPitch');
    if (savedPitch && pitchInput) {
        pitchInput.value = savedPitch;
        if (pitchValue) pitchValue.textContent = savedPitch + '%';
    }

    const savedVoice = localStorage.getItem('ttsVoice');
    if (savedVoice && voiceSelect.options.length > 1) {
        voiceSelect.value = savedVoice;
        updateStyleOptions(elements); // 更新风格
        const savedStyle = localStorage.getItem('ttsStyle');
        if (savedStyle && styleSelect) {
            styleSelect.value = savedStyle;
        }
    }
}

// 工具函数
function copyToClipboard(text) {
    navigator.clipboard.writeText(text).then(() => {
        showCustomAlert('链接已复制到剪贴板', 'success');
    }).catch(err => {
        console.error('复制失败:', err);
        showCustomAlert('复制失败', 'error');
    });
}

// 自定义 Alert
function showCustomAlert(message, type = 'info', title = '', duration = 3000) {
    let container = document.getElementById('custom-alert-container');
    if (!container) {
        container = document.createElement('div');
        container.id = 'custom-alert-container';
        document.body.appendChild(container);
    }

    const alert = document.createElement('div');
    alert.className = `custom-alert ${type}`;
    
    const icons = {
        success: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor"><path d="M12 22C6.477 22 2 17.523 2 12S6.477 2 12 2s10 4.477 10 10-4.477 10-10 10zm-.997-6l7.07-7.071-1.414-1.414-5.656 5.657-2.829-2.829-1.414 1.414L11.003 16z"/></svg>',
        error: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor"><path d="M12 22C6.477 22 2 17.523 2 12S6.477 2 12 2s10 4.477 10-10-4.477 10-10 10zm-1-7v2h2v-2h-2zm0-8v6h2V7h-2z"/></svg>',
        warning: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor"><path d="M12 22C6.477 22 2 17.523 2 12S6.477 2 12 2s10 4.477 10-10-4.477 10-10 10zm-1-7v2h2v-2h-2zm0-8v6h2V7h-2z"/></svg>',
        info: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor"><path d="M12 22C6.477 22 2 17.523 2 12S6.477 2 12 2s10 4.477 10-10-4.477 10-10 10zm-1-11v6h2v-6h-2zm0-4v2h2V7h-2z"/></svg>'
    };

    alert.innerHTML = `
        <div class="custom-alert-icon">${icons[type] || icons.info}</div>
        <div class="custom-alert-content">
            ${title ? `<h4>${title}</h4>` : ''}
            <p class="custom-alert-message">${message}</p>
        </div>
        <button class="custom-alert-close" aria-label="关闭">
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor"><path d="M12 10.586l4.95-4.95 1.414 1.414-4.95 4.95 4.95 4.95-1.414 1.414-4.95-4.95-4.95 4.95-1.414-1.414 4.95-4.95-4.95-4.95L7.05 5.636z"/></svg>
        </button>
        <div class="custom-alert-progress" style="animation-duration: ${duration}ms;"></div>
    `;
    
    container.appendChild(alert);

    const removeAlert = (element) => {
        element.classList.remove('show');
        setTimeout(() => {
            if (element.parentNode) element.parentNode.removeChild(element);
        }, 300);
    };

    const closeBtn = alert.querySelector('.custom-alert-close');
    closeBtn.addEventListener('click', () => removeAlert(alert));

    setTimeout(() => alert.classList.add('show'), 10);
    const timeout = setTimeout(() => removeAlert(alert), duration);
    
    alert.addEventListener('mouseover', () => clearTimeout(timeout));
}
