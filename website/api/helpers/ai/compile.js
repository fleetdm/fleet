module.exports = {


  friendlyName: 'Compile',


  description: 'Automatically generate code for use in a Sails app based on a human specification.',


  extendedDescription: '',


  inputs: {

    humanSpecification: {
      required: true,
      type: 'string',
      example: 'Sign up: Handle a signup form by creating a user in the database and set the `.userId` key in the session.',
    },

    purpose: {
      type: 'string',
      isIn: ['action', 'helper'],
      defaultsTo: 'action'
    },

  },


  exits: {

    success: {
      outputFriendlyName: 'Code file',
      outputDescription: 'The code for a Sails action or helper that implements the given human specification.',
      extendedDescription: 'The generated code is formatted for Sails v1 and above (aka using "actions2", aka the node-machine spec).'
    },

  },


  fn: async function ({ humanSpecification, purpose }) {

    return await ƒ.prompt.with({
      baseModel: 'o4-mini-2025-04-16',
      prompt:
        'Generate code for a sails app '+
          (purpose === 'action' ?
            'action (actions2 in sails v1+)'
            : 'helper (in sails v1+)'
          )+
        ' to accomplish the following specification:\n```\n'+
        humanSpecification+
        '\n```\nRespond only with pure JavaScript code, without code fences.  It is ok to use try/catch as needed, but never use .catch().  When it helps result in less code with equivalent rigor, take advantage of .intercept() or .tolerate() for error handling.  (Remember never to throw inside of the intercept or tolerate functions - instead for .intercept(), return the error to throw, and for .tolerate(), return the value the function should return)  Do not make a separate exit for 5xx server errors-- instead just throw to take advatage of the built-in error exit.  Do not include a second `exits` argument to `fn` -- instead just return or throw.  Instead of using the `inputs` argument to `fn`, specify it using syntax like `fn: ({ foo, bar })=>{ /* implementation */ }.  If writing or reading to the session, use the convention of `req.session.userId`.  Never use `fetch` for making outgoing HTTP requests -- instead, use sails.helpers.http if needed.',
    });
  }


};

