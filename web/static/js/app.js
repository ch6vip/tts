// 全局状态
let isSSMLMode = false;
let voicesData = [];
let lastAudioUrl = '';
let audioPlayer = null;
let isPlaying = false;
let historyData = [];

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
        speakButton: document.getElementById('speak'),
        downloadButton: document.getElementById('download'),
        copyLinkButton: document.getElementById('copyLink'),
        copyHttpTtsLinkButton: document.getElementById('copyHttpTtsLink'),
        copyIfreetimeLinkButton: document.getElementById('copyIfreetimeLink'),
        audioPlayer: document.getElementById('audioPlayer'),
        resultSection: document.getElementById('resultSection'),
        charCount: document.getElementById('charCount'),
        toggleInputModeBtn: document.getElementById('toggleInputMode'),
        ssmlHelp: document.getElementById('ssmlHelp'),
        metricsContainer: document.getElementById('metrics-container'),
        // 新增元素
        loadingOverlay: document.getElementById('loadingOverlay'),
        historyPanel: document.getElementById('historyPanel'),
        historyPanelClose: document.getElementById('historyPanelClose'),
        historyPanelContent: document.getElementById('historyPanelContent'),
        historyToggle: document.getElementById('historyToggle'),
        textInputWarning: document.getElementById('textInputWarning'),
        // 语音选择增强元素
        voiceSearch: document.getElementById('voiceSearch'),
        previewVoice: document.getElementById('previewVoice'),
        styleDescription: document.getElementById('styleDescription'),
        // 音频播放器元素
        playPauseBtn: document.getElementById('playPauseBtn'),
        playIcon: document.getElementById('playIcon'),
        pauseIcon: document.getElementById('pauseIcon'),
        progressBar: document.getElementById('progressBar'),
        progressBarFill: document.getElementById('progressBarFill'),
        progressHandle: document.getElementById('progressHandle'),
        currentTime: document.getElementById('currentTime'),
        duration: document.getElementById('duration'),
        volumeSlider: document.getElementById('volumeSlider'),
        volumeBar: document.getElementById('volumeBar'),
        volumeHandle: document.getElementById('volumeHandle'),
        playbackSpeed: document.getElementById('playbackSpeed'),
    };

    // 初始化
    initEventListeners(elements);
    initVoicesList(elements);
    loadFormData(elements);
    initMetrics(elements);
    initAudioPlayer(elements);
    loadHistoryData();
    initKeyboardShortcuts(elements);
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
    const { textInput, ssmlInput, charCount, textInputWarning } = elements;
    if (!charCount) return;
    const count = isSSMLMode ? ssmlInput.value.length : textInput.value.length;
    charCount.textContent = count;
    
    // 添加警告提示
    if (textInputWarning) {
        if (count > 4500) {
            charCount.classList.add('danger');
            charCount.classList.remove('warning');
            textInputWarning.classList.add('show');
            textInputWarning.textContent = '文本长度接近限制，建议适当缩减内容';
        } else if (count > 4000) {
            charCount.classList.add('warning');
            charCount.classList.remove('danger');
            textInputWarning.classList.add('show');
            textInputWarning.textContent = '文本长度较长，可能影响处理速度';
        } else {
            charCount.classList.remove('warning', 'danger');
            textInputWarning.classList.remove('show');
        }
    }
}

// 初始化事件监听器
function initEventListeners(elements) {
    const {
        speakButton, textInput, ssmlInput, rateInput, pitchInput, voiceSelect, styleSelect,
        toggleInputModeBtn, downloadButton, copyLinkButton, copyHttpTtsLinkButton,
        copyIfreetimeLinkButton
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
            updatePreviewButton(elements);
        });
    }

    if (styleSelect) {
        styleSelect.addEventListener('change', () => {
            saveFormData(elements);
            updateStyleDescription(elements);
        });
    }

    // 语音搜索功能
    if (voiceSearch) {
        voiceSearch.addEventListener('input', () => {
            filterVoices(elements);
        });
    }

    // 语音预览功能
    if (previewVoice) {
        previewVoice.addEventListener('click', () => {
            previewCurrentVoice(elements);
        });
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

            let httpTtsLink = `${window.location.origin}${config.basePath}/reader.json?&v=${voice}&r=${rate}&p=${pitch}&n=${displayName}`;
            if (style) httpTtsLink += `&s=${style}`;
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
            let ifreetimeLink = `${window.location.origin}${config.basePath}/ifreetime.json?&v=${voice}&r=${rate}&p=${pitch}&n=${displayName}`;
            if (style) ifreetimeLink += `&s=${style}`;
            window.open(ifreetimeLink, '_blank');
        });
    }

    // 历史记录相关事件
    if (historyToggle) {
        historyToggle.addEventListener('click', toggleHistoryPanel);
    }

    if (historyPanelClose) {
        historyPanelClose.addEventListener('click', toggleHistoryPanel);
    }

}

// 生成语音核心函数
async function generateSpeech(elements) {
    const {
        textInput, ssmlInput, voiceSelect, styleSelect, rateInput, pitchInput,
        audioPlayer, resultSection
    } = elements;

    // 空值檢查：確保所有關鍵的 DOM 元素都存在
    const requiredElements = {
        textInput: textInput,
        ssmlInput: ssmlInput,
        voiceSelect: voiceSelect,
        styleSelect: styleSelect,
        rateInput: rateInput,
        pitchInput: pitchInput,
        audioPlayer: audioPlayer,
        resultSection: resultSection
    };

    const missingElements = [];
    for (const [name, element] of Object.entries(requiredElements)) {
        if (!element) {
            missingElements.push(name);
        }
    }

    if (missingElements.length > 0) {
        console.error('缺少以下 DOM 元素:', missingElements);
        showCustomAlert(
            `頁面元素載入異常，缺少: ${missingElements.join(', ')}。請嘗試刷新頁面或聯繫管理員。`,
            'error'
        );
        return;
    }

    setButtonLoading(elements, true);
    showLoadingOverlay('正在合成语音，请稍候...');
    saveFormData(elements);

    try {
        const inputText = isSSMLMode ? ssmlInput.value.trim() : textInput.value.trim();
        if (!inputText) {
            showCustomAlert(isSSMLMode ? '请输入 SSML 内容' : '请输入要转换的文本', 'warning');
            return;
        }

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

        const url = `${config.basePath}/tts?${params.toString()}`;
        const response = await fetch(url);

        if (!response.ok) {
            const errorData = await response.json().catch(() => ({ error: `HTTP 错误: ${response.status}` }));
            showCustomAlert(errorData.error || '合成失败', 'error');
            return;
        }

        const blob = await response.blob();
        const audioUrl = URL.createObjectURL(blob);

        resultSection.classList.remove('hidden');
        lastAudioUrl = url; // 保存用于复制链接的原始URL
        
        // 添加到历史记录
        addToHistory(inputText, voiceSelect.value, styleSelect.value, rateInput.value, pitchInput.value);
        
        // 初始化自定义音频播放器
        initCustomAudioPlayer(audioUrl, elements);

    } catch (e) {
        console.error("生成失败:", e);
        showCustomAlert('发生网络错误，请稍后重试。', 'error');
    } finally {
        setButtonLoading(elements, false);
        hideLoadingOverlay();
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
        // 初始化预览按钮和风格描述
        updatePreviewButton(elements);
        updateStyleDescription(elements);
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

// 初始化系统监控
async function initMetrics(elements) {
    const { metricsContainer } = elements;
    if (!metricsContainer) return;

    const fetchAndRenderMetrics = async () => {
        try {
            const response = await fetch(`${config.basePath}/metrics`);
            if (!response.ok) {
                metricsContainer.innerHTML = '<p class="text-red-400">无法加载系统监控数据。</p>';
                return;
            }
            const data = await response.json();
            
            const metricsHtml = `
                <div class="grid grid-cols-2 gap-x-4 gap-y-2">
                    <span class="font-semibold">TTS 请求:</span><span>${data.tts.requests} (成功: ${data.tts.success_rate}%)</span>
                    <span class="font-semibold">平均延迟:</span><span>${data.tts.latency.avg}</span>
                    <span class="font-semibold">缓存命中率:</span><span>${data.cache.hit_rate}%</span>
                    <span class="font-semibold">内存分配:</span><span>${data.system.memory.alloc_mb} MB</span>
                    <span class="font-semibold">协程数:</span><span>${data.system.goroutines}</span>
                </div>
            `;
            metricsContainer.innerHTML = metricsHtml;
        } catch (error) {
            console.error('获取监控数据失败:', error);
            metricsContainer.innerHTML = '<p class="text-red-400">加载失败，请检查服务状态。</p>';
        }
    };

    fetchAndRenderMetrics();
    setInterval(fetchAndRenderMetrics, 5000); // 每5秒刷新一次
}

// 显示加载遮罩
function showLoadingOverlay(text = '正在处理您的请求...') {
    const overlay = document.getElementById('loadingOverlay');
    if (overlay) {
        const loadingText = overlay.querySelector('.loading-text');
        if (loadingText) loadingText.textContent = text;
        overlay.classList.add('show');
    }
}

// 隐藏加载遮罩
function hideLoadingOverlay() {
    const overlay = document.getElementById('loadingOverlay');
    if (overlay) {
        overlay.classList.remove('show');
    }
}

// 历史记录功能
function loadHistoryData() {
    const saved = localStorage.getItem('ttsHistory');
    if (saved) {
        try {
            historyData = JSON.parse(saved);
        } catch (e) {
            console.error('加载历史记录失败:', e);
            historyData = [];
        }
    }
    renderHistoryPanel();
}

function saveHistoryData() {
    localStorage.setItem('ttsHistory', JSON.stringify(historyData));
}

function addToHistory(text, voice, style, rate, pitch) {
    const historyItem = {
        id: Date.now(),
        text: text.substring(0, 100) + (text.length > 100 ? '...' : ''),
        fullText: text,
        voice: voice,
        style: style,
        rate: rate,
        pitch: pitch,
        date: new Date().toLocaleString('zh-CN')
    };
    
    historyData.unshift(historyItem);
    if (historyData.length > 20) {
        historyData = historyData.slice(0, 20);
    }
    
    saveHistoryData();
    renderHistoryPanel();
}

function renderHistoryPanel() {
    const content = document.getElementById('historyPanelContent');
    if (!content) return;
    
    if (historyData.length === 0) {
        content.innerHTML = '<div class="history-empty">暂无历史记录</div>';
        return;
    }
    
    content.innerHTML = '';
    historyData.forEach(item => {
        const historyEl = document.createElement('div');
        historyEl.className = 'history-item fade-in';
        historyEl.innerHTML = `
            <div class="history-item-text">${item.text}</div>
            <div class="history-item-meta">
                <div class="history-item-voice">
                    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" width="12" height="12">
                        <path d="M12 14c1.66 0 3-1.34 3-3V5c0-1.66-1.34-3-3-3S9 3.34 9 5v6c0 1.66 1.34 3 3 3z"/>
                        <path d="M17 11c0 2.76-2.24 5-5 5s-5-2.24-5-5H5c0 3.53 2.61 6.43 6 6.92V21h2v-3.08c3.39-.49 6-3.39 6-6.92h-2z"/>
                    </svg>
                    ${getVoiceDisplayName(item.voice)}
                </div>
                <div class="history-item-date">${item.date}</div>
            </div>
        `;
        
        historyEl.addEventListener('click', () => {
            loadHistoryItem(item);
        });
        
        content.appendChild(historyEl);
    });
}

function getVoiceDisplayName(voiceId) {
    const voice = voicesData.find(v => v.short_name === voiceId);
    return voice ? (voice.local_name || voice.display_name) : voiceId;
}

function loadHistoryItem(item) {
    const { textInput, voiceSelect, styleSelect, rateInput, pitchInput } = elements;
    
    if (textInput) textInput.value = item.fullText;
    if (voiceSelect) voiceSelect.value = item.voice;
    if (styleSelect) styleSelect.value = item.style;
    if (rateInput) rateInput.value = item.rate;
    if (pitchInput) pitchInput.value = item.pitch;
    
    // 更新显示值
    if (elements.rateValue) elements.rateValue.textContent = item.rate + '%';
    if (elements.pitchValue) elements.pitchValue.textContent = item.pitch + '%';
    
    // 更新字符计数
    updateCharCount(elements);
    
    // 更新风格选项
    updateStyleOptions(elements);
    
    // 关闭历史面板
    toggleHistoryPanel();
    
    showCustomAlert('已加载历史记录', 'success');
}

function toggleHistoryPanel() {
    const panel = document.getElementById('historyPanel');
    if (panel) {
        panel.classList.toggle('show');
    }
}

// 自定义音频播放器
function initAudioPlayer(elements) {
    // 初始化音频播放器事件
    if (elements.playPauseBtn) {
        elements.playPauseBtn.addEventListener('click', togglePlayPause);
    }
    
    if (elements.progressBar) {
        elements.progressBar.addEventListener('click', seekAudio);
    }
    
    if (elements.volumeSlider) {
        elements.volumeSlider.addEventListener('click', changeVolume);
    }
    
    if (elements.playbackSpeed) {
        elements.playbackSpeed.addEventListener('change', changePlaybackSpeed);
    }
}

function initCustomAudioPlayer(audioUrl, elements) {
    if (!elements.audioPlayer) return;
    
    audioPlayer = elements.audioPlayer;
    
    // 先停止当前播放
    if (!audioPlayer.paused) {
        audioPlayer.pause();
    }
    
    // 设置新的音频源
    audioPlayer.src = audioUrl;
    
    // 设置初始音量
    audioPlayer.volume = 0.7;
    
    // 添加音频事件监听器
    audioPlayer.addEventListener('loadedmetadata', () => {
        updateDurationDisplay();
    });
    
    audioPlayer.addEventListener('timeupdate', () => {
        updateProgressDisplay();
    });
    
    audioPlayer.addEventListener('ended', () => {
        resetPlayer();
    });
    
    // 标记是否已尝试自动播放
    let hasAttemptedAutoplay = false;
    
    audioPlayer.addEventListener('canplay', () => {
        if (hasAttemptedAutoplay) return; // 避免重复尝试
        hasAttemptedAutoplay = true;
        
        // 尝试自动播放
        attemptAutoplay();
    });
    
    // 重置播放状态
    isPlaying = false;
    updatePlayPauseButton();
}

// 尝试自动播放的函数
function attemptAutoplay() {
    if (!audioPlayer) return;
    
    const playPromise = audioPlayer.play();
    if (playPromise !== undefined) {
        playPromise.then(() => {
            // 播放成功
            isPlaying = true;
            updatePlayPauseButton();
        }).catch(error => {
            console.warn('自动播放失败:', error);
            // 重置播放状态
            isPlaying = false;
            updatePlayPauseButton();
            
            // 如果是因为浏览器策略导致的失败，添加用户交互监听
            if (error.name === 'NotAllowedError') {
                // 添加一次性点击事件监听器，以便用户交互后播放
                const enableAutoplay = () => {
                    // 移除事件监听器
                    document.removeEventListener('click', enableAutoplay);
                    document.removeEventListener('keydown', enableAutoplay);
                    document.removeEventListener('touchstart', enableAutoplay);
                    
                    // 再次尝试播放
                    if (audioPlayer && audioPlayer.paused) {
                        audioPlayer.play().then(() => {
                            isPlaying = true;
                            updatePlayPauseButton();
                            showCustomAlert('音频开始播放', 'success');
                        }).catch(err => {
                            console.error('用户交互后播放仍然失败:', err);
                            showCustomAlert('请点击播放按钮手动播放', 'warning');
                        });
                    }
                };
                
                // 添加多种用户交互事件监听器
                document.addEventListener('click', enableAutoplay, { once: true });
                document.addEventListener('keydown', enableAutoplay, { once: true });
                document.addEventListener('touchstart', enableAutoplay, { once: true });
                
                showCustomAlert('浏览器阻止了自动播放，请点击页面任意位置开始播放', 'warning');
            } else {
                // 其他错误
                showCustomAlert('音频播放失败，请点击播放按钮手动播放', 'error');
            }
        });
    }
}

function togglePlayPause() {
    if (!audioPlayer) return;
    
    if (isPlaying) {
        audioPlayer.pause();
    } else {
        audioPlayer.play();
    }
    
    isPlaying = !isPlaying;
    updatePlayPauseButton();
}

function updatePlayPauseButton() {
    const playIcon = document.getElementById('playIcon');
    const pauseIcon = document.getElementById('pauseIcon');
    const playPauseBtn = document.getElementById('playPauseBtn');
    
    if (!playIcon || !pauseIcon || !playPauseBtn) return;
    
    if (isPlaying) {
        playIcon.classList.add('hidden');
        pauseIcon.classList.remove('hidden');
        playPauseBtn.classList.add('playing');
    } else {
        playIcon.classList.remove('hidden');
        pauseIcon.classList.add('hidden');
        playPauseBtn.classList.remove('playing');
    }
}

function seekAudio(event) {
    if (!audioPlayer || !audioPlayer.duration) return;
    
    const progressBar = event.currentTarget;
    const rect = progressBar.getBoundingClientRect();
    const percent = (event.clientX - rect.left) / rect.width;
    const newTime = percent * audioPlayer.duration;
    
    audioPlayer.currentTime = newTime;
    updateProgressDisplay();
}

function changeVolume(event) {
    if (!audioPlayer) return;
    
    const volumeSlider = event.currentTarget;
    const rect = volumeSlider.getBoundingClientRect();
    const percent = (event.clientX - rect.left) / rect.width;
    
    audioPlayer.volume = Math.max(0, Math.min(1, percent));
    updateVolumeDisplay();
}

function changePlaybackSpeed(event) {
    if (!audioPlayer) return;
    
    const speed = parseFloat(event.target.value);
    audioPlayer.playbackRate = speed;
}

function updateProgressDisplay() {
    if (!audioPlayer) return;
    
    const currentTime = audioPlayer.currentTime || 0;
    const duration = audioPlayer.duration || 0;
    const percent = duration > 0 ? (currentTime / duration) * 100 : 0;
    
    // 更新进度条
    const progressBarFill = document.getElementById('progressBarFill');
    const progressHandle = document.getElementById('progressHandle');
    
    if (progressBarFill) progressBarFill.style.width = percent + '%';
    if (progressHandle) progressHandle.style.left = percent + '%';
    
    // 更新时间显示
    updateTimeDisplay(currentTime, duration);
}

function updateDurationDisplay() {
    if (!audioPlayer) return;
    
    const duration = audioPlayer.duration || 0;
    const currentTime = audioPlayer.currentTime || 0;
    updateTimeDisplay(currentTime, duration);
}

function updateTimeDisplay(currentTime, duration) {
    const currentTimeEl = document.getElementById('currentTime');
    const durationEl = document.getElementById('duration');
    
    if (currentTimeEl) currentTimeEl.textContent = formatTime(currentTime);
    if (durationEl) durationEl.textContent = formatTime(duration);
}

function updateVolumeDisplay() {
    if (!audioPlayer) return;
    
    const volume = audioPlayer.volume || 0;
    const percent = volume * 100;
    
    const volumeBar = document.getElementById('volumeBar');
    const volumeHandle = document.getElementById('volumeHandle');
    
    if (volumeBar) volumeBar.style.width = percent + '%';
    if (volumeHandle) volumeHandle.style.left = percent + '%';
}

function formatTime(seconds) {
    if (isNaN(seconds)) return '0:00';
    
    const minutes = Math.floor(seconds / 60);
    const secs = Math.floor(seconds % 60);
    return `${minutes}:${secs < 10 ? '0' : ''}${secs}`;
}

function resetPlayer() {
    isPlaying = false;
    updatePlayPauseButton();
    
    // 重置进度
    const progressBarFill = document.getElementById('progressBarFill');
    const progressHandle = document.getElementById('progressHandle');
    
    if (progressBarFill) progressBarFill.style.width = '0%';
    if (progressHandle) progressHandle.style.left = '0%';
}

// 快捷键支持
function initKeyboardShortcuts(elements) {
    document.addEventListener('keydown', (event) => {
        // Ctrl+Enter 合成语音
        if (event.ctrlKey && event.key === 'Enter') {
            event.preventDefault();
            if (elements.speakButton && !elements.speakButton.disabled) {
                generateSpeech(elements);
            }
        }
        
        // Ctrl+H 切换历史记录
        if (event.ctrlKey && event.key === 'h') {
            event.preventDefault();
            toggleHistoryPanel();
        }
        
        // ESC 关闭面板
        if (event.key === 'Escape') {
            const historyPanel = document.getElementById('historyPanel');
            if (historyPanel && historyPanel.classList.contains('show')) {
                toggleHistoryPanel();
            }
        }
    });
}

// 语音搜索和预览功能
function filterVoices(elements) {
    const { voiceSearch, voiceSelect } = elements;
    if (!voiceSearch || !voiceSelect || voicesData.length === 0) return;
    
    const searchTerm = voiceSearch.value.toLowerCase().trim();
    const options = voiceSelect.querySelectorAll('option');
    
    // 遍历所有选项
    options.forEach(option => {
        if (option.value === 'loading') return;
        
        const text = option.textContent.toLowerCase();
        const voiceId = option.value.toLowerCase();
        
        // 检查是否匹配搜索词
        const matchesSearch = searchTerm === '' ||
                            text.includes(searchTerm) ||
                            voiceId.includes(searchTerm);
        
        // 显示或隐藏选项
        option.style.display = matchesSearch ? '' : 'none';
        
        // 如果是 optgroup，检查是否有可见的子选项
        if (option.parentNode.tagName === 'OPTGROUP') {
            const optgroup = option.parentNode;
            const visibleOptions = Array.from(optgroup.querySelectorAll('option')).some(
                opt => opt.style.display !== 'none'
            );
            optgroup.style.display = visibleOptions ? '' : 'none';
        }
    });
}

function updatePreviewButton(elements) {
    const { previewVoice, voiceSelect } = elements;
    if (!previewVoice || !voiceSelect) return;
    
    const selectedVoice = voiceSelect.value;
    previewVoice.disabled = !selectedVoice || selectedVoice === 'loading';
    
    if (selectedVoice && selectedVoice !== 'loading') {
        const voiceData = voicesData.find(v => v.short_name === selectedVoice);
        const voiceName = voiceData ? (voiceData.local_name || voiceData.display_name) : selectedVoice;
        previewVoice.textContent = `预览: ${voiceName}`;
    } else {
        previewVoice.textContent = '预览声音';
    }
}

function previewCurrentVoice(elements) {
    const { voiceSelect, styleSelect, rateInput, pitchInput } = elements;
    if (!voiceSelect || !voiceSelect.value || voiceSelect.value === 'loading') return;
    
    // 预览文本
    const previewText = "您好，这是语音预览效果。";
    const voice = voiceSelect.value;
    const style = styleSelect.value || 'general';
    const rate = rateInput.value || '0';
    const pitch = pitchInput.value || '0';
    
    // 禁用预览按钮，显示加载状态
    const { previewVoice } = elements;
    if (previewVoice) {
        previewVoice.disabled = true;
        previewVoice.textContent = '预览中...';
    }
    
    // 构建请求参数
    const params = new URLSearchParams();
    params.append('t', previewText);
    params.append('v', voice);
    params.append('r', rate);
    params.append('p', pitch);
    params.append('s', style);
    
    // 发送请求
    fetch(`${config.basePath}/tts?${params.toString()}`)
        .then(response => {
            if (!response.ok) throw new Error('预览失败');
            return response.blob();
        })
        .then(blob => {
            const audioUrl = URL.createObjectURL(blob);
            const audio = new Audio(audioUrl);
            
            // 添加错误处理
            audio.addEventListener('error', () => {
                console.error('音频播放失败');
                URL.revokeObjectURL(audioUrl);
                if (previewVoice) {
                    previewVoice.disabled = false;
                    updatePreviewButton(elements);
                }
                showCustomAlert('语音预览播放失败', 'error');
            });
            
            // 音频播放完成后恢复按钮状态
            audio.addEventListener('ended', () => {
                URL.revokeObjectURL(audioUrl);
                if (previewVoice) {
                    previewVoice.disabled = false;
                    updatePreviewButton(elements);
                }
            });
            
            // 尝试播放音频
            audio.play().catch(error => {
                console.error('音频播放失败:', error);
                URL.revokeObjectURL(audioUrl);
                if (previewVoice) {
                    previewVoice.disabled = false;
                    updatePreviewButton(elements);
                }
                showCustomAlert('语音预览播放失败', 'error');
            });
        })
        .catch(error => {
            console.error('语音预览失败:', error);
            showCustomAlert('语音预览失败，请稍后重试', 'error');
            
            // 恢复按钮状态
            if (previewVoice) {
                previewVoice.disabled = false;
                updatePreviewButton(elements);
            }
        });
}

function updateStyleDescription(elements) {
    const { styleSelect, styleDescription } = elements;
    if (!styleSelect || !styleDescription) return;
    
    const selectedStyle = styleSelect.value;
    
    if (!selectedStyle || selectedStyle === 'general') {
        styleDescription.classList.add('hidden');
        return;
    }
    
    // 根据不同风格显示不同的描述
    const descriptions = {
        'cheerful': '欢快活泼的语气，适合表达积极向上的内容',
        'sad': '悲伤低沉的语气，适合表达哀伤或严肃的内容',
        'angry': '愤怒激动的语气，适合表达强烈情绪',
        'fearful': '恐惧紧张的语气，适合表达悬疑或不安的内容',
        'disgruntled': '不满抱怨的语气，适合表达批评或不满',
        'gentle': '温柔轻缓的语气，适合表达亲切或安抚的内容',
        'affectionate': '亲切关爱的语气，适合表达温暖或关怀的内容',
        'embarrassed': '尴尬害羞的语气，适合表达腼腆或不好意思的内容',
        'calm': '平静沉稳的语气，适合表达冷静或客观的内容',
        'news': '新闻播报的语气，正式、清晰、客观',
        'customerservice': '客服语气，友好、耐心、专业',
        'chat': '聊天语气，自然、轻松、随和',
        'narration': '叙述语气，流畅、清晰、富有表现力',
        'newscast': '新闻播报风格，正式、权威、清晰',
        'poetryreading': '诗歌朗诵风格，富有韵律感和情感表达',
        'documentary': '纪录片解说风格，沉稳、专业、有说服力'
    };
    
    const description = descriptions[selectedStyle] || '选择不同的情感风格来改变语音的表达方式';
    styleDescription.textContent = description;
    styleDescription.classList.remove('hidden');
}

// 在 initVoicesList 函数中添加初始化预览按钮的调用
function initVoicesListWithPreview(elements) {
    initVoicesList(elements).then(() => {
        updatePreviewButton(elements);
        updateStyleDescription(elements);
    });
}
