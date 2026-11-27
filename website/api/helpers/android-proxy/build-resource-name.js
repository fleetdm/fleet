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
      description: 'Pairs of collection name and resource ID.',
      example: ['devices', '3a1b2c3d4e5f', 'operations', '1700000000000'],
    },
  },


  exits: {
    success: {
      outputFriendlyName: 'Resource name',
      outputType: 'string',
    },
    invalidResourceId: {
      description: 'A provided ID is not a valid resource ID.'
    },
  },


  fn: async function ({androidEnterpriseId, segments}) {

    if (segments.length === 0 || segments.length % 2 !== 0) {
      throw new Error(`Expected pairs of collection name and resource ID, but got ${segments.length} segment(s).`);
    }

    // Every part must be exactly one path segment of the resource name.
    for (let part of [androidEnterpriseId, ...segments]) {
      if (!/^[A-Za-z0-9_.-]+$/.test(part) || part === '.' || part === '..') {
        throw 'invalidResourceId';
      }
    }

    return `enterprises/${androidEnterpriseId}/${segments.join('/')}`;

  }


};
