import htmxLib from "htmx.org/dist/htmx.esm.js";
import Socket from "./socket.js";
import _hyperscript from "hyperscript.org";
import EasyMDELib from "easymde";
import Alpine from "alpinejs";

declare global {
  interface Window {
    htmx: typeof htmx;
    Alpine: typeof Alpine;
  }
}

window.addEventListener("DOMContentLoaded", (): void => {
  window.htmx = htmxLib;
  window.EasyMDE = EasyMDELib;
  window.Alpine = Alpine;

  Socket.init();
  Alpine.start();
  _hyperscript.browserInit();

  // eslint-disable-next-line @typescript-eslint/ban-ts-comment
  // @ts-ignore
  window.htmx.config.globalViewTransitions = true;
});
