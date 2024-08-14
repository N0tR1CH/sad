import { WebSocketServer } from "ws";

const port = 8000;
const wsc = { port };
const wss = new WebSocketServer(wsc);

wss.on("open", () => {
  console.log("open");
});

wss.on("close", () => {
  console.log("disconnected");
});

wss.on("error", () => {
  console.log("error");
});
