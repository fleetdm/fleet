/**
 * <modal>
 * -----------------------------------------------------------------------------
 * A modal dialog pop-up.
 *
 * > Be careful adding other Vue.js lifecycle callbacks in this file!  The
 * > finnicky combination of Vue transitions and bootstrap modal animations used
 * > herein work, and are very well-tested in practical applications.  But any
 * > changes to that specific cocktail could be unpredictable, with unsavory
 * > consequences.
 *
 * @type {Component}
 *
 * @event close   [emitted when the closing process begins]
 * @event opened  [emitted when the opening process is completely done]
 * -----------------------------------------------------------------------------
 */

parasails.registerComponent('modal', {
  //  ╔═╗╦═╗╔═╗╔═╗╔═╗
  //  ╠═╝╠╦╝║ ║╠═╝╚═╗
  //  ╩  ╩╚═╚═╝╩  ╚═╝
  props: [
    'hideCloseButton'//« removes the default "x" button
  ],

  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: function (){
    return {
      // Spinlock used for preventing trying to close the bootstrap modal more than once.
      // (in practice it doesn't seem to hurt anything if it tries to close more than once,
      // but still.... better safe than sorry!)
      _bsModalIsAnimatingOut: false,

      isMobileSafari: false,//« more on this below
      originalScrollPosition: undefined,//« more on this below
    };
  },

  //  ╦ ╦╔╦╗╔╦╗╦
  //  ╠═╣ ║ ║║║║
  //  ╩ ╩ ╩ ╩ ╩╩═╝
  template: `
  <transition name="modal" v-on:leave="leave" v-bind:css="false">
    <div class="modal fade" tabindex="-1" role="dialog">
      <div class="petticoat"></div>
      <div class="modal-dialog custom-width position-relative" role="document" purpose="modal-dialog">
        <div class="modal-content" purpose="modal-content">
          <button type="button" style="top: 5px; right: 0; font-size: 28px; line-height: 1;" class="py-2 px-3 position-absolute" data-dismiss="modal" aria-label="Close" purpose="modal-close-button" v-if="!hideCloseButton">&times;</button>
          <slot></slot>
        </div><!-- /.modal-content -->
      </div><!-- /.modal-dialog -->
    </div><!-- /.modal -->
  </transition>
  `,

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    // If this is mobile safari, make note of it.
    this.isMobileSafari = (typeof bowser !== 'undefined') && bowser.mobile && bowser.safari;
    // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
    // ^^So there's a bug in mobile safari that misplaces the caret when the keyboard opening
    // causes the page to scroll, so we need to do some special tricks to keep it from getting ugly.
    // It's only in iOS 11... we think.  Hopefully it will be fixed.
    // In the mean time, we have to get wacky.
    //
    // > More info about the bug here:
    // > https://github.com/twbs/bootstrap/issues/24835#issuecomment-345974819
    // > https://stackoverflow.com/questions/46567233/how-to-fix-the-ios-11-input-element-in-fixed-modals-bug?rq=1
    //
    // FUTURE: maybe the bug will be fixed and we can remove this someday?
    // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

    if(this.isMobileSafari) {
      // Get our original scroll position before opening the modal and save it for later.
      this.originalScrollPosition = $(window).scrollTop();
    }
  },
  mounted: function(){
    // ^^ Note that this is not an `async function`.
    // This is just to be safe, since the timing here is a little tricky w/ the
    // animations and the fact that we're integrating with Bootstrap's modal.
    // (That said, it might work fine-- just hasn't been extensively tested.)

    // Immediately call out to the Bootstrap modal and tell it to show itself.
    $(this.$el).modal({
      // Set the modal backdrop to the 'static' option, which means it doesn't close the modal
      // when clicked.
      backdrop: 'static',
      show: true
    });

    // Attach listener for underlying custom modal closing event,
    // and when that happens, have Vue emit a custom "close" event.
    // (Note: This isn't just for convenience-- it's crucial that
    // the parent logic can use this event to update its scope.)
    $(this.$el).on('hide.bs.modal', ()=>{

      // Undo any mobile safari workarounds we may have added.
      // (i.e. shed the wackiness)
      if(this.isMobileSafari) {
        // Remove style overrides on our modal dialog.
        $(this.$el).css({
          'overflow-y': '',
          'position': '',
          'left': '',
          'top': '',
        });

        // Beckon to our siblings so they come out of hiding
        this.$get().parent().children().not(this.$el).css({
          'display': ''
        });

        // Scroll to our original position when the modal was summoned.
        window.scrollTo(0, this.originalScrollPosition);
      }//ﬁ

      this._bsModalIsAnimatingOut = true;
      this.$emit('close');

    });//œ

    // Attach listener for underlying custom modal "opened" event,
    // and when that happens, have Vue emit our own custom "opened" event.
    // This is so we know when the entry animation has completed, allows
    // us to do cool things like auto-focus the first input in a form modal.
    $(this.$el).on('shown.bs.modal', ()=>{

      // If this is mobile safari, let's get wacky.
      if(this.isMobileSafari) {
        // Scroll to the top of the page.
        window.scrollTo(0, 0);
        // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
        // ^^FUTURE: Don't actually do this -- instead, try setting `top` of the
        // modal to whatever the original scrollTop of our window was.  This
        // eliminates the need for auto-scrolling to the top and ripping you out
        // of the context you were in before the modal opens.  It would also allow
        // us to keep the nice animation when opening/closing modals on iOS.
        // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

        // Hide siblings to lop off any extra space at the bottom.
        this.$get().parent().children().not(this.$el).css({
          'display': 'none'
        });
        // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
        // ^^FUTURE: Instead of just hiding siblings, which isn't perfect and won't
        // always work for everyone, try grabbing outerHeight of the modal element
        // and using that to set an explicit height for the body.
        // (but also be sure to handle the case where the body is short!)
        // But for now, this should work as long as we have sticky footer styles.
        // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

        // Hard code some style overrides on our modal dialog.
        // Without these, it gets weird.
        $(this.$el).css({
          'overflow-y': 'auto!important',
          'position': 'absolute',
          'left': '0',
          'top': '0',
        });
      }//ﬁ

      // Focus our "focus-first" field, if relevant.
      // (but not on mobile, because it can get weird)
      if(typeof bowser !== 'undefined' && !bowser.mobile && this.$find('[focus-first]').length > 0) {
        this.$focus('[focus-first]');
      }

      this.$emit('opened');
      $(this.$el).off('shown.bs.modal');
    });//ƒ
  },
  // ^Note that there is no `beforeDestroy()` lifecycle callback in this
  // component. This is on purpose, since the timing vs. `leave()` gets tricky.

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {

    leave: function (el, done) {
      // > Note: This function signature comes from Vue.js's transition system.
      // > It will likely be replaced with `async function (el){…}` in a future
      // > release of Vue/Sails.js (i.e. no callback argument).

      // If this shutting down was spawned by the bootstrap modal's built-in logic,
      // then we'll have already begun animating the modal shut.  So we check our
      // spinlock to make sure.  If it turns out that we HAVEN'T started that process
      // yet, then we go ahead and start it now.
      if (!this._bsModalIsAnimatingOut) {
        $(this.$el).modal('hide');
      }//ﬁ

      // When the bootstrap modal finishes animating into nothingness, unbind all
      // the DOM events used by bootstrap, and then call `done()`, which passes
      // control back to Vue and lets it finish the job (i.e. afterLeave).
      //
      // > Note that the other lifecycle events like `destroyed` were actually
      // > already fired at this point.
      // >
      // > Also note that, since we're potentially long past the `destroyed` point
      // > of the lifecycle here, we can't call `.$emit()` anymore either.  So,
      // > for example, we wouldn't be able to emit a "fullyClosed" event --
      // > because by the time it'd be appropriate to emit the Vue event, our
      // > context for triggering it (i.e. the relevant instance of this component)
      // > will no longer be capable of emitting custom Vue events (because by then,
      // > it is no longer "reactive").
      // >
      // > For more info, see:
      // > https://github.com/vuejs/vue-router/issues/1302#issuecomment-291207073
      $(this.$el).on('hidden.bs.modal', ()=>{
        $(this.$el).off('hide.bs.modal');
        $(this.$el).off('hidden.bs.modal');
        $(this.$el).off('shown.bs.modal');
        done();
      });//_∏_

    },

  }
});
