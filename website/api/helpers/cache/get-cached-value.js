module.exports = {


  friendlyName: 'Get cached value',


  description: 'Retrieve a value from the app\'s cache, or undefined if no unexpired value is cached under the specified key.',


  extendedDescription: 'In production, values are stored in Redis (reusing the session store\'s connection), so cached values are shared between all servers running the website. If no Redis connection is available (e.g. in local development), values are stored in this Node process\'s memory instead.',


  sideEffects: 'cacheable',


  inputs: {
    key: {
      type: 'string',
      required: true,
      description: 'The key the value being retrieved is stored under.',
    },
  },


  exits: {

    success: {
      outputFriendlyName: 'Cached value',
      outputDescription: 'The cached value, or undefined if no unexpired value is cached under the specified key.',
    },

  },


  fn: async function ({key}) {

    // In production, the session hook stores its connected Redis client in `sails.config.session.client`.
    let redisClient = sails.config.session.client;
    if(redisClient) {
      // Note: cache keys are prefixed with "cache:" to keep them separate from session data ("sess:*") stored in the same Redis database.
      let serializedValue = await new Promise((resolve, reject)=>{
        redisClient.get(`cache:${key}`, (err, result)=>{
          if(err) {
            return reject(err);
          }
          return resolve(result);
        });
      }).catch((err)=>{
        // If a cached value could not be retrieved, log a warning and treat it as a cache miss rather than letting a caching error prevent the caller from continuing.
        sails.log.warn(`Could not retrieve a cached value (key: ${key}) from Redis. Full error: ${require('util').inspect(err)}`);
        return null;
      });
      if(!serializedValue) {
        return undefined;
      } else {
        console.log(serializedValue);
      }
      try {
        return JSON.parse(serializedValue);
      } catch(unusedErr) {
        // If the cached value could not be parsed, treat it as a cache miss.
        return undefined;
      }
    } else {
      if(!sails.inMemoryCache || !sails.inMemoryCache[key]) {
        return undefined;
      }
      let cacheEntry = sails.inMemoryCache[key];
      if(cacheEntry.expiresAt <= Date.now()) {
        delete sails.inMemoryCache[key];
        return undefined;
      }
      return cacheEntry.value;
    }

  }


};
