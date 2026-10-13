// On pages that have an inline demo form (`#demo-form`), turn every "Get a demo"
// link (including the ones in the site header) into a scroll to that form,
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
    let prefersReducedMotion = window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    demoForm.scrollIntoView({ behavior: prefersReducedMotion ? 'auto' : 'smooth', block: 'center' });
    let firstInput = demoForm.querySelector('input');
    if(firstInput) { firstInput.focus({ preventScroll: true }); }
  });
})();
