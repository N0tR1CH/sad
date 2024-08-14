interface ISocket {
  socket: WebSocket | null;
  socketURI: string;
  init: () => void;
  onOpen: () => void;
  onClose: () => void;
  onError: (e: Event) => void;
}

const Socket: ISocket = {
  socket: null,
  socketURI: "ws://localhost:8000/",

  init: () => {
    Socket.socket = new WebSocket(Socket.socketURI);
    Socket.socket.addEventListener("open", Socket.onOpen);
    Socket.socket.addEventListener("close", Socket.onClose);
  },

  onOpen: () => {
    console.log("connected");
  },

  onClose: () => {
    console.log("disconnected");
    if (Socket.socket.readyState === WebSocket.CLOSED) {
      setTimeout(() => {
        location.reload();
      }, 2000);
    }
  },

  onError: (e: Event) => {
    console.error("Websocket error observed:", e);
  },
};

export default Socket;
