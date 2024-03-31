import { assertEquals } from "https://deno.land/std@0.83.0/testing/asserts.ts";

import api, { login } from "../lib/api.ts";
import {
  launchExistingWorld,
  launchNewWorld,
  stopServer,
} from "../lib/easy.ts";
import { usingFakeMinecraftServer } from "../lib/env.ts";

console.log("Login");
const cookie = await login("user1", "password1");

const worldName = `test-${Date.now()}`;

console.log("Launch server");
await launchNewWorld(cookie, worldName);

const worldInfo = await api("GET /api/worldinfo", cookie);
assertEquals(worldInfo["version"], "1.20.1");
assertEquals(worldInfo["worldName"], worldName);

let worldVersion: string | null = null;

if (usingFakeMinecraftServer()) {
  const state = await fetch("http://localhost:25565/state").then((resp) =>
    resp.json()
  );
  assertEquals(state.worldVersionPrev, "");
  worldVersion = state.worldVersion as string;
}

console.log("Stop server");
await stopServer(cookie);

console.log("Launch server with another world (to purge cache)");
await launchNewWorld(cookie, `test-${Date.now()}`);
await stopServer(cookie);

console.log("Relaunch server");
await launchExistingWorld(cookie, worldName);

if (usingFakeMinecraftServer()) {
  const state = await fetch("http://localhost:25565/state").then((resp) =>
    resp.json()
  );
  assertEquals(state.worldVersionPrev, worldVersion);
}

console.log("Stop server");
await stopServer(cookie);

console.log("Success");
