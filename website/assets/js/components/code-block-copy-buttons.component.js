/**
 * <code-block-copy-buttons>
 * -----------------------------------------------------------------------------
 * A hovering copy button injected into code blocks that copies the text content when clicked.
 *
 * @type {Component}
 *
 * -----------------------------------------------------------------------------
 */

parasails.registerComponent('code-block-copy-buttons', {
  //  ╦ ╦╔╦╗╔╦╗╦
  //  ╠═╣ ║ ║║║║
  //  ╩ ╩ ╩ ╩ ╩╩═╝
  template: `<span></span>`,

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  mounted: async function(){
    // Only add copy buttons if the clipboard method is available.
    if(typeof navigator.clipboard !== undefined){
      // Prepend all <pre> elements with a code-block-copy-button elment, and add the position-relative bootstrap class.
      $('pre')
      .prepend('<div purpose="code-block-copy-button"></div>')
      .addClass('has-copy-button');// Note: we set this bootstrap class to correctly position the copy button without needing to adjust the styles of the page it is added to.

      // Now add click events to each code-block-copy-button element that copies the text content of the neighboring <code> element.
      $('[purpose="code-block-copy-button"]').on('click', async function() {
        let code = $(this).closest('pre').find('code').text();
        // Add the copied class to the copy button (which replaces the icon with a checkmark).
        $(this).addClass('copied');
        // Remove the copied class after 2 seconds.
        await setTimeout(()=>{
          $(this).removeClass('copied');
        }, 2000);
        navigator.clipboard.writeText(code);
      });
    }
  },
});
