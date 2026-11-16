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
    let rawType = this.type ? this.type.toLowerCase() : '';
    if (rawType.includes('token') || rawType.includes('public')) {
      rawType = 'public';
    }
    // "Global <role>" bubbles mark endpoints that fleet-level users with that role can't use.
    if (rawType.startsWith('global ')) {
      rawType = rawType.replace('global ', '');
    }
    // "+" isn't valid in a CSS class name.
    if (rawType === 'observer+') {
      rawType = 'observer-plus';
    }
    let roleLink = '';

    switch (rawType) {
      case 'admin':
      case 'maintainer':
      case 'observer':
      case 'observer-plus':
      case 'technician':
      case 'gitops':
        roleLink = '/guides/role-based-access#roles';
        break;
    }

    return {
      rawType: rawType,
      roleLink: roleLink
    };
  },

  //  ╦ ╦╔╦╗╔╦╗╦
  //  ╠═╣ ║ ║║║║
  //  ╩ ╩ ╩ ╩ ╩╩═╝
  template: `
    <a v-if="roleLink" class="role-link" :href="roleLink">
      <span purpose="bubble-heart" :class="rawType">{{type}}</span>
    </a>
    <span v-else>
      <span purpose="bubble-heart" :class="rawType">{{type}}</span>
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
