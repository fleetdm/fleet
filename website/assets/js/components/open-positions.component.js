/**
 * <rituals>
 * -----------------------------------------------------------------------------
 *
 *
 * @type {Component}
 *
 *
 * -----------------------------------------------------------------------------
 */

parasails.registerComponent('open-positions', {
  //  ╔═╗╦═╗╔═╗╔═╗╔═╗
  //  ╠═╝╠╦╝║ ║╠═╝╚═╗
  //  ╩  ╩╚═╚═╝╩  ╚═╝
  props: [
    'openPositions',
  ],

  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: function (){
    return {
      //…
    };
  },

  //  ╦ ╦╔╦╗╔╦╗╦
  //  ╠═╣ ║ ║║║║
  //  ╩ ╩ ╩ ╩ ╩╩═╝
  template: `
  <div>
    <div v-if="openPositions.length > 0">
      <p>Fleet is currently hiring for the following positions:</p>
      <ul >
        <li v-for="position in openPositions">
          <a :href="position.url">{{position.jobTitle}}</a>
        </li>
      </ul>
      <blockquote purpose="tip">
        <img src="/images/icon-info-16x16@2x.png" alt="An icon indicating that this section has important information">
        <div class="d-block">
          <p>
            <strong>🛸 Join us!</strong> &nbsp;Interested in joining the team at Fleet, or know someone who might be?  Click one of the positions to read the job description and apply.  Or <a href="/handbook/company#open-positions">copy a direct link to this page</a> to share a short summary about the company, including our vision, values, history, and all currently open positions.  Thank you for the help!
          </p>
        </div>
      </blockquote>
    </div>
    <div v-else>
      <blockquote purpose="tip">
        <img src="/images/icon-info-16x16@2x.png" alt="An icon indicating that this section has important information">
        <div class="d-block">
          <p>
            Fleet currently has no open positions. Interested in changing our mind? <a target="_blank" href="/contact#message">Message us your Linkedin profile</a>.
          </p>
        </div>
      </blockquote>
    </div>
  </div>
  `,

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function(){
    //…
  },
  beforeDestroy: function() {
    //…
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {


  }
});
