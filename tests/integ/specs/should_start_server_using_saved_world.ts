import {
  assertEquals,
  assertNotEquals,
} from "https://deno.land/std@0.83.0/testing/asserts.ts";

import api, { login } from "../lib/api.ts";
import {
  launchExistingWorld,
  launchNewWorld,
  stopServer,
} from "../lib/easy.ts";
import { usingFakeMinecraftServer } from "../lib/env.ts";
import { getState } from "../lib/mcproto.ts";

console.log("Login");
const accessToken = await login("admin", "password");

const worldName = `test-${Date.now()}`;

console.log("Launch server");
await launchNewWorld(accessToken, worldName);

const worldInfo = await api("GET /api/v1/worldinfo", accessToken);
assertEquals(worldInfo["version"], "1.20.1");
assertEquals(worldInfo["worldName"], worldName);

let worldVersion: string | null = null;

if (usingFakeMinecraftServer()) {
  const state = await getState();
  assertEquals(state.serverState!.worldVersionPrev, "");
  worldVersion = state.serverState!.worldVersion as string;
}

console.log("Stop server");
await stopServer(accessToken);

console.log("Launch server with another world (to purge cache)");
await launchNewWorld(accessToken, `test-${Date.now()}`);
if (usingFakeMinecraftServer()) {
  const state = await getState();
  assertNotEquals(state.serverState!.worldVersionPrev, worldVersion);
}
await stopServer(accessToken);

console.log("Relaunch server");
await launchExistingWorld(accessToken, worldName);

if (usingFakeMinecraftServer()) {
  const state = await getState();
  assertEquals(state.serverState!.worldVersionPrev, worldVersion);
}

console.log("Stop server");
await stopServer(accessToken);

console.log("Success");
