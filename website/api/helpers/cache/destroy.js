module.exports = {


  friendlyName: 'Destroy',


  description: 'Remove a value from the app\'s cache.',


  sideEffects: 'idempotent',


  inputs: {
    key: {
      type: 'string',
      required: true,
      description: 'The key of the cached value to remove.',
    },
  },


  exits: {

    success: {
      description: 'Any value cached under the specified key was removed.',
    },

  },


  fn: async function ({key}) {

    // In production, the session hook stores its connected Redis client in `sails.config.session.client`.
    let redisClient = sails.config.session.client;
    if(redisClient) {
      // Note: cache keys are prefixed with "cache:" to keep them separate from session data ("sess:*") stored in the same Redis database.
      await new Promise((resolve, reject)=>{
        redisClient.del(`cache:${key}`, (err)=>{
          if(err) {
            return reject(err);
          }
          return resolve();
        });
      }).catch((err)=>{
        // If the cached value could not be removed, log a warning and continue rather than letting a caching error prevent the caller from continuing.
        sails.log.warn(`Could not remove a cached value (key: ${key}) from Redis. Full error: ${require('util').inspect(err)}`);
      });
    } else if(sails.inMemoryCache) {
      delete sails.inMemoryCache[key];
    }

  }


};
