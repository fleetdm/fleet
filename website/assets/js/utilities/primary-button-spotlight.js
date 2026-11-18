/**
 * primary-button-spotlight.js
 *
 * Moves the hover spotlight on primary buttons to follow the cursor. Buttons opt in
 * through the `.primary-button()` LESS mixin, which sets the `--primary-button` custom
 * property; this sets `--spotlight-x` and `--spotlight-y` on whichever one is hovered.
 */
(() => {
  if (!window.matchMedia || !window.matchMedia('(hover: hover)').matches) {
    return;
  }
  let pendingEvent = null;
  document.addEventListener('pointermove', (event) => {
    if (event.pointerType !== 'mouse') {
      return;
    }
    if (!pendingEvent) {
      window.requestAnimationFrame(() => {
        let latestEvent = pendingEvent;
        pendingEvent = null;
        // Custom properties inherit, so a button's label also reports --primary-button.
        // The button is the outermost element (within a few levels) that has it.
        let button = null;
        let element = latestEvent.target;
        for (let depth = 0; element && element.nodeType === 1 && depth < 5; depth++, element = element.parentElement) {
          if (window.getComputedStyle(element).getPropertyValue('--primary-button').trim() === '1') {
            button = element;
          } else if (button) {
            break;
          }
        }
        if (button) {
          let bounds = button.getBoundingClientRect();
          button.style.setProperty('--spotlight-x', (latestEvent.clientX - bounds.left) + 'px');
          button.style.setProperty('--spotlight-y', (latestEvent.clientY - bounds.top) + 'px');
        }
      });
    }
    pendingEvent = event;
  }, { passive: true });
})();
