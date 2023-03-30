module.exports = {


  friendlyName: 'View launch party',


  description: 'Display "Launch party" page.',


  inputs: {
    showForm: {
      type: 'boolean',
      description: 'An optional boolean that if provided with other',
      defaultsTo: false
    },
    emailAddress: {
      type: 'string',
      description: 'If provided, this value will be used to prefill the emailAddress field in the waitlist form'
    },
    firstName: {
      type: 'string',
      description: 'If provided, this value will be used to prefill the first name field in the waitlist form'
    },
    lastName: {
      type: 'string',
      description: 'If provided, this value will be used to prefill the last name field in the waitlist form'
    }
  },


  exits: {

    success: {
      viewTemplatePath: 'pages/imagine/launch-party'
    }

  },


  fn: async function ({showForm, emailAddress, firstName, lastName}) {

    let showFormOnPageLoad = showForm;

    // If form inputs are provided via query string we'll prefill the inputs in the waitlist form. (e.g., A user is coming to this page from a personalized link in an email)
    let formDataProvidedViaQueryString = {};
    if(emailAddress){// Email address will always be provided if a user is coming here from an email link.
      formDataProvidedViaQueryString.emailAddress = emailAddress;
    }
    // If the first name provided is not '?' or Outreaches first name template, we'll prefill the first name in the waitlist form
    if(firstName && firstName !== '?' && firstName !== '{{first_name}}') {
      formDataProvidedViaQueryString.firstName = firstName;
    }
    // If the last name provided is not '?' or Outreaches last name template, we'll prefill the last name in the waitlist form
    if(lastName && lastName !== '?' && lastName !== '{{last_name}}') {
      formDataProvidedViaQueryString.lastName = lastName;
    }

    // Respond with view.
    return {
      showFormOnPageLoad,
      formDataProvidedViaQueryString,
    };

  }


};
