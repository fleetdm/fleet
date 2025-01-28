/**
 * parasails.js
 * (lightweight structures for apps with more than one page)
 *
 * v0.9.3
 *
 * Copyright 2014-present, Mike McNeil (@mikermcneil)
 * MIT License
 *
 * - https://sailsjs.com/about
 * - https://sailsjs.com/support
 * - https://www.npmjs.com/package/parasails
 *
 * > Parasails is a tiny (but opinionated) and pipeline-agnostic wrapper
 * > around Vue.js and Lodash, with optional participation from jQuery, bowser,
 * > and VueRouter.
 */
(function(factory, exposeUMD){
  exposeUMD(this, factory);
})(function (Vue, _, VueRouter, $, bowser){

  //  ██████╗ ██████╗ ██╗██╗   ██╗ █████╗ ████████╗███████╗
  //  ██╔══██╗██╔══██╗██║██║   ██║██╔══██╗╚══██╔══╝██╔════╝
  //  ██████╔╝██████╔╝██║██║   ██║███████║   ██║   █████╗
  //  ██╔═══╝ ██╔══██╗██║╚██╗ ██╔╝██╔══██║   ██║   ██╔══╝
  //  ██║     ██║  ██║██║ ╚████╔╝ ██║  ██║   ██║   ███████╗
  //  ╚═╝     ╚═╝  ╚═╝╚═╝  ╚═══╝  ╚═╝  ╚═╝   ╚═╝   ╚══════╝
  //
  //  ███████╗████████╗ █████╗ ████████╗███████╗
  //  ██╔════╝╚══██╔══╝██╔══██╗╚══██╔══╝██╔════╝
  //  ███████╗   ██║   ███████║   ██║   █████╗
  //  ╚════██║   ██║   ██╔══██║   ██║   ██╔══╝
  //  ███████║   ██║   ██║  ██║   ██║   ███████╗
  //  ╚══════╝   ╚═╝   ╚═╝  ╚═╝   ╚═╝   ╚══════╝
  //

  /**
   * Module state
   */

  // Keep track of whether or not a page script has already been loaded in the DOM.
  var didAlreadyLoadPageScript;

  // The variable we'll be exporting.
  var parasails;


  //  ██████╗ ██████╗ ██╗██╗   ██╗ █████╗ ████████╗███████╗
  //  ██╔══██╗██╔══██╗██║██║   ██║██╔══██╗╚══██╔══╝██╔════╝
  //  ██████╔╝██████╔╝██║██║   ██║███████║   ██║   █████╗
  //  ██╔═══╝ ██╔══██╗██║╚██╗ ██╔╝██╔══██║   ██║   ██╔══╝
  //  ██║     ██║  ██║██║ ╚████╔╝ ██║  ██║   ██║   ███████╗
  //  ╚═╝     ╚═╝  ╚═╝╚═╝  ╚═══╝  ╚═╝  ╚═╝   ╚═╝   ╚══════╝
  //
  //  ██╗   ██╗████████╗██╗██╗     ███████╗
  //  ██║   ██║╚══██╔══╝██║██║     ██╔════╝
  //  ██║   ██║   ██║   ██║██║     ███████╗
  //  ██║   ██║   ██║   ██║██║     ╚════██║
  //  ╚██████╔╝   ██║   ██║███████╗███████║
  //   ╚═════╝    ╚═╝   ╚═╝╚══════╝╚══════╝
  //

  /**
   * Module utilities (private)
   */

  function _ensureGlobalCache(){
    parasails._cache = parasails._cache || {};
  }

  function _exportOnGlobalCache(moduleName, moduleDefinition){
    _ensureGlobalCache();
    if (parasails._cache[moduleName]) { throw new Error('Something else (e.g. a utility or constant) has already been registered under that name (`'+moduleName+'`)'); }
    parasails._cache[moduleName] = moduleDefinition;
  }

  function _exposeBonusMethods(def, currentModuleEntityNoun){
    if (!currentModuleEntityNoun) { throw new Error('Consistency violation: Bad internal usage. '); }
    if (def.methods && def.methods.$get) { throw new Error('This '+currentModuleEntityNoun+' contains `methods` with a `$get` key, but you\'re not allowed to override that'); }
    if (def.methods && def.methods.$find) { throw new Error('This '+currentModuleEntityNoun+' contains `methods` with a `$find` key, but you\'re not allowed to override that'); }
    if (def.methods && def.methods.$focus) { throw new Error('This '+currentModuleEntityNoun+' contains `methods` with a `$focus` key, but you\'re not allowed to override that'); }
    if (def.methods && def.methods.forceRender) { throw new Error('This '+currentModuleEntityNoun+' contains `methods` with a `forceRender` key, but you\'re not allowed to override that'); }
    if (def.methods && def.methods.$forceRender) { throw new Error('This '+currentModuleEntityNoun+' contains `methods` with a `$forceRender` key, but that\'s too confusing to let stand (did you mean "forceRender"?  Besides, that method cannot be overridden anyway)'); }
    def.methods = def.methods || {};

    // Attach misc. methods:
    def.methods.forceRender = function (){
      this.$forceUpdate();
      var promise = this.$nextTick();
      return promise;
    };//ƒ


    // Attach jQuery-powered methods:
    if ($) {
      def.methods.$get = function (){
        var $rootEl = $(this.$el);
        if ($rootEl.length !== 1) { throw new Error('Cannot use .$get() - something is wrong with this '+currentModuleEntityNoun+'\'s top-level DOM element.  (It probably has not mounted yet!)'); }
        return $rootEl;
      };
      def.methods.$find = function (subSelector){
        if (!subSelector) { throw new Error('Cannot use .$find() because no sub-selector was provided.\nExample usage:\n    var $emailFields = this.$find(\'[name="emailAddress"]\');'); }
        var $rootEl = $(this.$el);
        if ($rootEl.length !== 1) { throw new Error('Cannot use .$find() - something is wrong with this '+currentModuleEntityNoun+'\'s top-level DOM element.  (It probably has not mounted yet!)'); }
        return $rootEl.find(subSelector);
      };
      def.methods.$focus = function (subSelector){
        if (!subSelector) { throw new Error('Cannot use .$focus() because no sub-selector was provided.\nExample usage:\n    this.$focus(\'[name="emailAddress"]\');'); }
        var $rootEl = $(this.$el);
        if ($rootEl.length !== 1) { throw new Error('Cannot use .$focus() - something is wrong with this '+currentModuleEntityNoun+'\'s top-level DOM element.  (It probably has not mounted yet!)'); }
        var $fieldToAutoFocus = $rootEl.find(subSelector);
        if ($fieldToAutoFocus.length === 0) { throw new Error('Could not autofocus-- no such element exists within this '+currentModuleEntityNoun+'.'); }
        // FUTURE: ^^ if that happens, try calling await this.forceRender() and then try again one more time before giving up
        if ($fieldToAutoFocus.length > 1) { throw new Error('Could not autofocus `'+subSelector+'`-- too many elements matched!'); }
        $fieldToAutoFocus.focus();
      };
    }
    else {
      def.methods.$get = function (){ throw new Error('Cannot use .$get() method because, at the time when this '+currentModuleEntityNoun+' was registered, jQuery (`$`) did not exist on the page yet.  (If you\'re using Sails, please check dependency loading order in pipeline.js and make sure jQuery is getting brought in before `parasails`.)'); };
      def.methods.$find = function (){ throw new Error('Cannot use .$find() method because, at the time when this '+currentModuleEntityNoun+' was registered, jQuery (`$`) did not exist on the page yet.  (If you\'re using Sails, please check dependency loading order in pipeline.js and make sure jQuery is getting brought in before `parasails`.)'); };
      def.methods.$focus = function (){ throw new Error('Cannot use .$focus() method because, at the time when this '+currentModuleEntityNoun+' was registered, jQuery (`$`) did not exist on the page yet.  (If you\'re using Sails, please check dependency loading order in pipeline.js and make sure jQuery is getting brought in before `parasails`.)'); };
    }
  }

  function _wrapMethodsAndVerifyNoArrowFunctions(def, currentModuleEntityNoun){
    if (!currentModuleEntityNoun) { throw new Error('Consistency violation: Bad internal usage. '); }

    // Preliminary sanity check:
    // Make sure top-level def doesn't have anything sketchy like "beforeMounted"
    // or "beforeDestroyed", because those definitely aren't real things.
    var RECOMMENDATIONS_BY_UNRECOGNIZED_KEY = {
      beforeMounted: 'beforeMount',
      beforeMounting: 'beforeMount',
      beforeDestroyed: 'beforeDestroy',
      beforeDestroying: 'beforeDestroy',
      events: 'methods',
      functions: 'methods',
      state: 'data',
      virtualPageRegExp: 'virtualPagesRegExp',
      virtualPageRegEx: 'virtualPagesRegExp',
      virtualPagesRegEx: 'virtualPagesRegExp',
      virtualPage: 'virtualPages',
      html5History: 'html5HistoryMode',
      historyMode: 'html5HistoryMode',
    };
    // > Note that this determination of whether to show a more precise
    // > "Did you mean?" error message is a case-_insensitive_ check.
    var lowercasedRecommendationsByKey = _.reduce(RECOMMENDATIONS_BY_UNRECOGNIZED_KEY, function(memo, correctAlias, incorrectKey){
      memo[incorrectKey.toLowerCase()] = correctAlias;
      return memo;
    }, {});
    _.each(def, function (x, propertyName) {
      if (x !== undefined) {
        if (_.contains(_.keys(RECOMMENDATIONS_BY_UNRECOGNIZED_KEY), propertyName) || _.contains(_.keys(lowercasedRecommendationsByKey), propertyName.toLowerCase())) {
          throw new Error('Detected unrecognized and potentially confusing key "'+propertyName+'" on the top level of '+currentModuleEntityNoun+' definition.  Did you mean "'+lowercasedRecommendationsByKey[propertyName.toLowerCase()]+'"?');
        }
      }
    });//∞

    // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
    // FUTURE: Maybe verify that neither beforeMount nor beforeDestroy are
    // `async function`s.  (These must be synchronous!  And it's easy to forget.)
    // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

    // In fact, in some cases, we'll go so far as to fail if we see any other
    // unrecognized top-level keys too:
    // > This is particularly useful for catching loose top-level properties
    // > that were intended to be within `data` or `methods`, etc.)
    if (currentModuleEntityNoun === 'page script' || currentModuleEntityNoun === 'component') {
      // FUTURE: don't allow page script only things on components

      var LEGAL_TOP_LVL_KEYS = [
        // Everyday page script stuff:
        'beforeMount',
        'mounted',
        'data',
        'methods',

        // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
        // FUTURE: Add `this.listen()` and `this.ignore()` -- see:
        // https://github.com/mikermcneil/parasails/commit/5b948a1a8a0945b19ccea6da3e7354255d3dc0b6
        // (and also 83c439dc1f3a0375e67066fffe9151581cbab639)
        //
        // Or better yet, just use a declarative approach instead and let
        // binding/unbinding happen behind the scenes.  e.g.:
        // ```
        // cloudEvents: {
        //   pet: (msg)=>{…},
        //   user: (msg)=>{…},
        //   organization: {
        //     destroyed: (msg)=>{…},
        //     updated: (msg)=>{…},
        //     bankWireReceived: (msg)=>{…},
        //     error: (msg)=>{…}
        //   },
        //   circus: {
        //     admissionPaymentReceived: (msg)=>{…}
        //   }
        // }
        // ```
        // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

        // Extra component stuff:
        'props',
        'template',
        'beforeDestroy',

        // Client-side router stuff:
        'router',
        'virtualPages',
        'html5HistoryMode',
        'beforeNavigate',
        'afterNavigate',
        'virtualPagesRegExp',

        // Misc. & relatively more uncommon Vue.js stuff
        'watch',
        'computed',
        'propsData',
        'components',
        'filters',
        'directives',
        'el',
        'render',
        'renderError',
        'comments',
        'inheritAttrs',
        'model',
        'functional',
        'delimiters',
        'name',
        'beforeCreate',
        'created',
        'beforeUpdate',
        'updated',
        'activated',
        'deactivated',
        'destroyed',
        'errorCaptured',
        'parent',
        'mixins',
        'extends',
        'provide',
        'inject'
      ];
      // FUTURE: change this to a case-insensitive check to do a better job helping
      // out a user who is trying to use e.g. "beforemount", without a capital "M"
      _.each(_.difference(_.keys(def), LEGAL_TOP_LVL_KEYS), function (propertyName) {
        if (def[propertyName] !== undefined) {
          throw new Error('Detected unrecognized key "'+propertyName+'" on the top level of '+currentModuleEntityNoun+' definition.  Did you perhaps intend for `'+propertyName+'` to be included as a nested key within `data` or `methods`?  Please check on that and try again.  If you\'re unsure, or you\'re deliberately attempting to use a Vue.js feature that relies on having a top-level property named `'+propertyName+'`, then please remove this check from the parasails.js library in your project, or drop by https://sailsjs.com/support for assistance.');
        }
      });//∞
    }//ﬁ

    // Mix in overridable default filters
    def.filters = def.filters || {};
    if (!def.filters.currency) {
      // FUTURE: mix in our go-to currency filter
    }//ﬁ
    if (!def.filters.round) {
      /**
       * Usage:
       * - `{{someValue | round}}`
       * - `{{someValue | round(1)}}`
       * - `{{someValue | round(2)}}`
       * - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
       * @param  {Ref} value
       * @param  {Number} accuracy
       * @param  {Boolean} chopTrailingZeros
       * @return {String}
       * - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
       * > The following is a modified version of:
       * > vue-round-filter@1.1.2
       * > c. 2016, Damian Martyniak (ISC License)
       * > https://github.com/rascada/vue-round-filter/blob/6529a384758b67ed54884b77e03dd73cc3dda215/index.js
       * > Originally updated for explicitness, to ensure Vue ≥2.x compatibility,
       * > and to change default usage to always preserve decimal accuracy.  It
       * > could likely wind up with additional updates over time.
       * > All edits are MIT licensed. Copyright (c) Mike McNeil, 2018-present
       */
      def.filters.round = function (value, accuracy, chopTrailingZeros) {
        if (typeof value !== 'number') {
          return ('' + value);
        }//•
        var result = value.toFixed(accuracy);
        if (chopTrailingZeros) {
          // (don't keep decimal accuracy, just chop off those trailing zeros)
          return ('' + (+result));
        } else {
          // (keep decimal accuracy)
          return ('' + result);
        }
      };//ƒ
    }//ﬁ

    // Wrap and verify methods:
    def.methods = def.methods || {};
    _.each(_.keys(def.methods), function (methodName) {
      if (!_.isFunction(def.methods[methodName])) {
        throw new Error('Unexpected definition for Vue method `'+methodName+'`.  Expecting a function, but got "'+def.methods[methodName]+'"');
      }

      var isArrowFunction;
      try {
        var asString = def.methods[methodName].toString();
        isArrowFunction = asString.match(/^\s*\(\s*/) || asString.match(/^\s*async\s*\(\s*/);
      } catch (err) {
        console.warn('Consistency violation: Encountered unexpected error when attempting to verify that Vue method `'+methodName+'` is not an arrow function.  (What browser is this?!)  Anyway, error details:', err);
      }

      if (isArrowFunction) {
        throw new Error('Unexpected definition for Vue method `'+methodName+'`.  Vue methods cannot be specified as arrow functions, because then you wouldn\'t have access to `this` (i.e. the Vue vm instance).  Please use a function like `function(){…}` or `async function(){…}` instead.');
      }

      // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
      // FUTURE:
      // Inject a wrapper function in order to provide more advanced / cleaner error handling.
      // (especially for AsyncFunctions)
      // ```
      // var _originalMethod = def.methods[methodName];
      // def.methods[methodName] = function(){
      //
      //   var rawResult;
      //   var originalCtx = this;
      //   (function(proceed){
      //     if (_originalMethod.constructor.name === 'AsyncFunction') {
      //       rawResult = _originalMethod.apply(originalCtx, arguments);
      //       // The result of an AsyncFunction is always a promise:
      //       rawResult.catch(function(err) {
      //         proceed(err);
      //       });//_∏_
      //       rawResult.then(function(actualResult){
      //         return proceed(undefined, actualResult);
      //       });
      //     }
      //     else {
      //       try {
      //         rawResult = _originalMethod.apply(originalCtx, arguments);
      //       } catch (err) { return proceed(err); }
      //       return proceed(undefined, rawResult);
      //     }
      //   })(function(err, actualResult){//eslint-disable-line no-unused-vars
      //     if (err) {
      //       // FUTURE: perform more advanced error handling here
      //       throw err;
      //     }
      //
      //     // Otherwise do nothing.
      //
      //   });//_∏_  (†)
      //
      //   // For compatibility, return the raw result.
      //   return rawResult;
      //
      // };//ƒ
      // ```
      // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -


    });//∞
  }


  //   ██████╗ ██╗      ██████╗ ██████╗  █████╗ ██╗         ███████╗██╗   ██╗███████╗███╗   ██╗████████╗███████╗
  //  ██╔════╝ ██║     ██╔═══██╗██╔══██╗██╔══██╗██║         ██╔════╝██║   ██║██╔════╝████╗  ██║╚══██╔══╝██╔════╝
  //  ██║  ███╗██║     ██║   ██║██████╔╝███████║██║         █████╗  ██║   ██║█████╗  ██╔██╗ ██║   ██║   ███████╗
  //  ██║   ██║██║     ██║   ██║██╔══██╗██╔══██║██║         ██╔══╝  ╚██╗ ██╔╝██╔══╝  ██║╚██╗██║   ██║   ╚════██║
  //  ╚██████╔╝███████╗╚██████╔╝██████╔╝██║  ██║███████╗    ███████╗ ╚████╔╝ ███████╗██║ ╚████║   ██║   ███████║
  //   ╚═════╝ ╚══════╝ ╚═════╝ ╚═════╝ ╚═╝  ╚═╝╚══════╝    ╚══════╝  ╚═══╝  ╚══════╝╚═╝  ╚═══╝   ╚═╝   ╚══════╝
  //

  ///////////////////////////////////////////////////////////////////////////////////////////
  // Capture uncaught errors (and trigger a fatal error if appropriate)
  //
  // A global error handler, "catcher of the uncaught", first of its name, and
  // bane to bugs.  This (normally-invisible) gutter lurks at the bottom of the
  // screen, but then springs to life if any uncaught errors are detected.
  //
  // > Out of the box, this behavior is deactivated any time Sails is running in
  // > in the "production" environment (`sails.config.environment !== 'production')
  // > (e.g. your local development machine, your staging server).  In order to do
  // > this, it checks `SAILS_LOCALS._environment`, a special variable set by the
  // > boilerplate "custom" hook (included in all Sails apps using the "Web App"
  // > template).  Note that this behavior is also disabled if `window.SAILS_LOCALS`
  // > is not set (e.g. any pages where "exposeLocalsToBrowser" is not in use).
  // > Finally, this behavior is also disabled if jQuery is not available.
  //
  // ---------------------------------------------------------------------------------------
  // In the off change your app is using a tool like Bugsnag/Sentry/Rollbars in
  // environments OTHER THAN PRODUCTION, and you'd like to disable this behavior,
  // the following section of code can simply be removed.
  // ---------------------------------------------------------------------------------------
  ///////////////////////////////////////////////////////////////////////////////////////////
  if ($ && typeof window !== 'undefined' && window.SAILS_LOCALS && window.SAILS_LOCALS._environment !== 'production') {

    var _displayErrorOverlay = function(errorSummary){

      if ($('#parasails-error-handler').length === 0) {
        // Very first error:
        $('<div id="parasails-error-handler">'+
          '<div role="error-handler-content">'+
            '<h1>Whoops</h1>'+
            '<p>'+
              '<span role="summary">An unexpected client-side error occurred.</span><br/>'+
              '<pre>'+_.escape(_.trunc(errorSummary, {length: 350}))+'</pre>'+
              '<span>Please check your browser\'s JavaScript console for further details.</span><br/>'+
              '<small>This message will not be displayed in production.  '+
              'If you\'re unsure, <a href="https://sailsjs.com/support">ask for help</a>.</small><br/>'+
              '<small>'+_.escape(new Date())+'</small>'+
            '</p>'+
          '</div>'+
        '</div>')
        .css({
          position: 'fixed',
          bottom: '0',
          height: '100%',
          width: '100%',
          'z-index': '9000',
          display: 'table',
          'background': 'radial-gradient(circle, rgba(0,0,0,0.98) 0%, rgba(35,8,8,0.87) 80%, rgba(20,5,5,0.85) 100%)',
          // (Thanks cssgradient.io!)
        })
        .appendTo('body');

        $('#parasails-error-handler [role="error-handler-content"]').css({
          display: 'table-cell',
          'vertical-align': 'middle',
          'text-align': 'center'
        });

        $('#parasails-error-handler [role="error-handler-content"] *').css({
          'font-family': '\'Consolas\', \'Courier\', \'courier\', serif',
          color: 'white'
        });

        $('#parasails-error-handler [role="error-handler-content"] small').css({
          color: '#cccccc'
        });

        $('#parasails-error-handler [role="error-handler-content"] pre').css({
          color: '#ff5555',
          display: 'block',
          'background': '#112f1f',
          'white-space': 'pre-wrap',
          'padding': '10px',
          'margin-left': 'auto',
          'margin-right': 'auto',
          'max-width': '500px',
          'min-width': '280px',
          'font-size': '11px'
        });

        $('#parasails-error-handler [role="error-handler-content"] a').css({
          'text-decoration': 'underline',
          color: '#cccccc'
        });

      } else {
        // Subsequent errors:
        $('#parasails-error-handler [role="summary"]')
        .text('Multiple unexpected client-side errors occurred.');
      }

      // Returning `true` would suppress the actual uncaught error from
      // showing up in the JavaScript console. But we don't want to play
      // with fire for now.  Just getting access to this is enough.
      // We allow the error to continue to be uncaught.
      return false;
    };//ƒ

    // Bind top-level error handler on the window.
    //
    // This NEVER prevents the error from continuing to be uncaught- it's just here
    // to ensure that if any JS errors occur, we notice them immediately, even if we
    // don't have Chrome dev tools open.
    //
    // For more info about `window.onerror`, see:
    // https://developer.mozilla.org/en-US/docs/Web/API/GlobalEventHandlers/onerror
    var originalWindowOnError = window.onerror;
    window.onerror = function(message, scriptSrc, lineNo, charNo, err) {
      _displayErrorOverlay(err&&err.message? err.message : message);
      if (_.isFunction(originalWindowOnError)) {
        return originalWindowOnError(message, scriptSrc, lineNo, charNo, err);
      }
    };//œ   </ on uncaught error >

    // Bind top-level uncaught promise rejection handler.
    // > https://developer.mozilla.org/en-US/docs/Web/Events/unhandledrejection
    // (Only works in desktop Chrome as of Oct 2018, but over time, this will
    // hopefully get better.  In the mean time, doesn't hurt anything, and it's
    // only for development anyway.)
    window.addEventListener('unhandledrejection', function (event) {
      _displayErrorOverlay(event&&event.reason? event.reason : event);
    });//œ  </ on unhandled promise rejection >

    // Configure Vue to share its beforeMount errors (and others) with us.
    // > https://vuejs.org/v2/api/#errorHandler
    Vue.config.errorHandler = function (err, unusedVm, errorSourceDisplayName) {
      if (err && err.message) {
        if (errorSourceDisplayName === 'render function') {
          err.message = 'In the HTML template (during render): '+err.message;
        } else {
          err.message = 'In '+errorSourceDisplayName+': '+err.message;
        }
      } else {
        var _originalNotActuallyErr = err;
        err = new Error(_originalNotActuallyErr);
        err.raw = _originalNotActuallyErr;
      }
      console.error(err);
      _displayErrorOverlay(err);
    };//ƒ

    // Also those Vue warnings -- but we'll treat them like errors because
    // we're serious about code quality.  (Plus, early detection of bugs
    // and typos saves so much time down the road!)
    // > `trace` is the component hierarchy trace
    Vue.config.warnHandler = function (msg, unusedVm, unusedTrace) {
      throw new Error(
        msg + '\n\n'+
        'Expand this error and check out the stack trace for more info.'
      );
    };//ƒ

  }//ﬁ

  //  ███████╗██╗  ██╗██████╗  ██████╗ ██████╗ ████████╗███████╗
  //  ██╔════╝╚██╗██╔╝██╔══██╗██╔═══██╗██╔══██╗╚══██╔══╝██╔════╝
  //  █████╗   ╚███╔╝ ██████╔╝██║   ██║██████╔╝   ██║   ███████╗
  //  ██╔══╝   ██╔██╗ ██╔═══╝ ██║   ██║██╔══██╗   ██║   ╚════██║
  //  ███████╗██╔╝ ██╗██║     ╚██████╔╝██║  ██║   ██║   ███████║
  //  ╚══════╝╚═╝  ╚═╝╚═╝      ╚═════╝ ╚═╝  ╚═╝   ╚═╝   ╚══════╝
  //

  /**
   * Module exports
   */

  parasails = {};


  /**
   * parasails.util
   *
   * Direct references to all registered utility methods from userland.
   *
   * @type {Dictionary}
   */

  parasails.util = {};


  /**
   * registerUtility()
   *
   * Build a callable utility function, then attach it to the global namespace
   * so that it can be accessed later via `.require()`.
   *
   * @param {String} utilityName
   * @param {Function} def
   */

  parasails.registerUtility = function(utilityName, def){

    // Usage
    if (!utilityName) { throw new Error('1st argument (utility name) is required'); }
    if (!def) { throw new Error('2nd argument (utility function definition) is required'); }
    if (!_.isFunction(def)) { throw new Error('2nd argument (utility function definition) should be a function'); }

    // Build callable utility
    var callableUtility = def;
    callableUtility.name = utilityName;

    // Attach to global cache
    _exportOnGlobalCache(utilityName, callableUtility);

    // Also expose on `parasails.util`
    parasails.util[utilityName] = callableUtility;

  };


  /**
   * registerConstant()
   *
   * Attach a constant to the global namespace so that it can be accessed
   * later via `.require()`.
   *
   * @param {String} constantName
   * @param {Ref} value
   */

  parasails.registerConstant = function(constantName, value){

    // Usage
    if (!constantName) { throw new Error('1st argument (constant name) is required'); }
    if (value === undefined) { throw new Error('2nd argument (the constant value) is required'); }

    // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
    // FUTURE: deep-freeze constant, if supported
    // (https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Object/freeze)
    // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

    // Attach to global cache
    _exportOnGlobalCache(constantName, value);

  };



  /**
   * registerComponent()
   *
   * Define a Vue component.
   *
   * @param {String} componentName   [In camelCase]
   * @param {Dictionary} def
   *
   * @returns {Ref}  [new vue component for this page]
   */

  parasails.registerComponent = function(componentName, def){

    // Expose extra methods on component def, if jQuery is available.
    _exposeBonusMethods(def, 'component');

    // Make sure none of the specified Vue methods are defined with any naughty arrow functions.
    _wrapMethodsAndVerifyNoArrowFunctions(def, 'component');

    // Wrap the `mounted` LC in order to decorate the top-level element with
    // a sniffable marker that can be unambiguously styled via a global selector
    // in the corresponding stylesheet for the component.
    var customMountedLC;
    if (def.mounted) {
      customMountedLC = def.mounted;
    }//ﬁ
    def.mounted = function(){

      // Attach `parasails-component="…"` DOM attribute to allow for painless
      // selecting from an optional, corresponding per-component stylesheet.
      this.$el.setAttribute('parasails-component', _.kebabCase(componentName));

      // Then call the original, custom "mounted" function, if there was one.
      if (customMountedLC) {
        customMountedLC.apply(this, []);
      }
    };//ƒ

    // Attach `goto` method, for convenience.
    if (def.methods && def.methods.goto) { throw new Error('Component definition contains `methods` with a `goto` key-- but you\'re not allowed to override that'); }
    if (def.methods && def.methods.gotoAndReplaceHistory) { throw new Error('Component definition contains `methods` with a `gotoAndReplaceHistory` key-- but you\'re not allowed to override that'); }
    def.methods = def.methods || {};
    def.methods.goto = function (rootRelativeUrl){
      // If the Bowser browser detection library is installed
      // (https://github.com/lancedikson/bowser/releases), check whether
      // we're in Edge or IE, in which case we'll add some special handling
      // for an edge case in `onbeforeunload` behavior.
      var isIEOrEdgeBrowser = bowser && (bowser.name === 'Internet Explorer' || bowser.name === 'Microsoft Edge');
      if(!isIEOrEdgeBrowser) {
        window.location = rootRelativeUrl;
      } else {
        try {
          // IE/Edge prefers "window.location.href"
          window.location.href = rootRelativeUrl;
        } catch(unusedErr) {
          // More helpful error message for unavoidable error during onbeforeunload edge case in IE/Edge
          throw new Error('`goto` failed in Edge or IE! If navigation was cancelled in `beforeunload`, you can probably ignore this message (see https://stackoverflow.com/questions/1509643/unknown-exception-when-cancelling-page-unload-with-location-href/1510074#1510074).');
        }
      }
    };
    def.methods.gotoAndReplaceHistory = function (rootRelativeUrl){
      window.location.replace(rootRelativeUrl);
    };

    // Finally, register as a global Vue component.
    Vue.component(componentName, def);

  };


  /**
   * require()
   *
   * Require a utility function or constant from the global namespace.
   *
   * @param {String} moduleName
   * @returns {Ref}  [e.g. the callable utility function, or the value of the constant]
   * @throws {Error} if no such module has been registered
   */

  parasails.require = function(moduleName) {

    // Usage
    if (!moduleName) { throw new Error('1st argument (module name -- i.e. the name of a utility or constant) is required'); }

    // Fetch from global cache
    _ensureGlobalCache();
    if (parasails._cache[moduleName] === undefined) {
      var err = new Error('No utility or constant is registered under that name (`'+moduleName+'`)');
      err.name = 'RequireError';
      err.code = 'MODULE_NOT_FOUND';
      throw err;
    }
    return parasails._cache[moduleName];

  };


  /**
   * registerPage()
   *
   * Define a page script, if applicable for the current contents of the DOM.
   *
   * @param {String} pageName
   * @param {Dictionary} def
   *
   * @returns {Ref}  [new vue app thing for this page]
   */

  parasails.registerPage = function(pageName, def){

    // Usage
    if (!pageName) { throw new Error('1st argument (page name) is required'); }
    if (!def) { throw new Error('2nd argument (page script definition) is required'); }

    // Don't look for a matching DOM element (by "id") within anything that has `parasails-has-no-page-script`
    // FUTURE: Move this check a bit further below, probably without defining the variable, and just add another avast (aka early return)
    var isWithinIgnoredElements;
    if ($) {
      // Note that, luckily, this works even without waiting for the DOM to be ready according to jQuery.  (i.e. $(()=>{ … }))
      isWithinIgnoredElements = (
        $('#'+pageName).parents().filter('[parasails-has-no-page-script]')
      ).length >= 1;
    } else {
      // For simplicity, this check is skipped if jQuery is not available.
      // FUTURE: Implement with vanilla JS here
      isWithinIgnoredElements = false;
    }

    // Only actually build+load this page script if it is relevant for the current contents of the DOM.
    if (!window.document.getElementById(pageName) || isWithinIgnoredElements) { return; }

    // Spinlock
    if (didAlreadyLoadPageScript) { throw new Error('Cannot load page script (`'+pageName+') because a page script has already been loaded on this page.'); }
    didAlreadyLoadPageScript = true;

    // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
    // FUTURE: Maybe set `parasails._mountedPage = pageName;` and use that in the `goto` method of components to still
    // do the check for a virtualPageRegExp and allow it to conditionaly do a "soft" client-side navigation to avoid page
    // reload. (Remember: There's only ever one registered page script mounted in the DOM when you're using parasails.)
    // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

    // Automatically set `el`
    if (def.el) { throw new Error('Page script definition contains `el`, but you\'re not allowed to override that'); }
    def.el = '#'+pageName;

    // Expose extra methods, if jQuery is available.
    _exposeBonusMethods(def, 'page script');

    // Make sure none of the specified Vue methods are defined with any naughty arrow functions.
    _wrapMethodsAndVerifyNoArrowFunctions(def, 'page script');

    // If bowser and jQuery are both around, sniff the user agent and determine
    // some additional information about the user agent device accessing the DOM.
    var bowserSniffClasses = '';
    var SNIFFER_CSS_CLASS_PREFIX = 'detected-';
    if (bowser && $) {

      if (bowser.tablet||bowser.mobile) {
        bowserSniffClasses += ' '+SNIFFER_CSS_CLASS_PREFIX+'mobile';
        // ^^Note: "detected-mobile" means ANY mobile OS/device (handset or tablet)
        //  [?] https://github.com/lancedikson/bowser/tree/6bbdaf99f0b36cf3a7b8a14feb0aa60d86d7e0dd#device-flags
        if (bowser.ios) {
          bowserSniffClasses += ' '+SNIFFER_CSS_CLASS_PREFIX+'ios';
        } else if (bowser.android) {
          bowserSniffClasses += ' '+SNIFFER_CSS_CLASS_PREFIX+'android';
        } else if (bowser.windowsphone) {
          bowserSniffClasses += ' '+SNIFFER_CSS_CLASS_PREFIX+'windowsphone';
        }

        if (bowser.tablet) {
          bowserSniffClasses += ' '+SNIFFER_CSS_CLASS_PREFIX+'tablet';
        } else if (bowser.mobile) {
          bowserSniffClasses += ' '+SNIFFER_CSS_CLASS_PREFIX+'handset';
        }
      }
      else {
        // Otherwise we're not on a mobile OS/browser/device.
        // But we can at least get a bit more intell on what's up:
        if (bowser.mac) {
          bowserSniffClasses += ' '+SNIFFER_CSS_CLASS_PREFIX+'mac';
        } else if (bowser.windows) {
          bowserSniffClasses += ' '+SNIFFER_CSS_CLASS_PREFIX+'windows';
        } else if (bowser.linux) {
          bowserSniffClasses += ' '+SNIFFER_CSS_CLASS_PREFIX+'linux';
        }
      }
    }//ﬁ

    // If we have jQuery available, then as soon as the DOM is ready, and if
    // appropriate based on browser device sniffing, attach special classes to
    // the <body> element.
    if ($ && bowserSniffClasses) {
      $(function(){
        $('body').addClass(bowserSniffClasses);
      });//_∏_
    }//ﬁ

    // Wrap the `mounted` LC:
    var customMountedLC;
    if (def.mounted) {
      customMountedLC = def.mounted;
    }//ﬁ
    def.mounted = function(){

      // Similar to above, attach special classes to the page script's top-level
      // DOM element, now that it has been mounted (again, only if appropriate
      // based on browser device sniffing, and only if jQuery is available.)
      if ($ && bowserSniffClasses) {
        this.$get().addClass(bowserSniffClasses);
      }//ﬁ

      // Then call the original, custom "mounted" function, if there was one.
      if (customMountedLC) {
        customMountedLC.apply(this, []);
        // ^FUTURE: consider whether it's worth dealing with the possibility
        // of uncaught promise rejections here (b/c it might be an `async function`!)
      }
    };//ƒ

    // Now, for convenience, automatically add built-in defaults to our `data`:
    if (def.data && def.data.pageName) { throw new Error('Page script definition contains `data` with a `pageName` key, but you\'re not allowed to override that'); }
    def.data = _.extend({
      pageName: pageName,
      _: _,
    }, def.data||{});
    if (bowser && !def.data.hasOwnProperty('bowser')) {
      def.data.bowser = bowser;
    }

    // And, as of Parasails ≥0.9, automatically merge in the contents of SAILS_LOCALS, if present.
    // > (this is so that you don't have to include boilerplate code inside beforeMount of page scripts
    // > to merge in data from the server)
    if (typeof window !== 'undefined' && _.isObject(window.SAILS_LOCALS) && !_.isArray(window.SAILS_LOCALS) && !_.isFunction(window.SAILS_LOCALS)) {
      _.extend(def.data, window.SAILS_LOCALS);
    }

    // Attach `goto` method, for convenience.
    if (def.methods && def.methods.goto) { throw new Error('Page script definition contains `methods` with a `goto` key-- but you\'re not allowed to override that'); }
    if (def.methods && def.methods.gotoAndReplaceHistory) { throw new Error('Page script definition contains `methods` with a `gotoAndReplaceHistory` key-- but you\'re not allowed to override that'); }
    def.methods = def.methods || {};
    if (VueRouter) {
      var _virtualPagesRegExp = def.virtualPagesRegExp;

      // The following inline function definition exists purely to avoid
      // duplication of code in `goto` and `gotoAndReplaceHistory` below:
      var _goto = function($router, rootRelativeUrlOrOpts, replaceHistory) {
        // FUTURE: add support for using '../' without reloading the page
        // (even though it doesn't technicaly match the regexp)
        if (!_virtualPagesRegExp || (_.isString(rootRelativeUrlOrOpts) && !rootRelativeUrlOrOpts.match(_virtualPagesRegExp))) {
          if (replaceHistory) {
            window.location.replace(rootRelativeUrlOrOpts);
          } else {
            window.location = rootRelativeUrlOrOpts;
          }
        } else {
          if (replaceHistory) {
            // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
            // FUTURE: also look into something like this for cleaner handling
            // of back/forward navigation for things like permalinked modals:
            // • https://stackoverflow.com/a/29227026/486547
            //
            // gotoAndReplaceHistory() works great for handling client-side
            // redirects within an afterNavigate function, but it doesn't work
            // great for removing stuff from the history stack.  Because that's
            // impossible, tbh.  So to truly solve that, you need a much more
            // opinionated solution (see above link & treeline2 for examples).
            // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
            return $router.replace(rootRelativeUrlOrOpts);
          } else {
            return $router.push(rootRelativeUrlOrOpts);
          }
        }
      };//ƒ
      def.methods.goto = function (rootRelativeUrlOrOpts){
        return _goto(this.$router, rootRelativeUrlOrOpts);
      };//ƒ
      def.methods.gotoAndReplaceHistory = function (rootRelativeUrlOrOpts){
        return _goto(this.$router, rootRelativeUrlOrOpts, true);
      };//ƒ
    }
    else {
      def.methods.goto = function (){ throw new Error('Cannot use .goto() method because, at the time when this page script was registered, VueRouter did not exist on the page yet. (If you\'re using Sails, please check dependency loading order in pipeline.js and make sure VueRouter is getting brought in before `parasails`.)'); };
      def.methods.gotoAndReplaceHistory = function (){ throw new Error('Cannot use .gotoAndReplaceHistory() method because, at the time when this page script was registered, VueRouter did not exist on the page yet. (If you\'re using Sails, please check dependency loading order in pipeline.js and make sure VueRouter is getting brought in before `parasails`.)'); };
    }

    // If virtualPages-related stuff was specified, check usage and tolerate shorthand.
    if (def.virtualPages === undefined) {
      if (def.virtualPagesRegExp) {
        def.virtualPages = true;
      }
    } else if (_.isObject(def.virtualPages) && !_.isArray(def.virtualPages) && !_.isFunction(def.virtualPages)) {
      throw new Error('This usage of `virtualPages` (as a dictionary) is no longer supported.  Instead, please use `virtualPages: true`.  [?] https://sailsjs.com/support');
      // (^^ old implementation removed in https://github.com/mikermcneil/parasails/commit/20af5992097de788b58ae2cb517675f235798879)
    } else if (!_.isBoolean(def.virtualPages)) {
      throw new Error('Cannot use `virtualPages` because the specified value doesn\'t match any recognized meaning.  Please specify either `true` (for the default handling) or a dictionary of client-side routing rules.');
    }//ﬁ
    if (def.virtualPages && def.router) { throw new Error('Cannot specify both `virtualPages` AND an actual Vue `router`!  Use one or the other.'); }
    if (def.router && !VueRouter) { throw new Error('Cannot use `router`, because that depends on the Vue Router.  But `VueRouter` does not exist on the page yet.  (If you\'re using Sails, please check dependency loading order in pipeline.js and make sure the VueRouter plugin is getting brought in before `parasails`.)'); }
    if (!def.virtualPages && def.html5HistoryMode !== undefined) { throw new Error('Cannot specify `html5HistoryMode` without also specifying `virtualPages`!'); }
    if (!def.virtualPages && def.beforeEach !== undefined) { throw new Error('Cannot specify `beforeEach` without also specifying `virtualPages`!'); }
    if ((def.beforeNavigate || def.afterNavigate) && def.virtualPages !== true) { throw new Error('Cannot specify `beforeNavigate` or `afterNavigate` unless you set `virtualPages: true`!'); }

    // If `virtualPages: true` was specified, then use reasonable defaults:
    //
    // > Note: This assumes that, somewhere within the parent page's template, there is:
    // > ```
    // > <router-view></router-view>
    // > ```
    if (def.virtualPages === true) {
      if (!VueRouter) { throw new Error('Cannot use `virtualPages`, because it depends on the Vue Router.  But `VueRouter` does not exist on the page yet.  (If you\'re using Sails, please check dependency loading order in pipeline.js and make sure the VueRouter plugin is getting brought in before `parasails`.)'); }
      if (def.beforeEach !== undefined) { throw new Error('Cannot specify `virtualPages: true` AND `beforeEach` at the same time!'); }
      if (!def.virtualPagesRegExp && def.html5HistoryMode === 'history') { throw new Error('If `html5HistoryMode: \'history\'` is specified, then virtualPagesRegExp must also be specified!'); }
      if (def.virtualPagesRegExp && !_.isRegExp(def.virtualPagesRegExp)) { throw new Error('Invalid `virtualPagesRegExp`: If specified, this must be a regular expression -- e.g. `/^\/manage\/access\/?([^\/]+)?/`'); }
      if (def.html5HistoryMode === undefined) {
        if (def.virtualPagesRegExp) {
          def.html5HistoryMode = 'history';
        } else {
          def.html5HistoryMode = 'hash';
        }
      } else if (def.html5HistoryMode !== 'history' && def.html5HistoryMode !== 'hash') { throw new Error('Invalid `html5HistoryMode`: If specified, this must be either "history" or "hash".'); }

      // Check for <router-view> element
      // (to provide a better error msg if it was omitted)
      var customBeforeMountLC;
      if (def.beforeMount) {
        customBeforeMountLC = def.beforeMount;
      }//ﬁ
      def.beforeMount = function(){

        // Inject additional code to check for <router-view> element:
        // console.log('this.$find(\'router-view\').length', this.$find('router-view').length);
        if (this.$find('router-view').length === 0) {
          throw new Error(
            'Cannot mount this page with `virtualPages: true` because no '+
            '<router-view> element exists in this page\'s HTML.\n'+
            'Please be sure the HTML includes:\n'+
            '\n'+
            '```\n'+
            '<router-view></router-view>\n'+
            '```\n'
          );
        }//•

        // Then call the original, custom "beforeMount" function, if there was one.
        if (customBeforeMountLC) {
          customBeforeMountLC.apply(this, []);
        }
      };//ƒ

      if (def.methods._handleVirtualNavigation) {
        throw new Error('Could not use `virtualPages: true`, because a conflicting `_handleVirtualNavigation` method is defined.  Please remove it, or do something else.');
      }

      // Set up local variables to refer to things in `def`, since it will be changing below.
      var pathMatchingRegExp;
      if (def.html5HistoryMode === 'history') {
        pathMatchingRegExp = def.virtualPagesRegExp;
      } else {
        pathMatchingRegExp = /.*/;
      }

      var beforeNavigate = def.beforeNavigate;
      var afterNavigate = def.afterNavigate;

      // Now modify the definition's methods and remove all relevant top-level props understood
      // by parasails (but not by Vue.js) to avoid creating any weird additional dependence on
      // parasails features beyond the expected usage.
      def.methods = _.extend(def.methods||{}, {
        _handleVirtualNavigation: function(virtualPageSlug){

          if (beforeNavigate) {
            var resultFromBeforeNavigate = beforeNavigate.apply(this, [ virtualPageSlug ]);
            if (resultFromBeforeNavigate === false) {
              return false;
            }//•
          }

          this.virtualPageSlug = virtualPageSlug;

          // console.log('navigate!  Got:', arguments);
          // console.log('Navigated. (Set `this.virtualPageSlug=\''+virtualPageSlug+'\'`)');

          if (afterNavigate) {
            afterNavigate.apply(this, [ virtualPageSlug ]);
          }

        }
      });

      // Automatically attach `virtualPageSlug` to `data`, for convenience.
      if (def.data && def.data.virtualPageSlug !== undefined && !_.isString(def.data.virtualPageSlug)) {
        throw new Error('Page script definition contains `data` with a `virtualPageSlug` key, but you\'re not allowed to set that yourself unless you use a string.  (And this is set to a non-string value: '+def.data.virtualPageSlug+')');
      } else if (def.data && def.data.virtualPageSlug === undefined) {
        def.data = _.extend({
          virtualPageSlug: undefined
        }, def.data||{});
      }//ﬁ

      // Now we'll replace `virtualPages` in our def with the thing that VueRouter actually expects:
      def = _.extend({
        router: new VueRouter({
          mode: def.html5HistoryMode,
          routes: [
            {
              path: '*',
              component: (function(){
                var vueComponentDef = {
                  render: function(){},
                  beforeRouteUpdate: function (to,from,next){
                    // this.$emit('navigate', to.path); <<old way
                    var path = to.path;
                    var matches = path.match(pathMatchingRegExp);
                    if (!matches) {
                      var err =new Error('Could not match current URL path (`'+path+'`) as a virtual page.  Please check the `virtualPagesRegExp` -- e.g. `/^\/foo\/bar\/?([^\/]+)?/`');
                      err.code = 'E_DID_NOT_MATCH_REGEXP';
                      throw err;
                    }//•

                    // console.log('this.$parent', this.$parent);
                    this.$parent._handleVirtualNavigation(matches[1]||'');
                    // this.$emit('navigate', {
                    //   rawPath: path,
                    //   virtualPageSlug: matches[1]||''
                    // });
                    return next();
                  },
                  mounted: function(){
                    // this.$emit('navigate', this.$route.path); <<old way
                    var path = this.$route.path;
                    var matches = path.match(pathMatchingRegExp);
                    if (!matches) {
                      var err =new Error('Could not match current URL path (`'+path+'`) as a virtual page.  Please check the `virtualPagesRegExp` -- e.g. `/^\/foo\/bar\/?([^\/]+)?/`');
                      err.code = 'E_DID_NOT_MATCH_REGEXP';
                      throw err;
                    }//•

                    this.$parent._handleVirtualNavigation(matches[1]||'');
                    // this.$emit('navigate', {
                    //   rawPath: path,
                    //   virtualPageSlug: matches[1]||''
                    // });
                  }
                };
                // Expose extra methods on virtual page script, if jQuery is available.
                _exposeBonusMethods(vueComponentDef, 'virtual page');

                // Make sure none of the specified Vue methods are defined with any naughty arrow functions.
                _wrapMethodsAndVerifyNoArrowFunctions(vueComponentDef, 'virtual page');

                return vueComponentDef;
              })()
            }
          ],
        })
      }, _.omit(def, ['virtualPages', 'virtualPagesRegExp', 'html5HistoryMode', 'beforeNavigate', 'afterNavigate']));
    }//ﬁ  </ def has `virtualPages` enabled >

    // Construct Vue instance for this page script.
    var vm = new Vue(def);

    return vm;

  };//ƒ



  /**
   * parasails.util.isMobile()
   *
   * Detect whether this is being accessed from a mobile browser/OS, which might
   * be a handset device (iPhone, etc.) OR a tablet device (iPad, etc.)
   *
   * > This relies on `bowser.mobile||bowser.tablet`.
   *
   * @returns {Boolean}
   */
  parasails.util.isMobile = function(){

    // If `bowser` is not available, throw an error.
    if(!bowser) {
      throw new Error('Cannot detect mobile-ness, because `bowser` global does not exist on the page yet. '+
        '(If you\'re using Sails, please check dependency loading order in pipeline.js and make sure '+
        'the Bowser library is getting brought in before `parasails`. If you have not included Bowser '+
        'in your project, you can find it at https://github.com/lancedikson/bowser/releases)');
    }

    return (!!bowser.mobile) || (!!bowser.tablet);

  };//ƒ
  // An extra alias, for convenience:
  parasails.isMobile = parasails.util.isMobile;


  /**
   * parasails.util.isValidEmailAddress()
   *
   * Determine whether the given value is a valid email address.
   *
   * > This code is taken directly from validator.js / anchor.
   * > It is implemented as a built-in, organic utility that may be overwritten
   * > in userland if desired.
   *
   * @param {String} value
   *
   * @returns {Boolean}
   */

  parasails.util.isValidEmailAddress = function(value){
    if (!value || typeof value !== 'string') { return false; }
    /* eslint-disable */
    return (function(){function _isByteLength(str,min,max){var len=encodeURI(str).split(/%..|./).length-1;return len>=min&&(typeof max==='undefined'||len<=max)}
    var emailUserUtf8Part=/^[a-z\d!#\$%&'\*\+\-\/=\?\^_`{\|}~\u00A0-\uD7FF\uF900-\uFDCF\uFDF0-\uFFEF]+$/i;var quotedEmailUserUtf8=/^([\s\x01-\x08\x0b\x0c\x0e-\x1f\x7f\x21\x23-\x5b\x5d-\x7e\u00A0-\uD7FF\uF900-\uFDCF\uFDF0-\uFFEF]|(\\[\x01-\x09\x0b\x0c\x0d-\x7f\u00A0-\uD7FF\uF900-\uFDCF\uFDF0-\uFFEF]))*$/i;function _isFQDN(str){var options={require_tld:!0,allow_underscores:!1,allow_trailing_dot:!1};if(options.allow_trailing_dot&&str[str.length-1]==='.'){str=str.substring(0,str.length-1)}
    var parts=str.split('.');if(options.require_tld){var tld=parts.pop();if(!parts.length||!/^([a-z\u00a1-\uffff]{2,}|xn[a-z0-9-]{2,})$/i.test(tld)){return!1}}
    for(var part,i=0;i<parts.length;i++){part=parts[i];if(options.allow_underscores){if(part.indexOf('__')>=0){return!1}
    part=part.replace(/_/g,'')}
    if(!/^[a-z\u00a1-\uffff0-9-]+$/i.test(part)){return!1}
    if(/[\uff01-\uff5e]/.test(part)){return!1}
    if(part[0]==='-'||part[part.length-1]==='-'||part.indexOf('---')>=0){return!1}}
    return!0};return function(str){var parts=str.split('@'),domain=parts.pop(),user=parts.join('@');var lower_domain=domain.toLowerCase();if(lower_domain==='gmail.com'||lower_domain==='googlemail.com'){user=user.replace(/\./g,'').toLowerCase()}
    if(!_isByteLength(user,0,64)||!_isByteLength(domain,0,256)){return!1}
    if(!_isFQDN(domain)){return!1}
    if(user[0]==='"'){user=user.slice(1,user.length-1);return quotedEmailUserUtf8.test(user)}
    var pattern=emailUserUtf8Part;var user_parts=user.split('.');for(var i=0;i<user_parts.length;i++){if(!pattern.test(user_parts[i])){return!1}}
    return!0}})()(value);
    /* eslint-enable */
  };//ƒ


  /**
   * parasails.utils
   *
   * A permanent alias for `parasails.util`.
   *
   * (Everyone gets these mixed up.)
   *
   * @type {:Dictionary}
   */

  parasails.utils = parasails.util;



  return parasails;

}, function (global, factory) {
  var Vue;
  var _;
  var VueRouter;
  var $;
  var bowser;

  //˙°˚°·.
  //‡CJS  ˚°˚°·˛
  if (typeof exports === 'object' && typeof module !== 'undefined') {
    var _require = require;// eslint-disable-line no-undef
    var _module = module;// eslint-disable-line no-undef
    // required deps:
    Vue = _require('vue');
    _ = _require('lodash');
    // optional deps:
    try { VueRouter = _require('vue-router'); } catch (e) { if (e.code === 'MODULE_NOT_FOUND') {/* ok */} else { throw e; } }
    try { $ = _require('jquery'); } catch (e) { if (e.code === 'MODULE_NOT_FOUND') {/* ok */} else { throw e; } }
    try { bowser = _require('bowser'); } catch (e) { if (e.code === 'MODULE_NOT_FOUND') {/* ok */} else { throw e; } }
    // export:
    _module.exports = factory(Vue, _, VueRouter, $, bowser);
  }
  //˙°˚°·
  //‡AMD ˚¸
  else if(typeof define === 'function' && define.amd) {// eslint-disable-line no-undef
    // Register as an anonymous module.
    define([], function () {// eslint-disable-line no-undef
      // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
      // FUTURE: maybe use optional dep. loading here instead?
      // e.g.  `function('vue', 'lodash', 'vue-router', 'jquery')`
      // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
      // required deps:
      if (!global.Vue) { throw new Error('`Vue` global does not exist on the page yet. (If you\'re using Sails, please check dependency loading order in pipeline.js and make sure the Vue.js library is getting brought in before `parasails`.)'); }
      Vue = global.Vue;
      if (!global._) { throw new Error('`_` global does not exist on the page yet. (If you\'re using Sails, please check dependency loading order in pipeline.js and make sure the Lodash library is getting brought in before `parasails`.)'); }
      _ = global._;
      // optional deps:
      VueRouter = global.VueRouter || undefined;
      $ = global.$ || global.jQuery || undefined;
      bowser = global.bowser || undefined;

      // So... there's not really a huge point to supporting AMD here--
      // except that if you're using it in your project, it makes this
      // module fit nicely with the others you're using.  And if you
      // really hate globals, I guess there's that.
      // ¯\_(ツ)_/¯
      return factory(Vue, _, VueRouter, $, bowser);
    });//ƒ
  }
  //˙°˚˙°·
  //‡NUDE ˚°·˛
  else {
    // required deps:
    if (!global.Vue) { throw new Error('`Vue` global does not exist on the page yet. (If you\'re using Sails, please check dependency loading order in pipeline.js and make sure the Vue.js library is getting brought in before `parasails`.)'); }
    Vue = global.Vue;
    if (!global._) { throw new Error('`_` global does not exist on the page yet. (If you\'re using Sails, please check dependency loading order in pipeline.js and make sure the Lodash library is getting brought in before `parasails`.)'); }
    _ = global._;
    // optional deps:
    VueRouter = global.VueRouter || undefined;
    $ = global.$ || global.jQuery || undefined;
    bowser = global.bowser || undefined;
    // export:
    if (global.parasails) { throw new Error('Conflicting global (`parasails`) already exists!'); }
    global.parasails = factory(Vue, _, VueRouter, $, bowser);
  }
});//…)
