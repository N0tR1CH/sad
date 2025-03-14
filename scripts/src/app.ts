import htmxLib from "htmx.org/dist/htmx.esm.js";
import Socket from "./socket.js";
import _hyperscript from "hyperscript.org";
import EasyMDELib from "easymde";
import Alpine from "alpinejs";
import Swal, { SweetAlertOptions } from "sweetalert2/src/sweetalert2.js";
import imageViewer from "./image_viewer.js";
import routesTable from "./routes_table.js";

declare global {
  interface Window {
    htmx: typeof htmx;
    Alpine: typeof Alpine;
    Swal: typeof Swal;
    sweetConfirm: (
      el: HTMLElement,
      options: SweetAlertOptions,
    ) => Promise<void>;
    runToast: (icon: string, title: string) => Promise<void>;
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

  if (window.location.href.startsWith("https://localhost")) {
    Socket.init();
  }

  // Alpine configuration
  Alpine.data("imageViewer", imageViewer);
  Alpine.data("routesTable", routesTable);
  Alpine.start();

  _hyperscript.browserInit();

  // toast helper
  window.runToast = async (icon: string, title: string): Promise<void> => {
    Swal.mixin({
      toast: true,
      position: "top-end",
      showConfirmButton: false,
      timer: 3000,
      timerProgressBar: true,
      didOpen: (toast) => {
        toast.onmouseenter = Swal.stopTimer;
        toast.onmouseleave = Swal.resumeTimer;
      },
    }).fire({ icon, title });
  };

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
