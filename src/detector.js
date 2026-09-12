const path = require('path');
const fs = require('fs');
const os = require('os');
const { execFileSync, spawn } = require('child_process');

// Name of the state marker written next to app.asar. It records what this tool
// last did, so language detection never has to guess from archive internals.
const STATE_MARKER_NAME = 'antigravity-i18n-state.json';

// Marker name used before the rename. It is still read so an install patched by
// an earlier release keeps its recorded language instead of reporting unknown.
const LEGACY_STATE_MARKER_NAMES = ['polygravity-state.json', 'antigravity-zh-state.json'];

// How long Antigravity is given to close on its own before it is force-killed.
const GRACEFUL_TIMEOUT_MS = 30000;
const GRACEFUL_POLL_INTERVAL_MS = 500;

function getPossibleAppDirs() {
    const platform = os.platform();
    const homedir = os.homedir();
    const dirs = [];

    // 1. Environment variable override
    if (process.env.ANTIGRAVITY_APP_DIR) {
        dirs.push(process.env.ANTIGRAVITY_APP_DIR);
    }

    if (platform === 'win32') {
        const localAppData = process.env.LOCALAPPDATA || path.join(homedir, 'AppData', 'Local');
        const programFiles = process.env.ProgramFiles || 'C:\\Program Files';
        const programFilesX86 = process.env['ProgramFiles(x86)'] || 'C:\\Program Files (x86)';

        dirs.push(
            path.join(localAppData, 'Programs', 'antigravity'),
            path.join(programFiles, 'Antigravity'),
            path.join(programFilesX86, 'Antigravity')
        );
    } else if (platform === 'darwin') {
        dirs.push(
            '/Applications/Antigravity.app/Contents/Resources',
            path.join(homedir, 'Applications', 'Antigravity.app', 'Contents', 'Resources')
        );
    } else {
        // Linux
        dirs.push(
            path.join(homedir, '.local', 'share', 'antigravity'),
            '/opt/antigravity',
            '/usr/lib/antigravity',
            '/usr/share/antigravity',
            path.join(homedir, 'antigravity'),
            // Manually installed apps often live in ~/Applications (Electron
            // updater installs Antigravity here with a capitalised name).
            path.join(homedir, 'Applications', 'Antigravity')
        );
    }

    return dirs;
}

function resolveAppPaths(customAppDir) {
    let appDir = customAppDir;

    if (!appDir) {
        const candidateDirs = getPossibleAppDirs();
        for (const candidate of candidateDirs) {
            if (fs.existsSync(candidate)) {
                // Check if resources/app.asar or app.asar directly exists
                if (fs.existsSync(path.join(candidate, 'resources', 'app.asar')) ||
                    fs.existsSync(path.join(candidate, 'app.asar'))) {
                    appDir = candidate;
                    break;
                }
            }
        }
    }

    if (!appDir || !fs.existsSync(appDir)) {
        throw new Error(
            `Could not locate Antigravity installation. Please specify the path using --app-dir <path> or set ANTIGRAVITY_APP_DIR environment variable.`
        );
    }

    appDir = path.resolve(appDir);
    let appDirStat;
    try {
        appDirStat = fs.statSync(appDir);
    } catch {
        appDirStat = null;
    }
    if (!appDirStat?.isDirectory()) {
        throw new Error(`Antigravity app directory is not a directory: ${appDir}`);
    }

    let asarPath = path.join(appDir, 'resources', 'app.asar');
    if (!fs.existsSync(asarPath) && fs.existsSync(path.join(appDir, 'app.asar'))) {
        asarPath = path.join(appDir, 'app.asar');
    }
    let asarStat;
    try {
        asarStat = fs.statSync(asarPath);
    } catch {
        asarStat = null;
    }
    if (!asarStat?.isFile()) {
        throw new Error(`Antigravity app.asar was not found or is not a file: ${asarPath}`);
    }

    const resourcesDir = path.dirname(asarPath);
    const cleanBackupPath = path.join(resourcesDir, 'app.asar.clean-backup');

    return {
        appDir,
        resourcesDir,
        asarPath,
        cleanBackupPath,
        statePath: path.join(resourcesDir, STATE_MARKER_NAME)
    };
}

/**
 * Read the state marker written by a previous run. Returns null when the marker
 * is absent or unreadable, which callers must treat as "unknown", not "English".
 *
 * @param {string} statePath
 * @returns {{ language: string, patchedAt?: string, asarSize?: number, version?: string }|null}
 */
function readState(statePath) {
    // The current marker wins; a marker left by a pre-rename release is read as
    // a fallback so an existing install does not lose its recorded language.
    const candidates = [statePath, ...LEGACY_STATE_MARKER_NAMES.map(
        (name) => path.join(path.dirname(statePath), name)
    )];
    for (const candidate of candidates) {
        try {
            const parsed = JSON.parse(fs.readFileSync(candidate, 'utf8'));
            if (parsed && typeof parsed.language === 'string') {
                return parsed;
            }
        } catch {
            // Try the next candidate.
        }
    }
    return null;
}

/**
 * Record what language this tool just applied. The asar size is stored so a
 * later run can notice that Antigravity replaced app.asar behind our back
 * (for example after an app update) and fall back to content detection.
 *
 * @param {string} statePath
 * @param {string} language - Installed locale code, or 'en' when restored.
 * @param {string} asarPath
 */
function writeState(statePath, language, asarPath) {
    let asarSize = null;
    try {
        asarSize = fs.statSync(asarPath).size;
    } catch {
        // Size is a best-effort staleness hint only.
    }

    const payload = {
        language,
        patchedAt: new Date().toISOString(),
        asarSize,
        toolVersion: getToolVersion()
    };

    try {
        fs.writeFileSync(statePath, JSON.stringify(payload, null, 2) + '\n', 'utf8');
        // A stale pre-rename marker beside the new one would keep being read as
        // a fallback, so it is removed once the new marker is safely written.
        for (const name of LEGACY_STATE_MARKER_NAMES) {
            const legacyPath = path.join(path.dirname(statePath), name);
            if (legacyPath !== statePath) {
                try {
                    fs.rmSync(legacyPath, { force: true });
                } catch {
                    // A leftover marker is harmless; the new one takes priority.
                }
            }
        }
    } catch {
        // A missing marker degrades detection but must never fail the patch.
    }
}

function getToolVersion() {
    try {
        return require('../package.json').version || null;
    } catch {
        return null;
    }
}

/**
 * Report whether any Antigravity process is still running.
 *
 * @returns {boolean}
 */
function isAntigravityRunning() {
    if (os.platform() === 'win32') {
        try {
            const out = execFileSync('tasklist.exe', ['/FI', 'IMAGENAME eq Antigravity.exe', '/NH'], {
                encoding: 'utf8',
                stdio: ['ignore', 'pipe', 'ignore'],
                shell: false
            });
            return /antigravity\.exe/i.test(out);
        } catch (err) {
            throw new Error(`Could not query Antigravity processes (tasklist): ${err.code || err.message}. Cannot confirm the app is stopped.`);
        }
    }

    // -x matches the executable name only, so this never matches our own
    // command line (which contains the string "antigravity-i18n"). Each name is
    // checked separately to keep process discovery shell-free.
    for (const name of ['Antigravity', 'antigravity']) {
        try {
            execFileSync('pgrep', ['-x', name], { stdio: 'ignore', shell: false });
            return true;
        } catch (err) {
            // pgrep uses status 1 for a successful query with no matches.
            // A missing command, denied access or another failure is unknown.
            if (err.status !== 1 || err.signal || err.code) {
                throw new Error(`Could not query Antigravity processes (pgrep): ${err.code || err.message}. Cannot confirm the app is stopped.`);
            }
        }
    }
    return false;
}

function tryExec(file, args) {
    try {
        execFileSync(file, args, { stdio: 'ignore', shell: false });
    } catch {
        // The process may already have exited; the follow-up probe decides.
    }
}

function sleepSync(ms) {
    Atomics.wait(new Int32Array(new SharedArrayBuffer(4)), 0, 0, ms);
}

/**
 * Ask Antigravity to close, then wait for it to exit before escalating.
 *
 * Patching rewrites app.asar in place, so the app genuinely must not be running.
 * But an unconditional force-kill discards unsaved editor state and in-flight
 * agent work, so a graceful request is always attempted first and a forced kill
 * only happens if the app is still alive after GRACEFUL_TIMEOUT_MS.
 *
 * @param {object} [options]
 * @param {boolean} [options.force=false] - Skip the graceful phase entirely.
 * @param {number} [options.timeoutMs]
 * @returns {{ wasRunning: boolean, stopped: boolean, forced: boolean }}
 */
function stopAntigravityProcesses(options = {}) {
    const platform = os.platform();
    const timeoutMs = Number.isFinite(options.timeoutMs) ? options.timeoutMs : GRACEFUL_TIMEOUT_MS;

    if (!isAntigravityRunning()) {
        return { wasRunning: false, stopped: true, forced: false };
    }

    if (!options.force) {
        // Phase 1: request a normal shutdown so the app can persist its state.
        try {
            if (platform === 'win32') {
                // No /F: this posts a close request to the window.
                tryExec('taskkill.exe', ['/IM', 'Antigravity.exe']);
            } else if (platform === 'darwin') {
                tryExec('osascript', ['-e', 'quit app "Antigravity"']);
            } else {
                tryExec('pkill', ['-TERM', '-x', 'Antigravity']);
                tryExec('pkill', ['-TERM', '-x', 'antigravity']);
            }
        } catch {
            // The app may refuse or have no window; the wait below decides.
        }

        const deadline = Date.now() + timeoutMs;
        while (Date.now() < deadline) {
            if (!isAntigravityRunning()) {
                return { wasRunning: true, stopped: true, forced: false };
            }
            sleepSync(GRACEFUL_POLL_INTERVAL_MS);
        }
    }

    // Phase 2: the app ignored the close request, so force it. Without this the
    // in-place app.asar rewrite would fail on a locked file.
    try {
        if (platform === 'win32') {
            tryExec('taskkill.exe', ['/F', '/IM', 'Antigravity.exe']);
        } else {
            tryExec('pkill', ['-9', '-x', 'Antigravity']);
            tryExec('pkill', ['-9', '-x', 'antigravity']);
        }
    } catch {
        // Ignore errors if the process exited between the check and the kill.
    }

    return { wasRunning: true, stopped: !isAntigravityRunning(), forced: true };
}

function launchAntigravity(appDir) {
    const platform = os.platform();
    try {
        if (platform === 'win32') {
            const exePath = path.join(appDir, 'Antigravity.exe');
            if (fs.existsSync(exePath)) {
                const child = spawn(exePath, [], {
                    detached: true,
                    stdio: 'ignore'
                });
                child.unref();
            }
        } else if (platform === 'darwin') {
            const appBundle = appDir.includes('.app') ? appDir.substring(0, appDir.indexOf('.app') + 4) : '/Applications/Antigravity.app';
            // The bundle path derives from --app-dir, so it is passed as an
            // argument rather than interpolated into a shell string.
            execFileSync('open', ['-a', appBundle], { stdio: 'ignore', shell: false });
        } else {
            // Linux
            const binPath = path.join(appDir, 'antigravity');
            if (fs.existsSync(binPath)) {
                const child = spawn(binPath, [], {
                    detached: true,
                    stdio: 'ignore'
                });
                child.unref();
            } else {
                const child = spawn('antigravity', [], {
                    detached: true,
                    stdio: 'ignore',
                    shell: false
                });
                child.unref();
            }
        }
    } catch {
        // Ignore launch errors
    }
}

module.exports = {
    getPossibleAppDirs,
    resolveAppPaths,
    stopAntigravityProcesses,
    launchAntigravity,
    isAntigravityRunning,
    readState,
    writeState,
    STATE_MARKER_NAME,
    LEGACY_STATE_MARKER_NAMES,
    GRACEFUL_TIMEOUT_MS
};
