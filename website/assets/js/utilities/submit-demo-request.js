/**
 * parasails.util.submitDemoRequest()
 *
 * Submits the "Talk to us" demo request, fires the same conversion events as
 * the form on /contact, and returns the Calendly URL to send the user to.
 *
 * @param {Dictionary} argins    The submitted form data (from <ajax-form>)
 * @returns {String}             The Calendly event URL to redirect to
 */
parasails.registerUtility('submitDemoRequest', async(argins)=>{
  if(typeof window.lintrk !== 'undefined') { window.lintrk('track', { conversion_id: 18587089 }); }// eslint-disable-line camelcase
  let report = await Cloud.deliverTalkToUsFormSubmission.with(argins);

  if(report.icp) {
    if(typeof gtag !== 'undefined') { gtag('event', 'fleet_website__contact_forms__demo__icp'); }
    if(typeof window.lintrk !== 'undefined') { window.lintrk('track', { conversion_id: 27493081 }); }// eslint-disable-line camelcase
  } else {
    if(typeof gtag !== 'undefined') { gtag('event', 'fleet_website__contact_forms__demo'); }
  }
  if(typeof gtag !== 'undefined') {
    gtag('event', 'conversion', {
      'send_to': 'AW-10788733823/aNrhCNSYrPobEP-GvJgo',
      'value': 1.0,
      'currency': 'USD'
    });
  }
  if(typeof qualified !== 'undefined') {
    qualified('saveFormData', {
      email: argins.emailAddress,
      name: argins.firstName +' '+ argins.lastName,
      company: argins.organization,
      how_many_hostsdevices_do_you_want_to_manage: argins.numberOfHosts,// eslint-disable-line camelcase
    });
    qualified('showFormExperience', 'experience-1772126772950');
  }

  return report.eventUrl;
});
