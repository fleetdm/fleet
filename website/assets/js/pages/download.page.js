parasails.registerPage('download', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    //…
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function() {
    if(typeof navigator.clipboard !== 'undefined' && typeof navigator.clipboard.writeText === 'function') {
      // Add a click event to the copy-button element that copies the text content of the neighboring <code> element.
      $('[purpose="copy-button"]').on('click', async function() {
        // Get the text content of the closest <code> element to the copy button.
        let code = $(this).closest('[purpose="codeblock"]').find('code').text();
        // Add the copied class to the copy button (which replaces the icon with a checkmark).
        $(this).addClass('copied');
        // Remove the copied class after 2 seconds.
        setTimeout(()=>{
          $(this).removeClass('copied');
        }, 2000);
        navigator.clipboard.writeText(code);
      });
    } else {
      // If the navigator.clipboard.writeText method is not available, remove the copy button.
      $('[purpose="copy-button"]').remove();
    }
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    //…
  }
});
