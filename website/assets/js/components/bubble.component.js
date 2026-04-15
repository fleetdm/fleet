/**
 * <bubble>
 * -----------------------------------------------------------------------------
 * A styled span used in documentation.
 *
 * @type {Component}
 *
 * @event click   [emitted when clicked]
 * -----------------------------------------------------------------------------
 */

parasails.registerComponent('bubble', {
  //  ╔═╗╦═╗╔═╗╔═╗╔═╗
  //  ╠═╝╠╦╝║ ║╠═╝╚═╗
  //  ╩  ╩╚═╚═╝╩  ╚═╝
  props: [
    'type',
  ],

  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: function (){
    let rawType = this.type ? this.type.replace(/\?$/, '').toLowerCase() : '';
    let roleLink = '';

    switch (this.type.toLowerCase()) {
      case 'admin':
        roleLink = '/guides/role-based-access#admin'
        break;
      case 'maintainer':
        roleLink = '/guides/role-based-access#maintainer'
        break;
      case 'observer':
        roleLink = '/guides/role-based-access#observer'
        break;
      case 'observer+':
        roleLink = '/guides/role-based-access#observer2'
        rawType = 'observer-plus';
        break;
      case 'technician':
        roleLink = '/guides/role-based-access#technician'
        break;
      case 'gitops':
        roleLink = '/guides/role-based-access#gitops'
        break;
    }

    return {
      rawType: rawType,
      isUncertain: this.type ? this.type.match(/\?$/g) ? true : false : '',
      roleLink: roleLink
    };
  },

  //  ╦ ╦╔╦╗╔╦╗╦
  //  ╠═╣ ║ ║║║║
  //  ╩ ╩ ╩ ╩ ╩╩═╝
  template: `
    <a v-if="roleLink" class="role-link" :href="roleLink">
      <span purpose="bubble-heart" :class="rawType+' '+[[isUncertain ? 'uncertain' : '']]" class="">{{type}}</span>
    </a>
    <span v-else>
      <span purpose="bubble-heart" :class="rawType+' '+[[isUncertain ? 'uncertain' : '']]" class="">{{type}}</span>
    </span>
  `,

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    if(this.type === undefined){
      throw new Error(`Incomplete usage of <bubble>: Please provide a 'type' that will be displayed as text inside the bubble. e.g., <bubble type="Observer"></bubble>`);
    }
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
    //…
  }
});
