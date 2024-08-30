import htmxLib from "htmx.org/dist/htmx.esm.js";
import Socket from "./socket.js";
import _hyperscript from "hyperscript.org";
import EasyMDELib from "easymde";
import Alpine from "alpinejs";
import Swal from "sweetalert2";

declare global {
  interface Window {
    htmx: typeof htmx;
    Alpine: typeof Alpine;
    Swal: typeof Swal;
  }
}

window.addEventListener("DOMContentLoaded", (): void => {
  window.htmx = htmxLib;
  window.EasyMDE = EasyMDELib;
  window.Alpine = Alpine;
  window.Swal = Swal;

  Socket.init();
  Alpine.start();
  _hyperscript.browserInit();

  // eslint-disable-next-line @typescript-eslint/ban-ts-comment
  // @ts-ignore
  window.htmx.config.globalViewTransitions = true;

  // Enable swap for 400 which helps with form errors
  document.body.addEventListener("htmx:beforeSwap", (e: CustomEvent): void => {
    if (e.detail.xhr.status === 400) {
      e.detail.shouldSwap = true;
      e.detail.isError = false;
    }
  });
});
