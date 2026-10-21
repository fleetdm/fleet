// On pages that have an inline demo form (`#demo-form`), turn every "Get a demo"
// link (including the ones in the site header) into a jump to that form,
// instead of sending the visitor to /contact.
(function() {
  document.addEventListener('click', (event)=>{
    if(event.defaultPrevented || event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) {
      return;
    }
    let link = event.target.closest && event.target.closest('a[href]');
    if(!link) { return; }
    let href = link.getAttribute('href');
    if(href !== '#demo-form' && href.indexOf('/contact?talkToUs') !== 0) { return; }
    let demoForm = document.getElementById('demo-form');
    if(!demoForm) { return; }

    event.preventDefault();
    if(typeof $ !== 'undefined') { $('#navbarToggleExternalContent').collapse('hide'); }// Close the mobile nav if it's open.
    let target = demoForm;
    let leadIns = document.querySelectorAll('[purpose="bottom-cta"]');
    let leadIn = leadIns.length ? leadIns[leadIns.length - 1] : null;
    let maxLeadInDistance = 800;
    if(leadIn && (leadIn.compareDocumentPosition(demoForm) & Node.DOCUMENT_POSITION_FOLLOWING) && demoForm.getBoundingClientRect().top - leadIn.getBoundingClientRect().top < maxLeadInDistance) {
      target = leadIn;
    }
    let header = document.querySelector('[purpose="header-container"]');
    let headerHeight = header ? header.getBoundingClientRect().height : 0;
    let breathingRoom = 24;
    window.scrollTo({ top: target.getBoundingClientRect().top + window.pageYOffset - headerHeight - breathingRoom, behavior: 'auto' });
    let firstInput = demoForm.querySelector('input');
    if(firstInput) { firstInput.focus({ preventScroll: true }); }
  });
})();
