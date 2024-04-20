import { assertEquals } from "https://deno.land/std@0.83.0/testing/asserts.ts";

import api, { login } from "../lib/api.ts";
import {
  createBaseConfig,
  stopServer,
  waitServerLaunched,
} from "../lib/easy.ts";
import { usingFakeMinecraftServer } from "../lib/env.ts";
import { getState } from "../lib/mcproto.ts";

console.log("Login");
const cookie = await login("user1", "password1");

const worldName = `test-${Date.now()}`;

console.log("Launch server");
const id = await createBaseConfig(cookie, worldName);

await api("PUT /api/config", cookie, {
  id,
  serverPropOverride: {
    "initial-enabled-packs": "vanilla,update_1_21,bundle",
  },
});
await api("POST /api/launch", cookie, { id });

await waitServerLaunched(cookie);

const worldInfo = await api("GET /api/worldinfo", cookie);
assertEquals(worldInfo["version"], "1.20.1");
assertEquals(worldInfo["worldName"], worldName);

if (usingFakeMinecraftServer()) {
  const state = await getState();
  assertEquals(
    state.serverState!.serverProps["initial-enabled-packs"],
    "vanilla,update_1_21,bundle",
  );
}

console.log("Stop server");
await stopServer(cookie);

console.log("Success");
