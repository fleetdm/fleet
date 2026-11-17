/**
 * <teams-chat-graphic>
 * -----------------------------------------------------------------------------
 * An animated Microsoft Teams chat with Fleet (CSS-only animation).
 *
 * @type {Component}
 * -----------------------------------------------------------------------------
 */

parasails.registerComponent('teamsChatGraphic', {
  //  ╔═╗╦═╗╔═╗╔═╗╔═╗
  //  ╠═╝╠╦╝║ ║╠═╝╚═╗
  //  ╩  ╩╚═╚═╝╩  ╚═╝
  props: [],

  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: function() {
    return {};
  },

  //  ╦ ╦╔╦╗╔╦╗╦
  //  ╠═╣ ║ ║║║║
  //  ╩ ╩ ╩ ╩ ╩╩═╝
  template: `
  <div purpose="teams-chat-graphic" id="teams-chat-graphic-component">
    <div purpose="teams-animation" role="img" aria-label="A Microsoft Teams chat with Fleet. An admin asks what it takes to meet CIS Level 2 on their Windows devices. Fleet reports that 2,184 of 2,350 Windows hosts pass Level 1, that Level 2 adds 38 settings, and that the biggest gaps are Credential Guard, telemetry limits, and unneeded services, then offers to propose changes to close those gaps. The admin says yes, and Fleet proposes 3 new profiles and 38 policies for their review.">
      <div purpose="teams-window" aria-hidden="true">
        <picture purpose="teams-chrome">
          <source media="(max-width: 767px)" srcset="/images/teams-chat-window-mobile-492x576.svg">
          <source media="(max-width: 991px)" srcset="/images/teams-chat-window-tablet-844x576.svg">
          <img alt="" src="/images/teams-chat-window-1044x520.svg">
        </picture>
        <div purpose="teams-messages">
          <div purpose="teams-message-stack">
            <div purpose="teams-first-message">
              <p><strong>@Fleet</strong> what do we need for CIS level 2 on our Windows devices?</p>
            </div>
            <div purpose="teams-first-reply">
              <div purpose="teams-typing">
                <img purpose="teams-avatar" alt="" src="/images/icon-fleet-avatar-28x28.svg">
                <div purpose="teams-typing-bubble"><span></span><span></span><span></span></div>
              </div>
              <div purpose="teams-reply">
                <img purpose="teams-avatar" alt="" src="/images/icon-fleet-avatar-28x28.svg">
                <div purpose="teams-reply-body">
                  <span purpose="teams-sender">Fleet</span>
                  <div purpose="teams-reply-bubble">
                    <p>2,184 of 2,350 Windows hosts pass Level 1. Level 2 adds 38 settings. The biggest gaps are Credential Guard, telemetry limits, and unneeded services.</p>
                  </div>
                  <div purpose="teams-follow-up">
                    <div purpose="teams-reply-bubble">
                      <p>Want me to propose changes that close those gaps?</p>
                    </div>
                  </div>
                </div>
              </div>
            </div>
            <div purpose="teams-second-message">
              <p>Yes</p>
            </div>
            <div purpose="teams-second-reply">
              <div purpose="teams-typing">
                <img purpose="teams-avatar" alt="" src="/images/icon-fleet-avatar-28x28.svg">
                <div purpose="teams-typing-bubble"><span></span><span></span><span></span></div>
              </div>
              <div purpose="teams-reply">
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
  `,

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function() {
    //…
  },
  beforeDestroy: function() {
    //…
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    //…
  }
});
