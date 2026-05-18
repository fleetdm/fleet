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
    let prefersReducedMotion = window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    let isProgrammaticScroll = false;
    let currentScrollRaf = null;

    // Mark one pill active. `forcedId` overrides scroll-position detection
    // (used on click so the target pill lights up before the scroll lands).
    // Pass null to clear all pills.
    let setActive = (forcedId) => {
      let activeId = forcedId;
      if (activeId === undefined) {
        activeId = null;
        let triggerY = window.innerHeight * 0.3;
        for (let i = 0; i < detailSections.length; i++) {
          let rect = detailSections[i].getBoundingClientRect();
          if (rect.top <= triggerY) {
            activeId = detailSections[i].id;
          }
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

    // Scroll listener updates the active pill — paused during programmatic
    // scrolls so the pill doesn't twitch through every section on the way.
    let ticking = false;
    let onScroll = () => {
      if (isProgrammaticScroll) { return; }
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

    // Cancel any in-flight scroll animation (used on click and when the user
    // takes manual control via wheel/touch).
    let cancelAnimation = () => {
      if (currentScrollRaf !== null) {
        window.cancelAnimationFrame(currentScrollRaf);
        currentScrollRaf = null;
      }
      isProgrammaticScroll = false;
    };
    window.addEventListener('wheel', cancelAnimation, { passive: true });
    window.addEventListener('touchstart', cancelAnimation, { passive: true });

    // Custom smooth scroll: ease-out-cubic, 500ms. Snappier than the
    // browser default, especially for long distances from the ecosystem map.
    let smoothScrollTo = (target) => {
      cancelAnimation();
      if (prefersReducedMotion) {
        target.scrollIntoView({ block: 'start' });
        return;
      }
      let duration = 500;
      let scrollMargin = parseInt(window.getComputedStyle(target).scrollMarginTop, 10) || 0;
      let startY = window.pageYOffset;
      let targetY = target.getBoundingClientRect().top + window.pageYOffset - scrollMargin;
      let distance = targetY - startY;
      let startTime = performance.now();
      let easeOutCubic = (t) => 1 - Math.pow(1 - t, 3);

      isProgrammaticScroll = true;
      let step = (currentTime) => {
        if (!isProgrammaticScroll) { return; }
        let elapsed = currentTime - startTime;
        let progress = Math.min(elapsed / duration, 1);
        window.scrollTo(0, startY + distance * easeOutCubic(progress));
        if (progress < 1) {
          currentScrollRaf = window.requestAnimationFrame(step);
        } else {
          currentScrollRaf = null;
          isProgrammaticScroll = false;
        }
      };
      currentScrollRaf = window.requestAnimationFrame(step);
    };

    // Wire up every in-page hash link (ecosystem-map tiles, pill-nav,
    // "back to ecosystem map" links).
    let hashLinks = Array.from(document.querySelectorAll('#how-fleet-fits-your-stack a[href^="#"]'));
    hashLinks.forEach((link) => {
      link.addEventListener('click', (event) => {
        let href = link.getAttribute('href');
        if (!href || href === '#') { return; }
        let target = document.querySelector(href);
        if (!target) { return; }
        event.preventDefault();

        // Highlight the destination immediately so the pill doesn't tick
        // through each section as we scroll past them.
        let targetId = href.replace('#', '');
        let isDetailTarget = detailSections.some((s) => s.id === targetId);
        setActive(isDetailTarget ? targetId : null);

        smoothScrollTo(target);
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
