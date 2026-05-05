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

parasails.registerComponent('rituals', {
  //  ╔═╗╦═╗╔═╗╔═╗╔═╗
  //  ╠═╝╠╦╝║ ║╠═╝╚═╗
  //  ╩  ╩╚═╚═╝╩  ╚═╝
  props: [
    'rituals',
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
  <p>Need to add a ritual? Here's a <a href="https://us-65885.app.gong.io/call?id=762393910108301882&email_type=collaborate-invitation-sent&xtid=59gug817r4n09e81ftf" target="_blank">video overview</a> of the process.</p>
    <table class="table table-responsive">
      <thead>
        <tr>
          <td>Task name</td>
          <td>Started on</td>
          <td>Frequency</td>
          <td>DRI</td>
          <td>Description</td>
        </tr>
      </thead>
      <tbody>
        <tr v-for="ritual in rituals">
          <td>{{ritual.task}}</td>
          <td>{{ritual.startedOn}}</td>
          <td>{{ritual.frequency}}</td>
          <td>{{ritual.dri}}</td>
          <td purpose="ritual-description" v-if="!ritual.moreInfoUrl">{{ritual.description}}</td>
          <td purpose="ritual-description" v-else><a :href="ritual.moreInfoUrl">{{ritual.description}}</a></td>
        </tr>
      </tbody>
    </table>
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
