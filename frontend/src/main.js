import './style.css';
import {ExecuteCommand, HideWindow, GetMountHistory, GetCustomers, GetVersion, SaveMount, RunMount, ExecuteConnect} from '../wailsjs/go/main/App';

GetVersion().then(v => {
    document.querySelectorAll('.version-label').forEach(el => el.textContent = v);
});

document.querySelector('#app').innerHTML = `
    <div class="container" id="page-execute">
        <div class="input-row">
            <input type="text" id="command" placeholder="Enter command..." autocomplete="off" spellcheck="false" />
        </div>
        <div class="output-box" id="output"></div>
        <div class="button-row">
            <button id="btn-execute">Execute</button>
            <button id="btn-close">Close</button>
        </div>
    </div>
    <div id="page-mount">
        <h3>Mount</h3>
        <span class="version-label"></span>
        <div class="mount-input-wrap">
            <input type="text" id="mount-input" placeholder="PBX name" autocomplete="off" spellcheck="false" />
            <div class="mount-suggestions" id="mount-suggestions"></div>
        </div>
        <div class="mount-buttons">
            <button id="btn-mount">Mount</button>
            <button id="btn-mount-close">Close</button>
        </div>
    </div>
    <div id="page-connect">
        <h3>Connect SSH</h3>
        <span class="version-label"></span>
        <div class="mount-input-wrap">
            <input type="text" id="connect-input" placeholder="PBX name" autocomplete="off" spellcheck="false" />
            <div class="mount-suggestions" id="connect-suggestions"></div>
        </div>
        <div class="mount-buttons">
            <button id="btn-connect">Connect</button>
            <button id="btn-connect-close">Close</button>
        </div>
    </div>
    <div id="page-result">
        <div class="result-output" id="result-output"></div>
        <div class="result-buttons">
            <button id="btn-result-ok">OK</button>
        </div>
    </div>
`;

// ── Execute page ─────────────────────────────────────────────────────────────

const commandInput = document.getElementById('command');
const outputEl     = document.getElementById('output');

commandInput.focus();

async function runCommand() {
    const cmd = commandInput.value.trim();
    if (!cmd) return;
    outputEl.textContent = 'Running...';
    outputEl.classList.remove('error');
    try {
        const result = await ExecuteCommand(cmd);
        outputEl.textContent = result || '(no output)';
        if (result.startsWith('Error:')) outputEl.classList.add('error');
    } catch (err) {
        outputEl.textContent = 'Error: ' + err;
        outputEl.classList.add('error');
    }
}

document.getElementById('btn-execute').addEventListener('click', runCommand);
document.getElementById('btn-close').addEventListener('click', () => HideWindow());
commandInput.addEventListener('keydown', (e) => { if (e.key === 'Enter') runCommand(); });

// ── Page switching ────────────────────────────────────────────────────────────

const EXEC_W = 500, EXEC_H = 260;
const MOUNT_W = 360, MOUNT_BASE_H = 200;
const SUGG_H  = 33;
const RESULT_W = 500, RESULT_H = 300;

const pageExecEl    = document.getElementById('page-execute');
const pageMountEl   = document.getElementById('page-mount');
const pageConnectEl = document.getElementById('page-connect');
const pageResultEl  = document.getElementById('page-result');

function showExecutePage() {
    pageMountEl.style.display   = 'none';
    pageConnectEl.style.display = 'none';
    pageResultEl.style.display  = 'none';
    pageExecEl.style.display    = 'flex';
    window.runtime.WindowSetSize(EXEC_W, EXEC_H);
    window.runtime.WindowSetTitle('Execute Command');
}

// ── Shared autocomplete helpers ───────────────────────────────────────────────

function makeAutocomplete(inputEl, suggestionsEl, getHistory, onSelect) {
    let activeIndex = -1;

    function hideSuggestions() {
        suggestionsEl.style.display = 'none';
        suggestionsEl.innerHTML = '';
        activeIndex = -1;
        window.runtime.WindowSetSize(MOUNT_W, MOUNT_BASE_H);
    }

    function setActive(index) {
        const items = suggestionsEl.querySelectorAll('.mount-suggestion');
        items.forEach((el, i) => el.classList.toggle('active', i === index));
        if (index >= 0 && index < items.length) inputEl.value = items[index].textContent;
    }

    function renderSuggestions(items) {
        suggestionsEl.innerHTML = '';
        activeIndex = -1;
        if (items.length === 0) { hideSuggestions(); return; }
        items.forEach((item) => {
            const div = document.createElement('div');
            div.className = 'mount-suggestion';
            div.textContent = item;
            div.addEventListener('mousedown', (e) => {
                e.preventDefault();
                inputEl.value = item;
                hideSuggestions();
                onSelect(item);
            });
            suggestionsEl.appendChild(div);
        });
        suggestionsEl.style.display = 'block';
        window.runtime.WindowSetSize(MOUNT_W, MOUNT_BASE_H + items.length * SUGG_H);
    }

    inputEl.addEventListener('input', () => {
        const val = inputEl.value;
        if (!val) { hideSuggestions(); return; }
        renderSuggestions(getHistory().filter(h => h.startsWith(val)).slice(0, 5));
    });

    inputEl.addEventListener('keydown', (e) => {
        const items   = suggestionsEl.querySelectorAll('.mount-suggestion');
        const showing = suggestionsEl.style.display === 'block' && items.length > 0;
        if (e.key === 'ArrowDown') {
            e.preventDefault();
            if (!showing) return;
            activeIndex = Math.min(activeIndex + 1, items.length - 1);
            setActive(activeIndex);
        } else if (e.key === 'ArrowUp') {
            e.preventDefault();
            if (!showing) return;
            activeIndex = Math.max(activeIndex - 1, -1);
            if (activeIndex === -1) items.forEach(el => el.classList.remove('active'));
            else setActive(activeIndex);
        } else if (e.key === 'Enter') {
            e.preventDefault();
            if (showing) hideSuggestions();
            onSelect(inputEl.value.trim());
        }
    });

    return { hideSuggestions, isSuggestionsVisible: () => suggestionsEl.style.display === 'block' };
}

// ── Mount page ───────────────────────────────────────────────────────────────

const mountInput      = document.getElementById('mount-input');
const mountSuggestEl  = document.getElementById('mount-suggestions');
let mountHistory = [];
let allSuggestions = [];

function mergeSuggestions(history, customers) {
    const seen = new Set(history);
    const extra = customers.filter(c => !seen.has(c));
    return [...history, ...extra];
}

const mountAC = makeAutocomplete(
    mountInput, mountSuggestEl,
    () => allSuggestions,
    (name) => { if (name) doMount(name); }
);

async function showMountPage() {
    const [history, customers] = await Promise.all([GetMountHistory(), GetCustomers()]);
    mountHistory = history;
    allSuggestions = mergeSuggestions(history, customers);
    pageExecEl.style.display    = 'none';
    pageConnectEl.style.display = 'none';
    pageResultEl.style.display  = 'none';
    pageMountEl.style.display   = 'flex';
    mountInput.value = '';
    mountAC.hideSuggestions();
    mountInput.focus();
}

function hideMountPage() {
    mountAC.hideSuggestions();
    showExecutePage();
    HideWindow();
}

async function doMount(name) {
    if (!name) return;
    await SaveMount(name);
    showResultPage(name);
}

document.getElementById('btn-mount').addEventListener('click', () => doMount(mountInput.value.trim()));
document.getElementById('btn-mount-close').addEventListener('click', hideMountPage);

// ── Connect SSH page ──────────────────────────────────────────────────────────

const connectInput    = document.getElementById('connect-input');
const connectSuggestEl = document.getElementById('connect-suggestions');

const connectAC = makeAutocomplete(
    connectInput, connectSuggestEl,
    () => allSuggestions,
    (name) => { if (name) doConnect(name); }
);

async function showConnectPage() {
    const [history, customers] = await Promise.all([GetMountHistory(), GetCustomers()]);
    mountHistory = history;
    allSuggestions = mergeSuggestions(history, customers);
    pageExecEl.style.display    = 'none';
    pageMountEl.style.display   = 'none';
    pageResultEl.style.display  = 'none';
    pageConnectEl.style.display = 'flex';
    connectInput.value = '';
    connectAC.hideSuggestions();
    connectInput.focus();
}

function hideConnectPage() {
    connectAC.hideSuggestions();
    showExecutePage();
    HideWindow();
}

function doConnect(name) {
    if (!name) return;
    hideConnectPage();
    ExecuteConnect(name); // Go: SaveMount + otevře terminál
}

document.getElementById('btn-connect').addEventListener('click', () => doConnect(connectInput.value.trim()));
document.getElementById('btn-connect-close').addEventListener('click', hideConnectPage);

// ── Result page ───────────────────────────────────────────────────────────────

const resultOutputEl = document.getElementById('result-output');
const resultOkBtn    = document.getElementById('btn-result-ok');

async function showResultPage(name) {
    pageExecEl.style.display    = 'none';
    pageMountEl.style.display   = 'none';
    pageConnectEl.style.display = 'none';
    mountAC.hideSuggestions();
    pageResultEl.style.display = 'flex';
    window.runtime.WindowSetSize(RESULT_W, RESULT_H);
    window.runtime.WindowSetTitle('Mount: ' + name);
    resultOutputEl.textContent = 'Running...';
    resultOkBtn.disabled = true;

    const output = await RunMount(name);
    resultOutputEl.textContent = output || '(no output)';
    resultOkBtn.disabled = false;
    resultOkBtn.focus();
}

function hideResultPage() {
    pageResultEl.style.display = 'none';
    showExecutePage();
    HideWindow();
}

resultOkBtn.addEventListener('click', hideResultPage);

// ── Global ESC / Enter handler (capture phase) ───────────────────────────────

document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
        e.preventDefault();
        if (pageResultEl.style.display === 'flex') {
            hideResultPage();
        } else if (pageMountEl.style.display === 'flex') {
            if (mountAC.isSuggestionsVisible()) mountAC.hideSuggestions();
            else hideMountPage();
        } else if (pageConnectEl.style.display === 'flex') {
            if (connectAC.isSuggestionsVisible()) connectAC.hideSuggestions();
            else hideConnectPage();
        } else {
            HideWindow();
        }
    } else if (e.key === 'Enter' && pageResultEl.style.display === 'flex') {
        e.preventDefault();
        hideResultPage();
    }
}, true);

// ── Wails events ──────────────────────────────────────────────────────────────

window.runtime.EventsOn('show-mount-dialog',  showMountPage);
window.runtime.EventsOn('show-connect-dialog', showConnectPage);
window.runtime.EventsOn('show-mount-result',  (name) => showResultPage(name));
