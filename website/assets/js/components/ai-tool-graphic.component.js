/**
 * <ai-tool-graphic>
 * -----------------------------------------------------------------------------
 * Animated mockups of Claude Code, Cursor, GitHub Copilot, and Codex working
 * with Fleet, with buttons to switch between them.
 *
 * @type {Component}
 * -----------------------------------------------------------------------------
 */

parasails.registerComponent('aiToolGraphic', {
  //  ╔═╗╦═╗╔═╗╔═╗╔═╗
  //  ╠═╝╠╦╝║ ║╠═╝╚═╗
  //  ╩  ╩╚═╚═╝╩  ╚═╝
  props: [],

  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: function() {
    return {
      selectedAiTool: 'claude',
      demoStep: 0,
      demoComposerText: '',
      demoScrollOffset: 0,
      demoFadingOut: false,
      demoResetting: false,
      demoReducedMotion: false,
      // Where the running animation is, so switching tools can pick up from the same point.
      demoEntryStartMs: 0,
      demoEntryMs: 0,
      demoEntryStartedAt: 0,
      // Each entry is [step to reveal (or text to type into the composer), ms to wait afterwards].
      // Every timeline has the admin's "Yes", which lines them up when the visitor switches tools.
      demoTimelines: {
        codex: [
          [null, 1200],
          [2, 1000],
          [3, 1100],
          [4, 500],
          [5, 600],
          [6, 1600],
          ['Yes', 400],
          [7, 800],
          [8, 900],
          [9, 1000],
          [10, 900],
          [11, 4500],
        ],
        'github-copilot': [
          [null, 1200],
          [2, 1000],
          [3, 1100],
          [4, 500],
          [5, 600],
          [6, 1600],
          ['Yes', 400],
          [7, 800],
          [8, 700],
          [9, 700],
          [10, 800],
          [11, 800],
          [12, 800],
          [13, 4500],
        ],
        cursor: [
          [null, 1200],
          [2, 1000],
          [3, 1100],
          [4, 500],
          [5, 600],
          [6, 1600],
          ['Yes', 400],
          [7, 800],
          [8, 900],
          [9, 700],
          [10, 1000],
          [11, 4500],
        ],
        claude: [
          [null, 1200],
          [2, 600],
          [3, 1000],
          [4, 1100],
          [5, 500],
          [6, 600],
          [7, 1600],
          ['Yes', 400],
          [8, 800],
          [9, 700],
          [10, 1000],
          [11, 800],
          [12, 4500],
        ],
      },
      // Bumped to cancel a running animation when the visitor switches tools.
      demoRunId: 0,
    };
  },

  //  ╦ ╦╔╦╗╔╦╗╦
  //  ╠═╣ ║ ║║║║
  //  ╩ ╩ ╩ ╩ ╩╩═╝
  template: `
  <div purpose="ai-tool-graphic" id="ai-tool-graphic-component">
    <div purpose="ai-tool-switcher" role="group" aria-label="AI tool">
      <button type="button" purpose="ai-tool-button" :class="{selected: selectedAiTool === 'claude'}" :aria-pressed="selectedAiTool === 'claude' ? 'true' : 'false'" @click="clickSelectAiTool('claude')"><img alt="" src="/images/icon-claude-logo-16x16.svg">Claude</button>
      <button type="button" purpose="ai-tool-button" :class="{selected: selectedAiTool === 'cursor'}" :aria-pressed="selectedAiTool === 'cursor' ? 'true' : 'false'" @click="clickSelectAiTool('cursor')"><img alt="" src="/images/icon-cursor-logo-16x16.svg">Cursor</button>
      <button type="button" purpose="ai-tool-button" :class="{selected: selectedAiTool === 'github-copilot'}" :aria-pressed="selectedAiTool === 'github-copilot' ? 'true' : 'false'" @click="clickSelectAiTool('github-copilot')"><img alt="" src="/images/icon-github-copilot-logo-16x16.svg">GitHub Copilot</button>
      <button type="button" purpose="ai-tool-button" :class="{selected: selectedAiTool === 'codex'}" :aria-pressed="selectedAiTool === 'codex' ? 'true' : 'false'" @click="clickSelectAiTool('codex')"><img alt="" src="/images/icon-codex-logo-16x16.svg">Codex</button>
    </div>
    <div purpose="cursor-animation" role="img" aria-label="A Cursor agent connected to Fleet. An admin asks which Macs in Engineering have an MCP server with shell access, and whether any of them are behind on macOS. Cursor checks with Fleet and reports that 14 of 1,120 Engineering Macs have one and 5 of those are still on macOS 15.7, then offers to propose a fix. The admin says yes, and Cursor writes the policy and opens a pull request for their review." :class="{'reduced-motion': demoReducedMotion}" v-if="selectedAiTool === 'cursor'">
      <div purpose="cursor-window" aria-hidden="true">
        <img purpose="ai-demo-window-controls" alt="" src="/images/icon-macos-window-controls-47x11.svg">
        <div purpose="cursor-sidebar">
          <div purpose="cursor-window-controls">
            <img alt="" src="/images/icon-cursor-panel-left-15x15.svg">
            <img alt="" src="/images/icon-cursor-search-15x15.svg">
            <img purpose="cursor-back" alt="" src="/images/icon-cursor-arrow-left-15x15.svg">
            <img alt="" src="/images/icon-cursor-arrow-right-15x15.svg">
          </div>
          <div purpose="cursor-sidebar-item"><img alt="" src="/images/icon-cursor-new-chat-13x13.svg"><span purpose="cursor-placeholder-bar"></span></div>
          <div purpose="cursor-sidebar-item"><img alt="" src="/images/icon-cursor-bot-14x14.svg"><span purpose="cursor-placeholder-bar"></span></div>
          <div purpose="cursor-sidebar-item"><img alt="" src="/images/icon-cursor-blocks-14x14.svg"><span purpose="cursor-placeholder-bar"></span></div>
          <div purpose="cursor-sidebar-label">
            <span>Repositories</span>
            <img alt="" src="/images/icon-cursor-list-filter-14x14.svg">
            <img alt="" src="/images/icon-cursor-folder-plus-14x14.svg">
          </div>
          <div purpose="cursor-sidebar-item"><img alt="" src="/images/icon-cursor-folder-14x14.svg"><span purpose="cursor-repo-name">fleet-gitops</span></div>
          <div purpose="cursor-sidebar-item" class="selected"><img alt="" src="/images/icon-cursor-dot-14x14.svg"><span purpose="cursor-session-name">Engineering Macs MCP audit</span></div>
          <div purpose="cursor-sidebar-item"><img alt="" src="/images/icon-cursor-dot-14x14.svg"><span purpose="cursor-placeholder-bar"></span></div>
          <div purpose="cursor-sidebar-item"><img alt="" src="/images/icon-cursor-dot-14x14.svg"><span purpose="cursor-placeholder-bar"></span></div>
          <div purpose="cursor-sidebar-item"><img alt="" src="/images/icon-cursor-dot-14x14.svg"><span purpose="cursor-placeholder-bar"></span></div>
          <div purpose="cursor-account">
            <span purpose="cursor-avatar"><img alt="" src="/images/avatar-claude-code-mike-24x29@2x.png"></span>
            <span purpose="cursor-placeholder-bar"></span>
            <img purpose="cursor-settings" alt="" src="/images/icon-cursor-settings-15x15.svg">
          </div>
        </div>
        <div purpose="cursor-content">
          <div purpose="cursor-header">
            <span purpose="cursor-tab">Engineering Macs MCP audit</span>
            <img purpose="cursor-ellipsis" alt="" src="/images/icon-cursor-ellipsis-16x16.svg">
            <img alt="" src="/images/icon-cursor-panel-right-15x15.svg">
          </div>
          <div purpose="cursor-body">
            <div purpose="cursor-main">
              <div purpose="cursor-conversation" ref="demoConversation">
                <div purpose="cursor-thread" ref="demoThread" :class="{resetting: demoResetting, 'fading-out': demoFadingOut}" :style="{transform: 'translateY(-' + demoScrollOffset + 'px)'}">
                  <div purpose="cursor-card"><p>Which Macs in Engineering have an MCP server with shell access, and are any of them behind on macOS?</p><img alt="" src="/images/icon-cursor-undo-14x14.svg"></div>
                  <div purpose="cursor-steps" v-show="demoStep >= 2">
                    <p purpose="cursor-step"><span>Called</span>&nbsp; fleet.run_live_query · 1,120 hosts</p>
                    <p purpose="cursor-step" v-show="demoStep >= 3"><span>Called</span>&nbsp; fleet.list_hosts · 5 results</p>
                  </div>
                  <p v-show="demoStep >= 4"><strong>14 of 1,120</strong> Engineering Macs have an MCP server with shell access. <strong>5</strong> of those are also behind on macOS, still on 15.7:</p>
                  <div purpose="cursor-table" v-show="demoStep >= 5">
                    <div purpose="cursor-table-header"><span>Host</span><span>macOS</span><span>MCP server</span></div>
                    <div purpose="cursor-table-row"><span><code>eng-mbp-0318</code></span><span>15.7</span><span><code>desktop-commander</code></span></div>
                    <div purpose="cursor-table-row"><span><code>eng-mbp-0442</code></span><span>15.7</span><span><code>iterm-mcp</code></span></div>
                    <div purpose="cursor-table-row"><span><code>eng-mbp-0517</code></span><span>15.7</span><span><code>mcp-server-commands</code></span></div>
                    <div purpose="cursor-table-row"><span><code>eng-mbp-0623</code></span><span>15.7</span><span><code>desktop-commander</code></span></div>
                    <div purpose="cursor-table-row"><span><code>eng-mbp-0781</code></span><span>15.7</span><span><code>mcp-shell</code></span></div>
                  </div>
                  <p v-show="demoStep >= 6">Want me to propose a fix?</p>
                  <div purpose="cursor-card" v-show="demoStep >= 7"><p>Yes</p><img alt="" src="/images/icon-cursor-undo-14x14.svg"></div>
                  <div purpose="cursor-file-card" v-show="demoStep >= 8">
                    <img alt="" src="/images/icon-cursor-file-diff-15x15.svg">
                    <span purpose="cursor-file-path">lib/policies/mcp-shell-access.yml</span>
                    <span purpose="cursor-added">+18</span>
                    <span purpose="cursor-removed">-0</span>
                  </div>
                  <div purpose="cursor-file-card" v-show="demoStep >= 9">
                    <img alt="" src="/images/icon-cursor-file-diff-15x15.svg">
                    <span purpose="cursor-file-path">teams/engineering.yml</span>
                    <span purpose="cursor-added">+6</span>
                    <span purpose="cursor-removed">-0</span>
                  </div>
                  <p purpose="cursor-step" v-show="demoStep >= 10"><span>Ran</span>&nbsp; gh pr create --base main</p>
                  <p v-show="demoStep >= 11">Done. I’ve proposed a policy that flags MCP shell access and a macOS update deadline for those 5 Macs, <span purpose="cursor-link">for your review</span>.</p>
                  <div purpose="cursor-feedback" v-show="demoStep >= 11">
                    <img alt="" src="/images/icon-cursor-thumbs-up-13x13.svg">
                    <img alt="" src="/images/icon-cursor-thumbs-down-13x13.svg">
                    <img alt="" src="/images/icon-cursor-copy-13x13.svg">
                    <img alt="" src="/images/icon-cursor-git-fork-13x13.svg">
                    <span>Just now</span>
                  </div>
                </div>
              </div>
              <div purpose="cursor-composer">
                <div purpose="cursor-input">
                  <span purpose="cursor-add"><img alt="" src="/images/icon-cursor-plus-14x14.svg"></span>
                  <span purpose="cursor-input-text" v-if="demoComposerText">{{demoComposerText}}<span purpose="ai-demo-caret"></span></span>
                  <span purpose="cursor-placeholder" v-else>Send follow-up</span>
                  <span purpose="cursor-model">Cursor Grok 4.6 Medium</span>
                  <img alt="" src="/images/icon-cursor-mic-15x15.svg">
                  <span purpose="cursor-send"><img alt="" src="/images/icon-cursor-audio-lines-14x14.svg"></span>
                </div>
                <div purpose="cursor-environment">
                  <img alt="" src="/images/icon-cursor-laptop-13x13.svg">
                  <span>This Mac</span>
                  <img alt="" src="/images/icon-cursor-chevron-down-12x12.svg">
                  <img purpose="cursor-context" alt="" src="/images/icon-cursor-context-12x12.svg">
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
    <div purpose="copilot-animation" role="img" aria-label="A GitHub Copilot session connected to Fleet. An admin asks which Macs in Engineering have an MCP server with shell access, and whether any of them are behind on macOS. Copilot checks with Fleet and reports that 14 of 1,120 Engineering Macs have one and 5 of those are still on macOS 15.7, then offers to propose a fix. The admin says yes, and Copilot creates the policy, commits the change, and opens a pull request for their review." :class="{'reduced-motion': demoReducedMotion}" v-else-if="selectedAiTool === 'github-copilot'">
      <div purpose="copilot-window" aria-hidden="true">
        <img purpose="ai-demo-window-controls" alt="" src="/images/icon-macos-window-controls-47x11.svg">
        <div purpose="copilot-sidebar">
          <div purpose="copilot-title-bar">
            <img alt="" src="/images/icon-github-copilot-panel-left-16x16.svg">
            <img purpose="copilot-back" alt="" src="/images/icon-github-copilot-chevron-left-16x16.svg">
            <img alt="" src="/images/icon-github-copilot-chevron-right-16x16.svg">
          </div>
          <div purpose="copilot-nav-item"><img alt="" src="/images/icon-github-copilot-house-16x16.svg"><span purpose="copilot-placeholder-bar"></span></div>
          <div purpose="copilot-nav-item"><img alt="" src="/images/icon-github-copilot-list-todo-16x16.svg"><span purpose="copilot-placeholder-bar"></span></div>
          <div purpose="copilot-nav-item"><img alt="" src="/images/icon-github-copilot-calendar-clock-16x16.svg"><span purpose="copilot-placeholder-bar"></span></div>
          <div purpose="copilot-nav-item"><img alt="" src="/images/icon-github-copilot-blocks-16x16.svg"><span purpose="copilot-placeholder-bar"></span></div>
          <div purpose="copilot-sessions-header">
            <span>Sessions</span>
            <img alt="" src="/images/icon-github-copilot-list-filter-16x16.svg">
            <img alt="" src="/images/icon-github-copilot-plus-outline-16x16.svg">
          </div>
          <div purpose="copilot-nav-item"><img alt="" src="/images/icon-github-copilot-message-square-16x16.svg"><span purpose="copilot-placeholder-bar"></span></div>
          <div purpose="copilot-nav-item"><img alt="" src="/images/icon-github-copilot-folder-16x16.svg"><span purpose="copilot-repo-name">fleet-gitops</span></div>
          <div purpose="copilot-nav-item" class="nested selected"><span purpose="copilot-session-name">Engineering Macs MCP audit</span></div>
          <div purpose="copilot-nav-item" class="nested"><span purpose="copilot-placeholder-bar"></span></div>
          <div purpose="copilot-nav-item" class="nested"><span purpose="copilot-placeholder-bar"></span></div>
          <div purpose="copilot-nav-item" class="nested"><span purpose="copilot-placeholder-bar"></span></div>
          <div purpose="copilot-nav-item" class="nested"><span purpose="copilot-placeholder-bar"></span></div>
          <div purpose="copilot-account">
            <span purpose="copilot-avatar"><img alt="" src="/images/avatar-claude-code-mike-24x29@2x.png"></span>
          </div>
        </div>
        <div purpose="copilot-main">
          <div purpose="copilot-header">
            <span purpose="copilot-session-title">Engineering Macs MCP audit</span>
            <span purpose="copilot-run-button"><span><img alt="" src="/images/icon-github-copilot-play-14x14.svg"></span><span><img alt="" src="/images/icon-github-copilot-chevron-down-14x14.svg"></span></span>
            <img purpose="copilot-panel-icon" alt="" src="/images/icon-github-copilot-panel-right-18x18.svg">
          </div>
          <div purpose="copilot-conversation" ref="demoConversation">
            <div purpose="copilot-thread" ref="demoThread" :class="{resetting: demoResetting, 'fading-out': demoFadingOut}" :style="{transform: 'translateY(-' + demoScrollOffset + 'px)'}">
              <div purpose="copilot-bubble"><p>Which Macs in Engineering have an MCP server with shell access, and are any of them behind on macOS?</p></div>
              <div purpose="copilot-activity-group" v-show="demoStep >= 2">
                <div purpose="copilot-activity"><img alt="" src="/images/icon-github-copilot-terminal-16x16.svg"><span>Run live query</span><code>mcp_servers WHERE shell_access = 1</code><span purpose="copilot-activity-meta">· 1,120 hosts</span></div>
                <div purpose="copilot-activity" v-show="demoStep >= 3"><img alt="" src="/images/icon-github-copilot-terminal-16x16.svg"><span>List hosts</span><code>team:Engineering os_version&lt;26</code><span purpose="copilot-activity-meta">· 5 results</span></div>
              </div>
              <p v-show="demoStep >= 4"><strong>14 of 1,120</strong> Engineering Macs have an MCP server with shell access. <strong>5</strong> of those are also behind on macOS, still on 15.7:</p>
              <div purpose="copilot-table" v-show="demoStep >= 5">
                <div purpose="copilot-table-header"><span>Host</span><span>macOS</span><span>MCP server</span></div>
                <div purpose="copilot-table-row"><span>eng-mbp-0318</span><span>15.7</span><span>desktop-commander</span></div>
                <div purpose="copilot-table-row"><span>eng-mbp-0442</span><span>15.7</span><span>iterm-mcp</span></div>
                <div purpose="copilot-table-row"><span>eng-mbp-0517</span><span>15.7</span><span>mcp-server-commands</span></div>
                <div purpose="copilot-table-row"><span>eng-mbp-0623</span><span>15.7</span><span>desktop-commander</span></div>
                <div purpose="copilot-table-row"><span>eng-mbp-0781</span><span>15.7</span><span>mcp-shell</span></div>
              </div>
              <p v-show="demoStep >= 6">Want me to propose a fix?</p>
              <div purpose="copilot-bubble" v-show="demoStep >= 7"><p>Yes</p></div>
              <div purpose="copilot-activity-group" v-show="demoStep >= 8">
                <div purpose="copilot-activity"><img alt="" src="/images/icon-github-copilot-file-diff-16x16.svg"><span>Create</span><span>lib/policies/mcp-shell-access.yml</span><span purpose="copilot-added">+18</span><span purpose="copilot-removed">-0</span></div>
                <div purpose="copilot-activity" v-show="demoStep >= 9"><img alt="" src="/images/icon-github-copilot-file-diff-16x16.svg"><span>Edit</span><span>teams/engineering.yml</span><span purpose="copilot-added">+6</span><span purpose="copilot-removed">-0</span></div>
                <div purpose="copilot-activity" v-show="demoStep >= 10"><img alt="" src="/images/icon-github-copilot-terminal-16x16.svg"><span>Commit changes</span><code>git commit -m "Add MCP shell access policy"</code><span purpose="copilot-activity-meta">· 2 files</span></div>
                <div purpose="copilot-activity" v-show="demoStep >= 11"><img alt="" src="/images/icon-github-copilot-terminal-16x16.svg"><span>Push branch</span><code>git push -u origin HEAD</code></div>
              </div>
              <div purpose="copilot-pull-request" v-show="demoStep >= 12">
                <div purpose="copilot-pull-request-title"><img alt="" src="/images/icon-github-copilot-pull-request-green-16x16.svg"><span>Pull request created</span><span purpose="copilot-added">+24</span><span purpose="copilot-removed">-0</span></div>
                <span purpose="copilot-pull-request-branches">mcp-shell-access → origin/main</span>
              </div>
              <p v-show="demoStep >= 13">Done. I’ve proposed a policy that flags MCP shell access and a macOS update deadline for those 5 Macs, <span purpose="copilot-link">for your review</span>.</p>
            </div>
          </div>
          <div purpose="copilot-composer">
            <div purpose="copilot-input">
              <span purpose="copilot-input-text" v-if="demoComposerText">{{demoComposerText}}<span purpose="ai-demo-caret"></span></span>
              <span purpose="copilot-placeholder" v-else>Ask anything. Use / for commands, @ files, &amp; sessions, # issues...</span>
              <div purpose="copilot-controls">
                <img alt="" src="/images/icon-github-copilot-plus-16x16.svg">
                <span purpose="copilot-picker">Interactive</span>
                <span purpose="copilot-picker">Copilot Fusion · Standard</span>
                <span purpose="copilot-send"><img alt="" src="/images/icon-github-copilot-arrow-up-16x16.svg"></span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
    <div purpose="codex-animation" role="img" aria-label="A Codex session connected to Fleet. An admin asks which Macs in Engineering have an MCP server with shell access, and whether any of them are behind on macOS. Codex checks with Fleet and reports that 14 of 1,120 Engineering Macs have one and 5 of those are still on macOS 15.7, then offers to propose a fix. The admin says yes, and Codex edits two files and opens a pull request for their review." :class="{'reduced-motion': demoReducedMotion}" v-else-if="selectedAiTool === 'codex'">
      <div purpose="codex-window" aria-hidden="true">
        <img purpose="ai-demo-window-controls" alt="" src="/images/icon-macos-window-controls-47x11.svg">
        <div purpose="codex-title-bar">
          <div purpose="codex-window-controls">
            <img alt="" src="/images/icon-codex-arrow-left-15x15.svg">
            <img alt="" src="/images/icon-codex-arrow-right-disabled-15x15.svg">
            <img alt="" src="/images/icon-codex-panel-left-15x15.svg">
          </div>
          <span purpose="codex-window-title">Engineering Macs MCP audit</span>
          <div purpose="codex-window-actions">
            <img purpose="codex-ellipsis" alt="" src="/images/icon-codex-ellipsis-16x16.svg">
            <img alt="" src="/images/icon-codex-list-checks-15x15.svg">
            <span purpose="codex-divider"></span>
            <img alt="" src="/images/icon-codex-square-plus-15x15.svg">
          </div>
        </div>
        <div purpose="codex-body">
          <div purpose="codex-rail">
            <span purpose="codex-rail-item" class="selected"><img alt="" src="/images/icon-codex-home-16x16.svg"></span>
            <span purpose="codex-rail-item"><img alt="" src="/images/icon-codex-images-16x16.svg"></span>
            <span purpose="codex-rail-item"><img alt="" src="/images/icon-codex-clock-16x16.svg"></span>
            <span purpose="codex-rail-item"><img alt="" src="/images/icon-codex-library-16x16.svg"></span>
            <span purpose="codex-rail-item"><img alt="" src="/images/icon-codex-at-sign-16x16.svg"></span>
            <span purpose="codex-rail-item"><img alt="" src="/images/icon-codex-ellipsis-16x16.svg"></span>
            <span purpose="codex-rail-divider"></span>
            <span purpose="codex-rail-item"><img alt="" src="/images/icon-codex-pull-request-16x16.svg"></span>
            <span purpose="codex-avatar">MT</span>
          </div>
          <div purpose="codex-panel">
            <div purpose="codex-sidebar">
              <div purpose="codex-sidebar-header">
                <span purpose="codex-sidebar-title"><span>Codex</span><img alt="" src="/images/icon-codex-chevron-down-14x14.svg"></span>
                <img alt="" src="/images/icon-codex-bell-15x15.svg">
                <img alt="" src="/images/icon-codex-search-15x15.svg">
              </div>
              <div purpose="codex-sidebar-item"><img alt="" src="/images/icon-codex-new-chat-15x15.svg"><span purpose="codex-placeholder-bar" style="width: 99em;"></span></div>
              <span purpose="codex-sidebar-label">Projects</span>
              <div purpose="codex-sidebar-item"><img alt="" src="/images/icon-codex-folder-15x15.svg"><span>fleet-gitops</span></div>
              <span purpose="codex-sidebar-label">Recents</span>
              <div purpose="codex-sidebar-item" class="selected"><span purpose="codex-session-name">Engineering Macs MCP audit</span></div>
              <div purpose="codex-sidebar-item"><span purpose="codex-placeholder-bar" style="width: 99em;"></span></div>
              <div purpose="codex-sidebar-item"><span purpose="codex-placeholder-bar" style="width: 119em;"></span></div>
              <div purpose="codex-sidebar-item"><span purpose="codex-placeholder-bar" style="width: 86em;"></span></div>
              <div purpose="codex-sidebar-item"><span purpose="codex-placeholder-bar" style="width: 112em;"></span></div>
              <div purpose="codex-sidebar-item"><span purpose="codex-placeholder-bar" style="width: 94em;"></span></div>
            </div>
            <div purpose="codex-main">
              <div purpose="codex-conversation" ref="demoConversation">
                <div purpose="codex-thread" ref="demoThread" :class="{resetting: demoResetting, 'fading-out': demoFadingOut}" :style="{transform: 'translateY(-' + demoScrollOffset + 'px)'}">
                  <div purpose="codex-bubble"><p>Which Macs in Engineering have an MCP server with shell access, and are any of them behind on macOS?</p></div>
                  <div purpose="codex-work" v-show="demoStep >= 2">
                    <div purpose="codex-worked-for"><span>Worked for 14s</span><img alt="" src="/images/icon-codex-chevron-right-14x14.svg"></div>
                    <p purpose="codex-call">Called <code>fleet.run_live_query</code>&nbsp; · 1,120 hosts</p>
                    <p purpose="codex-call" v-show="demoStep >= 3">Called <code>fleet.list_hosts</code>&nbsp; · 5 results</p>
                  </div>
                  <p v-show="demoStep >= 4"><strong>14 of 1,120</strong> Engineering Macs have an MCP server with shell access. <strong>5</strong> of those are also behind on macOS, still on 15.7:</p>
                  <div purpose="codex-table" v-show="demoStep >= 5">
                    <div purpose="codex-table-header"><span>Host</span><span>macOS</span><span>MCP server</span></div>
                    <div purpose="codex-table-row"><span><code>eng-mbp-0318</code></span><span>15.7</span><span><code>desktop-commander</code></span></div>
                    <div purpose="codex-table-row"><span><code>eng-mbp-0442</code></span><span>15.7</span><span><code>iterm-mcp</code></span></div>
                    <div purpose="codex-table-row"><span><code>eng-mbp-0517</code></span><span>15.7</span><span><code>mcp-server-commands</code></span></div>
                    <div purpose="codex-table-row"><span><code>eng-mbp-0623</code></span><span>15.7</span><span><code>desktop-commander</code></span></div>
                    <div purpose="codex-table-row"><span><code>eng-mbp-0781</code></span><span>15.7</span><span><code>mcp-shell</code></span></div>
                  </div>
                  <p v-show="demoStep >= 6">Want me to propose a fix?</p>
                  <div purpose="codex-bubble" v-show="demoStep >= 7"><p>Yes</p></div>
                  <div purpose="codex-worked-for" v-show="demoStep >= 8"><span>Worked for 38s</span><img alt="" src="/images/icon-codex-chevron-right-14x14.svg"></div>
                  <div purpose="codex-edited-files" v-show="demoStep >= 9">
                    <div purpose="codex-edited-summary">
                      <span purpose="codex-edited-icon"><img alt="" src="/images/icon-codex-file-diff-16x16.svg"></span>
                      <div>
                        <span purpose="codex-edited-title">Edited 2 files</span>
                        <span purpose="codex-edited-stats"><span purpose="codex-added">+24</span> <span purpose="codex-removed">−0</span></span>
                      </div>
                      <span purpose="codex-review-button">Review</span>
                    </div>
                    <div purpose="codex-edited-file"><span><span purpose="codex-path">lib/policies/</span>mcp-shell-access.yml</span><span><span purpose="codex-added">+18</span> <span purpose="codex-removed">−0</span></span></div>
                    <div purpose="codex-edited-file"><span><span purpose="codex-path">teams/</span>engineering.yml</span><span><span purpose="codex-added">+6</span> <span purpose="codex-removed">−0</span></span></div>
                  </div>
                  <div purpose="codex-pull-request" v-show="demoStep >= 10"><img alt="" src="/images/icon-codex-pull-request-14x14.svg"><span>Opened pull request</span><code>mcp-shell-access → main</code></div>
                  <p v-show="demoStep >= 11">Done. I’ve proposed a policy that flags MCP shell access and a macOS update deadline for those 5 Macs, <span purpose="codex-link">for your review</span>.</p>
                </div>
              </div>
              <div purpose="codex-composer">
                <div purpose="codex-input">
                  <span purpose="codex-input-text" v-if="demoComposerText">{{demoComposerText}}<span purpose="ai-demo-caret"></span></span>
                  <span purpose="codex-placeholder" v-else>Work with Codex</span>
                  <div purpose="codex-controls">
                    <img purpose="codex-plus" alt="" src="/images/icon-codex-plus-17x17.svg">
                    <span purpose="codex-approval"><img alt="" src="/images/icon-codex-hand-14x14.svg"><span>Ask for approval</span></span>
                    <span purpose="codex-model"><span>6 Luna <span>Medium</span></span><img alt="" src="/images/icon-codex-chevron-down-12x12.svg"></span>
                    <img alt="" src="/images/icon-codex-mic-16x16.svg">
                    <span purpose="codex-send"><img alt="" src="/images/icon-codex-arrow-up-16x16.svg"></span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
    <div purpose="claude-code-animation" role="img" aria-label="A Claude Code session connected to Fleet. An admin asks which Macs in Engineering have an MCP server with shell access, and whether any of them are behind on macOS. Claude checks with Fleet and reports that 14 of 1,120 Engineering Macs have one and 5 of those are still on macOS 15.7, then offers to propose a fix. The admin says yes, and Claude opens a pull request with a policy and a macOS update deadline for their review." :class="{'reduced-motion': demoReducedMotion}" v-else>
      <div purpose="claude-code-window" aria-hidden="true">
        <img purpose="ai-demo-window-controls" alt="" src="/images/icon-macos-window-controls-47x11.svg">
        <div purpose="claude-code-sidebar">
          <div purpose="claude-code-title-bar">
            <img alt="" src="/images/icon-claude-code-sidebar-18x18.svg">
            <img alt="" src="/images/icon-claude-code-back-16x16.svg">
            <img alt="" src="/images/icon-claude-code-forward-16x16.svg">
            <div purpose="claude-code-mode-toggle">
              <span><img alt="" src="/images/icon-claude-code-chat-16x16.svg"></span>
              <span purpose="claude-code-mode-selected"><img alt="" src="/images/icon-claude-code-code-16x16.svg"></span>
            </div>
          </div>
          <div purpose="claude-code-nav">
            <div purpose="claude-code-nav-item"><img alt="" src="/images/icon-claude-code-plus-16x16.svg"><span purpose="claude-code-placeholder-bar"></span></div>
            <div purpose="claude-code-nav-item"><img alt="" src="/images/icon-claude-code-search-16x16.svg"><span purpose="claude-code-placeholder-bar"></span></div>
            <div purpose="claude-code-nav-item"><img alt="" src="/images/icon-claude-code-artifacts-16x16.svg"><span purpose="claude-code-placeholder-bar"></span></div>
            <div purpose="claude-code-nav-item"><img alt="" src="/images/icon-claude-code-bell-16x16.svg"><span purpose="claude-code-placeholder-bar"></span></div>
            <div purpose="claude-code-nav-item"><img alt="" src="/images/icon-claude-code-briefcase-16x16.svg"><span purpose="claude-code-placeholder-bar"></span></div>
            <div purpose="claude-code-nav-item"><img alt="" src="/images/icon-claude-code-more-14x14.svg"><span purpose="claude-code-placeholder-bar"></span></div>
          </div>
          <div purpose="claude-code-project-header">
            <span purpose="claude-code-project-name"><span>fleet-gitops</span><img alt="" src="/images/icon-claude-code-caret-10x10.svg"></span>
            <img alt="" src="/images/icon-claude-code-plus-16x16.svg">
          </div>
          <div purpose="claude-code-sessions">
            <div purpose="claude-code-session" class="selected"><span purpose="claude-code-session-name">Engineering Macs MCP audit</span><img purpose="claude-code-session-icon" alt="" src="/images/icon-claude-code-pull-request-16x16.svg"></div>
            <div purpose="claude-code-session"><span purpose="claude-code-placeholder-bar"></span><img purpose="claude-code-session-icon" alt="" src="/images/icon-claude-code-branch-14x14.svg"></div>
            <div purpose="claude-code-session"><span purpose="claude-code-placeholder-bar"></span><img purpose="claude-code-session-icon" alt="" src="/images/icon-claude-code-branch-14x14.svg"></div>
            <div purpose="claude-code-session"><span purpose="claude-code-placeholder-bar"></span><img purpose="claude-code-session-icon" alt="" src="/images/icon-claude-code-pull-request-16x16.svg"></div>
          </div>
          <div purpose="claude-code-account">
            <span purpose="claude-code-avatar"><img alt="" src="/images/avatar-claude-code-mike-24x29@2x.png"></span>
            <span purpose="claude-code-placeholder-bar"></span>
            <img alt="" src="/images/icon-claude-code-bug-15x15.svg">
          </div>
        </div>
        <div purpose="claude-code-main">
          <div purpose="claude-code-header">
            <span purpose="claude-code-session-title"><span>Engineering Macs MCP audit</span><img alt="" src="/images/icon-claude-code-chevron-down-14x14.svg"></span>
            <span purpose="claude-code-panel-toggle"><img alt="" src="/images/icon-claude-code-panel-list-15x15.svg"></span>
            <img alt="" src="/images/icon-claude-code-terminal-18x18.svg">
            <img alt="" src="/images/icon-claude-code-diff-16x16.svg">
            <img alt="" src="/images/icon-claude-code-globe-18x18.svg">
            <img alt="" src="/images/icon-claude-code-folder-18x18.svg">
            <img alt="" src="/images/icon-claude-code-more-18x18.svg">
          </div>
          <div purpose="claude-code-conversation" ref="demoConversation">
            <div purpose="claude-code-thread" ref="demoThread" :class="{resetting: demoResetting, 'fading-out': demoFadingOut}" :style="{transform: 'translateY(-' + demoScrollOffset + 'px)'}">
              <div purpose="claude-code-user-message"><p>Which Macs in Engineering have an MCP server with shell access, and are any of them behind on macOS?</p></div>
              <p v-show="demoStep >= 2">I’ll check with Fleet.</p>
              <div purpose="claude-code-tool-calls" v-show="demoStep >= 3">
                <div purpose="claude-code-tool-row" v-show="demoStep >= 3">
                  <img alt="" src="/images/icon-claude-code-tool-16x16.svg">
                  <span purpose="claude-code-tool-label">Fleet · Run live query</span>
                  <code>mcp_servers WHERE shell_access = 1</code>
                  <span purpose="claude-code-tool-meta">· 1,120 hosts</span>
                </div>
                <div purpose="claude-code-tool-row" v-show="demoStep >= 4">
                  <img alt="" src="/images/icon-claude-code-tool-16x16.svg">
                  <span purpose="claude-code-tool-label">Fleet · List hosts</span>
                  <code>team:Engineering os_version&lt;26</code>
                  <span purpose="claude-code-tool-meta">· 5 results</span>
                </div>
              </div>
              <p v-show="demoStep >= 5"><strong>14 of 1,120</strong> Engineering Macs have an MCP server with shell access. <strong>5</strong> of those are also behind on macOS, still on 15.7:</p>
              <div purpose="claude-code-table" v-show="demoStep >= 6">
                <div purpose="claude-code-table-header"><span>Host</span><span>macOS</span><span>MCP server</span></div>
                <div purpose="claude-code-table-row"><span>eng-mbp-0318</span><span>15.7</span><span>desktop-commander</span></div>
                <div purpose="claude-code-table-row"><span>eng-mbp-0442</span><span>15.7</span><span>iterm-mcp</span></div>
                <div purpose="claude-code-table-row"><span>+ 3 more</span><span></span><span></span></div>
              </div>
              <p v-show="demoStep >= 7">Want me to propose a fix?</p>
              <div purpose="claude-code-user-message" v-show="demoStep >= 8"><p>Yes</p></div>
              <div purpose="claude-code-tool-calls" v-show="demoStep >= 9">
                <div purpose="claude-code-tool-row" v-show="demoStep >= 9">
                  <img alt="" src="/images/icon-claude-code-edit-16x16.svg">
                  <span purpose="claude-code-tool-label">Write</span>
                  <code>lib/policies/mcp-shell-access.yml</code>
                  <span purpose="claude-code-tool-added">+18</span>
                </div>
                <div purpose="claude-code-tool-row" v-show="demoStep >= 10">
                  <img alt="" src="/images/icon-claude-code-edit-16x16.svg">
                  <span purpose="claude-code-tool-label">Edit</span>
                  <code>teams/engineering.yml</code>
                  <span purpose="claude-code-tool-added">+6</span>
                </div>
              </div>
              <div purpose="claude-code-pull-request" v-show="demoStep >= 11">
                <img alt="" src="/images/icon-claude-code-pull-request-16x16.svg">
                <div>
                  <span purpose="claude-code-pull-request-title">Pull request opened</span>
                  <span purpose="claude-code-pull-request-branch">mcp-shell-access → main</span>
                </div>
              </div>
              <p v-show="demoStep >= 12">Done. I’ve proposed a policy that flags MCP shell access and a macOS update deadline for those 5 Macs, <span purpose="claude-code-link">for your review</span>.</p>
            </div>
          </div>
          <div purpose="claude-code-composer">
            <div purpose="claude-code-input">
              <img alt="" src="/images/icon-claude-code-plus-18x18.svg">
              <span purpose="claude-code-input-text" v-if="demoComposerText">{{demoComposerText}}<span purpose="ai-demo-caret"></span></span>
              <span purpose="claude-code-placeholder" v-else>Type / for commands</span>
              <img alt="" src="/images/icon-claude-code-mic-18x18.svg">
            </div>
            <div purpose="claude-code-session-settings">
              <span purpose="claude-code-permission-mode">Auto</span>
              <span>Opus 5.5 <span purpose="claude-code-effort">Medium</span></span>
              <img alt="" src="/images/icon-claude-code-spinner-14x14.svg">
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
  `,

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  mounted: async function() {
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
      this.demoReducedMotion = true;
      this.demoStep = 99;
      return;
    }
    window.addEventListener('resize', this.updateDemoScrollOffset);
    this.runAiDemo();
  },
  beforeDestroy: function() {
    window.removeEventListener('resize', this.updateDemoScrollOffset);
    // Stops a running animation.
    this.demoRunId++;
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {

    clickSelectAiTool: async function(tool) {
      if (tool === this.selectedAiTool) {
        return;
      }
      let previousTool = this.selectedAiTool;
      this.selectedAiTool = tool;
      if (this.demoReducedMotion) {
        return;
      }
      this.demoRunId++;
      if (this.demoFadingOut || this.demoEntryMs === 0) {
        await this.resetAiDemo();
        this.runAiDemo();
        return;
      }
      // Carry the position over by how far it is to (or past) the admin's "Yes",
      // since the tools take a different number of steps on either side of it.
      let elapsedMs = this.demoEntryStartMs + Math.min(Date.now() - this.demoEntryStartedAt, this.demoEntryMs);
      let [previousBeforeYesMs, previousTotalMs] = [0, 0];
      let [beforeYesMs, totalMs] = [0, 0];
      for (let [timelineTool, isPrevious] of [[previousTool, true], [tool, false]]) {
        let runningMs = 0;
        let yesMs = 0;
        for (let [stepOrText, msToWait] of this.demoTimelines[timelineTool]) {
          if (stepOrText === 'Yes') {
            yesMs = runningMs;
          }
          runningMs += (typeof stepOrText === 'string' ? stepOrText.length * 120 : 0) + msToWait;
        }
        if (isPrevious) {
          [previousBeforeYesMs, previousTotalMs] = [yesMs, runningMs];
        } else {
          [beforeYesMs, totalMs] = [yesMs, runningMs];
        }
      }
      let startAtMs;
      if (elapsedMs < previousBeforeYesMs) {
        startAtMs = elapsedMs / previousBeforeYesMs * beforeYesMs;
      } else {
        startAtMs = beforeYesMs + (elapsedMs - previousBeforeYesMs) / (previousTotalMs - previousBeforeYesMs) * (totalMs - beforeYesMs);
      }
      this.runAiDemo(Math.min(startAtMs, totalMs - 1));
    },

    runAiDemo: async function(startAtMs = 0) {
      let runId = this.demoRunId;
      // Resolves to false once another run has replaced this one.
      let wait = async (ms) => {
        await new Promise((resolve) => setTimeout(resolve, ms));
        return runId === this.demoRunId;
      };
      // When picking up partway through, jump straight to that point with
      // transitions off, so switching tools only swaps the UI.
      let isJumping = startAtMs > 0;
      if (isJumping) {
        this.demoResetting = true;
        this.demoStep = 0;
        this.demoComposerText = '';
      }
      let entryStartMs = 0;
      for (let [stepOrText, msToWait] of this.demoTimelines[this.selectedAiTool]) {
        let typingMs = typeof stepOrText === 'string' ? stepOrText.length * 120 : 0;
        let entryMs = typingMs + msToWait;
        let msIntoEntry = Math.max(0, startAtMs - entryStartMs);
        if (msIntoEntry >= entryMs) {
          if (typeof stepOrText === 'string') {
            this.demoComposerText = stepOrText;
          } else if (stepOrText !== null) {
            this.demoComposerText = '';
            this.demoStep = stepOrText;
          }
          entryStartMs += entryMs;
          continue;
        }
        if (isJumping) {
          isJumping = false;
          await this.$nextTick();
          this.updateDemoScrollOffset();
          if (!await wait(50)) {
            return;
          }
          this.demoResetting = false;
        }
        // Typing that was cut off by the switch starts over.
        if (msIntoEntry < typingMs) {
          msIntoEntry = 0;
        }
        this.demoEntryStartMs = entryStartMs;
        this.demoEntryMs = entryMs;
        this.demoEntryStartedAt = Date.now() - msIntoEntry;
        if (typeof stepOrText === 'string') {
          if (msIntoEntry === 0) {
            for (let i = 1; i <= stepOrText.length; i++) {
              this.demoComposerText = stepOrText.slice(0, i);
              if (!await wait(120)) {
                return;
              }
            }
          } else {
            this.demoComposerText = stepOrText;
          }
        } else if (stepOrText !== null) {
          // Sending a message clears the composer, like pressing return.
          this.demoComposerText = '';
          this.demoStep = stepOrText;
          await this.$nextTick();
          this.updateDemoScrollOffset();
        }
        if (!await wait(msToWait - Math.max(0, msIntoEntry - typingMs))) {
          return;
        }
        entryStartMs += entryMs;
      }
      this.demoFadingOut = true;
      if (!await wait(600)) {
        return;
      }
      await this.resetAiDemo();
      if (runId === this.demoRunId) {
        this.runAiDemo();
      }
    },

    resetAiDemo: async function() {
      this.demoResetting = true;
      this.demoEntryMs = 0;
      this.demoStep = 0;
      this.demoComposerText = '';
      this.demoScrollOffset = 0;
      await this.$nextTick();
      // Let the browser apply the reset before transitions come back on, then
      // fade the thread back in. It starts with the first message already sent.
      await new Promise((resolve) => setTimeout(resolve, 50));
      this.demoResetting = false;
      this.demoFadingOut = false;
    },

    updateDemoScrollOffset: function() {
      let conversation = this.$refs.demoConversation;
      let thread = this.$refs.demoThread;
      if (!conversation || !thread || this.demoReducedMotion) {
        return;
      }
      // 1em is one Figma pixel; keep a 14px gap below the newest message.
      let bottomGap = 14 * parseFloat(window.getComputedStyle(conversation).fontSize);
      this.demoScrollOffset = Math.max(0, thread.offsetTop + thread.offsetHeight + bottomGap - conversation.clientHeight);
    },

  }
});
