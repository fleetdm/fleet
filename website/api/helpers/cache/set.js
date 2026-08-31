module.exports = {


  friendlyName: 'Set',


  description: 'Store a JSON-serializable value in the app\'s cache under a specified key, with a TTL.',


  extendedDescription: 'In production, values are stored in Redis (reusing the session store\'s connection), so cached values are shared between all servers running the website. If no Redis connection is available (e.g. in local development), values are stored in this Node process\'s memory instead.',


  sideEffects: 'idempotent',


  inputs: {
    key: {
      type: 'string',
      required: true,
      description: 'The key to store the provided value under.',
    },
    value: {
      type: 'json',
      required: true,
      description: 'The value to store in the cache.',
    },
    ttl: {
      type: 'number',
      required: true,
      description: 'The number of seconds the provided value will stay in the cache.',
    },
  },


  exits: {

    success: {
      description: 'The provided value was stored in the cache.',
    },

  },


  fn: async function ({key, value, ttl}) {

    let ttlInSeconds = Math.floor(ttl);
    if(ttlInSeconds < 1) {
      return;
    }

    // In production, the session hook stores its connected Redis client in `sails.config.session.client`.
    let redisClient = sails.config.session.client;
    if(redisClient) {
      // Note: cache keys are prefixed with "cache:" to keep them separate from session data ("sess:*") stored in the same Redis database.
      await new Promise((resolve, reject)=>{
        redisClient.setex(`cache:${key}`, ttlInSeconds, JSON.stringify(value), (err)=>{
          if(err) {
            return reject(err);
          }
          return resolve();
        });
      }).catch((err)=>{
        // If the value could not be cached, log a warning and continue rather than letting a caching error prevent the caller from continuing.
        sails.log.warn(`Could not store a cached value (key: ${key}) in Redis. Full error: ${require('util').inspect(err)}`);
      });
    } else {
      if(!sails.inMemoryCache) {
        sails.inMemoryCache = {};
      }
      sails.inMemoryCache[key] = {
        value: value,
        expiresAt: Date.now() + ttlInSeconds * 1000,
      };
    }

  }


};
