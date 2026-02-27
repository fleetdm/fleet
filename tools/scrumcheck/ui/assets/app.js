    (function () {
      const bridgeSession = document.body.dataset.bridgeSession || '';
      function bridgeJSONHeaders() {
        return {
          'Content-Type': 'application/json',
          'X-Qacheck-Session': bridgeSession,
        };
      }
      const buttons = document.querySelectorAll('.menu-btn');
      const panels = document.querySelectorAll('.panel');
      function activate(tabName) {
        buttons.forEach((btn) => {
          btn.classList.toggle('active', btn.dataset.tab === tabName);
        });
        panels.forEach((panel) => {
          panel.classList.toggle('active', panel.id === 'tab-' + tabName);
        });
      }
      buttons.forEach((btn) => {
        btn.addEventListener('click', () => activate(btn.dataset.tab));
      });

      // connectBridgeEvents wires an SSE stream so the UI can react to live
      // bridge operations (refreshes, apply actions, and shutdown events).
      function connectBridgeEvents() {
        const bridgeURL = document.body.dataset.bridgeUrl || window.location.origin || '';
        if (!bridgeURL) {
          return;
        }
        try {
          const streamURL = bridgeSession
            ? bridgeURL + '/api/events?session=' + encodeURIComponent(bridgeSession)
            : bridgeURL + '/api/events';
          const stream = new EventSource(streamURL);
          stream.addEventListener('log', async () => {
            try {
              await fetchStateAndRender(false);
            } catch (_) {
              // Keep stream alive even if a specific refresh attempt fails.
            }
          });
        } catch (_) {
          // SSE support is best-effort; manual refresh still works.
        }
      }

      function setButtonDone(btn, text) {
        btn.classList.remove('failed');
        btn.classList.add('done');
        btn.textContent = text;
        btn.disabled = true;
      }

      function setButtonFailed(btn) {
        btn.classList.remove('done');
        btn.classList.add('failed');
        btn.textContent = 'Failed (retry)';
        btn.disabled = false;
      }

      function setButtonWorking(btn, text) {
        btn.classList.remove('done', 'failed');
        btn.textContent = text;
        btn.disabled = true;
      }

      function setTabClean(tabName, isClean) {
        const btn = document.querySelector('.menu-btn[data-tab="' + tabName + '"]');
        if (!btn) return;
        const dot = btn.querySelector('.status-dot');
        if (!dot) return;
        dot.classList.toggle('ok', Boolean(isClean));
      }

      async function refreshTimestampPanel(forceRefresh) {
        const subtle = document.getElementById('timestamp-subtle');
        const content = document.getElementById('timestamp-content');
        if (!subtle || !content) return;

        const bridgeURL = document.body.dataset.bridgeUrl || window.location.origin || '';
        if (!bridgeURL || !bridgeSession) {
          subtle.innerHTML = 'Bridge unavailable.';
          content.innerHTML = '<p class="empty">🔴 Could not load timestamp check (missing bridge session).</p>';
          setTabClean('timestamp', false);
          return;
        }

        try {
          const query = forceRefresh ? '?refresh=1' : '';
          const res = await fetch(bridgeURL + '/api/check/timestamp' + query, {
            method: 'GET',
            headers: { 'X-Qacheck-Session': bridgeSession },
          });
          if (!res.ok) {
            throw new Error('Bridge error ' + res.status);
          }
          const payload = await res.json();
          const rawURL = String(payload.url || '');
          const safeURL = safeHTTPURL(rawURL);
          const minDays = Number(payload.min_days || 0);
          const displayURL = safeURL || rawURL || '(invalid url)';
          subtle.innerHTML = 'Checks that <a href="' + escHTML(safeURL || '#') + '" target="_blank" rel="noopener noreferrer">' + escHTML(displayURL) + '</a> expires at least ' + minDays + ' days from now.';

          if (payload.error) {
            content.innerHTML = '<p class="empty">🔴 Could not validate timestamp expiry: ' + escHTML(payload.error) + '</p>';
            setTabClean('timestamp', false);
            return;
          }

          const stateText = payload.ok ? '🟢 OK' : '🔴 Failing threshold';
          const expires = escHTML(payload.expires_at || '(unknown)');
          const daysLeft = Number(payload.days_left || 0).toFixed(1);
          const hoursLeft = Number(payload.duration_hours || 0).toFixed(1);

          content.innerHTML =
            '<p><strong>' + stateText + '</strong></p>' +
            '<ul>' +
            '<li>Expires: ' + expires + '</li>' +
            '<li>Days remaining: ' + daysLeft + '</li>' +
            '<li>Hours remaining: ' + hoursLeft + '</li>' +
            '<li>Minimum required days: ' + minDays + '</li>' +
            '</ul>';
          setTabClean('timestamp', Boolean(payload.ok));
        } catch (err) {
          subtle.textContent = 'Checks timestamp expiry from the bridge.';
          content.innerHTML = '<p class="empty">🔴 Could not load timestamp check: ' + escHTML(err) + '</p>';
          setTabClean('timestamp', false);
        }
      }

      function escHTML(value) {
        const text = String(value == null ? '' : value);
        return text
          .replaceAll('&', '&amp;')
          .replaceAll('<', '&lt;')
          .replaceAll('>', '&gt;')
          .replaceAll('"', '&quot;')
          .replaceAll("'", '&#39;');
      }

      function safeHTTPURL(value) {
        const raw = String(value == null ? '' : value).trim();
        if (!raw) return '';
        try {
          const parsed = new URL(raw, window.location.origin);
          if (parsed.protocol === 'http:' || parsed.protocol === 'https:') {
            return parsed.href;
          }
        } catch (_) {}
        return '';
      }

      function renderSafeExternalLink(rawURL) {
        const display = String(rawURL || '');
        const safeHref = safeHTTPURL(display) || '#';
        return '<a href="' + escHTML(safeHref) + '" target="_blank" rel="noopener noreferrer">' + escHTML(display || '(invalid url)') + '</a>';
      }

      function renderUnreleasedItem(item, cssClass) {
        const status = item.Status ? item.Status : '(unset)';
        const projectText = Number(item.ProjectNum || 0) > 0 ? String(item.ProjectNum) : '(not on selected project)';
        const assignees = Array.isArray(item.Assignees) && item.Assignees.length > 0 ? item.Assignees.join(', ') : '(none)';
        const labels = Array.isArray(item.Labels) && item.Labels.length > 0 ? item.Labels.join(', ') : '(none)';

        return '' +
          '<article class="item ' + cssClass + '">' +
            '<div><strong>#' + escHTML(item.Number) + ' - ' + escHTML(item.Title) + '</strong></div>' +
            '<div>' + renderSafeExternalLink(item.URL) + '</div>' +
            '<ul>' +
              '<li>Status: ' + escHTML(status) + '</li>' +
              '<li>Project: ' + escHTML(projectText) + '</li>' +
              '<li>Repository: ' + escHTML(item.Repo) + '</li>' +
              '<li>Assignees: ' + escHTML(assignees) + '</li>' +
              '<li>Labels: ' + escHTML(labels) + '</li>' +
            '</ul>' +
          '</article>';
      }

      function renderReleaseStoryTODOItem(item) {
        const status = item.Status ? item.Status : '(unset)';
        const assignees = Array.isArray(item.Assignees) && item.Assignees.length > 0 ? item.Assignees.join(', ') : '(empty)';
        const labels = Array.isArray(item.Labels) && item.Labels.length > 0 ? item.Labels.join(', ') : '(none)';
        const preview = Array.isArray(item.BodyPreview) && item.BodyPreview.length > 0 ? item.BodyPreview : ['(empty)'];
        const previewLines = preview.map((line) => '<li>' + escHTML(line) + '</li>').join('');

        return '' +
          '<article class="item red-bug">' +
            '<div><strong>#' + escHTML(item.Number) + ' - ' + escHTML(item.Title) + '</strong></div>' +
            '<div>' + renderSafeExternalLink(item.URL) + '</div>' +
            '<ul>' +
              '<li>Status: ' + escHTML(status) + '</li>' +
              '<li>Repository: ' + escHTML(item.Repo) + '</li>' +
              '<li>Assignees: ' + escHTML(assignees) + '</li>' +
              '<li>Labels: ' + escHTML(labels) + '</li>' +
              '<li>Snippet:</li>' +
              previewLines +
            '</ul>' +
          '</article>';
      }

      async function refreshReleaseStoryTODOPanel(forceRefresh) {
        const root = document.getElementById('release-story-todo-content');
        if (!root) return;

        const bridgeURL = document.body.dataset.bridgeUrl || window.location.origin || '';
        if (!bridgeURL || !bridgeSession) {
          root.innerHTML = '<p class="empty">🔴 Could not load release stories TODO (missing bridge session).</p>';
          setTabClean('release-story-todo', false);
          return;
        }

        try {
          const query = forceRefresh ? '?refresh=1' : '';
          const res = await fetch(bridgeURL + '/api/check/release-story-todo' + query, {
            method: 'GET',
            headers: { 'X-Qacheck-Session': bridgeSession },
          });
          if (!res.ok) {
            throw new Error('Bridge error ' + res.status);
          }
          const payload = await res.json();
          const projects = Array.isArray(payload.projects) ? payload.projects : [];
          let totalItems = 0;
          projects.forEach((proj) => {
            const columns = Array.isArray(proj.Columns) ? proj.Columns : [];
            columns.forEach((col) => {
              const items = Array.isArray(col.Items) ? col.Items : [];
              totalItems += items.length;
            });
          });
          setTabClean('release-story-todo', totalItems === 0);
          if (projects.length === 0) {
            root.innerHTML = '<p class="empty">🟢 No release stories with TODO found.</p>';
            return;
          }

          root.innerHTML = projects.map((proj) => {
            const columns = Array.isArray(proj.Columns) ? proj.Columns : [];
            const renderedCols = columns.map((col) => {
              const items = Array.isArray(col.Items) ? col.Items : [];
              let html = '<div class="status"><h3>' + escHTML(col.Label) + '</h3>';
              if (items.length === 0) {
                html += '<p class="empty">🟢 No items in this group.</p>';
              } else {
                items.forEach((it) => {
                  html += renderReleaseStoryTODOItem(it);
                });
              }
              html += '</div>';
              return html;
            }).join('');

            return '' +
              '<div class="project">' +
                '<h3>Project ' + escHTML(proj.ProjectNum) + '</h3>' +
                renderedCols +
              '</div>';
          }).join('');
        } catch (err) {
          root.innerHTML = '<p class="empty">🔴 Could not load release stories TODO: ' + escHTML(err) + '</p>';
          setTabClean('release-story-todo', false);
        }
      }

      function renderMissingSprintItem(item, showActions) {
        const status = item.Status ? item.Status : '(unset)';
        const currentSprint = item.CurrentSprint ? item.CurrentSprint : '(unknown)';
        const milestone = item.Milestone ? item.Milestone : '(empty)';
        const assignees = Array.isArray(item.Assignees) && item.Assignees.length > 0 ? item.Assignees.join(', ') : '(empty)';
        const labels = Array.isArray(item.Labels) && item.Labels.length > 0 ? item.Labels.join(', ') : '(empty)';
        const preview = Array.isArray(item.BodyPreview) && item.BodyPreview.length > 0 ? item.BodyPreview : ['(empty)'];
        const previewLines = preview.map((line) => '<li>' + escHTML(line) + '</li>').join('');
        const actionHTML = showActions ? (
          '<div class="actions">' +
            '<button class="fix-btn apply-sprint-btn" data-item-id="' + escHTML(item.ItemID) + '">Set current sprint</button>' +
          '</div>'
        ) : '';

        return '' +
          '<article class="item">' +
            '<div><strong>#' + escHTML(item.Number) + ' - ' + escHTML(item.Title) + '</strong></div>' +
            '<div>' + renderSafeExternalLink(item.URL) + '</div>' +
            '<ul>' +
              '<li>Status: ' + escHTML(status) + '</li>' +
              '<li>Current sprint: ' + escHTML(currentSprint) + '</li>' +
              '<li>Milestone: ' + escHTML(milestone) + '</li>' +
              '<li>Assignees: ' + escHTML(assignees) + '</li>' +
              '<li>Labels: ' + escHTML(labels) + '</li>' +
              '<li>Snippet:</li>' +
              previewLines +
            '</ul>' +
            actionHTML +
          '</article>';
      }

      async function refreshMissingSprintPanel(forceRefresh) {
        const root = document.getElementById('missing-sprint-content');
        if (!root) return;

        const bridgeURL = document.body.dataset.bridgeUrl || window.location.origin || '';
        if (!bridgeURL || !bridgeSession) {
          root.innerHTML = '<p class="empty">🔴 Could not load missing sprint data (missing bridge session).</p>';
          setTabClean('sprint', false);
          return;
        }

        try {
          const query = forceRefresh ? '?refresh=1' : '';
          const res = await fetch(bridgeURL + '/api/check/missing-sprint' + query, {
            method: 'GET',
            headers: { 'X-Qacheck-Session': bridgeSession },
          });
          if (!res.ok) {
            throw new Error('Bridge error ' + res.status);
          }
          const payload = await res.json();
          const projects = Array.isArray(payload.projects) ? payload.projects : [];
          let totalItems = 0;
          projects.forEach((proj) => {
            const columns = Array.isArray(proj.Columns) ? proj.Columns : [];
            columns.forEach((col) => {
              const items = Array.isArray(col.Items) ? col.Items : [];
              totalItems += items.length;
            });
          });
          setTabClean('sprint', totalItems === 0);
          if (projects.length === 0) {
            root.innerHTML = '<p class="empty">🟢 No missing sprint items found.</p>';
            return;
          }

          const showActions = Boolean(bridgeSession);
          root.innerHTML = projects.map((proj) => {
            const columns = Array.isArray(proj.Columns) ? proj.Columns : [];
            const renderedCols = columns.map((col) => {
              const items = Array.isArray(col.Items) ? col.Items : [];
              const colAction = showActions && items.length > 0
                ? '<button class="fix-btn apply-sprint-column-btn">Set current sprint for column</button>'
                : '';
              let html = '<div class="status"><div class="column-head"><h3>' + escHTML(col.Label) + '</h3>' + colAction + '</div>';
              if (items.length === 0) {
                html += '<p class="empty">🟢 No items in this group.</p>';
              } else {
                items.forEach((it) => {
                  html += renderMissingSprintItem(it, showActions);
                });
              }
              html += '</div>';
              return html;
            }).join('');

            return '' +
              '<div class="project">' +
                '<h3>Project ' + escHTML(proj.ProjectNum) + '</h3>' +
                renderedCols +
              '</div>';
          }).join('');
        } catch (err) {
          root.innerHTML = '<p class="empty">🔴 Could not load missing sprint data: ' + escHTML(err) + '</p>';
          setTabClean('sprint', false);
        }
      }

      async function refreshUnreleasedPanel(forceRefresh) {
        const root = document.getElementById('unreleased-content');
        if (!root) return;

        const bridgeURL = document.body.dataset.bridgeUrl || window.location.origin || '';
        if (!bridgeURL || !bridgeSession) {
          root.innerHTML = '<p class="empty">🔴 Could not load unreleased bugs (missing bridge session).</p>';
          setTabClean('unassigned-unreleased', false);
          return;
        }

        try {
          const query = forceRefresh ? '?refresh=1' : '';
          const res = await fetch(bridgeURL + '/api/check/unassigned-unreleased' + query, {
            method: 'GET',
            headers: { 'X-Qacheck-Session': bridgeSession },
          });
          if (!res.ok) {
            throw new Error('Bridge error ' + res.status);
          }
          const payload = await res.json();
          const groups = Array.isArray(payload.groups) ? payload.groups : [];
          let totalRed = 0;
          groups.forEach((group) => {
            const columns = Array.isArray(group.Columns) ? group.Columns : [];
            columns.forEach((col) => {
              const redItems = Array.isArray(col.RedItems) ? col.RedItems : [];
              totalRed += redItems.length;
            });
          });
          setTabClean('unassigned-unreleased', totalRed === 0);
          if (groups.length === 0) {
            root.innerHTML = '<p class="empty">🟢 No unassigned unreleased bugs found.</p>';
            return;
          }

          root.innerHTML = groups.map((group) => {
            const columns = Array.isArray(group.Columns) ? group.Columns : [];
            const renderedCols = columns.map((col) => {
              const redItems = Array.isArray(col.RedItems) ? col.RedItems : [];
              const greenItems = Array.isArray(col.GreenItems) ? col.GreenItems : [];
              let html = '<div class="status"><h3>' + escHTML(col.Label) + '</h3>';
              redItems.forEach((it) => {
                html += renderUnreleasedItem(it, 'red-bug');
              });
              greenItems.forEach((it) => {
                html += renderUnreleasedItem(it, 'green-bug');
              });
              if (redItems.length === 0 && greenItems.length === 0) {
                html += '<p class="empty">🟢 No items in this group.</p>';
              }
              html += '</div>';
              return html;
            }).join('');

            return '' +
              '<div class="project">' +
                '<h3>Group: ' + escHTML(group.GroupLabel) + '</h3>' +
                renderedCols +
              '</div>';
          }).join('');
        } catch (err) {
          root.innerHTML = '<p class="empty">🔴 Could not load unreleased bugs: ' + escHTML(err) + '</p>';
          setTabClean('unassigned-unreleased', false);
        }
      }

      function listOrEmpty(values, emptyText) {
        return Array.isArray(values) && values.length > 0 ? values.join(', ') : emptyText;
      }

      function renderCounts(state) {
        const root = document.getElementById('counts-content');
        if (!root || !state) return;
        root.innerHTML = '' +
          '<span class="pill">Release stories with TODO (selected projects): ' + escHTML(state.TotalReleaseStoryTODO) + '</span>' +
          '<span class="pill">Generic query issues: ' + escHTML(state.TotalGenericQueries) + '</span>' +
          '<span class="pill">Awaiting QA violations: ' + escHTML(state.TotalAwaiting) + '</span>' +
          '<span class="pill">Stale Awaiting QA items: ' + escHTML(state.TotalStale) + '</span>' +
          '<span class="pill">Missing milestones (selected projects): ' + escHTML(state.TotalNoMilestone) + '</span>' +
          '<span class="pill">Missing sprint (selected projects): ' + escHTML(state.TotalNoSprint) + '</span>' +
          '<span class="pill">Missing assignee (selected projects): ' + escHTML(state.TotalMissingAssignee) + '</span>' +
          '<span class="pill">Assigned to me (selected projects): ' + escHTML(state.TotalAssignedToMe) + '</span>' +
          '<span class="pill">Unassigned unreleased bugs (selected projects): ' + escHTML(state.TotalUnassignedUnreleased) + '</span>' +
          '<span class="pill">Tracked unreleased bugs (assigned): ' + escHTML(state.TotalTrackedUnreleased) + '</span>' +
          '<span class="pill">Release label issues (selected projects): ' + escHTML(state.TotalRelease) + '</span>' +
          '<span class="pill">Drafting checklist violations: ' + escHTML(state.TotalDrafting) + '</span>';
      }

      function renderAwaitingFromState(state) {
        const root = document.getElementById('awaiting-content');
        if (!root) return;
        const sections = Array.isArray(state.AwaitingSections) ? state.AwaitingSections : [];
        if (sections.length === 0) {
          root.innerHTML = '<p class="empty">🟢 No project data found.</p>';
          setTabClean('awaiting', true);
          return;
        }
        let total = 0;
        root.innerHTML = sections.map((sec) => {
          const items = Array.isArray(sec.Items) ? sec.Items : [];
          total += items.length;
          if (items.length === 0) {
            return '<div class="project"><h3>Project ' + escHTML(sec.ProjectNum) + '</h3><p class="empty">🟢 No violations in this project.</p></div>';
          }
          return '<div class="project"><h3>Project ' + escHTML(sec.ProjectNum) + '</h3>' + items.map((it) => {
            const unchecked = Array.isArray(it.Unchecked) ? it.Unchecked : [];
            const uncheckedHTML = unchecked.length > 0 ? '<ul>' + unchecked.map((u) => '<li>[ ] ' + escHTML(u) + '</li>').join('') + '</ul>' : '';
            return '' +
              '<article class="item">' +
                '<div><strong>#' + escHTML(it.Number) + ' - ' + escHTML(it.Title) + '</strong></div>' +
                '<div>' + renderSafeExternalLink(it.URL) + '</div>' +
                '<ul><li>Assignees: ' + escHTML(listOrEmpty(it.Assignees, '(empty)')) + '</li></ul>' +
                uncheckedHTML +
              '</article>';
          }).join('') + '</div>';
        }).join('');
        setTabClean('awaiting', total === 0);
      }

      function renderStaleFromState(state) {
        const root = document.getElementById('stale-content');
        if (!root) return;
        const sections = Array.isArray(state.StaleSections) ? state.StaleSections : [];
        if (sections.length === 0) {
          root.innerHTML = '<p class="empty">🟢 No project data found.</p>';
          setTabClean('stale', true);
          return;
        }
        let total = 0;
        root.innerHTML = sections.map((sec) => {
          const items = Array.isArray(sec.Items) ? sec.Items : [];
          total += items.length;
          if (items.length === 0) {
            return '<div class="project"><h3>Project ' + escHTML(sec.ProjectNum) + '</h3><p class="empty">🟢 No stale items in this project.</p></div>';
          }
          return '<div class="project"><h3>Project ' + escHTML(sec.ProjectNum) + '</h3>' + items.map((it) => (
            '<article class="item">' +
              '<div><strong>#' + escHTML(it.Number) + ' - ' + escHTML(it.Title) + '</strong></div>' +
              '<div>' + renderSafeExternalLink(it.URL) + '</div>' +
              '<ul><li>Last updated: ' + escHTML(it.LastUpdated) + '</li><li>Age: ' + escHTML(it.StaleDays) + ' days</li></ul>' +
            '</article>'
          )).join('') + '</div>';
        }).join('');
        setTabClean('stale', total === 0);
      }

      function renderMilestoneFromState(state) {
        const root = document.getElementById('milestone-content');
        if (!root) return;
        const projects = Array.isArray(state.MissingMilestone) ? state.MissingMilestone : [];
        if (projects.length === 0) {
          root.innerHTML = '<p class="empty">🟢 No missing milestones found.</p>';
          setTabClean('milestone', true);
          return;
        }
        let total = 0;
        root.innerHTML = projects.map((proj) => {
          const cols = Array.isArray(proj.Columns) ? proj.Columns : [];
          const colsHTML = cols.map((col) => {
            const items = Array.isArray(col.Items) ? col.Items : [];
            total += items.length;
            const colButton = items.length > 0 ? '<button class="fix-btn apply-milestone-column-btn">Apply selected milestones in column</button>' : '';
            const itemsHTML = items.length === 0 ? '<p class="empty">🟢 No items in this group.</p>' : items.map((it) => {
              const suggestions = Array.isArray(it.Suggestions) ? it.Suggestions : [];
              const options = suggestions.map((s) => '<option value="' + escHTML(s.Title) + '" data-number="' + escHTML(s.Number) + '">' + escHTML(s.Title) + '</option>').join('');
              const actionHTML = suggestions.length > 0
                ? '<div class="actions"><select class="fix-btn milestone-select" data-issue="' + escHTML(it.Number) + '" data-repo="' + escHTML(it.Repo) + '">' + options + '</select><button class="fix-btn apply-milestone-btn">Apply milestone</button></div>'
                : '<div class="actions"><span class="copied-note">No milestone suggestions found for this repo.</span></div>';
              const preview = Array.isArray(it.BodyPreview) && it.BodyPreview.length > 0 ? it.BodyPreview : ['(empty)'];
              const previewHTML = preview.map((p) => '<li>' + escHTML(p) + '</li>').join('');
              return '' +
                '<article class="item">' +
                  '<div><strong>#' + escHTML(it.Number) + ' - ' + escHTML(it.Title) + '</strong></div>' +
                  '<div>' + renderSafeExternalLink(it.URL) + '</div>' +
                  '<ul><li>Status: ' + escHTML(it.Status || '(unset)') + '</li><li>Repository: ' + escHTML(it.Repo) + '</li><li>Assignees: ' + escHTML(listOrEmpty(it.Assignees, '(empty)')) + '</li><li>Labels: ' + escHTML(listOrEmpty(it.Labels, '(empty)')) + '</li><li>Snippet:</li>' + previewHTML + '</ul>' +
                  actionHTML +
                '</article>';
            }).join('');
            return '<div class="status"><div class="column-head"><h3>' + escHTML(col.Label) + '</h3>' + colButton + '</div>' + itemsHTML + '</div>';
          }).join('');
          return '<div class="project"><h3>Project ' + escHTML(proj.ProjectNum) + '</h3>' + colsHTML + '</div>';
        }).join('');
        setTabClean('milestone', total === 0);
      }

      function renderDraftingFromState(state) {
        const root = document.getElementById('drafting-content');
        if (!root) return;
        const sections = Array.isArray(state.DraftingSections) ? state.DraftingSections : [];
        if (sections.length === 0) {
          root.innerHTML = '<p class="empty">🟢 No drafting violations.</p>';
          setTabClean('drafting', true);
          return;
        }
        let total = 0;
        root.innerHTML = sections.map((sec) => {
          const items = Array.isArray(sec.Items) ? sec.Items : [];
          total += items.length;
          const itemsHTML = items.length === 0 ? '<p class="empty">🟢 No violations in this status.</p>' : items.map((it) => {
            const unchecked = Array.isArray(it.Unchecked) ? it.Unchecked : [];
            const checksHTML = unchecked.map((c) => '<div class="checklist-row"><span class="checklist-text">• [ ] ' + escHTML(c) + '</span><button class="fix-btn apply-drafting-check-btn" data-repo="' + escHTML(it.Repo) + '" data-issue="' + escHTML(it.Number) + '" data-check="' + escHTML(c) + '">Check on GitHub</button></div>').join('');
            return '<article class="item"><div><strong>#' + escHTML(it.Number) + ' - ' + escHTML(it.Title) + '</strong></div><div>' + renderSafeExternalLink(it.URL) + '</div><ul><li>Assignees: ' + escHTML(listOrEmpty(it.Assignees, '(empty)')) + '</li><li>Labels: ' + escHTML(listOrEmpty(it.Labels, '(empty)')) + '</li></ul><div>' + checksHTML + '</div></article>';
          }).join('');
          return '<div class="status"><h3>' + escHTML(sec.Emoji) + ' ' + escHTML(sec.Status) + '</h3><p class="subtle">' + escHTML(sec.Intro) + '</p>' + itemsHTML + '</div>';
        }).join('');
        setTabClean('drafting', total === 0);
      }

      function renderAssigneeSection(rootID, tabKey, projects, showMineBadge) {
        const root = document.getElementById(rootID);
        if (!root) return;
        if (!Array.isArray(projects) || projects.length === 0) {
          root.innerHTML = '<p class="empty">🟢 No items found.</p>';
          setTabClean(tabKey, true);
          return;
        }
        let total = 0;
        root.innerHTML = projects.map((proj) => {
          const cols = Array.isArray(proj.Columns) ? proj.Columns : [];
          const colsHTML = cols.map((col) => {
            const items = Array.isArray(col.Items) ? col.Items : [];
            total += items.length;
            const colBtn = items.length > 0 ? '<button class="fix-btn apply-assignee-column-btn">Assign selected in column</button>' : '';
            const itemsHTML = items.length === 0 ? '<p class="empty">🟢 No items in this group.</p>' : items.map((it) => {
              const options = (Array.isArray(it.SuggestedAssignees) ? it.SuggestedAssignees : []).map((s) => '<option value="' + escHTML(s.Login) + '">' + escHTML(s.Login) + '</option>').join('');
              const actions = options ? '<div class="actions"><select class="fix-btn assignee-select" data-issue="' + escHTML(it.Number) + '" data-repo="' + escHTML(it.Repo) + '">' + options + '</select><button class="fix-btn apply-assignee-btn">Assign</button></div>' : '<div class="actions"><span class="copied-note">No assignee options found for this repo.</span></div>';
              const badge = showMineBadge ? '<div class="mine-badge">Assigned to me</div>' : '';
              return '<article class="item' + (it.AssignedToMe ? ' assigned-to-me' : '') + '"><div><strong>#' + escHTML(it.Number) + ' - ' + escHTML(it.Title) + '</strong></div><div>' + renderSafeExternalLink(it.URL) + '</div><ul><li>Status: ' + escHTML(it.Status || '(unset)') + '</li><li>Repository: ' + escHTML(it.Repo) + '</li><li>Current assignees: ' + escHTML(listOrEmpty(it.CurrentAssignees, '(none)')) + '</li></ul>' + badge + actions + '</article>';
            }).join('');
            return '<div class="status"><div class="column-head"><h3>' + escHTML(col.Label) + '</h3>' + colBtn + '</div>' + itemsHTML + '</div>';
          }).join('');
          return '<div class="project"><h3>Project ' + escHTML(proj.ProjectNum) + '</h3>' + colsHTML + '</div>';
        }).join('');
        setTabClean(tabKey, total === 0);
      }

      function renderReleaseFromState(state) {
        const root = document.getElementById('release-content');
        if (!root) return;
        const projects = Array.isArray(state.ReleaseLabel) ? state.ReleaseLabel : [];
        if (projects.length === 0) {
          root.innerHTML = '<p class="empty">🟢 No release-label issues found.</p>';
          setTabClean('release', true);
          return;
        }
        let total = 0;
        root.innerHTML = projects.map((proj) => {
          const items = Array.isArray(proj.Items) ? proj.Items : [];
          total += items.length;
          const btn = items.length > 0 ? '<button class="fix-btn apply-release-project-btn">Apply release label</button>' : '';
          const itemsHTML = items.length === 0 ? '<p class="empty">🟢 No release-label issues in this project.</p>' : items.map((it) => '<article class="item release-item" data-repo="' + escHTML(it.Repo) + '" data-issue="' + escHTML(it.Number) + '"><div><strong>#' + escHTML(it.Number) + ' - ' + escHTML(it.Title) + '</strong></div><div>' + renderSafeExternalLink(it.URL) + '</div><ul><li>Status: ' + escHTML(it.Status || '(unset)') + '</li><li>Labels: ' + escHTML(listOrEmpty(it.CurrentLabels, '(none)')) + '</li></ul></article>').join('');
          return '<div class="project"><div class="column-head"><h3>Project ' + escHTML(proj.ProjectNum) + '</h3>' + btn + '</div>' + itemsHTML + '</div>';
        }).join('');
        setTabClean('release', total === 0);
      }

      // Render the generic-query check panel.
      // Each configured query expansion is shown in declaration order with:
      // title, expanded query text, and matched tickets.
      function renderGenericQueriesFromState(state) {
        const root = document.getElementById('generic-queries-content');
        if (!root) return;
        const queries = Array.isArray(state.GenericQueries) ? state.GenericQueries : [];
        if (queries.length === 0) {
          root.innerHTML = '<p class="empty">🟢 No generic queries configured.</p>';
          setTabClean('generic-queries', true);
          return;
        }
        let total = 0;
        root.innerHTML = queries.map((query) => {
          const items = Array.isArray(query.Items) ? query.Items : [];
          total += items.length;
          const itemsHTML = items.length === 0
            ? '<p class="empty">🟢 No issues found for this query.</p>'
            : items.map((it) => (
              '<article class="item">' +
                '<div><strong>#' + escHTML(it.Number) + ' - ' + escHTML(it.Title) + '</strong></div>' +
                '<div>' + renderSafeExternalLink(it.URL) + '</div>' +
                '<ul><li>Status: ' + escHTML(it.Status || '(unset)') + '</li><li>Repository: ' + escHTML(it.Repo || '(unknown)') + '</li><li>Assignees: ' + escHTML(listOrEmpty(it.Assignees, '(none)')) + '</li><li>Labels: ' + escHTML(listOrEmpty(it.Labels, '(none)')) + '</li></ul>' +
              '</article>'
            )).join('');
          return '<div class="project"><h3>' + escHTML(query.Title || '(untitled query)') + '</h3><p class="subtle"><strong>Query:</strong> <code>' + escHTML(query.Query || '') + '</code></p>' + itemsHTML + '</div>';
        }).join('');
        setTabClean('generic-queries', total === 0);
      }

      // Fetch bridge-backed state and re-render all dynamic panels in one pass.
      async function fetchStateAndRender(forceRefresh) {
        const bridgeURL = document.body.dataset.bridgeUrl || window.location.origin || '';
        if (!bridgeURL || !bridgeSession) {
          throw new Error('Bridge unavailable');
        }
        const query = forceRefresh ? '?refresh=1' : '';
        const res = await fetch(bridgeURL + '/api/state' + query, {
          method: 'GET',
          headers: { 'X-Qacheck-Session': bridgeSession },
        });
        if (!res.ok) {
          const body = await res.text();
          throw new Error('Bridge error ' + res.status + ': ' + body);
        }
        const payload = await res.json();
        const state = payload.state || {};
        renderCounts(state);
        renderAwaitingFromState(state);
        renderStaleFromState(state);
        renderMilestoneFromState(state);
        installMilestoneFiltering();
        renderDraftingFromState(state);
        renderAssigneeSection('missing-assignee-content', 'missing-assignee', state.MissingAssignee, false);
        renderAssigneeSection('assigned-to-me-content', 'assigned-to-me', state.AssignedToMe, true);
        installAssigneeFiltering();
        renderReleaseFromState(state);
        renderGenericQueriesFromState(state);
        await refreshReleaseStoryTODOPanel(forceRefresh);
        await refreshMissingSprintPanel(forceRefresh);
        await refreshTimestampPanel(forceRefresh);
        await refreshUnreleasedPanel(forceRefresh);
      }

      function installMilestoneFiltering() {
        const actionBlocks = document.querySelectorAll('.actions');
        actionBlocks.forEach((actions) => {
          const searchInput = actions.querySelector('.milestone-search');
          const select = actions.querySelector('.milestone-select');
          if (!searchInput || !select) return;

          const allOptions = Array.from(select.options).map((opt) => ({
            title: opt.value,
            number: opt.dataset.number || '',
          }));

          function renderFiltered(term) {
            const q = term.trim().toLowerCase();
            const filtered = allOptions.filter((o) => o.title.toLowerCase().includes(q));
            select.innerHTML = '';
            if (filtered.length === 0) {
              const none = document.createElement('option');
              none.textContent = 'No matching milestones';
              none.value = '';
              none.dataset.number = '';
              select.appendChild(none);
              return;
            }
            filtered.forEach((o) => {
              const opt = document.createElement('option');
              opt.value = o.title;
              opt.textContent = o.title;
              opt.dataset.number = o.number;
              select.appendChild(opt);
            });
          }

          searchInput.addEventListener('input', () => renderFiltered(searchInput.value));
        });
      }

      function installAssigneeFiltering() {
        const actionBlocks = document.querySelectorAll('.actions');
        actionBlocks.forEach((actions) => {
          const searchInput = actions.querySelector('.assignee-search');
          const select = actions.querySelector('.assignee-select');
          if (!searchInput || !select) return;

          const allOptions = Array.from(select.options).map((opt) => ({
            login: opt.value,
          }));

          function renderFiltered(term) {
            const q = term.trim().toLowerCase();
            const filtered = allOptions.filter((o) => o.login.toLowerCase().includes(q));
            select.innerHTML = '';
            if (filtered.length === 0) {
              const none = document.createElement('option');
              none.textContent = 'No matching assignees';
              none.value = '';
              select.appendChild(none);
              return;
            }
            filtered.forEach((o) => {
              const opt = document.createElement('option');
              opt.value = o.login;
              opt.textContent = o.login;
              select.appendChild(opt);
            });
          }

          searchInput.addEventListener('input', () => renderFiltered(searchInput.value));
        });
      }

      async function applyMilestoneButton(btn) {
        const actions = btn.closest('.actions');
        const select = actions && actions.querySelector('.milestone-select');
        if (!select) return false;
        const issue = select.dataset.issue || '';
        const repo = select.dataset.repo || '';
        const milestoneTitle = select.value || '';
        const milestoneNumber = parseInt((select.selectedOptions[0] && select.selectedOptions[0].dataset.number) || '', 10);
        if (!issue || !repo || !milestoneTitle || Number.isNaN(milestoneNumber)) return false;

        const bridgeURL = document.body.dataset.bridgeUrl || window.location.origin || '';
        if (!bridgeURL || !bridgeSession) {
          window.alert('Bridge unavailable. Re-run scrumcheck and keep terminal open.');
          return false;
        }

        const endpoint = bridgeURL + '/api/apply-milestone';
        const payload = { repo: repo, issue: issue, milestone_number: milestoneNumber };
        setButtonWorking(btn, 'Applying...');
        try {
          const res = await fetch(endpoint, {
            method: 'POST',
            headers: bridgeJSONHeaders(),
            body: JSON.stringify(payload),
          });
          if (!res.ok) {
            const body = await res.text();
            throw new Error('Bridge error ' + res.status + ': ' + body);
          }
          setButtonDone(btn, 'Done');
          return true;
        } catch (err) {
          window.alert('Could not apply milestone. ' + err);
          setButtonFailed(btn);
          return false;
        }
      }

      async function applyDraftingCheckButton(btn) {
        const bridgeURL = document.body.dataset.bridgeUrl || window.location.origin || '';
        if (!bridgeURL || !bridgeSession) {
          window.alert('Bridge unavailable. Re-run scrumcheck and keep terminal open.');
          return false;
        }
        const repo = btn.dataset.repo || '';
        const issue = btn.dataset.issue || '';
        const checkText = btn.dataset.check || '';
        if (!repo || !issue || !checkText) return false;

        const endpoint = bridgeURL + '/api/apply-checklist';
        setButtonWorking(btn, 'Checking...');
        try {
          const res = await fetch(endpoint, {
            method: 'POST',
            headers: bridgeJSONHeaders(),
            body: JSON.stringify({ repo: repo, issue: issue, check_text: checkText }),
          });
          if (!res.ok) {
            const body = await res.text();
            throw new Error('Bridge error ' + res.status + ': ' + body);
          }
          const payload = await res.json();
          if (!payload.updated) {
            if (payload.already_checked) {
              setButtonDone(btn, 'Done');
            } else {
              setButtonFailed(btn);
            }
            return false;
          }
          setButtonDone(btn, 'Done');
          const row = btn.closest('.checklist-row');
          const textEl = row && row.querySelector('.checklist-text');
          if (textEl) {
            textEl.textContent = '• [x] ' + checkText;
          }
          return true;
        } catch (err) {
          window.alert('Could not apply checklist update. ' + err);
          setButtonFailed(btn);
          return false;
        }
      }

      async function applySprintButton(btn) {
        const bridgeURL = document.body.dataset.bridgeUrl || window.location.origin || '';
        if (!bridgeURL || !bridgeSession) {
          window.alert('Bridge unavailable. Re-run scrumcheck and keep terminal open.');
          return false;
        }
        const itemID = btn.dataset.itemId || '';
        if (!itemID) return false;

        const endpoint = bridgeURL + '/api/apply-sprint';
        setButtonWorking(btn, 'Setting...');
        try {
          const res = await fetch(endpoint, {
            method: 'POST',
            headers: bridgeJSONHeaders(),
            body: JSON.stringify({ item_id: itemID }),
          });
          if (!res.ok) {
            const body = await res.text();
            throw new Error('Bridge error ' + res.status + ': ' + body);
          }
          setButtonDone(btn, 'Done');
          return true;
        } catch (err) {
          window.alert('Could not set sprint. ' + err);
          setButtonFailed(btn);
          return false;
        }
      }
      document.addEventListener('click', async (event) => {
        const milestoneBtn = event.target.closest('.apply-milestone-btn');
        if (milestoneBtn) {
          await applyMilestoneButton(milestoneBtn);
          return;
        }

        const milestoneColBtn = event.target.closest('.apply-milestone-column-btn');
        if (milestoneColBtn) {
          const statusCard = milestoneColBtn.closest('.status');
          if (!statusCard) return;
          const rowButtons = Array.from(statusCard.querySelectorAll('.apply-milestone-btn'));
          if (rowButtons.length === 0) return;
          setButtonWorking(milestoneColBtn, 'Applying column...');
          let ok = true;
          for (const rowBtn of rowButtons) {
            const rowOK = await applyMilestoneButton(rowBtn);
            ok = ok && rowOK;
          }
          if (ok) setButtonDone(milestoneColBtn, 'Done'); else setButtonFailed(milestoneColBtn);
          return;
        }

        const draftingBtn = event.target.closest('.apply-drafting-check-btn');
        if (draftingBtn) {
          await applyDraftingCheckButton(draftingBtn);
          return;
        }

        const rowBtn = event.target.closest('.apply-sprint-btn');
        if (rowBtn) {
          await applySprintButton(rowBtn);
          return;
        }

        const colBtn = event.target.closest('.apply-sprint-column-btn');
        if (colBtn) {
          const statusCard = colBtn.closest('.status');
          if (!statusCard) return;
          const rowButtons = Array.from(statusCard.querySelectorAll('.apply-sprint-btn'));
          if (rowButtons.length === 0) return;
          setButtonWorking(colBtn, 'Setting column...');
          let ok = true;
          for (const rowBtnEl of rowButtons) {
            const rowOK = await applySprintButton(rowBtnEl);
            ok = ok && rowOK;
          }
          if (ok) setButtonDone(colBtn, 'Done'); else setButtonFailed(colBtn);
          return;
        }

        const assigneeBtn = event.target.closest('.apply-assignee-btn');
        if (assigneeBtn) {
          await applyAssigneeButton(assigneeBtn);
          return;
        }

        const assigneeColBtn = event.target.closest('.apply-assignee-column-btn');
        if (assigneeColBtn) {
          const statusCard = assigneeColBtn.closest('.status');
          if (!statusCard) return;
          const rowButtons = Array.from(statusCard.querySelectorAll('.apply-assignee-btn'));
          if (rowButtons.length === 0) return;
          setButtonWorking(assigneeColBtn, 'Assigning column...');
          let ok = true;
          for (const rowBtnEl of rowButtons) {
            const rowOK = await applyAssigneeButton(rowBtnEl);
            ok = ok && rowOK;
          }
          if (ok) setButtonDone(assigneeColBtn, 'Done'); else setButtonFailed(assigneeColBtn);
          return;
        }

        const releaseProjectBtn = event.target.closest('.apply-release-project-btn');
        if (releaseProjectBtn) {
          const project = releaseProjectBtn.closest('.project');
          if (!project) return;
          const items = Array.from(project.querySelectorAll('.release-item'));
          if (items.length === 0) return;
          setButtonWorking(releaseProjectBtn, 'Applying...');
          try {
            let ok = true;
            for (const item of items) {
              const itemOK = await applyReleaseItem(item);
              ok = ok && itemOK;
            }
            if (ok) setButtonDone(releaseProjectBtn, 'Done'); else setButtonFailed(releaseProjectBtn);
          } catch (err) {
            window.alert('Could not apply release label. ' + err);
            setButtonFailed(releaseProjectBtn);
          }
          return;
        }
      });

      async function applyAssigneeButton(btn) {
        const actions = btn.closest('.actions');
        const select = actions && actions.querySelector('.assignee-select');
        if (!select) return false;
        const issue = select.dataset.issue || '';
        const repo = select.dataset.repo || '';
        const assignee = select.value || '';
        if (!issue || !repo || !assignee) return false;

        const bridgeURL = document.body.dataset.bridgeUrl || window.location.origin || '';
        if (!bridgeURL || !bridgeSession) {
          window.alert('Bridge unavailable. Re-run scrumcheck and keep terminal open.');
          return false;
        }
        const endpoint = bridgeURL + '/api/add-assignee';
        const payload = { repo: repo, issue: issue, assignee: assignee };
        setButtonWorking(btn, 'Assigning...');
        try {
          const res = await fetch(endpoint, {
            method: 'POST',
            headers: bridgeJSONHeaders(),
            body: JSON.stringify(payload),
          });
          if (!res.ok) {
            const body = await res.text();
            throw new Error('Bridge error ' + res.status + ': ' + body);
          }
          setButtonDone(btn, 'Done');
          return true;
        } catch (err) {
          window.alert('Could not assign user. ' + err);
          setButtonFailed(btn);
          return false;
        }
      }


      async function applyReleaseItem(itemEl) {
        const repo = itemEl.dataset.repo || '';
        const issue = itemEl.dataset.issue || '';
        if (!repo || !issue) return false;

        const bridgeURL = document.body.dataset.bridgeUrl || window.location.origin || '';
        if (!bridgeURL || !bridgeSession) {
          window.alert('Bridge unavailable. Re-run scrumcheck and keep terminal open.');
          return false;
        }
        const endpoint = bridgeURL + '/api/apply-release-label';
        const res = await fetch(endpoint, {
          method: 'POST',
          headers: bridgeJSONHeaders(),
          body: JSON.stringify({ repo: repo, issue: issue }),
        });
        if (!res.ok) {
          const body = await res.text();
          throw new Error('Bridge error ' + res.status + ': ' + body);
        }
        return true;
      }


      const closeSessionButton = document.getElementById('close-session-btn');
      if (closeSessionButton) {
        closeSessionButton.addEventListener('click', async () => {
          const bridgeURL = document.body.dataset.bridgeUrl || window.location.origin || '';
          if (!bridgeURL || !bridgeSession) {
            window.alert('Bridge unavailable.');
            return;
          }
          setButtonWorking(closeSessionButton, 'Closing...');
          try {
            const res = await fetch(bridgeURL + '/api/close', {
              method: 'POST',
              headers: bridgeJSONHeaders(),
              body: JSON.stringify({ reason: 'closed from UI' }),
            });
            if (!res.ok) {
              const body = await res.text();
              throw new Error('Bridge error ' + res.status + ': ' + body);
            }
            document.querySelectorAll('.apply-milestone-btn, .apply-milestone-column-btn, .apply-drafting-check-btn, .apply-sprint-btn, .apply-sprint-column-btn, .apply-assignee-btn, .apply-assignee-column-btn, .apply-release-project-btn').forEach((el) => {
              el.disabled = true;
            });
            setButtonDone(closeSessionButton, 'Done');
          } catch (err) {
            window.alert('Could not close session. ' + err);
            setButtonFailed(closeSessionButton);
          }
        });
      }

      const refreshButtons = document.querySelectorAll('.refresh-check-btn');
      refreshButtons.forEach((btn) => {
        btn.addEventListener('click', async () => {
          setButtonWorking(btn, 'Refreshing...');
          try {
            await fetchStateAndRender(true);
            btn.classList.remove('done', 'failed');
            btn.textContent = 'Refresh';
            btn.disabled = false;
          } catch (_) {
            setButtonFailed(btn);
          }
        });
      });

      fetchStateAndRender(false).catch((err) => {
        console.error(err);
      });
      connectBridgeEvents();
    })();
