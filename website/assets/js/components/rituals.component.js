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
  <div class="table-responsive">
    <table class="table">
      <thead>
        <tr>
          <td>Task name</td>
          <td>Started on</td>
          <td>Frequency</td>
          <td>Description</td>
          <td>DRI</td>
        </tr>
      </thead>
      <tbody>
        <tr v-for="ritual in rituals">
          <td>{{ritual.task}}</td>
          <td>{{ritual.startedOn}}</td>
          <td>{{ritual.frequency}}</td>
          <td style="max-width: 200px" v-if="!ritual.moreInfoUrl">{{ritual.description}}</td>
          <td style="max-width: 200px" v-else><a :href="ritual.moreInfoUrl">{{ritual.description}}</a></td>
          <td v-if="!Array.isArray(ritual.dri)">{{ritual.dri}}</td>
          <td v-else><p v-for="dri in ritual.dri">{{dri}}</p></td>
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
