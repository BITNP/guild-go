export function usePanelTilt(panel: Ref<HTMLElement | null>) {
  let targetX = 0;
  let targetY = 0;
  let pointerX = 0;
  let pointerY = 0;

  let frame = 0;
  let lastFrame = performance.now();
  let reduceMotion = false;

  function updatePointerTilt(event: PointerEvent) {
    targetX = (event.clientX / Math.max(window.innerWidth, 1)) * 2 - 1;
    targetY = (event.clientY / Math.max(window.innerHeight, 1)) * 2 - 1;
  }

  function animate(now: number) {
    const delta = Math.min(0.05, (now - lastFrame) / 1000);

    lastFrame = now;
    const damping = 1 - Math.exp(-delta * 5);

    pointerX += (targetX - pointerX) * damping;
    pointerY += (targetY - pointerY) * damping;

    panel.value?.style.setProperty("--panel-tilt-x", `${pointerY * 0.8}deg`);
    panel.value?.style.setProperty("--panel-tilt-y", `${-pointerX * 1.15}deg`);

    frame = requestAnimationFrame(animate);
  }

  onMounted(() => {
    reduceMotion = window.matchMedia(
      "(prefers-reduced-motion: reduce)",
    ).matches;

    window.addEventListener("pointermove", updatePointerTilt, {
      passive: true,
    });

    if (!reduceMotion) {
      lastFrame = performance.now();
      frame = requestAnimationFrame(animate);
    }
  });

  onBeforeUnmount(() => {
    window.removeEventListener("pointermove", updatePointerTilt);
    cancelAnimationFrame(frame);
  });
}
