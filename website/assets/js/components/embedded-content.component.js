/**
 * <embedded-content>
 * -----------------------------------------------------------------------------
 * An embedded third-party iframe (e.g. a YouTube video) that is only loaded if
 * the visitor has consented to cookies, or when they choose to load it.
 *
 * > The build-static-content script replaces `<div purpose="embedded-content">`
 * > elements in Markdown content with this component.
 *
 * @type {Component}
 *
 * -----------------------------------------------------------------------------
 */

parasails.registerComponent('embeddedContent', {
  //  ╔═╗╦═╗╔═╗╔═╗╔═╗
  //  ╠═╝╠╦╝║ ║╠═╝╚═╗
  //  ╩  ╩╚═╚═╝╩  ╚═╝
  props: [
    'src', // Required: The URL of the embedded content.
    'title', // Optional: The title of the iframe.
  ],

  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: function (){
    return {
      hasCookieConsent: false,
      wasLoadedManually: false,
      canOpenCookieSettings: false,
      // The CookieYes category that embedded content is gated on.
      requiredConsentCategory: 'functional',
    };
  },

  //  ╦ ╦╔╦╗╔╦╗╦
  //  ╠═╣ ║ ║║║║
  //  ╩ ╩ ╩ ╩ ╩╩═╝
  template: `
  <div purpose="embedded-content">
    <iframe :src="iframeSrc" :title="title" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" allowfullscreen v-if="hasCookieConsent || wasLoadedManually"></iframe>
    <div purpose="embedded-content-overlay" v-else>
      <div purpose="overlay-title">This {{contentType}} is blocked</div>
      <div purpose="overlay-text" v-if="canOpenCookieSettings">Load it manually, or update your cookie settings to allow {{isVideo ? 'video playback' : 'embedded content'}}.</div>
      <div purpose="overlay-text" v-else>Load it manually to view this {{contentType}}.</div>
      <div purpose="overlay-actions">
        <button type="button" purpose="overlay-button" @click="clickLoadContent()">Load {{contentType}}</button>
        <button type="button" purpose="overlay-button" @click="clickOpenCookieSettings()" v-if="canOpenCookieSettings">Cookie settings</button>
      </div>
    </div>
  </div>
  `,

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  mounted: function(){
    // The CookieYes script is only added to the page in production. Without it there is no consent to ask for, so embedded content always loads.
    this.hasCookieConsent = !document.getElementById('cookieyes') || this._getHasCookieConsent();
    this.canOpenCookieSettings = typeof window.revisitCkyConsent === 'function';
    // CookieYes may finish loading after this component is mounted, and visitors can change their preferences without reloading the page.
    document.addEventListener('cookieyes_banner_load', this._handleCookieBannerLoad);
    document.addEventListener('cookieyes_consent_update', this._handleCookieConsentUpdate);
  },
  beforeDestroy: function(){
    document.removeEventListener('cookieyes_banner_load', this._handleCookieBannerLoad);
    document.removeEventListener('cookieyes_consent_update', this._handleCookieConsentUpdate);
  },

  //  ╔═╗╔═╗╔╦╗╔═╗╦ ╦╔╦╗╔═╗╔╦╗
  //  ║  ║ ║║║║╠═╝║ ║ ║ ║╣  ║║
  //  ╚═╝╚═╝╩ ╩╩  ╚═╝ ╩ ╚═╝═╩╝
  computed: {
    isVideo: function(){
      return /youtube(-nocookie)?\.com\/embed\//.test(this.src);
    },
    contentType: function(){
      return this.isVideo ? 'video' : 'content';
    },
    iframeSrc: function(){
      // The visitor already clicked once to load a blocked video, so start playing it instead of making them click again.
      if(this.isVideo && this.wasLoadedManually) {
        return this.src + (this.src.includes('?') ? '&' : '?') + 'autoplay=1';
      }
      return this.src;
    },
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    clickLoadContent: function(){
      this.wasLoadedManually = true;
    },
    clickOpenCookieSettings: function(){
      window.revisitCkyConsent();
    },
    _getHasCookieConsent: function(){
      if(typeof window.getCkyConsent === 'function') {
        return !!window.getCkyConsent().categories[this.requiredConsentCategory];
      }
      // CookieYes has not finished loading, so check the cookie it stores this category's consent in. (e.g. "cookieyes-functional=yes")
      return document.cookie.split('; ').includes('cookieyes-' + this.requiredConsentCategory + '=yes');
    },
    _handleCookieBannerLoad: function(event){
      this.canOpenCookieSettings = typeof window.revisitCkyConsent === 'function';
      if(event.detail && event.detail.categories && event.detail.categories[this.requiredConsentCategory]) {
        this.hasCookieConsent = true;
      }
    },
    _handleCookieConsentUpdate: function(event){
      // Content that is already loaded is left in place if consent is withdrawn.
      if(event.detail && Array.isArray(event.detail.accepted) && event.detail.accepted.includes(this.requiredConsentCategory)) {
        this.hasCookieConsent = true;
      }
    },
  },
});
