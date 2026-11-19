import { Centrifuge } from "centrifuge";
import { authorized } from "./api";

export function assessmentChannel(owner: string, signal: AbortSignal) {
  const channel = "user:" + owner + ":assessments";
  async function token(path: string, body: object) {
    return (
      await authorized<{ token: string }>("/realtime/" + path, body, signal)
    ).token;
  }
  const ws = new URL("/connection/websocket", location.origin);
  ws.protocol = location.protocol === "https:" ? "wss:" : "ws:";
  const client = new Centrifuge(import.meta.env.VITE_WS_URL || ws.href, {
    getToken: () => token("connection-token", {}),
  });
  const subscription = client.newSubscription(channel, {
    getToken: () => token("subscription-token", { channel }),
  });
  return { client, subscription };
}
