// Frontend controller for antigravity-i18n.
//
// Field naming follows what Wails actually puts on the wire: types declared
// with json tags (LocaleInfo) arrive lower camel case, while plain structs
// (Status, ApplyResult, RestoreResult) arrive with their Go field names.
(function () {
  'use strict';

  var currentStatus = null;
  var locales = [];
  var selectedLocale = '';
  var isOperating = false;
  var operationLogs = [];

  var elAppVersionBadge = document.getElementById('appVersionBadge');
  var elAppVersion = document.getElementById('appVersion');
  var elAppPathDisplay = document.getElementById('appPathDisplay');
  var elBtnChangePath = document.getElementById('btnChangePath');
  var elBtnAbout = document.getElementById('btnAbout');
  var elStatusPill = document.getElementById('statusPill');
  var elStatusLabel = document.getElementById('statusLabel');
  var elStatusDesc = document.getElementById('statusDesc');
  var elProgressBox = document.getElementById('progressBox');
  var elProgressStage = document.getElementById('progressStage');
  var elProgressDetail = document.getElementById('progressDetail');
  var elBtnPrimaryAction = document.getElementById('btnPrimaryAction');
  var elBtnSecondaryAction = document.getElementById('btnSecondaryAction');
  var elBtnRestore = document.getElementById('btnRestore');
  var elLocaleChips = document.getElementById('localeChips');
  var elLocaleNote = document.getElementById('localeNote');
  var elBtnToggleLogs = document.getElementById('btnToggleLogs');
  var elLogDrawer = document.getElementById('logDrawer');
  var elLogContent = document.getElementById('logContent');
  var elBtnCopyLogs = document.getElementById('btnCopyLogs');
  var elBtnCloseLogs = document.getElementById('btnCloseLogs');
  var elAboutModal = document.getElementById('aboutModal');
  var elBtnCloseAbout = document.getElementById('btnCloseAbout');
  var elApplyConfirmModal = document.getElementById('applyConfirmModal');
  var elApplyConfirmTitle = document.getElementById('applyConfirmTitle');
  var elApplyConfirmBody = document.getElementById('applyConfirmBody');
  var elBtnCancelApply = document.getElementById('btnCancelApply');
  var elBtnConfirmApply = document.getElementById('btnConfirmApply');
  var elMsgModal = document.getElementById('msgModal');
  var elMsgTitle = document.getElementById('msgTitle');
  var elMsgBody = document.getElementById('msgBody');
  var elBtnMsgOk = document.getElementById('btnMsgOk');

  function backend() {
    return window.go && window.go.main && window.go.main.App ? window.go.main.App : null;
  }

  function errorText(err) {
    if (!err) return '未知错误';
    if (typeof err === 'string') return err;
    return err.message || String(err);
  }

  function appendLog(line) {
    operationLogs.push(line);
    elLogContent.textContent = operationLogs.join('\n');
    elLogContent.scrollTop = elLogContent.scrollHeight;
  }

  function showMessage(title, body) {
    elMsgTitle.textContent = title;
    elMsgBody.textContent = body;
    elMsgModal.classList.remove('hidden');
  }

  function setOperatingState(operating, stage, detail) {
    isOperating = operating;
    if (operating) {
      elProgressBox.classList.remove('hidden');
      elProgressStage.textContent = stage || '正在处理...';
      elProgressDetail.textContent = detail || '请稍候...';
    } else {
      elProgressBox.classList.add('hidden');
    }
    [elBtnPrimaryAction, elBtnSecondaryAction, elBtnRestore, elBtnChangePath].forEach(function (btn) {
      btn.disabled = operating;
    });
    Array.prototype.forEach.call(elLocaleChips.querySelectorAll('.chip'), function (chip) {
      chip.disabled = operating;
    });
  }

  function isPatched(st) {
    return !!(st && st.CurrentLanguage && st.CurrentLanguage !== 'en');
  }

  function refreshStatus() {
    var app = backend();
    if (!app) {
      elStatusPill.className = 'status-pill danger';
      elStatusLabel.textContent = '无法连接';
      elStatusDesc.textContent = '窗口运行库没有就绪，请重新启动本工具。';
      return Promise.resolve();
    }
    return app.GetStatus().then(function (st) {
      currentStatus = st;
      renderStatus(st);
      renderChips();
    }).catch(function (err) {
      currentStatus = null;
      elStatusPill.className = 'status-pill danger';
      elStatusLabel.textContent = '读取失败';
      elStatusDesc.textContent = errorText(err);
      appendLog('[错误] 读取状态失败: ' + errorText(err));
    });
  }

  function renderStatus(st) {
    if (!st) return;

    elAppPathDisplay.textContent = st.AppDir || '未检测到 Antigravity 安装位置';
    elAppPathDisplay.title = st.AppDir || '';
    elAppVersion.textContent = st.AppVersion ? 'v' + st.AppVersion : '版本未知';

    elStatusPill.className = 'status-pill';
    elBtnSecondaryAction.classList.add('hidden');
    elBtnRestore.classList.add('hidden');

    if (isPatched(st)) {
      elStatusPill.classList.add('success');
      elStatusLabel.textContent = '已本地化 · ' + st.CurrentLanguage;
      elStatusDesc.textContent = '当前界面语言为 ' + (st.LocaleName || st.CurrentLanguage) +
        '，可随时还原为官方英文版。';
      elBtnPrimaryAction.textContent = '重新应用';
      elBtnSecondaryAction.classList.remove('hidden');
      elBtnSecondaryAction.textContent = '打开 Antigravity';
      elBtnRestore.classList.remove('hidden');
    } else if (st.HasCleanBackup) {
      elStatusPill.classList.add('default');
      elStatusLabel.textContent = '官方英文';
      elStatusDesc.textContent = '当前是官方原始版本，干净备份可用，可以选择并应用界面语言。';
      elBtnPrimaryAction.textContent = '应用中文';
      elBtnSecondaryAction.classList.remove('hidden');
      elBtnSecondaryAction.textContent = '打开 Antigravity';
    } else {
      elStatusPill.classList.add('warning');
      elStatusLabel.textContent = '官方英文';
      elStatusDesc.textContent = '当前是官方原始版本；应用时会先创建一份干净备份供还原使用。';
      elBtnPrimaryAction.textContent = '应用中文';
    }

    if (st.CleanBackupStatus && st.CleanBackupStatus !== 'valid') {
      elStatusDesc.textContent += ' 干净备份状态：' + st.CleanBackupStatus + '。';
    }
    if (st.StateIsStale) {
      elStatusDesc.textContent += ' 状态文件与实际内容不一致，以 app.asar 的实际内容为准。';
    }
  }
  // The button names the locale the click will install, not the one already
  // installed. Saying "应用中文" while the selection is Japanese misreports what
  // the next click does.
  function selectedLabel() {
    for (var i = 0; i < locales.length; i++) {
      if (locales[i].code === selectedLocale) return locales[i].name;
    }
    return selectedLocale;
  }

  function localeLabel(code) {
    for (var i = 0; i < locales.length; i++) {
      if (locales[i].code === code) return locales[i].name + ' (' + code + ')';
    }
    return code;
  }

  function renderPrimaryLabel() {
    var label = selectedLocale === 'en' ? 'English' : selectedLabel();
    if (isPatched(currentStatus)) {
      elBtnPrimaryAction.textContent = currentStatus.CurrentLanguage === selectedLocale
        ? '重新应用 ' + label
        : '切换到 ' + label;
      return;
    }
    elBtnPrimaryAction.textContent = selectedLocale === 'en' ? '保持英文' : '应用 ' + label;
  }

  function renderChips() {
    elLocaleChips.innerHTML = '';
    if (!locales.length) {
      elLocaleNote.textContent = '没有可用的语言包';
      return;
    }
    var patched = isPatched(currentStatus);
    locales.forEach(function (loc) {
      var chip = document.createElement('button');
      chip.type = 'button';
      chip.className = 'chip' + (loc.coverage < 100 ? ' partial' : '');
      chip.setAttribute('aria-pressed', String(loc.code === selectedLocale));
      chip.disabled = isOperating;
      chip.title = loc.englishName;

      var name = document.createElement('span');
      name.textContent = loc.name;
      var cov = document.createElement('span');
      cov.className = 'cov';
      cov.textContent = loc.coverage + '%';
      chip.appendChild(name);
      chip.appendChild(cov);

      chip.addEventListener('click', function () {
        selectedLocale = loc.code;
        renderChips();
      });
      elLocaleChips.appendChild(chip);
    });
    elLocaleNote.textContent = patched
      ? '点击其他语言可直接切换'
      : '共 ' + locales.length + ' 个选项，选中 English 即保持官方英文';
    renderPrimaryLabel();
  }

  function loadLocales() {
    var app = backend();
    if (!app) return Promise.resolve();
    return app.GetLocales().then(function (list) {
      locales = list || [];
      if (!selectedLocale) {
        // English leads the list, so the fallback must name a real language pack
        // rather than whichever entry happens to be first.
        selectedLocale = isPatched(currentStatus)
          ? currentStatus.CurrentLanguage
          : 'zh-CN';
      }
      renderChips();
    }).catch(function (err) {
      elLocaleNote.textContent = '读取语言列表失败';
      appendLog('[错误] 读取语言列表失败: ' + errorText(err));
    });
  }

  function doApply() {
    if (isOperating) return;
    var app = backend();
    if (!app) return;
    var code = selectedLocale || 'zh-CN';

    // English is not a language pack: it is the untranslated build that restore
    // reinstalls. Routing it through apply could only fail on an unknown locale.
    if (code === 'en') {
      confirmAction('保持官方英文', code, function () {
        elApplyConfirmModal.classList.add('hidden');
        doRestore();
      });
      return;
    }

    confirmAction('切换到 ' + selectedLabel() + '?' , code, function () {
      elApplyConfirmModal.classList.add('hidden');
      runApply(app, code);
    });
  }

  // Both apply and restore stop Antigravity, and the process is killed outright
  // once the graceful window expires, so the user is asked before anything is
  // touched. The backend refuses to run without confirmation as well.
  function confirmAction(title, code, onConfirm) {
    elApplyConfirmTitle.textContent = title;
    elApplyConfirmBody.textContent = code === 'en'
      ? '将用官方英文版 app.asar 替换当前文件，并移除全部本地化补丁。'
      : '将改写 app.asar 中的界面语言资源，Antigravity 需要先关闭。';
    elBtnConfirmApply.onclick = onConfirm;
    elApplyConfirmModal.classList.remove('hidden');
  }

  function runApply(app, code) {
    setOperatingState(true, '正在改写 app.asar', '正在准备语言包与翻译资源...');
    appendLog('[操作] 应用语言包: ' + code);

    app.ApplyLocale(code, true).then(function (res) {
      var name = res && res.LocaleName ? res.LocaleName : code;
      appendLog('[完成] 已切换为 ' + name);
      showMessage('已应用', '界面语言已切换为 ' + name + '。' +
        (res && res.Relaunched ? 'Antigravity 已重新启动。' : '未能自动打开 Antigravity，请手动启动。'));
    }).catch(function (err) {
      var msg = errorText(err);
      appendLog('[错误] ' + msg);
      showMessage('应用失败', msg);
    }).then(function () {
      setOperatingState(false);
      return refreshStatus();
    });
  }

  function doRestore() {
    if (isOperating) return;
    var app = backend();
    if (!app) return;

    setOperatingState(true, '正在还原官方版本', '正在恢复干净备份并重算校验...');
    appendLog('[操作] 还原为官方英文');

    app.RestoreLocale(true).then(function (res) {
      appendLog('[完成] 已还原为官方英文');
      var message = res && res.AlreadyEnglish ? '当前已是官方英文版本。' : 'app.asar 已恢复为官方英文版本。';
      if (res && !res.AlreadyEnglish && !res.Relaunched) message += '未能自动打开 Antigravity，请手动启动。';
      showMessage('已还原', message);
    }).catch(function (err) {
      var msg = errorText(err);
      appendLog('[错误] ' + msg);
      showMessage('还原失败', msg);
    }).then(function () {
      setOperatingState(false);
      return refreshStatus();
    });
  }

  function launchApp() {
    var app = backend();
    if (!app) return;
    app.LaunchAntigravity().then(function () {
      appendLog('[操作] 已启动 Antigravity');
    }).catch(function (err) {
      showMessage('启动失败', errorText(err));
    });
  }

  function handlePrimaryAction() {
    if (isOperating) return;
    if (!currentStatus) {
      refreshStatus();
      return;
    }
    if (!currentStatus.AppDir) {
      handleChangePath();
      return;
    }
    doApply();
  }

  function handleChangePath() {
    var app = backend();
    if (!app) return;
    app.SelectAppDir().then(function (selected) {
      if (!selected) return null;
      setOperatingState(true, '正在切换安装位置', selected);
      return app.SetAppDir(selected).then(function (st) {
        currentStatus = st;
        renderStatus(st);
        return refreshStatus();
      });
    }).catch(function (err) {
      var msg = errorText(err);
      appendLog('[错误] ' + msg);
      showMessage('切换位置失败', msg);
    }).then(function () {
      setOperatingState(false);
    });
  }

  elBtnPrimaryAction.addEventListener('click', handlePrimaryAction);
  elBtnSecondaryAction.addEventListener('click', launchApp);
  elBtnChangePath.addEventListener('click', handleChangePath);
  // The trailing "restore" link and the primary button now ask the same
  // question through the same dialog, so there is one wording for a shutdown.
  elBtnRestore.addEventListener('click', function () {
    confirmAction('还原为官方英文？', 'en', function () {
      elApplyConfirmModal.classList.add('hidden');
      doRestore();
    });
  });
  elBtnCancelApply.addEventListener('click', function () {
    elApplyConfirmModal.classList.add('hidden');
    elBtnConfirmApply.onclick = null;
  });

  elBtnAbout.addEventListener('click', function () {
    elAboutModal.classList.remove('hidden');
  });
  elBtnCloseAbout.addEventListener('click', function () {
    elAboutModal.classList.add('hidden');
  });

  elBtnMsgOk.addEventListener('click', function () {
    elMsgModal.classList.add('hidden');
  });

  elBtnToggleLogs.addEventListener('click', function () {
    elLogDrawer.classList.toggle('hidden');
  });
  elBtnCloseLogs.addEventListener('click', function () {
    elLogDrawer.classList.add('hidden');
  });
  elBtnCopyLogs.addEventListener('click', function () {
    var text = operationLogs.join('\n');
    // navigator.clipboard is only exposed in a secure context and can be
    // absent in older WebView2 builds. Wails ships its own clipboard call, so
    // fall back to it instead of leaving the button doing nothing.
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text).then(function () {
        showMessage('已复制', '运行日志已复制到剪贴板。');
      }).catch(function () {
        copyLogsWithRuntime(text);
      });
      return;
    }
    copyLogsWithRuntime(text);
  });

  function copyLogsWithRuntime(text) {
    var rt = window.runtime;
    if (rt && typeof rt.ClipboardSetText === 'function') {
      try {
        var result = rt.ClipboardSetText(text);
        if (result && typeof result.then === 'function') {
          result.then(function (ok) {
            showMessage(ok === false ? '复制失败' : '已复制',
              ok === false ? '无法访问剪贴板，请手动选择日志内容。' : '运行日志已复制到剪贴板。');
          }).catch(function () {
            showMessage('复制失败', '无法访问剪贴板，请手动选择日志内容。');
          });
          return;
        }
        showMessage('已复制', '运行日志已复制到剪贴板。');
        return;
      } catch (err) {
        // Fall through to the manual-selection hint below.
      }
    }
    showMessage('复制失败', '无法访问剪贴板，请手动选择日志内容。');
  }

  function setupWailsEvents() {
    if (window.runtime && window.runtime.EventsOn) {
      window.runtime.EventsOn('progress', function (data) {
        if (!data || !isOperating) return;
        if (data.stage) elProgressStage.textContent = data.stage;
        if (data.detail) elProgressDetail.textContent = data.detail;
        appendLog('[进度] ' + (data.stage || '') + (data.detail ? ': ' + data.detail : ''));
      });
    }
  }

  window.addEventListener('DOMContentLoaded', function () {
    setupWailsEvents();
    // Wait briefly for the Wails IPC layer to attach window.go.
    setTimeout(function () {
      var app = backend();
      if (app) {
        app.GetVersion().then(function (v) {
          if (v) elAppVersionBadge.textContent = 'v' + v;
        }).catch(function () {});
      }
      refreshStatus().then(loadLocales);
    }, 120);
  });
})();
