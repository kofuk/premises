import { assertEquals } from "https://deno.land/std@0.83.0/testing/asserts.ts";

import api, { login } from "../lib/api.ts";
import { initConfig, stopServer, waitServerLaunched } from "../lib/easy.ts";
import { usingFakeMinecraftServer } from "../lib/env.ts";
import { getState } from "../lib/mcproto.ts";

console.log("Login");
const accessToken = await login("admin", "password");

const worldName = `test-${Date.now()}`;

console.log("Launch server");
await initConfig(accessToken, worldName);

await api("PUT /api/v1/config", accessToken, {
  serverPropOverride: {
    "initial-enabled-packs": "vanilla,update_1_21",
  },
});
await api("POST /api/v1/launch", accessToken);

await waitServerLaunched(accessToken);

const worldInfo = await api("GET /api/v1/worldinfo", accessToken);
assertEquals(worldInfo["version"], "1.20.1");
assertEquals(worldInfo["worldName"], worldName);

if (usingFakeMinecraftServer()) {
  const state = await getState();
  assertEquals(
    state.serverState!.serverProps["initial-enabled-packs"],
    "vanilla,update_1_21",
  );
}

console.log("Stop server");
await stopServer(accessToken);

console.log("Success");
