import htmxLib from "htmx.org/dist/htmx.esm.js";
import Socket from "./socket.js";
import _hyperscript from "hyperscript.org";
import EasyMDELib from "easymde";
import Alpine from "alpinejs";
import Swal, { SweetAlertOptions } from "sweetalert2/src/sweetalert2.js";

declare global {
  interface Window {
    htmx: typeof htmx;
    Alpine: typeof Alpine;
    Swal: typeof Swal;
    sweetConfirm: (
      el: HTMLElement,
      options: SweetAlertOptions,
    ) => Promise<void>;
  }
}

window.addEventListener("DOMContentLoaded", async (): Promise<void> => {
  window.htmx = htmxLib;
  window.EasyMDE = EasyMDELib;
  window.Alpine = Alpine;
  window.Swal = Swal;
  window.sweetConfirm = async (el, options) => {
    const alertResult = await Swal.fire(options);
    if (alertResult.isConfirmed) {
      el.dispatchEvent(new Event("confirmed"));
    }
  };

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
