import { WebSocketServer } from "ws";

type WebSocketConfig = { port: number };

const port = 8000;
const wsc: WebSocketConfig = { port };
const wss: WebSocketServer = new WebSocketServer(wsc);

wss.on("open", () => {
  console.log("open");
});

wss.on("close", () => {
  console.log("disconnected");
});

wss.on("error", () => {
  console.log("error");
});
