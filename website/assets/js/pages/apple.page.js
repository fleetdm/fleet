parasails.registerPage('apple-page', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
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
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
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
