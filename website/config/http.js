/**
 * HTTP Server Settings
 * (sails.config.http)
 *
 * Configuration for the underlying HTTP server in Sails.
 * (for additional recommended settings, see `config/env/production.js`)
 *
 * For more information on configuration, check out:
 * https://sailsjs.com/config/http
 */

module.exports.http = {

  /****************************************************************************
  *                                                                           *
  * Sails/Express middleware to run for every HTTP request.                   *
  * (Only applies to HTTP requests -- not virtual WebSocket requests.)        *
  *                                                                           *
  * https://sailsjs.com/documentation/concepts/middleware                     *
  *                                                                           *
  ****************************************************************************/

  middleware: {

    /***************************************************************************
    *                                                                          *
    * The order in which middleware should be run for HTTP requests.           *
    * (This Sails app's routes are handled by the "router" middleware below.)  *
    *                                                                          *
    ***************************************************************************/

    order: [
      'cookieParser',
      'session',
      'bodyParser',
      'compress',
      'poweredBy',
      'router',
      'www',
      'favicon',
      'middlewareErrorHandler'
    ],


    /***************************************************************************
    *                                                                          *
    * The body parser that will handle incoming multipart HTTP requests.       *
    *                                                                          *
    * https://sailsjs.com/config/http#?customizing-the-body-parser             *
    *                                                                          *
    ***************************************************************************/

    bodyParser: (function _configureBodyParser(){
      var skipper = require('skipper');
      var middlewareFn = skipper({
        strict: true,
        limit: '10MB',// [?] https://github.com/expressjs/body-parser/tree/ee91374eae1555af679550b1d2fb5697d9924109#limit-1
        onBodyParserError: (err, req, res)=>{
          // If an error occurs while parsing an incoming request body, we'll return a badRequest response if error.statusCode is between 400-500
          if (_.isNumber(err.statusCode) && err.statusCode >= 400 && err.statusCode < 500) {
            return res.status(400).send(err.message);
          } else {
            sails.log.error('Sending 500 ("Server Error") response: \n', err);
            return res.status(500).send();
          }
        }
      });
      return middlewareFn;
    })(),

    // Note: this middleware function will run for every HTTP request, but will only handle errors thrown by the serve-static middleware if a user requests an invalid byte range of a static asset.
    middlewareErrorHandler: function(err, req, res, next) {
      // If this is a 'RangeNotSatisfiableError' error, respond with a 416 status code.
      if (err.message === 'Range Not Satisfiable') {
        return res.status(416).send();
      } else {
        return next(err);
      }
    },

  },

};
