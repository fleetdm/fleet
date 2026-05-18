parasails.registerPage('how-fleet-fits-your-stack', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    //…
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function() {
    let detailSections = Array.from(document.querySelectorAll('[purpose="detail-section"]'));
    let navLinks = Array.from(document.querySelectorAll('[purpose="detail-nav"] a'));

    // Highlight the pill matching the section nearest the top of the viewport.
    let setActive = () => {
      let triggerY = window.innerHeight * 0.3;
      let activeId = null;
      for (let i = 0; i < detailSections.length; i++) {
        let rect = detailSections[i].getBoundingClientRect();
        if (rect.top <= triggerY) {
          activeId = detailSections[i].id;
        }
      }
      navLinks.forEach((link) => {
        if (link.dataset.target === activeId) {
          link.classList.add('is-active');
        } else {
          link.classList.remove('is-active');
        }
      });
    };

    let ticking = false;
    let onScroll = () => {
      if (!ticking) {
        window.requestAnimationFrame(() => {
          setActive();
          ticking = false;
        });
        ticking = true;
      }
    };
    window.addEventListener('scroll', onScroll, { passive: true });
    setActive();

    // Smooth-scroll for all in-page anchor links on this page (hub cards,
    // pill-nav, and "back to overview" links).
    let hashLinks = Array.from(document.querySelectorAll('#how-fleet-fits-your-stack a[href^="#"]'));
    hashLinks.forEach((link) => {
      link.addEventListener('click', (event) => {
        let href = link.getAttribute('href');
        if (!href || href === '#') { return; }
        let target = document.querySelector(href);
        if (!target) { return; }
        event.preventDefault();
        target.scrollIntoView({ behavior: 'smooth', block: 'start' });
        if (window.history && window.history.pushState) {
          window.history.pushState(null, '', href);
        }
      });
    });
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    //…
  }
});
