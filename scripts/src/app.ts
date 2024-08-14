import htmxLib from "htmx.org/dist/htmx.esm.js";
import Socket from "./socket.js";
import _hyperscript from "hyperscript.org";

window.addEventListener("DOMContentLoaded", (): void => {
  Socket.init();
  window.htmx = htmxLib;
  _hyperscript.browserInit();
});
