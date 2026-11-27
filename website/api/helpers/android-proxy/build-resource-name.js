module.exports = {


  friendlyName: 'Build resource name',


  description: 'Builds the Android Management API resource name for a resource that belongs to an Android enterprise.',


  inputs: {
    androidEnterpriseId: {
      type: 'string',
      required: true,
      description: 'The ID of the Android enterprise that the resource belongs to.'
    },
    segments: {
      type: ['string'],
      required: true,
      description: 'Alternating collection names and resource IDs.',
      example: ['devices', '3a1b2c3d4e5f', 'operations', '1700000000000'],
    },
  },


  exits: {
    success: {
      outputFriendlyName: 'Resource name',
      outputType: 'string',
    },
    invalidResourceId: {
      description: 'A provided segment is not a valid resource ID.'
    },
  },


  fn: async function ({androidEnterpriseId, segments}) {

    // Each segment must be exactly one path segment of the resource name.
    for (let segment of segments) {
      if (!/^[A-Za-z0-9_.-]+$/.test(segment) || segment === '.' || segment === '..') {
        throw 'invalidResourceId';
      }
    }

    return `enterprises/${androidEnterpriseId}/${segments.join('/')}`;

  }


};
