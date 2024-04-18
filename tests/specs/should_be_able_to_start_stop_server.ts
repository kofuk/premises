import { assertEquals } from "https://deno.land/std@0.83.0/testing/asserts.ts";

import api, { login } from "../lib/api.ts";
import { launchNewWorld, stopServer } from "../lib/easy.ts";
import { usingFakeMinecraftServer } from "../lib/env.ts";

console.log("Login");
const cookie = await login("user1", "password1");

const worldName = `test-${Date.now()}`;

console.log("Launch server");
await launchNewWorld(cookie, worldName);

console.log("Check launched world");
const worldInfo = await api("GET /api/worldinfo", cookie);
assertEquals(worldInfo["version"], "1.20.1");
assertEquals(worldInfo["worldName"], worldName);

if (usingFakeMinecraftServer()) {
  const state = await fetch("http://127.0.0.2:25565/state").then((resp) =>
    resp.json()
  );
  assertEquals(state.worldVersionPrev, "");
  assertEquals(state.version, "1.20.1");
}

console.log("Stop server");
await stopServer(cookie);

console.log("Success");
