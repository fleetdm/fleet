/**
 * <ai-tool-graphic>
 * -----------------------------------------------------------------------------
 * Animated mockups of Slack, Microsoft Teams, Claude Code, Cursor, GitHub
 * Copilot, and Codex working with Fleet, with buttons to switch between them.
 *
 * @type {Component}
 * -----------------------------------------------------------------------------
 */

parasails.registerComponent('aiToolGraphic', {
  //  ╔═╗╦═╗╔═╗╔═╗╔═╗
  //  ╠═╝╠╦╝║ ║╠═╝╚═╗
  //  ╩  ╩╚═╚═╝╩  ╚═╝
  props: [
    'hideToolSwitcher',// « Shows only the Slack demo, without buttons to switch tools.
    'hideTeams',// « Leaves out the Microsoft Teams demo (e.g. on Apple-focused pages).
  ],

  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: function() {
    return {
      selectedAiTool: 'slack',
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
        slack: [
          [null, 1200],
          [2, 3000],
          ['Yes', 400],
          [3, 1600],
          [4, 1500],
        ],
        teams: [
          [null, 800],
          [2, 1800],
          [3, 1000],
          [4, 1500],
          ['Yes', 400],
          [5, 600],
          [6, 1800],
          [7, 1500],
        ],
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
          [11, 1500],
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
          [13, 1500],
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
          [11, 1500],
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
          [12, 1500],
        ],
      },
      // The shared ending after every tool: the pull request, its review and merge, then Fleet.
      demoEnding: '',
      demoPrStep: 0,
      demoReviewPanelOpen: false,
      demoApproved: false,
      demoMerged: false,
      demoGitops: '',
      demoCueVisible: false,
      demoFleetProgress: 0,
      demoPointer: {visible: false, pressed: false, x: 0, y: 0, moveMs: 0},
      // Which tab the pull request shows: 'conversation' or 'files'.
      demoPrView: 'conversation',
      // How far the Conversation tab has scrolled, to keep the merge box in view.
      demoPrScroll: 0,
      demoPrScrollInstant: false,
      // Set briefly after an ending, so the tool pops back in when the demo loops.
      demoPoppingIn: false,
      // Split diff rows: [kind, old line, old text, new line, new text], or ['hunk', header].
      // Long paths have a zero-width space after the folder so they wrap there.
      demoPrDiffs: {
        macos: [
          ['hunk', '@@ -81,3 +81,6 @@ controls:'],
          ['ctx', '81', '  apple_settings:', '81', '  apple_settings:'],
          ['ctx', '82', '    configuration_profiles:', '82', '    configuration_profiles:'],
          ['ctx', '83', '      - path: ../lib/macos/configuration-profiles/\u200bfirewall.mobileconfig', '83', '      - path: ../lib/macos/configuration-profiles/\u200bfirewall.mobileconfig'],
          ['add', '', '', '84', '      - path: ../lib/macos/configuration-profiles/\u200brestrictions.mobileconfig'],
          ['add', '', '', '85', '      - path: ../lib/macos/configuration-profiles/\u200bdiagnostics.mobileconfig'],
          ['add', '', '', '86', '      - path: ../lib/macos/configuration-profiles/\u200bsharing.mobileconfig'],
          ['hunk', '@@ -183,2 +186,3 @@ policies:'],
          ['ctx', '183', 'policies:', '186', 'policies:'],
          ['ctx', '184', '  - path: ../lib/macos/policies/latest-macos.yml', '187', '  - path: ../lib/macos/policies/latest-macos.yml'],
          ['add', '', '', '188', '  - path: ../lib/macos/policies/cis-level-2.yml'],
        ],
        windows: [
          ['hunk', '@@ -81,3 +81,6 @@ controls:'],
          ['ctx', '81', '  windows_settings:', '81', '  windows_settings:'],
          ['ctx', '82', '    configuration_profiles:', '82', '    configuration_profiles:'],
          ['ctx', '83', '      - path: ../lib/windows/configuration-profiles/\u200bwindows-firewall.xml', '83', '      - path: ../lib/windows/configuration-profiles/\u200bwindows-firewall.xml'],
          ['add', '', '', '84', '      - path: ../lib/windows/configuration-profiles/\u200bcredential-guard.xml'],
          ['add', '', '', '85', '      - path: ../lib/windows/configuration-profiles/\u200bdiagnostic-data.xml'],
          ['add', '', '', '86', '      - path: ../lib/windows/configuration-profiles/\u200bdisable-services.xml'],
          ['hunk', '@@ -183,2 +186,3 @@ policies:'],
          ['ctx', '183', 'policies:', '186', 'policies:'],
          ['ctx', '184', '  - path: ../lib/windows/policies/latest-windows.yml', '187', '  - path: ../lib/windows/policies/latest-windows.yml'],
          ['add', '', '', '188', '  - path: ../lib/windows/policies/cis-level-2.yml'],
        ],
      },
      // [policy name, passing hosts before the change, passing hosts after], out of 2,350 hosts.
      demoFleetPolicies: {
        macos: [
          ['CIS - Ensure iCloud Drive Document and Desktop Sync Is Disabled', 944, 2346],
          ['CIS - Ensure Sending Diagnostic and Usage Data to Apple Is Disabled', 319, 2341],
          ['CIS - Ensure Limit Ad Tracking Is Enabled', 362, 2344],
          ['CIS - Ensure Remote Login Is Disabled', 478, 2348],
          ['CIS - Ensure Screen Sharing Is Disabled', 710, 2350],
          ['CIS - Ensure File Sharing Is Disabled', 839, 2347],
          ['CIS - Ensure Bluetooth Sharing Is Disabled', 1060, 2349],
          ['CIS - Ensure Media Sharing Is Disabled', 595, 2345],
          ['CIS - Ensure AirDrop Is Disabled', 1146, 2342],
          ['CIS - Ensure Siri Is Disabled', 1370, 2348],
          ['CIS - Ensure Share iCloud Analytics Is Disabled', 420, 2350],
          ['CIS - Ensure Improve Assistive Voice Features Is Disabled', 1246, 2347],
        ],
        windows: [
          ['CIS - Ensure Credential Guard Is Enabled', 830, 2343],
          ['CIS - Ensure Virtualization Based Security Is Enabled', 870, 2345],
          ['CIS - Ensure Allow Diagnostic Data Is Set to Off', 319, 2341],
          ['CIS - Ensure Remote Registry Is Disabled', 478, 2348],
          ['CIS - Ensure Remote Desktop Services Is Disabled', 700, 2350],
          ['CIS - Ensure Print Spooler Is Disabled', 640, 2346],
          ['CIS - Ensure Windows Error Reporting Is Disabled', 960, 2349],
          ['CIS - Ensure Advertising ID Is Disabled', 1110, 2347],
          ['CIS - Ensure Cortana Is Disabled', 1170, 2350],
          ['CIS - Ensure Consumer Experiences Are Turned Off', 1045, 2344],
          ['CIS - Ensure SMBv1 Client Driver Is Disabled', 1390, 2348],
          ['CIS - Ensure Xbox Accessory Management Service Is Disabled', 630, 2350],
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
    <div purpose="ai-tool-switcher" role="group" aria-label="AI tool" v-if="!hideToolSwitcher">
      <button type="button" purpose="ai-tool-button" :class="{selected: selectedAiTool === 'slack'}" :aria-pressed="selectedAiTool === 'slack' ? 'true' : 'false'" @click="clickSelectAiTool('slack')"><img alt="" src="/images/ai-demo/icon-slack-logo-16x16.svg">Slack</button>
      <button type="button" purpose="ai-tool-button" :class="{selected: selectedAiTool === 'teams'}" :aria-pressed="selectedAiTool === 'teams' ? 'true' : 'false'" @click="clickSelectAiTool('teams')" v-if="!hideTeams"><img alt="" src="/images/ai-demo/icon-microsoft-teams-logo-16x16.svg">Teams</button>
      <button type="button" purpose="ai-tool-button" :class="{selected: selectedAiTool === 'github-copilot'}" :aria-pressed="selectedAiTool === 'github-copilot' ? 'true' : 'false'" @click="clickSelectAiTool('github-copilot')"><img alt="" src="/images/ai-demo/icon-microsoft-copilot-logo-16x16@2x.png">Copilot</button>
      <button type="button" purpose="ai-tool-button" :class="{selected: selectedAiTool === 'claude'}" :aria-pressed="selectedAiTool === 'claude' ? 'true' : 'false'" @click="clickSelectAiTool('claude')"><img alt="" src="/images/ai-demo/icon-claude-logo-16x16.svg">Claude</button>
      <button type="button" purpose="ai-tool-button" :class="{selected: selectedAiTool === 'cursor'}" :aria-pressed="selectedAiTool === 'cursor' ? 'true' : 'false'" @click="clickSelectAiTool('cursor')"><img alt="" src="/images/ai-demo/icon-cursor-logo-16x16.svg">Cursor</button>
      <button type="button" purpose="ai-tool-button" :class="{selected: selectedAiTool === 'codex'}" :aria-pressed="selectedAiTool === 'codex' ? 'true' : 'false'" @click="clickSelectAiTool('codex')"><img alt="" src="/images/ai-demo/icon-codex-logo-16x16.svg">Codex</button>
    </div>
    <div purpose="ai-demo-stage" ref="demoStage">
      <div purpose="cursor-animation" role="img" aria-label="A Cursor agent connected to Fleet. An admin asks what it takes to meet CIS Level 2 on their Macs. Cursor checks with Fleet and reports that 2,184 of 2,350 Macs pass Level 1, that Level 2 adds 38 settings, and that the biggest gaps are iCloud Drive sync, telemetry limits, and unneeded services, then offers to propose changes to close those gaps. The admin says yes, and Cursor writes the changes and opens a pull request that proposes 3 new profiles and 38 policies for their review." :class="{'reduced-motion': demoReducedMotion, 'popped-out': demoEnding, 'popping-in': demoPoppingIn}" v-if="selectedAiTool === 'cursor'">
        <div purpose="cursor-window" aria-hidden="true">
          <div purpose="cursor-sidebar">
            <div purpose="cursor-window-controls">
              <img alt="" src="/images/ai-demo/icon-cursor-panel-left-15x15.svg">
              <img alt="" src="/images/ai-demo/icon-cursor-search-15x15.svg">
              <img purpose="cursor-back" alt="" src="/images/ai-demo/icon-cursor-arrow-left-15x15.svg">
              <img alt="" src="/images/ai-demo/icon-cursor-arrow-right-15x15.svg">
            </div>
            <div purpose="cursor-sidebar-item"><img alt="" src="/images/ai-demo/icon-cursor-new-chat-13x13.svg"><span purpose="cursor-placeholder-bar"></span></div>
            <div purpose="cursor-sidebar-item"><img alt="" src="/images/ai-demo/icon-cursor-bot-14x14.svg"><span purpose="cursor-placeholder-bar"></span></div>
            <div purpose="cursor-sidebar-item"><img alt="" src="/images/ai-demo/icon-cursor-blocks-14x14.svg"><span purpose="cursor-placeholder-bar"></span></div>
            <div purpose="cursor-sidebar-label">
              <span>Repositories</span>
              <img alt="" src="/images/ai-demo/icon-cursor-list-filter-14x14.svg">
              <img alt="" src="/images/ai-demo/icon-cursor-folder-plus-14x14.svg">
            </div>
            <div purpose="cursor-sidebar-item"><img alt="" src="/images/ai-demo/icon-cursor-folder-14x14.svg"><span purpose="cursor-repo-name">fleet-gitops</span></div>
            <div purpose="cursor-sidebar-item" class="selected"><img alt="" src="/images/ai-demo/icon-cursor-dot-14x14.svg"><span purpose="cursor-session-name">CIS Level 2 for Macs</span></div>
            <div purpose="cursor-sidebar-item"><img alt="" src="/images/ai-demo/icon-cursor-dot-14x14.svg"><span purpose="cursor-placeholder-bar"></span></div>
            <div purpose="cursor-sidebar-item"><img alt="" src="/images/ai-demo/icon-cursor-dot-14x14.svg"><span purpose="cursor-placeholder-bar"></span></div>
            <div purpose="cursor-sidebar-item"><img alt="" src="/images/ai-demo/icon-cursor-dot-14x14.svg"><span purpose="cursor-placeholder-bar"></span></div>
            <div purpose="cursor-account">
              <span purpose="cursor-avatar"><img alt="" src="/images/ai-demo/avatar-mike-24x29@2x.png"></span>
              <span purpose="cursor-placeholder-bar"></span>
              <img purpose="cursor-settings" alt="" src="/images/ai-demo/icon-cursor-settings-15x15.svg">
            </div>
          </div>
          <div purpose="cursor-content">
            <div purpose="cursor-header">
              <span purpose="cursor-tab">CIS Level 2 for Macs</span>
              <img purpose="cursor-ellipsis" alt="" src="/images/ai-demo/icon-cursor-ellipsis-16x16.svg">
              <img alt="" src="/images/ai-demo/icon-cursor-panel-right-15x15.svg">
            </div>
            <div purpose="cursor-body">
              <div purpose="cursor-main">
                <div purpose="cursor-conversation" ref="demoConversation">
                  <div purpose="cursor-thread" ref="demoThread" :class="{resetting: demoResetting, 'fading-out': demoFadingOut}" :style="{transform: 'translateY(-' + demoScrollOffset + 'px)'}">
                    <div purpose="cursor-card"><p>What do we need for CIS Level 2 on our Macs?</p><img alt="" src="/images/ai-demo/icon-cursor-undo-14x14.svg"></div>
                    <div purpose="cursor-steps" v-show="demoStep >= 2">
                      <p purpose="cursor-step"><span>Called</span>&nbsp; fleet.get_policy_results · 2,350 hosts</p>
                      <p purpose="cursor-step" v-show="demoStep >= 3"><span>Called</span>&nbsp; fleet.get_cis_benchmark · 38 settings</p>
                    </div>
                    <p v-show="demoStep >= 4"><strong>2,184 of 2,350</strong> Macs pass Level 1. Level 2 adds <strong>38</strong> settings. The biggest gaps are:</p>
                    <div purpose="cursor-table" v-show="demoStep >= 5">
                      <div purpose="cursor-table-header"><span>Setting</span><span>Failing</span><span>Fix</span></div>
                      <div purpose="cursor-table-row"><span>iCloud Drive sync</span><span>1,406</span><span>Restrictions profile</span></div>
                      <div purpose="cursor-table-row"><span>Telemetry limits</span><span>2,031</span><span>Diagnostics profile</span></div>
                      <div purpose="cursor-table-row"><span>Unneeded services</span><span>1,872</span><span>Sharing profile</span></div>
                    </div>
                    <p v-show="demoStep >= 6">Want me to propose changes that close those gaps?</p>
                    <div purpose="cursor-card" v-show="demoStep >= 7"><p>Yes</p><img alt="" src="/images/ai-demo/icon-cursor-undo-14x14.svg"></div>
                    <div purpose="cursor-file-card" v-show="demoStep >= 8">
                      <img alt="" src="/images/ai-demo/icon-cursor-file-diff-15x15.svg">
                      <span purpose="cursor-file-path">4 files · 3 profiles, 1 policy file</span>
                      <span purpose="cursor-added">+507</span>
                      <span purpose="cursor-removed">-0</span>
                    </div>
                    <div purpose="cursor-file-card" v-show="demoStep >= 9">
                      <img alt="" src="/images/ai-demo/icon-cursor-file-diff-15x15.svg">
                      <span purpose="cursor-file-path">fleets/workstations.yml</span>
                      <span purpose="cursor-added">+4</span>
                      <span purpose="cursor-removed">-0</span>
                    </div>
                    <p purpose="cursor-step" v-show="demoStep >= 10"><span>Ran</span>&nbsp; gh pr create --base main</p>
                    <p v-show="demoStep >= 11">Done. I’ve proposed 3 new profiles and 38 policies <span purpose="cursor-link">for your review</span>.</p>
                    <div purpose="cursor-feedback" v-show="demoStep >= 11">
                      <img alt="" src="/images/ai-demo/icon-cursor-thumbs-up-13x13.svg">
                      <img alt="" src="/images/ai-demo/icon-cursor-thumbs-down-13x13.svg">
                      <img alt="" src="/images/ai-demo/icon-cursor-copy-13x13.svg">
                      <img alt="" src="/images/ai-demo/icon-cursor-git-fork-13x13.svg">
                      <span>Just now</span>
                    </div>
                  </div>
                </div>
                <div purpose="cursor-composer">
                  <div purpose="cursor-input">
                    <span purpose="cursor-add"><img alt="" src="/images/ai-demo/icon-cursor-plus-14x14.svg"></span>
                    <span purpose="cursor-input-text" v-if="demoComposerText">{{demoComposerText}}<span purpose="ai-demo-caret"></span></span>
                    <span purpose="cursor-placeholder" v-else>Send follow-up</span>
                    <span purpose="cursor-model">Cursor Grok 4.6 Medium</span>
                    <img alt="" src="/images/ai-demo/icon-cursor-mic-15x15.svg">
                    <span purpose="cursor-send"><img alt="" src="/images/ai-demo/icon-cursor-audio-lines-14x14.svg"></span>
                  </div>
                  <div purpose="cursor-environment">
                    <img alt="" src="/images/ai-demo/icon-cursor-laptop-13x13.svg">
                    <span>This Mac</span>
                    <img alt="" src="/images/ai-demo/icon-cursor-chevron-down-12x12.svg">
                    <img purpose="cursor-context" alt="" src="/images/ai-demo/icon-cursor-context-12x12.svg">
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div purpose="copilot-animation" role="img" aria-label="A GitHub Copilot session connected to Fleet. An admin asks what it takes to meet CIS Level 2 on their Macs. Copilot checks with Fleet and reports that 2,184 of 2,350 Macs pass Level 1, that Level 2 adds 38 settings, and that the biggest gaps are iCloud Drive sync, telemetry limits, and unneeded services, then offers to propose changes to close those gaps. The admin says yes, and Copilot commits the changes and opens a pull request that proposes 3 new profiles and 38 policies for their review." :class="{'reduced-motion': demoReducedMotion, 'popped-out': demoEnding, 'popping-in': demoPoppingIn}" v-else-if="selectedAiTool === 'github-copilot'">
        <div purpose="copilot-window" aria-hidden="true">
          <div purpose="copilot-sidebar">
            <div purpose="copilot-title-bar">
              <img alt="" src="/images/ai-demo/icon-github-copilot-panel-left-16x16.svg">
              <img purpose="copilot-back" alt="" src="/images/ai-demo/icon-github-copilot-chevron-left-16x16.svg">
              <img alt="" src="/images/ai-demo/icon-github-copilot-chevron-right-16x16.svg">
            </div>
            <div purpose="copilot-nav-item"><img alt="" src="/images/ai-demo/icon-github-copilot-house-16x16.svg"><span purpose="copilot-placeholder-bar"></span></div>
            <div purpose="copilot-nav-item"><img alt="" src="/images/ai-demo/icon-github-copilot-list-todo-16x16.svg"><span purpose="copilot-placeholder-bar"></span></div>
            <div purpose="copilot-nav-item"><img alt="" src="/images/ai-demo/icon-github-copilot-calendar-clock-16x16.svg"><span purpose="copilot-placeholder-bar"></span></div>
            <div purpose="copilot-nav-item"><img alt="" src="/images/ai-demo/icon-github-copilot-blocks-16x16.svg"><span purpose="copilot-placeholder-bar"></span></div>
            <div purpose="copilot-sessions-header">
              <span>Sessions</span>
              <img alt="" src="/images/ai-demo/icon-github-copilot-list-filter-16x16.svg">
              <img alt="" src="/images/ai-demo/icon-github-copilot-plus-outline-16x16.svg">
            </div>
            <div purpose="copilot-nav-item"><img alt="" src="/images/ai-demo/icon-github-copilot-message-square-16x16.svg"><span purpose="copilot-placeholder-bar"></span></div>
            <div purpose="copilot-nav-item"><img alt="" src="/images/ai-demo/icon-github-copilot-folder-16x16.svg"><span purpose="copilot-repo-name">fleet-gitops</span></div>
            <div purpose="copilot-nav-item" class="nested selected"><span purpose="copilot-session-name">CIS Level 2 for Macs</span></div>
            <div purpose="copilot-nav-item" class="nested"><span purpose="copilot-placeholder-bar"></span></div>
            <div purpose="copilot-nav-item" class="nested"><span purpose="copilot-placeholder-bar"></span></div>
            <div purpose="copilot-nav-item" class="nested"><span purpose="copilot-placeholder-bar"></span></div>
            <div purpose="copilot-nav-item" class="nested"><span purpose="copilot-placeholder-bar"></span></div>
            <div purpose="copilot-account">
              <span purpose="copilot-avatar"><img alt="" src="/images/ai-demo/avatar-mike-24x29@2x.png"></span>
            </div>
          </div>
          <div purpose="copilot-main">
            <div purpose="copilot-header">
              <span purpose="copilot-session-title">CIS Level 2 for Macs</span>
              <span purpose="copilot-run-button"><span><img alt="" src="/images/ai-demo/icon-github-copilot-play-14x14.svg"></span><span><img alt="" src="/images/ai-demo/icon-github-copilot-chevron-down-14x14.svg"></span></span>
              <img purpose="copilot-panel-icon" alt="" src="/images/ai-demo/icon-github-copilot-panel-right-18x18.svg">
            </div>
            <div purpose="copilot-conversation" ref="demoConversation">
              <div purpose="copilot-thread" ref="demoThread" :class="{resetting: demoResetting, 'fading-out': demoFadingOut}" :style="{transform: 'translateY(-' + demoScrollOffset + 'px)'}">
                <div purpose="copilot-bubble"><p>What do we need for CIS Level 2 on our Macs?</p></div>
                <div purpose="copilot-activity-group" v-show="demoStep >= 2">
                  <div purpose="copilot-activity"><img alt="" src="/images/ai-demo/icon-github-copilot-terminal-16x16.svg"><span>Get policy results</span><code>CIS Level 1</code><span purpose="copilot-activity-meta">· 2,350 hosts</span></div>
                  <div purpose="copilot-activity" v-show="demoStep >= 3"><img alt="" src="/images/ai-demo/icon-github-copilot-terminal-16x16.svg"><span>Compare benchmark</span><code>CIS Level 2</code><span purpose="copilot-activity-meta">· 38 settings</span></div>
                </div>
                <p v-show="demoStep >= 4"><strong>2,184 of 2,350</strong> Macs pass Level 1. Level 2 adds <strong>38</strong> settings. The biggest gaps are:</p>
                <div purpose="copilot-table" v-show="demoStep >= 5">
                  <div purpose="copilot-table-header"><span>Setting</span><span>Failing</span><span>Fix</span></div>
                  <div purpose="copilot-table-row"><span>iCloud Drive sync</span><span>1,406</span><span>Restrictions profile</span></div>
                  <div purpose="copilot-table-row"><span>Telemetry limits</span><span>2,031</span><span>Diagnostics profile</span></div>
                  <div purpose="copilot-table-row"><span>Unneeded services</span><span>1,872</span><span>Sharing profile</span></div>
                </div>
                <p v-show="demoStep >= 6">Want me to propose changes that close those gaps?</p>
                <div purpose="copilot-bubble" v-show="demoStep >= 7"><p>Yes</p></div>
                <div purpose="copilot-activity-group" v-show="demoStep >= 8">
                  <div purpose="copilot-activity"><img alt="" src="/images/ai-demo/icon-github-copilot-file-diff-16x16.svg"><span>Create</span><span>4 files · 3 profiles, 1 policy file</span><span purpose="copilot-added">+507</span><span purpose="copilot-removed">-0</span></div>
                  <div purpose="copilot-activity" v-show="demoStep >= 9"><img alt="" src="/images/ai-demo/icon-github-copilot-file-diff-16x16.svg"><span>Edit</span><span>fleets/workstations.yml</span><span purpose="copilot-added">+4</span><span purpose="copilot-removed">-0</span></div>
                  <div purpose="copilot-activity" v-show="demoStep >= 10"><img alt="" src="/images/ai-demo/icon-github-copilot-terminal-16x16.svg"><span>Commit changes</span><code>git commit -m "Add CIS Level 2 profiles and policies"</code><span purpose="copilot-activity-meta">· 5 files</span></div>
                  <div purpose="copilot-activity" v-show="demoStep >= 11"><img alt="" src="/images/ai-demo/icon-github-copilot-terminal-16x16.svg"><span>Push branch</span><code>git push -u origin HEAD</code></div>
                </div>
                <div purpose="copilot-pull-request" v-show="demoStep >= 12">
                  <div purpose="copilot-pull-request-title"><img alt="" src="/images/ai-demo/icon-github-copilot-pull-request-green-16x16.svg"><span>Pull request created</span><span purpose="copilot-added">+511</span><span purpose="copilot-removed">-0</span></div>
                  <span purpose="copilot-pull-request-branches">cis-level-2 → origin/main</span>
                </div>
                <p v-show="demoStep >= 13">Done. I’ve proposed 3 new profiles and 38 policies <span purpose="copilot-link">for your review</span>.</p>
              </div>
            </div>
            <div purpose="copilot-composer">
              <div purpose="copilot-input">
                <span purpose="copilot-input-text" v-if="demoComposerText">{{demoComposerText}}<span purpose="ai-demo-caret"></span></span>
                <span purpose="copilot-placeholder" v-else>Ask anything. Use / for commands, @ files, &amp; sessions, # issues...</span>
                <div purpose="copilot-controls">
                  <img alt="" src="/images/ai-demo/icon-github-copilot-plus-16x16.svg">
                  <span purpose="copilot-picker">Interactive</span>
                  <span purpose="copilot-picker">Copilot Fusion · Standard</span>
                  <span purpose="copilot-send"><img alt="" src="/images/ai-demo/icon-github-copilot-arrow-up-16x16.svg"></span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div purpose="codex-animation" role="img" aria-label="A Codex session connected to Fleet. An admin asks what it takes to meet CIS Level 2 on their Macs. Codex checks with Fleet and reports that 2,184 of 2,350 Macs pass Level 1, that Level 2 adds 38 settings, and that the biggest gaps are iCloud Drive sync, telemetry limits, and unneeded services, then offers to propose changes to close those gaps. The admin says yes, and Codex edits the files and opens a pull request that proposes 3 new profiles and 38 policies for their review." :class="{'reduced-motion': demoReducedMotion, 'popped-out': demoEnding, 'popping-in': demoPoppingIn}" v-else-if="selectedAiTool === 'codex'">
        <div purpose="codex-window" aria-hidden="true">
          <div purpose="codex-title-bar">
            <div purpose="codex-window-controls">
              <img alt="" src="/images/ai-demo/icon-codex-arrow-left-15x15.svg">
              <img alt="" src="/images/ai-demo/icon-codex-arrow-right-disabled-15x15.svg">
              <img alt="" src="/images/ai-demo/icon-codex-panel-left-15x15.svg">
            </div>
            <span purpose="codex-window-title">CIS Level 2 for Macs</span>
            <div purpose="codex-window-actions">
              <img purpose="codex-ellipsis" alt="" src="/images/ai-demo/icon-codex-ellipsis-16x16.svg">
              <img alt="" src="/images/ai-demo/icon-codex-list-checks-15x15.svg">
              <span purpose="codex-divider"></span>
              <img alt="" src="/images/ai-demo/icon-codex-square-plus-15x15.svg">
            </div>
          </div>
          <div purpose="codex-body">
            <div purpose="codex-rail">
              <span purpose="codex-rail-item" class="selected"><img alt="" src="/images/ai-demo/icon-codex-home-16x16.svg"></span>
              <span purpose="codex-rail-item"><img alt="" src="/images/ai-demo/icon-codex-images-16x16.svg"></span>
              <span purpose="codex-rail-item"><img alt="" src="/images/ai-demo/icon-codex-clock-16x16.svg"></span>
              <span purpose="codex-rail-item"><img alt="" src="/images/ai-demo/icon-codex-library-16x16.svg"></span>
              <span purpose="codex-rail-item"><img alt="" src="/images/ai-demo/icon-codex-at-sign-16x16.svg"></span>
              <span purpose="codex-rail-item"><img alt="" src="/images/ai-demo/icon-codex-ellipsis-16x16.svg"></span>
              <span purpose="codex-rail-divider"></span>
              <span purpose="codex-rail-item"><img alt="" src="/images/ai-demo/icon-codex-pull-request-16x16.svg"></span>
              <span purpose="codex-avatar">MT</span>
            </div>
            <div purpose="codex-panel">
              <div purpose="codex-sidebar">
                <div purpose="codex-sidebar-header">
                  <span purpose="codex-sidebar-title"><span>Codex</span><img alt="" src="/images/ai-demo/icon-codex-chevron-down-14x14.svg"></span>
                  <img alt="" src="/images/ai-demo/icon-codex-bell-15x15.svg">
                  <img alt="" src="/images/ai-demo/icon-codex-search-15x15.svg">
                </div>
                <div purpose="codex-sidebar-item"><img alt="" src="/images/ai-demo/icon-codex-new-chat-15x15.svg"><span purpose="codex-placeholder-bar"></span></div>
                <span purpose="codex-sidebar-label">Projects</span>
                <div purpose="codex-sidebar-item"><img alt="" src="/images/ai-demo/icon-codex-folder-15x15.svg"><span>fleet-gitops</span></div>
                <span purpose="codex-sidebar-label">Recents</span>
                <div purpose="codex-sidebar-item" class="selected"><span purpose="codex-session-name">CIS Level 2 for Macs</span></div>
                <div purpose="codex-sidebar-item"><span purpose="codex-placeholder-bar" style="width: 99em;"></span></div>
                <div purpose="codex-sidebar-item"><span purpose="codex-placeholder-bar" style="width: 119em;"></span></div>
                <div purpose="codex-sidebar-item"><span purpose="codex-placeholder-bar" style="width: 86em;"></span></div>
                <div purpose="codex-sidebar-item"><span purpose="codex-placeholder-bar" style="width: 112em;"></span></div>
                <div purpose="codex-sidebar-item"><span purpose="codex-placeholder-bar" style="width: 94em;"></span></div>
              </div>
              <div purpose="codex-main">
                <div purpose="codex-conversation" ref="demoConversation">
                  <div purpose="codex-thread" ref="demoThread" :class="{resetting: demoResetting, 'fading-out': demoFadingOut}" :style="{transform: 'translateY(-' + demoScrollOffset + 'px)'}">
                    <div purpose="codex-bubble"><p>What do we need for CIS Level 2 on our Macs?</p></div>
                    <div purpose="codex-work" v-show="demoStep >= 2">
                      <div purpose="codex-worked-for"><span>Worked for 14s</span><img alt="" src="/images/ai-demo/icon-codex-chevron-right-14x14.svg"></div>
                      <p purpose="codex-call">Called <code>fleet.get_policy_results</code>&nbsp; · 2,350 hosts</p>
                      <p purpose="codex-call" v-show="demoStep >= 3">Called <code>fleet.get_cis_benchmark</code>&nbsp; · 38 settings</p>
                    </div>
                    <p v-show="demoStep >= 4"><strong>2,184 of 2,350</strong> Macs pass Level 1. Level 2 adds <strong>38</strong> settings. The biggest gaps are:</p>
                    <div purpose="codex-table" v-show="demoStep >= 5">
                      <div purpose="codex-table-header"><span>Setting</span><span>Failing</span><span>Fix</span></div>
                      <div purpose="codex-table-row"><span>iCloud Drive sync</span><span>1,406</span><span>Restrictions profile</span></div>
                      <div purpose="codex-table-row"><span>Telemetry limits</span><span>2,031</span><span>Diagnostics profile</span></div>
                      <div purpose="codex-table-row"><span>Unneeded services</span><span>1,872</span><span>Sharing profile</span></div>
                    </div>
                    <p v-show="demoStep >= 6">Want me to propose changes that close those gaps?</p>
                    <div purpose="codex-bubble" v-show="demoStep >= 7"><p>Yes</p></div>
                    <div purpose="codex-worked-for" v-show="demoStep >= 8"><span>Worked for 38s</span><img alt="" src="/images/ai-demo/icon-codex-chevron-right-14x14.svg"></div>
                    <div purpose="codex-edited-files" v-show="demoStep >= 9">
                      <div purpose="codex-edited-summary">
                        <span purpose="codex-edited-icon"><img alt="" src="/images/ai-demo/icon-codex-file-diff-16x16.svg"></span>
                        <div>
                          <span purpose="codex-edited-title">Edited 5 files</span>
                          <span purpose="codex-edited-stats"><span purpose="codex-added">+511</span> <span purpose="codex-removed">−0</span></span>
                        </div>
                        <span purpose="codex-review-button">Review</span>
                      </div>
                      <div purpose="codex-edited-file"><span>4 files · 3 profiles, 1 policy file</span><span><span purpose="codex-added">+507</span> <span purpose="codex-removed">−0</span></span></div>
                      <div purpose="codex-edited-file"><span><span purpose="codex-path">fleets/</span>workstations.yml</span><span><span purpose="codex-added">+4</span> <span purpose="codex-removed">−0</span></span></div>
                    </div>
                    <div purpose="codex-pull-request" v-show="demoStep >= 10"><img alt="" src="/images/ai-demo/icon-codex-pull-request-14x14.svg"><span>Opened pull request</span><code>cis-level-2 → main</code></div>
                    <p v-show="demoStep >= 11">Done. I’ve proposed 3 new profiles and 38 policies <span purpose="codex-link">for your review</span>.</p>
                  </div>
                </div>
                <div purpose="codex-composer">
                  <div purpose="codex-input">
                    <span purpose="codex-input-text" v-if="demoComposerText">{{demoComposerText}}<span purpose="ai-demo-caret"></span></span>
                    <span purpose="codex-placeholder" v-else>Work with Codex</span>
                    <div purpose="codex-controls">
                      <img purpose="codex-plus" alt="" src="/images/ai-demo/icon-codex-plus-17x17.svg">
                      <span purpose="codex-approval"><img alt="" src="/images/ai-demo/icon-codex-hand-14x14.svg"><span>Ask for approval</span></span>
                      <span purpose="codex-model"><span>6 Luna <span>Medium</span></span><img alt="" src="/images/ai-demo/icon-codex-chevron-down-12x12.svg"></span>
                      <img alt="" src="/images/ai-demo/icon-codex-mic-16x16.svg">
                      <span purpose="codex-send"><img alt="" src="/images/ai-demo/icon-codex-arrow-up-16x16.svg"></span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div purpose="slack-animation" role="img" aria-label="A Slack thread in the #ask-fleet channel. An admin asks what it takes to meet CIS Level 2 on their Macs. Fleet reports that 2,184 of 2,350 Macs pass Level 1, that Level 2 adds 38 settings, and that the biggest gaps are iCloud Drive sync, telemetry limits, and unneeded services, then offers to propose changes to close those gaps. The admin says yes, and Fleet proposes 3 new profiles and 38 policies for their review." :class="{'reduced-motion': demoReducedMotion, 'popped-out': demoEnding, 'popping-in': demoPoppingIn}" v-else-if="selectedAiTool === 'slack'">
        <div purpose="slack-window" aria-hidden="true">
          <div purpose="slack-title-bar">
            <img alt="" src="/images/ai-demo/icon-slack-back-20x20.svg">
            <img alt="" src="/images/ai-demo/icon-slack-forward-20x20.svg">
            <img alt="" src="/images/ai-demo/icon-slack-history-20x20.svg">
            <span purpose="slack-search"><img alt="" src="/images/ai-demo/icon-slack-search-14x14.svg"></span>
          </div>
          <div purpose="slack-body">
            <div purpose="slack-rail">
              <span purpose="slack-rail-item" class="selected"><img alt="" src="/images/ai-demo/icon-slack-home-20x20.svg"></span>
              <span purpose="slack-rail-item"><img alt="" src="/images/ai-demo/icon-slack-messages-20x20.svg"></span>
              <span purpose="slack-rail-item"><img alt="" src="/images/ai-demo/icon-slack-activity-20x20.svg"></span>
              <span purpose="slack-rail-item"><img alt="" src="/images/ai-demo/icon-slack-bookmark-20x20.svg"></span>
              <span purpose="slack-rail-item"><img alt="" src="/images/ai-demo/icon-slack-more-20x20.svg"></span>
            </div>
            <div purpose="slack-workspace">
              <div purpose="slack-sidebar">
                <div purpose="slack-sidebar-header">
                  <span purpose="slack-workspace-name"><span purpose="slack-sidebar-bar"></span><img alt="" src="/images/ai-demo/icon-slack-chevron-down-12x12.svg"></span>
                  <span purpose="slack-new-message"><img alt="" src="/images/ai-demo/icon-slack-new-message-12x12.svg"></span>
                </div>
                <div purpose="slack-sidebar-group">
                  <div purpose="slack-sidebar-item"><img alt="" src="/images/ai-demo/icon-slack-unread-14x14.svg"><span purpose="slack-sidebar-bar"></span></div>
                  <div purpose="slack-sidebar-item"><img alt="" src="/images/ai-demo/icon-slack-thread-14x14.svg"><span purpose="slack-sidebar-bar"></span></div>
                  <div purpose="slack-sidebar-item"><img alt="" src="/images/ai-demo/icon-slack-send-14x14.svg"><span purpose="slack-sidebar-bar"></span></div>
                </div>
                <div purpose="slack-sidebar-section"><img alt="" src="/images/ai-demo/icon-slack-caret-18x18.svg"><span>Channels</span></div>
                <div purpose="slack-sidebar-channels">
                  <div purpose="slack-sidebar-item" class="selected"><img alt="" src="/images/ai-demo/icon-slack-hashtag-selected-14x14.svg"><span>ask fleet</span></div>
                  <div purpose="slack-sidebar-item"><img alt="" src="/images/ai-demo/icon-slack-hashtag-14x14.svg"><span purpose="slack-sidebar-bar"></span></div>
                  <div purpose="slack-sidebar-item"><img alt="" src="/images/ai-demo/icon-slack-hashtag-14x14.svg"><span purpose="slack-sidebar-bar"></span></div>
                  <div purpose="slack-sidebar-item"><img alt="" src="/images/ai-demo/icon-slack-lock-14x14.svg"><span purpose="slack-sidebar-bar"></span></div>
                </div>
                <div purpose="slack-sidebar-item" class="direct-messages"><img alt="" src="/images/ai-demo/icon-slack-message-14x14.svg"><span purpose="slack-sidebar-bar"></span></div>
              </div>
              <div purpose="slack-channel">
                <div purpose="slack-pane-header">
                  <span purpose="slack-channel-name"><img alt="" src="/images/ai-demo/icon-slack-hashtag-thick-13x13.svg"><span>ask fleet</span><img alt="" src="/images/ai-demo/icon-slack-channel-chevron-13x13.svg"></span>
                </div>
                <div purpose="slack-channel-message"><img alt="" src="/images/ai-demo/avatar-slack-allen-28x28@2x.png"><div><span style="width: 162em;"></span><span style="width: 135em;"></span><span style="width: 99em;"></span><span style="width: 135em;"></span></div></div>
                <div purpose="slack-channel-message"><img alt="" src="/images/ai-demo/avatar-slack-zay-28x28@2x.png"><div><span style="width: 162em;"></span><span style="width: 135em;"></span><span style="width: 99em;"></span><span style="width: 135em;"></span></div></div>
                <div purpose="slack-channel-message"><img alt="" src="/images/ai-demo/avatar-slack-mike-m-28x28@2x.png"><div><span style="width: 162em;"></span><span style="width: 135em;"></span><span style="width: 99em;"></span><span style="width: 135em;"></span></div></div>
              </div>
              <div purpose="slack-thread-pane">
                <div purpose="slack-pane-header"><span purpose="slack-thread-title">Thread</span></div>
                <div purpose="slack-conversation" ref="demoConversation">
                  <div purpose="slack-thread" ref="demoThread" :class="{resetting: demoResetting, 'fading-out': demoFadingOut}" :style="{transform: 'translateY(-' + demoScrollOffset + 'px)'}">
                    <div purpose="slack-message">
                      <img purpose="slack-avatar" alt="" src="/images/ai-demo/avatar-slack-mike-32x32@2x.png">
                      <div purpose="slack-message-content">
                        <span purpose="slack-sender">Mike</span>
                        <p><span purpose="slack-mention">@fleet</span> what do we need for CIS Level 2 on our Macs?</p>
                        <div purpose="slack-reactions"><span purpose="slack-reaction"><span>✅</span><span>1</span></span><span purpose="slack-reaction"><img alt="" src="/images/ai-demo/icon-slack-add-reaction-14x14.svg"></span></div>
                      </div>
                    </div>
                    <div purpose="slack-message" v-show="demoStep >= 2">
                      <img purpose="slack-avatar" alt="" src="/images/ai-demo/icon-slack-fleet-avatar-32x32.svg">
                      <div purpose="slack-message-content">
                        <span purpose="slack-sender">Fleet <span purpose="slack-app-badge">APP</span></span>
                        <p>2,184 of 2,350 Macs pass Level 1. Level 2 adds 38 settings. The biggest gaps are iCloud Drive sync, telemetry limits, and unneeded services. Want me to propose changes that close those gaps?</p>
                      </div>
                    </div>
                    <div purpose="slack-message" v-show="demoStep >= 3">
                      <img purpose="slack-avatar" alt="" src="/images/ai-demo/avatar-slack-mike-32x32@2x.png">
                      <div purpose="slack-message-content">
                        <span purpose="slack-sender">Mike</span>
                        <p>Yes</p>
                        <div purpose="slack-reactions"><span purpose="slack-reaction"><span>✅</span><span>1</span></span><span purpose="slack-reaction"><img alt="" src="/images/ai-demo/icon-slack-add-reaction-14x14.svg"></span></div>
                      </div>
                    </div>
                    <div purpose="slack-message" v-show="demoStep >= 4">
                      <img purpose="slack-avatar" alt="" src="/images/ai-demo/icon-slack-fleet-avatar-32x32.svg">
                      <div purpose="slack-message-content">
                        <span purpose="slack-sender">Fleet <span purpose="slack-app-badge">APP</span></span>
                        <p>Done. I’ve proposed 3 new profiles and 38 policies <span purpose="slack-link">for your review</span>.</p>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div purpose="teams-animation" role="img" aria-label="A Microsoft Teams chat with Fleet. An admin asks what it takes to meet CIS Level 2 on their Windows devices. Fleet reports that 2,184 of 2,350 Windows hosts pass Level 1, that Level 2 adds 38 settings, and that the biggest gaps are Credential Guard, telemetry limits, and unneeded services, then offers to propose changes to close those gaps. The admin says yes, and Fleet proposes 3 new profiles and 38 policies for their review." :class="{'reduced-motion': demoReducedMotion, 'popped-out': demoEnding, 'popping-in': demoPoppingIn}" v-else-if="selectedAiTool === 'teams'">
        <div purpose="teams-window" aria-hidden="true">
          <div purpose="teams-title-bar">
            <img purpose="teams-logo" alt="" src="/images/ai-demo/icon-teams-logo-28x28.svg">
            <div purpose="teams-title-nav">
              <img alt="" src="/images/ai-demo/icon-teams-back-21x21.svg">
              <img alt="" src="/images/ai-demo/icon-teams-forward-21x21.svg">
              <span purpose="teams-search"><img alt="" src="/images/ai-demo/icon-teams-search-18x18.svg"><span>Search</span></span>
            </div>
          </div>
          <div purpose="teams-body">
            <div purpose="teams-app-bar">
              <span purpose="teams-app-bar-item"><img alt="" src="/images/ai-demo/icon-teams-bell-21x21.svg"></span>
              <span purpose="teams-app-bar-item" class="selected"><img alt="" src="/images/ai-demo/icon-teams-chat-21x21.svg"></span>
              <span purpose="teams-app-bar-item"><img alt="" src="/images/ai-demo/icon-teams-calendar-21x21.svg"></span>
              <span purpose="teams-app-bar-item"><img alt="" src="/images/ai-demo/icon-teams-calls-21x21.svg"></span>
              <span purpose="teams-app-bar-item"><img alt="" src="/images/ai-demo/icon-teams-files-21x21.svg"></span>
              <span purpose="teams-app-bar-item"><img alt="" src="/images/ai-demo/icon-teams-more-21x21.svg"></span>
              <span purpose="teams-app-bar-item"><img alt="" src="/images/ai-demo/icon-teams-apps-21x21.svg"></span>
            </div>
            <div purpose="teams-sidebar">
              <div purpose="teams-sidebar-header">
                <span purpose="teams-sidebar-title">Chat</span>
                <span purpose="teams-icon-button"><img alt="" src="/images/ai-demo/icon-teams-filter-17x17.svg"></span>
                <span purpose="teams-icon-button"><img alt="" src="/images/ai-demo/icon-teams-compose-15x15.svg"></span>
              </div>
              <span purpose="teams-sidebar-label">Recent</span>
              <div purpose="teams-sidebar-item"><span purpose="teams-placeholder-avatar"></span><span purpose="teams-placeholder-bar"></span></div>
              <div purpose="teams-sidebar-item" class="selected"><img alt="" src="/images/icon-fleet-avatar-28x28.svg"><span purpose="teams-chat-name">Fleet</span></div>
              <div purpose="teams-sidebar-item"><span purpose="teams-placeholder-avatar"></span><span purpose="teams-placeholder-bar"></span></div>
              <span purpose="teams-sidebar-label">Chats</span>
              <div purpose="teams-sidebar-item"><span purpose="teams-placeholder-avatar"></span><span purpose="teams-placeholder-bar"></span></div>
              <div purpose="teams-sidebar-item"><span purpose="teams-placeholder-avatar"></span><span purpose="teams-placeholder-bar"></span></div>
              <div purpose="teams-sidebar-item"><span purpose="teams-placeholder-avatar"></span><span purpose="teams-placeholder-bar"></span></div>
            </div>
            <div purpose="teams-main">
              <div purpose="teams-header">
                <img alt="" src="/images/icon-fleet-avatar-28x28.svg">
                <span purpose="teams-chat-title">Fleet</span>
                <span purpose="teams-tab">Chat</span>
              </div>
              <div purpose="teams-conversation" ref="demoConversation">
                <div purpose="teams-thread" ref="demoThread" :class="{resetting: demoResetting, 'fading-out': demoFadingOut}" :style="{transform: 'translateY(-' + demoScrollOffset + 'px)'}">
                  <div purpose="teams-first-message">
                    <p><strong>@Fleet</strong> what do we need for CIS level 2 on our Windows devices?</p>
                  </div>
                  <div purpose="teams-first-reply" v-show="demoStep >= 2">
                    <div purpose="teams-typing" v-show="demoStep === 2">
                      <img purpose="teams-avatar" alt="" src="/images/icon-fleet-avatar-28x28.svg">
                      <div purpose="teams-typing-bubble"><span></span><span></span><span></span></div>
                    </div>
                    <div purpose="teams-reply" v-show="demoStep >= 3">
                      <img purpose="teams-avatar" alt="" src="/images/icon-fleet-avatar-28x28.svg">
                      <div purpose="teams-reply-body">
                        <span purpose="teams-sender">Fleet</span>
                        <div purpose="teams-reply-bubble">
                          <p>2,184 of 2,350 Windows hosts pass Level 1. Level 2 adds 38 settings. The biggest gaps are Credential Guard, telemetry limits, and unneeded services.</p>
                        </div>
                        <div purpose="teams-follow-up" v-show="demoStep >= 4">
                          <div purpose="teams-reply-bubble">
                            <p>Want me to propose changes that close those gaps?</p>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                  <div purpose="teams-second-message" v-show="demoStep >= 5">
                    <p>Yes</p>
                  </div>
                  <div purpose="teams-second-reply" v-show="demoStep >= 6">
                    <div purpose="teams-typing" v-show="demoStep === 6">
                      <img purpose="teams-avatar" alt="" src="/images/icon-fleet-avatar-28x28.svg">
                      <div purpose="teams-typing-bubble"><span></span><span></span><span></span></div>
                    </div>
                    <div purpose="teams-reply" v-show="demoStep >= 7">
                      <img purpose="teams-avatar" alt="" src="/images/icon-fleet-avatar-28x28.svg">
                      <div purpose="teams-reply-body">
                        <span purpose="teams-sender">Fleet</span>
                        <div purpose="teams-reply-bubble">
                          <p>Done. I’ve proposed 3 new profiles and 38 policies <span purpose="teams-link">for your review</span>.</p>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div purpose="claude-code-animation" role="img" aria-label="A Claude Code session connected to Fleet. An admin asks what it takes to meet CIS Level 2 on their Macs. Claude checks with Fleet and reports that 2,184 of 2,350 Macs pass Level 1, that Level 2 adds 38 settings, and that the biggest gaps are iCloud Drive sync, telemetry limits, and unneeded services, then offers to propose changes to close those gaps. The admin says yes, and Claude opens a pull request that proposes 3 new profiles and 38 policies for their review." :class="{'reduced-motion': demoReducedMotion, 'popped-out': demoEnding, 'popping-in': demoPoppingIn}" v-else>
        <div purpose="claude-code-window" aria-hidden="true">
          <div purpose="claude-code-sidebar">
            <div purpose="claude-code-title-bar">
              <img alt="" src="/images/ai-demo/icon-claude-code-sidebar-18x18.svg">
              <img alt="" src="/images/ai-demo/icon-claude-code-back-16x16.svg">
              <img alt="" src="/images/ai-demo/icon-claude-code-forward-16x16.svg">
              <div purpose="claude-code-mode-toggle">
                <span><img alt="" src="/images/ai-demo/icon-claude-code-chat-16x16.svg"></span>
                <span purpose="claude-code-mode-selected"><img alt="" src="/images/ai-demo/icon-claude-code-code-16x16.svg"></span>
              </div>
            </div>
            <div purpose="claude-code-nav">
              <div purpose="claude-code-nav-item"><img alt="" src="/images/ai-demo/icon-claude-code-plus-16x16.svg"><span purpose="claude-code-placeholder-bar"></span></div>
              <div purpose="claude-code-nav-item"><img alt="" src="/images/ai-demo/icon-claude-code-search-16x16.svg"><span purpose="claude-code-placeholder-bar"></span></div>
              <div purpose="claude-code-nav-item"><img alt="" src="/images/ai-demo/icon-claude-code-artifacts-16x16.svg"><span purpose="claude-code-placeholder-bar"></span></div>
              <div purpose="claude-code-nav-item"><img alt="" src="/images/ai-demo/icon-claude-code-bell-16x16.svg"><span purpose="claude-code-placeholder-bar"></span></div>
              <div purpose="claude-code-nav-item"><img alt="" src="/images/ai-demo/icon-claude-code-briefcase-16x16.svg"><span purpose="claude-code-placeholder-bar"></span></div>
              <div purpose="claude-code-nav-item"><img alt="" src="/images/ai-demo/icon-claude-code-more-14x14.svg"><span purpose="claude-code-placeholder-bar"></span></div>
            </div>
            <div purpose="claude-code-project-header">
              <span purpose="claude-code-project-name"><span>fleet-gitops</span><img alt="" src="/images/ai-demo/icon-claude-code-caret-10x10.svg"></span>
              <img alt="" src="/images/ai-demo/icon-claude-code-plus-16x16.svg">
            </div>
            <div purpose="claude-code-sessions">
              <div purpose="claude-code-session" class="selected"><span purpose="claude-code-session-name">CIS Level 2 for Macs</span><img purpose="claude-code-session-icon" alt="" src="/images/ai-demo/icon-claude-code-pull-request-16x16.svg"></div>
              <div purpose="claude-code-session"><span purpose="claude-code-placeholder-bar"></span><img purpose="claude-code-session-icon" alt="" src="/images/ai-demo/icon-claude-code-branch-14x14.svg"></div>
              <div purpose="claude-code-session"><span purpose="claude-code-placeholder-bar"></span><img purpose="claude-code-session-icon" alt="" src="/images/ai-demo/icon-claude-code-branch-14x14.svg"></div>
              <div purpose="claude-code-session"><span purpose="claude-code-placeholder-bar"></span><img purpose="claude-code-session-icon" alt="" src="/images/ai-demo/icon-claude-code-pull-request-16x16.svg"></div>
            </div>
            <div purpose="claude-code-account">
              <span purpose="claude-code-avatar"><img alt="" src="/images/ai-demo/avatar-mike-24x29@2x.png"></span>
              <span purpose="claude-code-placeholder-bar"></span>
              <img alt="" src="/images/ai-demo/icon-claude-code-bug-15x15.svg">
            </div>
          </div>
          <div purpose="claude-code-main">
            <div purpose="claude-code-header">
              <span purpose="claude-code-session-title"><span>CIS Level 2 for Macs</span><img alt="" src="/images/ai-demo/icon-claude-code-chevron-down-14x14.svg"></span>
              <span purpose="claude-code-panel-toggle"><img alt="" src="/images/ai-demo/icon-claude-code-panel-list-15x15.svg"></span>
              <img alt="" src="/images/ai-demo/icon-claude-code-terminal-18x18.svg">
              <img alt="" src="/images/ai-demo/icon-claude-code-diff-16x16.svg">
              <img alt="" src="/images/ai-demo/icon-claude-code-globe-18x18.svg">
              <img alt="" src="/images/ai-demo/icon-claude-code-folder-18x18.svg">
              <img alt="" src="/images/ai-demo/icon-claude-code-more-18x18.svg">
            </div>
            <div purpose="claude-code-conversation" ref="demoConversation">
              <div purpose="claude-code-thread" ref="demoThread" :class="{resetting: demoResetting, 'fading-out': demoFadingOut}" :style="{transform: 'translateY(-' + demoScrollOffset + 'px)'}">
                <div purpose="claude-code-user-message"><p>What do we need for CIS Level 2 on our Macs?</p></div>
                <p v-show="demoStep >= 2">I’ll check with Fleet.</p>
                <div purpose="claude-code-tool-calls" v-show="demoStep >= 3">
                  <div purpose="claude-code-tool-row" v-show="demoStep >= 3">
                    <img alt="" src="/images/ai-demo/icon-claude-code-tool-16x16.svg">
                    <span purpose="claude-code-tool-label">Fleet · Get policy results</span>
                    <code>CIS Level 1</code>
                    <span purpose="claude-code-tool-meta">· 2,350 hosts</span>
                  </div>
                  <div purpose="claude-code-tool-row" v-show="demoStep >= 4">
                    <img alt="" src="/images/ai-demo/icon-claude-code-tool-16x16.svg">
                    <span purpose="claude-code-tool-label">Fleet · Compare benchmark</span>
                    <code>CIS Level 2</code>
                    <span purpose="claude-code-tool-meta">· 38 settings</span>
                  </div>
                </div>
                <p v-show="demoStep >= 5"><strong>2,184 of 2,350</strong> Macs pass Level 1. Level 2 adds <strong>38</strong> settings. The biggest gaps are:</p>
                <div purpose="claude-code-table" v-show="demoStep >= 6">
                  <div purpose="claude-code-table-header"><span>Setting</span><span>Failing</span><span>Fix</span></div>
                  <div purpose="claude-code-table-row"><span>iCloud Drive sync</span><span>1,406</span><span>Restrictions profile</span></div>
                  <div purpose="claude-code-table-row"><span>Telemetry limits</span><span>2,031</span><span>Diagnostics profile</span></div>
                  <div purpose="claude-code-table-row"><span>Unneeded services</span><span>1,872</span><span>Sharing profile</span></div>
                </div>
                <p v-show="demoStep >= 7">Want me to propose changes that close those gaps?</p>
                <div purpose="claude-code-user-message" v-show="demoStep >= 8"><p>Yes</p></div>
                <div purpose="claude-code-tool-calls" v-show="demoStep >= 9">
                  <div purpose="claude-code-tool-row" v-show="demoStep >= 9">
                    <img alt="" src="/images/ai-demo/icon-claude-code-edit-16x16.svg">
                    <span purpose="claude-code-tool-label">Write</span>
                    <code>4 files · 3 profiles, 1 policy file</code>
                    <span purpose="claude-code-tool-added">+507</span>
                  </div>
                  <div purpose="claude-code-tool-row" v-show="demoStep >= 10">
                    <img alt="" src="/images/ai-demo/icon-claude-code-edit-16x16.svg">
                    <span purpose="claude-code-tool-label">Edit</span>
                    <code>fleets/workstations.yml</code>
                    <span purpose="claude-code-tool-added">+4</span>
                  </div>
                </div>
                <div purpose="claude-code-pull-request" v-show="demoStep >= 11">
                  <img alt="" src="/images/ai-demo/icon-claude-code-pull-request-16x16.svg">
                  <div>
                    <span purpose="claude-code-pull-request-title">Pull request opened</span>
                    <span purpose="claude-code-pull-request-branch">cis-level-2 → main</span>
                  </div>
                </div>
                <p v-show="demoStep >= 12">Done. I’ve proposed 3 new profiles and 38 policies <span purpose="claude-code-link">for your review</span>.</p>
              </div>
            </div>
            <div purpose="claude-code-composer">
              <div purpose="claude-code-input">
                <img alt="" src="/images/ai-demo/icon-claude-code-plus-18x18.svg">
                <span purpose="claude-code-input-text" v-if="demoComposerText">{{demoComposerText}}<span purpose="ai-demo-caret"></span></span>
                <span purpose="claude-code-placeholder" v-else>Type / for commands</span>
                <img alt="" src="/images/ai-demo/icon-claude-code-mic-18x18.svg">
              </div>
              <div purpose="claude-code-session-settings">
                <span purpose="claude-code-permission-mode">Auto</span>
                <span>Opus 5.5 <span purpose="claude-code-effort">Medium</span></span>
                <img alt="" src="/images/ai-demo/icon-claude-code-spinner-14x14.svg">
              </div>
            </div>
          </div>
        </div>
      </div>
      <div purpose="ai-demo-ending" aria-hidden="true" :class="{'fading-out': demoFadingOut}" v-if="demoEnding">
        <div purpose="ai-demo-pr-window" :class="{'popped-out': demoEnding === 'fleet'}">
          <div purpose="ai-demo-pr-header">
            <p purpose="ai-demo-pr-title">Add CIS Level 2 profiles and policies <span>#1420</span></p>
            <div purpose="ai-demo-pr-meta">
              <span purpose="ai-demo-pr-state" :class="{merged: demoMerged}">
                <svg v-if="demoMerged" viewBox="0 0 16 16"><path d="M5.45 5.154A4.25 4.25 0 0 0 9.25 7.5h1.378a2.251 2.251 0 1 1 0 1.5H9.25A5.734 5.734 0 0 1 5 7.123v3.505a2.25 2.25 0 1 1-1.5 0V5.372a2.25 2.25 0 1 1 1.95-.218ZM4.25 13.5a.75.75 0 1 0 0-1.5.75.75 0 0 0 0 1.5Zm8.5-4.5a.75.75 0 1 0 0-1.5.75.75 0 0 0 0 1.5ZM4.25 4a.75.75 0 1 0 0-1.5.75.75 0 0 0 0 1.5Z"/></svg>
                <svg v-else viewBox="0 0 16 16"><path d="M1.5 3.25a2.25 2.25 0 1 1 3 2.122v5.256a2.251 2.251 0 1 1-1.5 0V5.372A2.25 2.25 0 0 1 1.5 3.25Zm5.677-.177L9.573.677A.25.25 0 0 1 10 .854V2.5h1A2.5 2.5 0 0 1 13.5 5v5.628a2.251 2.251 0 1 1-1.5 0V5a1 1 0 0 0-1-1h-1v1.646a.25.25 0 0 1-.427.177L7.177 3.427a.25.25 0 0 1 0-.354ZM3.75 2.5a.75.75 0 1 0 0 1.5.75.75 0 0 0 0-1.5Zm0 9.5a.75.75 0 1 0 0 1.5.75.75 0 0 0 0-1.5Zm8.25.75a.75.75 0 1 0 1.5 0 .75.75 0 0 0-1.5 0Z"/></svg>
                <span>{{demoMerged ? 'Merged' : 'Open'}}</span>
              </span>
              <span purpose="ai-demo-pr-summary">{{demoMerged ? 'Merged' : 'Merge'}} <code>cis-level-2</code> into <code>main</code> · {{demoMerged ? 'Just now' : 'Opened just now'}}</span>
            </div>
            <div purpose="ai-demo-pr-tabs">
              <div purpose="ai-demo-pr-tab-list">
                <span purpose="ai-demo-pr-tab" :class="{selected: demoPrView === 'conversation'}" ref="demoConversationTab"><svg viewBox="0 0 16 16"><path d="M1.75 1h8.5c.966 0 1.75.784 1.75 1.75v5.5A1.75 1.75 0 0 1 10.25 10H7.061l-2.574 2.573A1.458 1.458 0 0 1 2 11.543V10h-.25A1.75 1.75 0 0 1 0 8.25v-5.5C0 1.784.784 1 1.75 1ZM1.5 2.75v5.5c0 .138.112.25.25.25h1a.75.75 0 0 1 .75.75v2.19l2.72-2.72a.749.749 0 0 1 .53-.22h3.5a.25.25 0 0 0 .25-.25v-5.5a.25.25 0 0 0-.25-.25h-8.5a.25.25 0 0 0-.25.25Zm13 2a.25.25 0 0 0-.25-.25h-.5a.75.75 0 0 1 0-1.5h.5c.966 0 1.75.784 1.75 1.75v5.5A1.75 1.75 0 0 1 14.25 12H14v1.543a1.458 1.458 0 0 1-2.487 1.03L9.22 12.28a.749.749 0 0 1 .326-1.275.749.749 0 0 1 .734.215l2.22 2.22v-2.19a.75.75 0 0 1 .75-.75h1a.25.25 0 0 0 .25-.25Z"/></svg><span>Conversation</span><span purpose="ai-demo-pr-count">0</span></span>
                <span purpose="ai-demo-pr-tab"><svg viewBox="0 0 16 16"><path d="M11.93 8.5a4.002 4.002 0 0 1-7.86 0H.75a.75.75 0 0 1 0-1.5h3.32a4.002 4.002 0 0 1 7.86 0h3.32a.75.75 0 0 1 0 1.5Zm-1.43-.75a2.5 2.5 0 1 0-5 0 2.5 2.5 0 0 0 5 0Z"/></svg><span><span purpose="ai-demo-ghost" class="tab" style="width: 4.92em;"></span></span></span>
                <span purpose="ai-demo-pr-tab"><svg viewBox="0 0 16 16"><path d="M2.5 1.75v11.5c0 .138.112.25.25.25h3.17a.75.75 0 0 1 0 1.5H2.75A1.75 1.75 0 0 1 1 13.25V1.75C1 .784 1.784 0 2.75 0h8.5C12.216 0 13 .784 13 1.75v7.736a.75.75 0 0 1-1.5 0V1.75a.25.25 0 0 0-.25-.25h-8.5a.25.25 0 0 0-.25.25Zm13.274 9.537v-.001l-4.557 4.45a.75.75 0 0 1-1.055-.008l-1.943-1.95a.75.75 0 0 1 1.062-1.058l1.419 1.425 4.026-3.932a.75.75 0 1 1 1.048 1.074ZM4.75 4h4.5a.75.75 0 0 1 0 1.5h-4.5a.75.75 0 0 1 0-1.5ZM4 7.75A.75.75 0 0 1 4.75 7h2a.75.75 0 0 1 0 1.5h-2A.75.75 0 0 1 4 7.75Z"/></svg><span><span purpose="ai-demo-ghost" class="tab" style="width: 4.92em;"></span></span></span>
                <span purpose="ai-demo-pr-tab" :class="{selected: demoPrView === 'files'}" ref="demoFilesTab"><svg viewBox="0 0 16 16"><path d="M1 1.75C1 .784 1.784 0 2.75 0h7.586c.464 0 .909.184 1.237.513l2.914 2.914c.329.328.513.773.513 1.237v9.586A1.75 1.75 0 0 1 13.25 16H2.75A1.75 1.75 0 0 1 1 14.25Zm1.75-.25a.25.25 0 0 0-.25.25v12.5c0 .138.112.25.25.25h10.5a.25.25 0 0 0 .25-.25V4.664a.25.25 0 0 0-.073-.177l-2.914-2.914a.25.25 0 0 0-.177-.073ZM8 3.25a.75.75 0 0 1 .75.75v1.5h1.5a.75.75 0 0 1 0 1.5h-1.5v1.5a.75.75 0 0 1-1.5 0V7h-1.5a.75.75 0 0 1 0-1.5h1.5V4A.75.75 0 0 1 8 3.25Zm-3 8a.75.75 0 0 1 .75-.75h4.5a.75.75 0 0 1 0 1.5h-4.5a.75.75 0 0 1-.75-.75Z"/></svg><span>Files changed</span><span purpose="ai-demo-pr-count">5</span></span>
              </div>
              <span purpose="ai-demo-pr-diffstat"><span purpose="ai-demo-pr-added">+511</span> <span purpose="ai-demo-pr-removed">−0</span><span purpose="ai-demo-pr-diffbar"><span></span><span></span><span></span><span></span><span></span></span></span>
            </div>
          </div>
          <div purpose="ai-demo-pr-body" v-if="demoPrView === 'conversation'">
            <div purpose="ai-demo-pr-main" ref="demoPrMain">
              <div purpose="ai-demo-pr-timeline" ref="demoPrTimeline" :class="{instant: demoPrScrollInstant}" :style="{transform: 'translateY(-' + demoPrScroll + 'px)'}">
                <div purpose="ai-demo-pr-comment" v-show="demoPrStep >= 1">
                  <img alt="" src="/images/ai-demo/avatar-slack-mike-32x32@2x.png">
                  <div purpose="ai-demo-pr-card" class="comment">
                    <div purpose="ai-demo-pr-comment-header"><p><strong>mike</strong><span purpose="ai-demo-ghost" style="width: 11.54em; margin-left: 0.62em;"></span></p></div>
                    <div purpose="ai-demo-pr-comment-body">
                      <p>Adds 3 profiles and 38 policies so workstations meet CIS Level 2.</p>
                      <ul purpose="ai-demo-pr-files"><li v-for="width in [418, 410, 382, 245, 166]" :key="width"><code><span purpose="ai-demo-ghost" :style="{width: width / 12 + 'em'}"></span></code></li></ul>
                    </div>
                  </div>
                </div>
                <div purpose="ai-demo-pr-events" v-show="demoPrStep >= 2">
                  <div purpose="ai-demo-pr-event"><span purpose="ai-demo-pr-event-icon"><svg viewBox="0 0 16 16"><path d="M8 2c1.981 0 3.671.992 4.933 2.078 1.27 1.091 2.187 2.345 2.637 3.023a1.62 1.62 0 0 1 0 1.798c-.45.678-1.367 1.932-2.637 3.023C11.67 13.008 9.981 14 8 14c-1.981 0-3.671-.992-4.933-2.078C1.797 10.83.88 9.576.43 8.898a1.62 1.62 0 0 1 0-1.798c.45-.677 1.367-1.931 2.637-3.022C4.33 2.992 6.019 2 8 2ZM1.679 7.932a.12.12 0 0 0 0 .136c.411.622 1.241 1.75 2.366 2.717C5.176 11.758 6.527 12.5 8 12.5c1.473 0 2.825-.742 3.955-1.715 1.124-.967 1.954-2.096 2.366-2.717a.12.12 0 0 0 0-.136c-.412-.621-1.242-1.75-2.366-2.717C10.824 4.242 9.473 3.5 8 3.5c-1.473 0-2.825.742-3.955 1.715-1.124.967-1.954 2.096-2.366 2.717ZM8 10a2 2 0 1 1-.001-3.999A2 2 0 0 1 8 10Z"/></svg></span><img alt="" src="/images/ai-demo/avatar-slack-mike-32x32@2x.png"><p><strong>mike</strong> requested a review from <strong>priya</strong> just now</p></div>
                  <div purpose="ai-demo-pr-event" class="approved" v-if="demoApproved"><img purpose="ai-demo-pr-event-reviewer" alt="" src="/images/ai-demo/avatar-swan-40x40@2x.png"><span purpose="ai-demo-pr-event-icon"><svg viewBox="0 0 16 16"><path d="M13.78 4.22a.75.75 0 0 1 0 1.06l-7.25 7.25a.75.75 0 0 1-1.06 0L2.22 9.28a.751.751 0 0 1 .018-1.042.751.751 0 0 1 1.042-.018L6 10.94l6.72-6.72a.75.75 0 0 1 1.06 0Z"/></svg></span><p><strong>priya</strong> approved these changes just now</p></div>
                  <div purpose="ai-demo-pr-event" class="merged" v-if="demoMerged"><span purpose="ai-demo-pr-event-icon"><svg viewBox="0 0 16 16"><path d="M5.45 5.154A4.25 4.25 0 0 0 9.25 7.5h1.378a2.251 2.251 0 1 1 0 1.5H9.25A5.734 5.734 0 0 1 5 7.123v3.505a2.25 2.25 0 1 1-1.5 0V5.372a2.25 2.25 0 1 1 1.95-.218ZM4.25 13.5a.75.75 0 1 0 0-1.5.75.75 0 0 0 0 1.5Zm8.5-4.5a.75.75 0 1 0 0-1.5.75.75 0 0 0 0 1.5ZM4.25 4a.75.75 0 1 0 0-1.5.75.75 0 0 0 0 1.5Z"/></svg></span><img alt="" src="/images/ai-demo/avatar-swan-20x20@2x.png"><p><strong>priya</strong> merged commit <code>3f9c2a1</code> into <code>main</code> just now</p></div>
                  <div purpose="ai-demo-pr-event" :class="{pending: !demoGitops}" v-if="demoMerged"><span purpose="ai-demo-pr-event-icon"><svg viewBox="0 0 16 16"><path d="M14.064 0h.186C15.216 0 16 .784 16 1.75v.186a8.752 8.752 0 0 1-2.564 6.186l-.458.459c-.314.314-.641.616-.979.904v3.207c0 .608-.315 1.172-.833 1.49l-2.774 1.707a.749.749 0 0 1-1.11-.418l-.954-3.102a1.214 1.214 0 0 1-.145-.125L3.754 9.816a1.218 1.218 0 0 1-.124-.145L.528 8.717a.749.749 0 0 1-.418-1.11l1.71-2.774A1.748 1.748 0 0 1 3.31 4h3.204c.288-.338.59-.665.904-.979l.459-.458A8.749 8.749 0 0 1 14.064 0ZM8.938 3.623h-.002l-.458.458c-.76.76-1.437 1.598-2.02 2.5l-1.5 2.317 2.143 2.143 2.317-1.5c.902-.583 1.74-1.26 2.499-2.02l.459-.458a7.25 7.25 0 0 0 2.123-5.127V1.75a.25.25 0 0 0-.25-.25h-.186a7.249 7.249 0 0 0-5.125 2.123ZM3.56 14.56c-.732.732-2.334 1.045-3.005 1.148a.234.234 0 0 1-.201-.064.234.234 0 0 1-.064-.201c.103-.671.416-2.273 1.15-3.003a1.502 1.502 0 1 1 2.12 2.12Zm6.94-3.935c-.088.06-.177.118-.266.175l-2.35 1.521.548 1.783 1.949-1.2a.25.25 0 0 0 .119-.213ZM3.678 8.116 5.2 5.766c.058-.09.117-.178.176-.266H3.309a.25.25 0 0 0-.213.119l-1.2 1.95ZM12 5a1 1 0 1 1-2 0 1 1 0 0 1 2 0Z"/></svg></span><p><strong>github-actions</strong> deployed to <strong>fleet</strong> just now · with <strong>GitHub Actions</strong></p><span purpose="ai-demo-pr-deploy-state" :class="{active: demoGitops === 'applied'}">{{demoGitops === 'applied' ? 'Active' : 'In progress'}}</span></div>
                </div>
                <div purpose="ai-demo-pr-merge" :class="{ready: demoApproved && !demoMerged, merged: demoMerged}" v-show="demoPrStep >= 2">
                  <span purpose="ai-demo-pr-merge-tile"><svg viewBox="0 0 16 16"><path d="M5.45 5.154A4.25 4.25 0 0 0 9.25 7.5h1.378a2.251 2.251 0 1 1 0 1.5H9.25A5.734 5.734 0 0 1 5 7.123v3.505a2.25 2.25 0 1 1-1.5 0V5.372a2.25 2.25 0 1 1 1.95-.218ZM4.25 13.5a.75.75 0 1 0 0-1.5.75.75 0 0 0 0 1.5Zm8.5-4.5a.75.75 0 1 0 0-1.5.75.75 0 0 0 0 1.5ZM4.25 4a.75.75 0 1 0 0-1.5.75.75 0 0 0 0 1.5Z"/></svg></span>
                  <div purpose="ai-demo-pr-merge-box" v-if="demoMerged">
                    <div purpose="ai-demo-pr-merge-section" class="merged"><div><strong>Pull request successfully merged and closed</strong><span><span purpose="ai-demo-ghost" style="width: 18em;"></span></span></div></div>
                  </div>
                  <div purpose="ai-demo-pr-merge-box" v-else>
                    <div purpose="ai-demo-pr-merge-section" v-if="demoApproved"><span purpose="ai-demo-pr-merge-icon" class="success"><svg viewBox="0 0 16 16"><path d="M13.78 4.22a.75.75 0 0 1 0 1.06l-7.25 7.25a.75.75 0 0 1-1.06 0L2.22 9.28a.751.751 0 0 1 .018-1.042.751.751 0 0 1 1.042-.018L6 10.94l6.72-6.72a.75.75 0 0 1 1.06 0Z"/></svg></span><div><strong>Changes approved</strong><span><span purpose="ai-demo-ghost" style="width: 20em;"></span></span></div><svg purpose="ai-demo-pr-chevron" viewBox="0 0 16 16"><path d="M4 6l4 4 4-4" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg></div>
                    <div purpose="ai-demo-pr-merge-section" v-else><span purpose="ai-demo-pr-merge-icon" class="failed"><svg viewBox="0 0 16 16"><path d="M3.72 3.72a.75.75 0 0 1 1.06 0L8 6.94l3.22-3.22a.749.749 0 0 1 1.275.326.749.749 0 0 1-.215.734L9.06 8l3.22 3.22a.749.749 0 0 1-.326 1.275.749.749 0 0 1-.734-.215L8 9.06l-3.22 3.22a.751.751 0 0 1-1.042-.018.751.751 0 0 1-.018-1.042L6.94 8 3.72 4.78a.75.75 0 0 1 0-1.06Z"/></svg></span><div><strong>Review required</strong><span><span purpose="ai-demo-ghost" style="width: 20em;"></span></span></div><svg purpose="ai-demo-pr-chevron" viewBox="0 0 16 16"><path d="M4 6l4 4 4-4" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg></div>
                    <div purpose="ai-demo-pr-merge-detail"><template v-if="demoApproved"><svg purpose="ai-demo-pr-detail-check" viewBox="0 0 16 16"><path d="M13.78 4.22a.75.75 0 0 1 0 1.06l-7.25 7.25a.75.75 0 0 1-1.06 0L2.22 9.28a.751.751 0 0 1 .018-1.042.751.751 0 0 1 1.042-.018L6 10.94l6.72-6.72a.75.75 0 0 1 1.06 0Z"/></svg><span>1 approval</span></template><template v-else><svg viewBox="0 0 16 16"><path d="M10.561 8.073a6.005 6.005 0 0 1 3.432 5.142.75.75 0 1 1-1.498.07 4.5 4.5 0 0 0-8.99 0 .75.75 0 0 1-1.498-.07 6.004 6.004 0 0 1 3.431-5.142 3.999 3.999 0 1 1 5.123 0ZM10.5 5a2.5 2.5 0 1 0-5 0 2.5 2.5 0 0 0 5 0Z"/></svg><span>1 pending review</span></template><svg viewBox="0 0 16 16"><path d="M6 4l4 4-4 4" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg></div>
                    <div purpose="ai-demo-pr-merge-section"><span purpose="ai-demo-pr-merge-icon" class="success"><svg viewBox="0 0 16 16"><path d="M13.78 4.22a.75.75 0 0 1 0 1.06l-7.25 7.25a.75.75 0 0 1-1.06 0L2.22 9.28a.751.751 0 0 1 .018-1.042.751.751 0 0 1 1.042-.018L6 10.94l6.72-6.72a.75.75 0 0 1 1.06 0Z"/></svg></span><div><strong>All checks have passed</strong><span><span purpose="ai-demo-ghost" style="width: 16.67em;"></span></span></div><svg purpose="ai-demo-pr-chevron" viewBox="0 0 16 16"><path d="M6 4l4 4-4 4" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg></div>
                    <div purpose="ai-demo-pr-merge-footer"><span purpose="ai-demo-pr-button" :class="{primary: demoApproved}" ref="demoMergeButton">Merge pull request</span><p><span purpose="ai-demo-ghost" style="width: 20em;"></span></p></div>
                  </div>
                </div>
              </div>
            </div>
            <div purpose="ai-demo-pr-sidebar" :class="{shown: demoPrStep >= 3}">
              <span purpose="ai-demo-pr-sidebar-label">Reviewers</span>
              <div purpose="ai-demo-pr-reviewer"><img alt="" src="/images/ai-demo/avatar-swan-20x20@2x.png"><span>priya</span><svg v-if="demoApproved" viewBox="0 0 16 16" purpose="ai-demo-pr-icon-check"><path d="M3.5 8.5L6.5 11.5L12.5 4.5" fill="none" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round"/></svg><span purpose="ai-demo-pr-pending-dot" v-else></span></div>
              <span purpose="ai-demo-pr-sidebar-divider"></span>
              <span purpose="ai-demo-pr-sidebar-label">Labels</span>
              <div purpose="ai-demo-pr-labels"><span purpose="ai-demo-ghost" class="pill" style="width: 30em;"></span><span purpose="ai-demo-ghost" class="pill" style="width: 46em;"></span></div>
            </div>
          </div>
          <div purpose="ai-demo-pr-files-view" v-else>
            <div purpose="ai-demo-pr-files-toolbar">
              <span purpose="ai-demo-pr-select">All commits <svg viewBox="0 0 16 16"><path d="M4 6l4 4 4-4" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg></span>
              <span purpose="ai-demo-pr-button" class="primary" ref="demoFilesSubmitButton">Submit review <svg viewBox="0 0 16 16"><path d="M4 6l4 4 4-4" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg></span>
              <div purpose="ai-demo-pr-review-panel" class="dropdown" v-if="demoReviewPanelOpen">
                  <strong>Finish your review</strong>
                  <span purpose="ai-demo-pr-comment-box">Leave a comment</span>
                  <div purpose="ai-demo-pr-radio"><svg viewBox="0 0 14 14"><circle cx="7" cy="7" r="6" fill="#ffffff" stroke="#8c959f"/></svg><div><strong>Comment</strong><span><span purpose="ai-demo-ghost" style="width: 21em;"></span></span></div></div>
                  <div purpose="ai-demo-pr-radio"><svg viewBox="0 0 14 14"><circle cx="7" cy="7" r="6.5" fill="#0969da"/><circle cx="7" cy="7" r="2.5" fill="#ffffff"/></svg><div><strong>Approve</strong><span><span purpose="ai-demo-ghost" style="width: 21em;"></span></span></div></div>
                  <div purpose="ai-demo-pr-radio"><svg viewBox="0 0 14 14"><circle cx="7" cy="7" r="6" fill="#ffffff" stroke="#8c959f"/></svg><div><strong>Request changes</strong><span><span purpose="ai-demo-ghost" style="width: 22.64em;"></span></span></div></div>
                  <span purpose="ai-demo-pr-button" class="primary" ref="demoSubmitButton">Submit review</span>
                </div>
            </div>
            <div purpose="ai-demo-pr-file-list">
              <div purpose="ai-demo-pr-file" v-show="demoPrStep >= 1">
                <div purpose="ai-demo-pr-file-header"><svg viewBox="0 0 16 16"><path d="M4 6l4 4 4-4" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg><code>fleets/workstations.yml</code><span purpose="ai-demo-pr-added">+4</span><span purpose="ai-demo-pr-removed">−0</span><span purpose="ai-demo-pr-viewed-box"><svg viewBox="0 0 12 12"><rect x="0.75" y="0.75" width="10.5" height="10.5" rx="2" fill="none" stroke="#59636e" stroke-width="1.5"/></svg>Viewed</span></div>
                <div purpose="ai-demo-pr-diff-row" :class="row[0]" v-for="(row, index) in demoPrDiffs[selectedAiTool === 'teams' ? 'windows' : 'macos']" :key="index">
                  <template v-if="row[0] === 'hunk'"><code><span purpose="ai-demo-ghost" style="width: 14em;"></span></code></template>
                  <template v-else><span>{{row[1]}}</span><span></span><code><span purpose="ai-demo-ghost" :style="demoGhostStyle(row[2])" v-if="row[0] === 'ctx'"></span></code><span>{{row[3]}}</span><span>{{row[0] === 'add' ? '+' : ''}}</span><code><span purpose="ai-demo-ghost" :style="demoGhostStyle(row[4])" v-if="row[0] === 'ctx'"></span><template v-else>{{row[4]}}</template></code></template>
                </div>
              </div>
            </div>
          </div>
        </div>
        <div purpose="ai-demo-fleet-window" v-if="demoEnding === 'fleet'">
          <div purpose="ai-demo-fleet-page">
            <div purpose="ai-demo-fleet-nav">
              <img alt="" src="/images/logo-brandmark-32x32@2x.png">
              <span><span purpose="ai-demo-ghost" style="width: 4em;"></span></span><span><span purpose="ai-demo-ghost" style="width: 4em;"></span></span><span><span purpose="ai-demo-ghost" style="width: 4em;"></span></span><span><span purpose="ai-demo-ghost" style="width: 4em;"></span></span><span class="selected">Policies</span>
              <span purpose="ai-demo-fleet-gitops"><svg viewBox="0 0 16 16"><path d="M5.45 5.154A4.25 4.25 0 0 0 9.25 7.5h1.378a2.251 2.251 0 1 1 0 1.5H9.25A5.734 5.734 0 0 1 5 7.123v3.505a2.25 2.25 0 1 1-1.5 0V5.372a2.25 2.25 0 1 1 1.95-.218ZM4.25 13.5a.75.75 0 1 0 0-1.5.75.75 0 0 0 0 1.5Zm8.5-4.5a.75.75 0 1 0 0-1.5.75.75 0 0 0 0 1.5ZM4.25 4a.75.75 0 1 0 0-1.5.75.75 0 0 0 0 1.5Z"/></svg><span>GitOps mode</span></span>
              <span purpose="ai-demo-fleet-account"><img alt="" src="/images/ai-demo/avatar-swan-20x20@2x.png"><svg viewBox="0 0 16 16"><path d="M4 6l4 4 4-4" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg></span>
            </div>
            <div purpose="ai-demo-fleet-content">
              <p purpose="ai-demo-fleet-title"><span>💻</span>Workstations <svg viewBox="0 0 16 16"><path d="M4 6l4 4 4-4" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg></p>
              <p purpose="ai-demo-fleet-description"><span purpose="ai-demo-ghost" style="width: 23.26em;"></span></p>
              <div purpose="ai-demo-fleet-toolbar"><span>38 policies</span><span purpose="ai-demo-fleet-search"><svg viewBox="0 0 16 16"><circle cx="7" cy="7" r="4.5" fill="none" stroke="#515774" stroke-width="1.5"/><path d="M10.5 10.5L13.5 13.5" stroke="#515774" stroke-width="1.5" stroke-linecap="round"/></svg>CIS</span></div>
              <div purpose="ai-demo-fleet-table">
                <div purpose="ai-demo-fleet-row" class="header"><span>Name</span><span><svg viewBox="0 0 16 16"><circle cx="8" cy="8" r="7" fill="#3db67b"/><path d="M5 8.2l2 2 4-4.2" fill="none" stroke="#ffffff" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>Pass</span><span><svg viewBox="0 0 16 16"><circle cx="8" cy="8" r="6.25" fill="none" stroke="#515774" stroke-width="1.5"/><path d="M8 4.75v3.75" stroke="#515774" stroke-width="1.5" stroke-linecap="round"/><circle cx="8" cy="11" r="0.9" fill="#515774"/></svg>Fail</span></div>
                <div purpose="ai-demo-fleet-row" v-for="policy in demoFleetPolicies[selectedAiTool === 'teams' ? 'windows' : 'macos']" :key="policy[0]"><span>{{policy[0]}}</span><span>{{formatDemoHosts(policy, true)}}</span><span>{{formatDemoHosts(policy, false)}}</span></div>
              </div>
            </div>
          </div>
          <span purpose="ai-demo-fleet-cue" v-show="demoCueVisible"><svg viewBox="0 0 14 14"><circle cx="7" cy="7" r="5.75" fill="none" stroke="#ffffff" stroke-width="1.5"/><path d="M7 4v3l2 1.5" fill="none" stroke="#ffffff" stroke-width="1.5" stroke-linecap="round"/></svg>1 hour later</span>
        </div>
      </div>
      <img purpose="ai-demo-pointer" alt="" src="/images/ai-demo/icon-pointer-20x28.svg" :class="{visible: demoPointer.visible}" :style="{transform: 'translate(' + demoPointer.x + 'px, ' + demoPointer.y + 'px)' + (demoPointer.pressed ? ' scale(0.85)' : ''), transitionDuration: demoPointer.moveMs + 'ms'}">
    </div>
  </div>
  `,

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    if (bowser.windows && !this.hideToolSwitcher && !this.hideTeams) {
      this.selectedAiTool = 'teams';
    }
  },
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
      // The shared ending covers every tool, so switching during it starts the new tool over.
      if (this.demoFadingOut || this.demoEntryMs === 0 || this.demoEnding) {
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
      if (!await this.runAiDemoEnding(wait)) {
        return;
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
      // Start the tool's fade-in in the same render that clears the ending, or
      // it shows at full opacity for a frame first.
      if (this.demoEnding) {
        this.demoPoppingIn = true;
        setTimeout(() => {
          this.demoPoppingIn = false;
        }, 650);
      }
      this.demoResetting = true;
      this.demoEntryMs = 0;
      this.demoStep = 0;
      this.demoComposerText = '';
      this.demoScrollOffset = 0;
      this.demoEnding = '';
      this.demoPrView = 'conversation';
      this.demoPrScroll = 0;
      this.demoPrScrollInstant = false;
      this.demoPrStep = 0;
      this.demoReviewPanelOpen = false;
      this.demoApproved = false;
      this.demoMerged = false;
      this.demoGitops = '';
      this.demoCueVisible = false;
      this.demoFleetProgress = 0;
      this.demoPointer = {visible: false, pressed: false, x: 0, y: 0, moveMs: 0};
      await this.$nextTick();
      // Let the browser apply the reset before transitions come back on, then
      // fade the thread back in. It starts with the first message already sent.
      await new Promise((resolve) => setTimeout(resolve, 50));
      this.demoResetting = false;
      this.demoFadingOut = false;
    },

    // Plays the shared ending: the pointer opens the pull request, approves and
    // merges it, then the demo switches to Fleet. Resolves to false if cancelled.
    runAiDemoEnding: async function(wait) {
      let stage = this.$refs.demoStage;
      let link = stage && stage.querySelector('[purpose$="-link"]');
      if (!link) {
        return true;
      }
      let stageRect = stage.getBoundingClientRect();
      this.demoPointer = {visible: false, pressed: false, x: stageRect.width * 0.82, y: stageRect.height * 0.98, moveMs: 0};
      await this.$nextTick();
      this.demoPointer.visible = true;
      if (!await this.moveDemoPointerTo(link, 900, wait)) {
        return false;
      }
      if (!await this.pressDemoPointer(wait)) {
        return false;
      }
      this.demoPrView = 'conversation';
      this.demoEnding = 'pr';
      for (let step = 1; step <= 3; step++) {
        if (!await wait(step === 1 ? 450 : 250)) {
          return false;
        }
        this.demoPrStep = step;
      }
      if (!await wait(900)) {
        return false;
      }
      await this.$nextTick();
      // Read the conversation, then check the diff before approving.
      if (!await this.moveDemoPointerTo(this.$refs.demoFilesTab, 800, wait) || !await this.pressDemoPointer(wait)) {
        return false;
      }
      this.demoPrView = 'files';
      await this.$nextTick();
      if (!await wait(1400) || !await this.moveDemoPointerTo(this.$refs.demoFilesSubmitButton, 800, wait) || !await this.pressDemoPointer(wait)) {
        return false;
      }
      this.demoReviewPanelOpen = true;
      await this.$nextTick();
      if (!await wait(500) || !await this.moveDemoPointerTo(this.$refs.demoSubmitButton, 650, wait) || !await this.pressDemoPointer(wait)) {
        return false;
      }
      this.demoReviewPanelOpen = false;
      this.demoApproved = true;
      await this.$nextTick();
      // The merge button is back on the Conversation tab.
      if (!await wait(250) || !await this.moveDemoPointerTo(this.$refs.demoConversationTab, 550, wait) || !await this.pressDemoPointer(wait)) {
        return false;
      }
      this.demoPrView = 'conversation';
      await this.$nextTick();
      this.updateDemoPrScroll();
      if (!await wait(500) || !await this.moveDemoPointerTo(this.$refs.demoMergeButton, 450, wait) || !await this.pressDemoPointer(wait)) {
        return false;
      }
      // The merged box is much shorter than the merge box. Settle the scroll in the same frame,
      // with room already held for the deploy event, so the conversation doesn't slide back up.
      this.demoMerged = true;
      this.demoPrScrollInstant = true;
      await this.$nextTick();
      this.updateDemoPrScroll();
      this.demoPointer.visible = false;
      if (!await wait(50)) {
        return false;
      }
      this.demoPrScrollInstant = false;
      if (!await wait(350)) {
        return false;
      }
      this.demoGitops = 'applying';
      if (!await wait(700)) {
        return false;
      }
      this.demoGitops = 'applied';
      if (!await wait(600)) {
        return false;
      }
      this.demoEnding = 'fleet';
      if (!await wait(600)) {
        return false;
      }
      this.demoCueVisible = true;
      if (!await wait(400)) {
        return false;
      }
      // Pass counts climb with an ease-out, as if an hour of check-ins happened.
      for (let i = 1; i <= 30; i++) {
        this.demoFleetProgress = 1 - Math.pow(1 - i / 30, 3);
        if (!await wait(60)) {
          return false;
        }
      }
      return await wait(4500);
    },

    moveDemoPointerTo: async function(element, ms, wait) {
      let stage = this.$refs.demoStage;
      if (!element || !stage) {
        return true;
      }
      let stageRect = stage.getBoundingClientRect();
      // A link that wraps has one box per line, so aim at its first line.
      let rect = element.getClientRects()[0] || element.getBoundingClientRect();
      this.demoPointer.moveMs = ms;
      this.demoPointer.x = rect.left - stageRect.left + Math.min(rect.width * 0.6, rect.width - 8);
      this.demoPointer.y = rect.top - stageRect.top + rect.height * 0.55;
      return await wait(ms + 100);
    },

    pressDemoPointer: async function(wait) {
      this.demoPointer.moveMs = 120;
      this.demoPointer.pressed = true;
      if (!await wait(140)) {
        return false;
      }
      this.demoPointer.pressed = false;
      return await wait(120);
    },

    // Scrolls the Conversation tab just enough to show the bottom of the timeline.
    updateDemoPrScroll: function() {
      let main = this.$refs.demoPrMain;
      let timeline = this.$refs.demoPrTimeline;
      if (!main || !timeline) {
        return;
      }
      let gap = 16 * parseFloat(window.getComputedStyle(main).fontSize);
      this.demoPrScroll = Math.max(0, timeline.offsetHeight + gap - main.clientHeight);
    },

    // A ghost bar sized to stand in for a line of text: one em per monospace character
    // by default, with the line's indent kept.
    demoGhostStyle: function(text, emPerCharacter = 0.6) {
      let clean = String(text).replace(/\u200b/g, '');
      let indent = clean.length - clean.replace(/^ +/, '').length;
      return {marginLeft: indent * emPerCharacter + 'em', width: clean.trim().length * emPerCharacter + 'em'};
    },

    formatDemoHosts: function(policy, isPassing) {
      let passing = Math.round(policy[1] + (policy[2] - policy[1]) * this.demoFleetProgress);
      let hosts = isPassing ? passing : 2350 - passing;
      return hosts.toLocaleString('en-US') + (hosts === 1 ? ' host' : ' hosts');
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
