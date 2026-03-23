// AppWrap Web UI - Vanilla JS SPA

(function () {
    'use strict';

    // ===== Event Kind Constants (match Go EventKind iota) =====
    const EventKind = {
        INFO: 0,
        PROGRESS: 1,
        LOG_LINE: 2,
        WARNING: 3,
        ERROR: 4,
        COMPLETE: 5,
    };

    const kindClass = {
        [EventKind.INFO]: 'info',
        [EventKind.PROGRESS]: 'progress',
        [EventKind.LOG_LINE]: 'log',
        [EventKind.WARNING]: 'warning',
        [EventKind.ERROR]: 'error',
        [EventKind.COMPLETE]: 'complete',
    };

    // ===== API Helper =====
    async function api(method, path, body) {
        const opts = {
            method,
            headers: { 'Content-Type': 'application/json' },
        };
        if (body) opts.body = JSON.stringify(body);
        const res = await fetch('/api' + path, opts);
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || `HTTP ${res.status}`);
        return data;
    }

    // ===== Toast System =====
    function toast(msg, type = 'info', duration = 4000) {
        const container = document.getElementById('toast-container');
        const el = document.createElement('div');
        el.className = `toast ${type}`;
        el.innerHTML = `<span>${esc(msg)}</span><button class="toast-close" onclick="this.parentElement.remove()">&times;</button>`;
        container.appendChild(el);
        setTimeout(() => el.remove(), duration);
    }

    // ===== Escape HTML =====
    function esc(s) {
        if (s == null) return '';
        const d = document.createElement('div');
        d.textContent = String(s);
        return d.innerHTML;
    }

    // ===== WebSocket Manager =====
    function connectWS(opId, onEvent, onDone) {
        const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
        const ws = new WebSocket(`${proto}//${location.host}/ws/events/${opId}`);
        ws.onmessage = (e) => {
            try {
                const evt = JSON.parse(e.data);
                if (evt.message === '__done__') {
                    if (onDone) onDone();
                    ws.close();
                    return;
                }
                onEvent(evt);
                if (evt.kind === EventKind.COMPLETE) {
                    if (onDone) onDone();
                }
            } catch (err) {
                console.error('WS parse error:', err);
            }
        };
        ws.onerror = () => toast('WebSocket error', 'error');
        ws.onclose = () => { if (onDone) onDone(); };
        return ws;
    }

    // ===== Log Panel Component =====
    function createLogPanel(parentEl) {
        const panel = document.createElement('div');
        panel.className = 'log-panel';
        parentEl.appendChild(panel);

        const progressBar = document.createElement('div');
        progressBar.className = 'progress-bar';
        progressBar.innerHTML = '<div class="progress-bar-fill"></div>';
        parentEl.insertBefore(progressBar, panel);
        const fill = progressBar.querySelector('.progress-bar-fill');

        return {
            el: panel,
            addEvent(evt) {
                const cls = kindClass[evt.kind] || 'log';
                const line = document.createElement('div');
                line.className = `log-line ${cls}`;
                let prefix = '';
                if (evt.kind === EventKind.PROGRESS && evt.percent) {
                    prefix = `[${evt.percent}%] `;
                    fill.style.width = evt.percent + '%';
                }
                line.textContent = prefix + (evt.message || '');
                panel.appendChild(line);
                panel.scrollTop = panel.scrollHeight;
            },
            clear() {
                panel.innerHTML = '';
                fill.style.width = '0%';
            },
        };
    }

    // ===== Router =====
    const content = () => document.getElementById('content');
    let currentPage = '';

    const routes = {
        dashboard: renderDashboard,
        scan: renderScan,
        build: renderBuild,
        run: renderRun,
        inspect: renderInspect,
        keygen: renderKeygen,
        profiles: renderProfiles,
        containers: renderContainers,
    };

    function navigate(page) {
        if (!routes[page]) page = 'dashboard';
        currentPage = page;

        // Update nav
        document.querySelectorAll('.nav-link').forEach((link) => {
            link.classList.toggle('active', link.dataset.page === page);
        });

        // Render
        content().innerHTML = '';
        routes[page]();
    }

    // Listen for hash changes
    window.addEventListener('hashchange', () => {
        const page = location.hash.replace('#', '') || 'dashboard';
        navigate(page);
    });

    // Nav clicks
    document.querySelectorAll('.nav-link').forEach((link) => {
        link.addEventListener('click', (e) => {
            e.preventDefault();
            const page = link.dataset.page;
            location.hash = '#' + page;
        });
    });

    // ===== Status Check =====
    async function checkStatus() {
        try {
            const status = await api('GET', '/status');
            const el = document.getElementById('docker-status');
            const badge = document.getElementById('version-badge');
            if (status.docker) {
                el.className = 'docker-status connected';
                el.querySelector('.status-text').textContent = 'Docker Connected';
            } else {
                el.className = 'docker-status disconnected';
                el.querySelector('.status-text').textContent = 'Docker Unavailable';
            }
            if (status.version) badge.textContent = 'v' + status.version;
            return status;
        } catch {
            const el = document.getElementById('docker-status');
            el.className = 'docker-status disconnected';
            el.querySelector('.status-text').textContent = 'Server Error';
            return { docker: false, version: '?' };
        }
    }

    // ===== Page: Dashboard =====
    async function renderDashboard() {
        const el = content();
        el.innerHTML = `
            <div class="page-header">
                <h1>Dashboard</h1>
                <p>AppWrap container builder overview</p>
            </div>
            <div class="card-grid" id="dash-stats">
                <div class="stat-card">
                    <span class="stat-label">Docker</span>
                    <span class="stat-value" id="dash-docker"><span class="spinner"></span></span>
                    <span class="stat-desc">Engine status</span>
                </div>
                <div class="stat-card">
                    <span class="stat-label">Profiles</span>
                    <span class="stat-value" id="dash-profiles"><span class="spinner"></span></span>
                    <span class="stat-desc">Available profiles</span>
                </div>
                <div class="stat-card">
                    <span class="stat-label">Containers</span>
                    <span class="stat-value" id="dash-containers"><span class="spinner"></span></span>
                    <span class="stat-desc">Running containers</span>
                </div>
            </div>
            <div class="card">
                <div class="card-header"><h2>Quick Actions</h2></div>
                <div class="btn-group">
                    <button class="btn btn-primary" onclick="location.hash='#scan'">Scan Application</button>
                    <button class="btn btn-secondary" onclick="location.hash='#build'">Build Image</button>
                    <button class="btn btn-secondary" onclick="location.hash='#run'">Run Container</button>
                </div>
            </div>
        `;

        // Load stats
        try {
            const status = await checkStatus();
            document.getElementById('dash-docker').textContent = status.docker ? 'Online' : 'Offline';
            document.getElementById('dash-docker').style.color = status.docker ? 'var(--success)' : 'var(--danger)';
        } catch { document.getElementById('dash-docker').textContent = '?'; }

        try {
            const profiles = await api('GET', '/profiles');
            document.getElementById('dash-profiles').textContent = profiles.length;
        } catch { document.getElementById('dash-profiles').textContent = '0'; }

        try {
            const containers = await api('GET', '/containers');
            const running = containers.filter((c) => c.state === 'running').length;
            document.getElementById('dash-containers').textContent = running;
        } catch { document.getElementById('dash-containers').textContent = '0'; }
    }

    // ===== App Browser Modal =====
    function openAppBrowser(onSelect) {
        const overlay = document.createElement('div');
        overlay.className = 'modal-overlay';
        overlay.innerHTML = `
            <div class="modal">
                <div class="modal-header">
                    <h2>Browse Installed Applications</h2>
                    <button class="modal-close" id="modal-close-btn">&times;</button>
                </div>
                <div class="modal-filter">
                    <input type="text" id="app-filter" placeholder="Search apps by name or publisher..." autofocus>
                    <div class="filter-count" id="app-count"><span class="spinner"></span> Scanning installed apps...</div>
                </div>
                <div class="modal-body" id="app-list-body">
                    <div class="loading-center"><span class="spinner"></span> Discovering applications...</div>
                </div>
                <div class="modal-footer">
                    <button class="btn btn-secondary" id="modal-cancel-btn">Cancel</button>
                    <button class="btn btn-primary" id="modal-select-btn" disabled>Select</button>
                </div>
            </div>
        `;
        document.body.appendChild(overlay);

        let allApps = [];
        let selectedApp = null;

        function close() { overlay.remove(); }

        overlay.querySelector('#modal-close-btn').addEventListener('click', close);
        overlay.querySelector('#modal-cancel-btn').addEventListener('click', close);
        overlay.addEventListener('click', (e) => { if (e.target === overlay) close(); });

        overlay.querySelector('#modal-select-btn').addEventListener('click', () => {
            if (selectedApp && selectedApp.exePath) {
                onSelect(selectedApp.exePath);
                close();
            }
        });

        function renderApps(apps) {
            const body = overlay.querySelector('#app-list-body');
            if (apps.length === 0) {
                body.innerHTML = '<div class="loading-center">No matching applications found</div>';
                return;
            }
            body.innerHTML = apps.map((app, i) => `
                <div class="app-list-item" data-idx="${i}">
                    <span class="app-name">${esc(app.name)}</span>
                    <span class="app-source">${esc(app.source)}</span>
                    <span class="app-meta">${esc(app.publisher || '')}${app.version ? ' v' + esc(app.version) : ''}</span>
                    ${app.exePath ? '<span class="app-path">' + esc(app.exePath) + '</span>' : '<span class="app-meta" style="color:var(--warning)">No .exe found</span>'}
                </div>
            `).join('');

            body.querySelectorAll('.app-list-item').forEach(item => {
                item.addEventListener('click', () => {
                    body.querySelectorAll('.app-list-item').forEach(el => el.classList.remove('selected'));
                    item.classList.add('selected');
                    const idx = parseInt(item.dataset.idx, 10);
                    selectedApp = apps[idx];
                    overlay.querySelector('#modal-select-btn').disabled = !selectedApp.exePath;
                });
                item.addEventListener('dblclick', () => {
                    const idx = parseInt(item.dataset.idx, 10);
                    const app = apps[idx];
                    if (app.exePath) {
                        onSelect(app.exePath);
                        close();
                    }
                });
            });
        }

        // Load apps
        api('GET', '/apps').then(apps => {
            allApps = apps || [];
            overlay.querySelector('#app-count').textContent = allApps.length + ' applications found';
            renderApps(allApps);
        }).catch(err => {
            overlay.querySelector('#app-list-body').innerHTML =
                '<div class="loading-center" style="color:var(--danger)">Error: ' + esc(err.message) + '</div>';
            overlay.querySelector('#app-count').textContent = 'Failed to scan';
        });

        // Filter
        overlay.querySelector('#app-filter').addEventListener('input', (e) => {
            const q = e.target.value.toLowerCase().trim();
            if (!q) {
                renderApps(allApps);
                overlay.querySelector('#app-count').textContent = allApps.length + ' applications found';
                return;
            }
            const filtered = allApps.filter(a =>
                (a.name || '').toLowerCase().includes(q) ||
                (a.publisher || '').toLowerCase().includes(q)
            );
            renderApps(filtered);
            overlay.querySelector('#app-count').textContent = filtered.length + ' of ' + allApps.length + ' shown';
        });
    }

    // ===== Page: Scan =====
    function renderScan() {
        const el = content();
        el.innerHTML = `
            <div class="page-header">
                <h1>Scan Application</h1>
                <p>Analyze a Windows executable and generate a container profile</p>
            </div>
            <div class="card">
                <div class="form-group">
                    <label for="scan-target">Target Path</label>
                    <div style="display:flex;gap:8px">
                        <input type="text" id="scan-target" placeholder="C:\\path\\to\\app.exe or .lnk" style="flex:1">
                        <button class="btn btn-secondary" id="scan-browse-btn" type="button">Browse Installed Apps</button>
                    </div>
                </div>
                <div class="form-row">
                    <div class="form-group">
                        <label for="scan-strategy">Strategy</label>
                        <select id="scan-strategy">
                            <option value="">Auto-detect</option>
                            <option value="wine">Wine (Linux)</option>
                            <option value="windows-servercore">Windows Server Core</option>
                            <option value="windows-nanoserver">Windows Nano Server</option>
                        </select>
                    </div>
                    <div class="form-group">
                        <label for="scan-format">Output Format</label>
                        <select id="scan-format">
                            <option value="yaml">YAML</option>
                            <option value="json">JSON</option>
                        </select>
                    </div>
                    <div class="form-group">
                        <label for="scan-firewall">Firewall</label>
                        <select id="scan-firewall">
                            <option value="">Disabled</option>
                            <option value="deny">Default Deny</option>
                            <option value="allow">Default Allow</option>
                        </select>
                    </div>
                </div>
                <div class="form-row">
                    <div class="form-group">
                        <label for="scan-output">Output Path (optional)</label>
                        <input type="text" id="scan-output" placeholder="Leave blank for auto">
                    </div>
                    <div class="form-group">
                        <label for="scan-vpn">VPN Config (optional)</label>
                        <input type="text" id="scan-vpn" placeholder="Path to WireGuard .conf">
                    </div>
                </div>
                <div class="form-row">
                    <div class="checkbox-group">
                        <input type="checkbox" id="scan-encrypt">
                        <label for="scan-encrypt">Enable Encryption</label>
                    </div>
                    <div class="checkbox-group">
                        <input type="checkbox" id="scan-verbose">
                        <label for="scan-verbose">Verbose Output</label>
                    </div>
                </div>
                <div class="btn-group">
                    <button class="btn btn-primary" id="scan-btn">Start Scan</button>
                </div>
            </div>
            <div class="card" id="scan-log-card" style="display:none">
                <div class="card-header"><h2>Scan Output</h2></div>
                <div id="scan-log-container"></div>
            </div>
        `;

        // Browse installed apps button
        document.getElementById('scan-browse-btn').addEventListener('click', () => {
            openAppBrowser((exePath) => {
                document.getElementById('scan-target').value = exePath;
            });
        });

        let logPanel = null;
        document.getElementById('scan-btn').addEventListener('click', async () => {
            const target = document.getElementById('scan-target').value.trim();
            if (!target) { toast('Target path is required', 'warning'); return; }

            const btn = document.getElementById('scan-btn');
            btn.disabled = true;
            btn.innerHTML = '<span class="spinner"></span> Scanning...';

            const logCard = document.getElementById('scan-log-card');
            logCard.style.display = '';
            const logContainer = document.getElementById('scan-log-container');
            logContainer.innerHTML = '';
            logPanel = createLogPanel(logContainer);

            try {
                const res = await api('POST', '/scan', {
                    targetPath: target,
                    strategy: document.getElementById('scan-strategy').value,
                    format: document.getElementById('scan-format').value,
                    outputPath: document.getElementById('scan-output').value.trim(),
                    encrypt: document.getElementById('scan-encrypt').checked,
                    firewall: document.getElementById('scan-firewall').value,
                    vpnConfig: document.getElementById('scan-vpn').value.trim(),
                    verbose: document.getElementById('scan-verbose').checked,
                });

                connectWS(res.operationId, (evt) => logPanel.addEvent(evt), () => {
                    btn.disabled = false;
                    btn.textContent = 'Start Scan';
                    toast('Scan completed', 'success');
                });
            } catch (err) {
                toast(err.message, 'error');
                btn.disabled = false;
                btn.textContent = 'Start Scan';
            }
        });
    }

    // ===== Page: Build =====
    function renderBuild() {
        const el = content();
        el.innerHTML = `
            <div class="page-header">
                <h1>Build Image</h1>
                <p>Build a Docker image from a container profile</p>
            </div>
            <div class="card">
                <div class="form-group">
                    <label for="build-profile">Profile Path</label>
                    <input type="text" id="build-profile" placeholder="path/to/app-profile.yaml">
                </div>
                <div class="form-row">
                    <div class="form-group">
                        <label for="build-tag">Image Tag (optional)</label>
                        <input type="text" id="build-tag" placeholder="myapp:latest">
                    </div>
                    <div class="form-group">
                        <label for="build-gendir">Generate Only Dir (optional)</label>
                        <input type="text" id="build-gendir" placeholder="Leave blank to build with Docker">
                    </div>
                </div>
                <div class="checkbox-group">
                    <input type="checkbox" id="build-nocache">
                    <label for="build-nocache">No Cache</label>
                </div>
                <div class="btn-group">
                    <button class="btn btn-primary" id="build-btn">Start Build</button>
                </div>
            </div>
            <div class="card" id="build-log-card" style="display:none">
                <div class="card-header"><h2>Build Output</h2></div>
                <div id="build-log-container"></div>
            </div>
        `;

        let logPanel = null;
        document.getElementById('build-btn').addEventListener('click', async () => {
            const profilePath = document.getElementById('build-profile').value.trim();
            if (!profilePath) { toast('Profile path is required', 'warning'); return; }

            const btn = document.getElementById('build-btn');
            btn.disabled = true;
            btn.innerHTML = '<span class="spinner"></span> Building...';

            const logCard = document.getElementById('build-log-card');
            logCard.style.display = '';
            const logContainer = document.getElementById('build-log-container');
            logContainer.innerHTML = '';
            logPanel = createLogPanel(logContainer);

            try {
                const res = await api('POST', '/build', {
                    profilePath,
                    tag: document.getElementById('build-tag').value.trim(),
                    noCache: document.getElementById('build-nocache').checked,
                    generateDir: document.getElementById('build-gendir').value.trim(),
                });

                connectWS(res.operationId, (evt) => logPanel.addEvent(evt), () => {
                    btn.disabled = false;
                    btn.textContent = 'Start Build';
                    toast('Build completed', 'success');
                });
            } catch (err) {
                toast(err.message, 'error');
                btn.disabled = false;
                btn.textContent = 'Start Build';
            }
        });
    }

    // ===== Page: Run =====
    function renderRun() {
        const el = content();
        el.innerHTML = `
            <div class="page-header">
                <h1>Run Container</h1>
                <p>Start a containerized application</p>
            </div>
            <div class="card">
                <div class="form-group">
                    <label for="run-image">Image</label>
                    <input type="text" id="run-image" placeholder="myapp:latest">
                </div>
                <div class="form-row">
                    <div class="form-group">
                        <label for="run-display">Display Mode</label>
                        <select id="run-display">
                            <option value="none">None</option>
                            <option value="vnc">VNC (port 5901)</option>
                            <option value="novnc">noVNC (port 6080)</option>
                            <option value="rdp">RDP (port 3389)</option>
                        </select>
                    </div>
                    <div class="form-group">
                        <label for="run-name">Container Name (optional)</label>
                        <input type="text" id="run-name" placeholder="my-container">
                    </div>
                </div>
                <div class="form-row">
                    <div class="form-group">
                        <label for="run-profile">Security Profile (optional)</label>
                        <input type="text" id="run-profile" placeholder="path/to/profile.yaml">
                    </div>
                    <div class="form-group">
                        <label for="run-agekey">Age Key File (optional)</label>
                        <input type="text" id="run-agekey" placeholder="path/to/age-key.txt">
                    </div>
                </div>
                <div class="form-row">
                    <div class="checkbox-group">
                        <input type="checkbox" id="run-detach" checked>
                        <label for="run-detach">Detached Mode</label>
                    </div>
                    <div class="checkbox-group">
                        <input type="checkbox" id="run-remove">
                        <label for="run-remove">Auto-remove on Exit</label>
                    </div>
                </div>
                <div class="btn-group">
                    <button class="btn btn-primary" id="run-btn">Run</button>
                </div>
            </div>
        `;

        document.getElementById('run-btn').addEventListener('click', async () => {
            const image = document.getElementById('run-image').value.trim();
            if (!image) { toast('Image is required', 'warning'); return; }

            const btn = document.getElementById('run-btn');
            btn.disabled = true;
            btn.innerHTML = '<span class="spinner"></span> Starting...';

            try {
                await api('POST', '/run', {
                    image,
                    display: document.getElementById('run-display').value,
                    name: document.getElementById('run-name').value.trim(),
                    detach: document.getElementById('run-detach').checked,
                    remove: document.getElementById('run-remove').checked,
                    profile: document.getElementById('run-profile').value.trim(),
                    ageKey: document.getElementById('run-agekey').value.trim(),
                });
                toast('Container started successfully', 'success');
            } catch (err) {
                toast(err.message, 'error');
            } finally {
                btn.disabled = false;
                btn.textContent = 'Run';
            }
        });
    }

    // ===== Page: Inspect =====
    function renderInspect() {
        const el = content();
        el.innerHTML = `
            <div class="page-header">
                <h1>Inspect Binary</h1>
                <p>Analyze a PE executable's architecture and imports</p>
            </div>
            <div class="card">
                <div class="form-group">
                    <label for="inspect-target">Target Path</label>
                    <input type="text" id="inspect-target" placeholder="C:\\path\\to\\app.exe">
                </div>
                <div class="btn-group">
                    <button class="btn btn-primary" id="inspect-btn">Inspect</button>
                </div>
            </div>
            <div id="inspect-results"></div>
        `;

        document.getElementById('inspect-btn').addEventListener('click', async () => {
            const target = document.getElementById('inspect-target').value.trim();
            if (!target) { toast('Target path is required', 'warning'); return; }

            const btn = document.getElementById('inspect-btn');
            btn.disabled = true;
            btn.innerHTML = '<span class="spinner"></span> Inspecting...';
            const results = document.getElementById('inspect-results');
            results.innerHTML = '';

            try {
                const r = await api('GET', '/inspect?target=' + encodeURIComponent(target));
                const systemImports = r.Imports ? r.Imports.filter(i => i.IsSystem) : [];
                const appImports = r.Imports ? r.Imports.filter(i => !i.IsSystem) : [];

                results.innerHTML = `
                    <div class="card">
                        <div class="card-header"><h2>Binary Information</h2></div>
                        <dl class="detail-list">
                            <span class="label">File Name</span><span class="value">${esc(r.FileName)}</span>
                            <span class="label">Full Path</span><span class="value">${esc(r.FullPath)}</span>
                            <span class="label">Architecture</span><span class="value">${esc(r.Arch)}</span>
                            <span class="label">Subsystem</span><span class="value">${esc(r.Subsystem)}</span>
                            <span class="label">Total Imports</span><span class="value">${r.Imports ? r.Imports.length : 0}</span>
                            <span class="label">App DLLs</span><span class="value">${appImports.length}</span>
                            <span class="label">System DLLs</span><span class="value">${systemImports.length}</span>
                        </dl>
                    </div>
                    ${appImports.length > 0 ? `
                    <div class="card">
                        <div class="card-header"><h2>Application DLLs (${appImports.length})</h2></div>
                        <div class="table-wrapper"><table>
                            <thead><tr><th>DLL Name</th><th>Type</th></tr></thead>
                            <tbody>${appImports.map(i => `<tr><td>${esc(i.Name)}</td><td><span class="badge badge-created">App</span></td></tr>`).join('')}</tbody>
                        </table></div>
                    </div>` : ''}
                    ${systemImports.length > 0 ? `
                    <div class="card">
                        <div class="card-header"><h2>System DLLs (${systemImports.length})</h2></div>
                        <div class="table-wrapper"><table>
                            <thead><tr><th>DLL Name</th><th>Type</th></tr></thead>
                            <tbody>${systemImports.map(i => `<tr><td>${esc(i.Name)}</td><td><span class="badge badge-stopped">System</span></td></tr>`).join('')}</tbody>
                        </table></div>
                    </div>` : ''}
                `;
            } catch (err) {
                toast(err.message, 'error');
            } finally {
                btn.disabled = false;
                btn.textContent = 'Inspect';
            }
        });
    }

    // ===== Page: Keygen =====
    function renderKeygen() {
        const el = content();
        el.innerHTML = `
            <div class="page-header">
                <h1>Generate Keys</h1>
                <p>Create an Age keypair for container encryption</p>
            </div>
            <div class="card">
                <div class="form-group">
                    <label for="keygen-dir">Output Directory</label>
                    <input type="text" id="keygen-dir" placeholder="." value=".">
                </div>
                <div class="btn-group">
                    <button class="btn btn-primary" id="keygen-btn">Generate Keys</button>
                </div>
            </div>
            <div id="keygen-results"></div>
        `;

        document.getElementById('keygen-btn').addEventListener('click', async () => {
            const btn = document.getElementById('keygen-btn');
            btn.disabled = true;
            btn.innerHTML = '<span class="spinner"></span> Generating...';
            const results = document.getElementById('keygen-results');

            try {
                const r = await api('POST', '/keygen', {
                    outputDir: document.getElementById('keygen-dir').value.trim() || '.',
                });
                results.innerHTML = `
                    <div class="card">
                        <div class="card-header"><h2>Keys Generated</h2></div>
                        <dl class="detail-list">
                            <span class="label">Recipient</span><span class="value">${esc(r.Recipient)}</span>
                            <span class="label">Key File</span><span class="value">${esc(r.KeyFile)}</span>
                        </dl>
                    </div>
                `;
                toast('Keys generated successfully', 'success');
            } catch (err) {
                toast(err.message, 'error');
            } finally {
                btn.disabled = false;
                btn.textContent = 'Generate Keys';
            }
        });
    }

    // ===== Page: Profiles =====
    async function renderProfiles() {
        const el = content();
        el.innerHTML = `
            <div class="page-header">
                <h1>Profiles</h1>
                <p>Manage container profiles</p>
            </div>
            <div class="card">
                <div class="card-header">
                    <h2>Available Profiles</h2>
                    <button class="btn btn-sm btn-secondary" id="profiles-refresh">Refresh</button>
                </div>
                <div id="profiles-list"><div class="loading-center"><span class="spinner"></span> Loading profiles...</div></div>
            </div>
        `;

        async function loadProfiles() {
            const container = document.getElementById('profiles-list');
            try {
                const profiles = await api('GET', '/profiles');
                if (profiles.length === 0) {
                    container.innerHTML = `
                        <div class="empty-state">
                            <h3>No profiles found</h3>
                            <p>Scan an application to create a profile</p>
                            <button class="btn btn-primary btn-sm" onclick="location.hash='#scan'" style="margin-top:12px">Scan Application</button>
                        </div>
                    `;
                    return;
                }
                container.innerHTML = `
                    <div class="table-wrapper"><table>
                        <thead><tr>
                            <th>Name</th>
                            <th>Application</th>
                            <th>Strategy</th>
                            <th>Arch</th>
                            <th>Format</th>
                            <th>Actions</th>
                        </tr></thead>
                        <tbody>${profiles.map(p => `
                            <tr>
                                <td>${esc(p.name)}</td>
                                <td>${esc(p.appName)}</td>
                                <td>${esc(p.strategy)}</td>
                                <td>${esc(p.arch)}</td>
                                <td>${esc(p.format)}</td>
                                <td>
                                    <button class="btn btn-sm btn-secondary" onclick="location.hash='#build'">Build</button>
                                    <button class="btn btn-sm btn-danger" data-delete="${esc(p.name)}">Delete</button>
                                </td>
                            </tr>
                        `).join('')}</tbody>
                    </table></div>
                `;

                // Delete buttons
                container.querySelectorAll('[data-delete]').forEach(btn => {
                    btn.addEventListener('click', async () => {
                        const name = btn.dataset.delete;
                        if (!confirm(`Delete profile "${name}"?`)) return;
                        try {
                            await api('DELETE', '/profiles/' + encodeURIComponent(name));
                            toast('Profile deleted', 'success');
                            loadProfiles();
                        } catch (err) {
                            toast(err.message, 'error');
                        }
                    });
                });
            } catch (err) {
                container.innerHTML = `<div class="empty-state"><h3>Error loading profiles</h3><p>${esc(err.message)}</p></div>`;
            }
        }

        loadProfiles();
        document.getElementById('profiles-refresh').addEventListener('click', loadProfiles);
    }

    // ===== Page: Containers =====
    async function renderContainers() {
        const el = content();
        el.innerHTML = `
            <div class="page-header">
                <h1>Containers</h1>
                <p>Manage Docker containers</p>
            </div>
            <div class="card">
                <div class="card-header">
                    <h2>All Containers</h2>
                    <button class="btn btn-sm btn-secondary" id="containers-refresh">Refresh</button>
                </div>
                <div id="containers-list"><div class="loading-center"><span class="spinner"></span> Loading containers...</div></div>
            </div>
        `;

        async function loadContainers() {
            const container = document.getElementById('containers-list');
            try {
                const containers = await api('GET', '/containers');
                if (containers.length === 0) {
                    container.innerHTML = `
                        <div class="empty-state">
                            <h3>No containers found</h3>
                            <p>Build and run an application to see containers here</p>
                        </div>
                    `;
                    return;
                }
                container.innerHTML = `
                    <div class="table-wrapper"><table>
                        <thead><tr>
                            <th>ID</th>
                            <th>Name</th>
                            <th>Image</th>
                            <th>Status</th>
                            <th>State</th>
                            <th>Ports</th>
                            <th>Actions</th>
                        </tr></thead>
                        <tbody>${containers.map(c => `
                            <tr>
                                <td title="${esc(c.id)}">${esc(c.id.substring(0, 12))}</td>
                                <td>${esc(c.name)}</td>
                                <td>${esc(c.image)}</td>
                                <td>${esc(c.status)}</td>
                                <td><span class="badge badge-${c.state}">${esc(c.state)}</span></td>
                                <td>${esc(c.ports || '-')}</td>
                                <td>
                                    ${c.state === 'running' ? `<button class="btn btn-sm btn-danger" data-stop="${esc(c.id)}">Stop</button>` : ''}
                                </td>
                            </tr>
                        `).join('')}</tbody>
                    </table></div>
                `;

                // Stop buttons
                container.querySelectorAll('[data-stop]').forEach(btn => {
                    btn.addEventListener('click', async () => {
                        const id = btn.dataset.stop;
                        btn.disabled = true;
                        btn.innerHTML = '<span class="spinner"></span>';
                        try {
                            await api('POST', '/containers/' + encodeURIComponent(id) + '/stop');
                            toast('Container stopped', 'success');
                            loadContainers();
                        } catch (err) {
                            toast(err.message, 'error');
                            btn.disabled = false;
                            btn.textContent = 'Stop';
                        }
                    });
                });
            } catch (err) {
                container.innerHTML = `<div class="empty-state"><h3>Error loading containers</h3><p>${esc(err.message)}</p></div>`;
            }
        }

        loadContainers();
        document.getElementById('containers-refresh').addEventListener('click', loadContainers);
    }

    // ===== Init =====
    checkStatus();
    const startPage = location.hash.replace('#', '') || 'dashboard';
    navigate(startPage);

    // Refresh docker status every 30s
    setInterval(checkStatus, 30000);
})();
