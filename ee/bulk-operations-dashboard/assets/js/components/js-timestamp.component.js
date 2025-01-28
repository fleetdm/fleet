/**
 * <js-timestamp>
 * -----------------------------------------------------------------------------
 * A human-readable, self-updating "timeago" timestamp, with some special rules:
 *
 * • Within 24 hours, displays in "timeago" format.
 * • Within a month, displays month, day, and time of day.
 * • Within a year, displays just the month and day.
 * • Older/newer than that, displays the month and day with the full year.
 *
 * @type {Component}
 * -----------------------------------------------------------------------------
 */

parasails.registerComponent('jsTimestamp', {

  //  ╔═╗╦═╗╔═╗╔═╗╔═╗
  //  ╠═╝╠╦╝║ ║╠═╝╚═╗
  //  ╩  ╩╚═╚═╝╩  ╚═╝
  props: [
    'at',// « The JS timestamp to format
    'short',// « Whether to shorten the formatted date by not including the time of day (may only be used with timeago, and even then only applicable in certain situations)
    'format',// « one of: 'calendar', 'timeago' (defaults to 'timeago'.  Otherwise, the "calendar" format displays data as US-style calendar dates with a four-character year, separated by dashes.  In other words: "MM-DD-YYYY")
  ],

  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: function (){
    return {
      formatType: undefined,
      formattedTimestamp: '',
      interval: undefined
    };
  },

  //  ╦ ╦╔╦╗╔╦╗╦
  //  ╠═╣ ║ ║║║║
  //  ╩ ╩ ╩ ╩ ╩╩═╝
  template: `
  <span>{{formattedTimestamp}}</span>
  `,

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    if (this.at === undefined) {
      throw new Error('Incomplete usage of <js-timestamp>:  Please specify `at` as a JS timestamp (i.e. epoch ms, a number).  For example: `<js-timestamp :at="something.createdAt">`');
    }
    if(this.format === undefined) {
      this.formatType = 'timeago';
    } else  {
      if(!_.contains(['calendar', 'timeago'], this.format)) { throw new Error('Unsupported `format` ('+this.format+') passed in to the JS timestamp component! Must be either \'calendar\' or \'timeago\'.'); }
      this.formatType = this.format;
    }

    // If timeago timestamp, update the timestamp every minute.
    if(this.formatType === 'timeago') {
      this._formatTimeago();
      this.interval = setInterval(async()=>{
        try {
          this._formatTimeago();
          await this.forceRender();
        } catch (err) {
          err.raw = err;
          throw new Error('Encountered unexpected error while attempting to automatically re-render <js-timestamp> in the background, as the seconds tick by.  '+err.message);
        }
      },60*1000);//œ
    }

    // If calendar timestamp, just set it the once.
    // (Also don't allow usage with `short`.)
    if(this.formatType === 'calendar') {
      this.formattedTimestamp = moment(this.at).format('MM-DD-YYYY');
      if (this.short) {
        throw new Error('Invalid usage of <js-timestamp>:  Cannot use `short="true"` at the same time as `format="calendar"`.');
      }
    }
  },

  beforeDestroy: function() {
    if(this.formatType === 'timeago') {
      clearInterval(this.interval);
    }
  },

  watch: {
    at: function() {
      // Render to account for after-mount programmatic changes to `at`.
      if(this.formatType === 'timeago') {
        this._formatTimeago();
      } else if(this.formatType === 'calendar') {
        this.formattedTimestamp = moment(this.at).format('MM-DD-YYYY');
      } else {
        throw new Error();
      }
    }
  },


  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {

    _formatTimeago: function() {
      var now = new Date().getTime();
      var timeDifference = Math.abs(now - this.at);

      // If the timestamp is less than a day old, format as time ago.
      if(timeDifference < 1000*60*60*24) {
        this.formattedTimestamp = moment(this.at).fromNow();
      } else {
        // If the timestamp is less than a month-ish old, we'll include the
        // time of day in the formatted timestamp.
        let includeTime = !this.short && timeDifference < 1000*60*60*24*31;

        // If the timestamp is from a different year, we'll include the year
        // in the formatted timestamp.
        let includeYear = moment(now).format('YYYY') !== moment(this.at).format('YYYY');

        this.formattedTimestamp = moment(this.at).format('MMMM DD'+(includeYear ? ', YYYY' : '')+(includeTime ? ' [at] h:mma' : ''));
      }

    }

  }

});
