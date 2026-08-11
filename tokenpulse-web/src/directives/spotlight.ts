import type { Directive } from 'vue';

const spotlight: Directive<HTMLElement> = {
  mounted(element) {
    element.classList.add('spotlight-surface');
    element.addEventListener('pointermove', updateSpotlight);
  },
  beforeUnmount(element) {
    element.removeEventListener('pointermove', updateSpotlight);
  },
};

function updateSpotlight(event: PointerEvent): void {
  const element = event.currentTarget as HTMLElement;
  const bounds = element.getBoundingClientRect();
  element.style.setProperty('--spotlight-x', `${event.clientX - bounds.left}px`);
  element.style.setProperty('--spotlight-y', `${event.clientY - bounds.top}px`);
}

export default spotlight;
