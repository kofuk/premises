import { assertEquals } from "https://deno.land/std@0.83.0/testing/asserts.ts";

import api, { login, params, streamEvent } from "../lib/api.ts";
import codes from "../lib/codes.ts";

console.log("Login");
const cookie = await login("user1", "password1");

const worldName = `test-${(Math.random() * 1000000) >> 0}`;

console.log("Launch server");
await api(
  "POST /api/launch",
  cookie,
  params({
    "machine-type": "2g",
    "server-version": "1.20.1",
    "prefer-detect": "true",
    "world-source": "new-world",
    "world-name": worldName,
    "seed": "",
    "level-type": "default",
  }),
);

console.log("Wait server launched");
for (let i = 0; i < 18; i++) {
  console.log("...");
  await new Promise((resolve) => setTimeout(resolve, 10 * 1000));

  const { pageCode } = await streamEvent("/api/streaming/events", cookie);
  if (pageCode === codes.PAGE.RUNNING) {
    break;
  } else if (pageCode === codes.PAGE.MANUAL_SETUP) {
    throw new Error("Unexpected page");
  }
}
const { pageCode } = await streamEvent("/api/streaming/events", cookie);
assertEquals(pageCode, codes.PAGE.RUNNING);

// It takes some time to initialize world info data.
console.log("Check launched world");
for (let i = 0; i < 10; i++) {
  try {
    await api("GET /api/worldinfo", cookie);
    break;
  } catch (_: unknown) {
    // Check error later
  }

  await new Promise((resolve) => setTimeout(resolve, 2 * 1000));
}
const worldInfo = await api("GET /api/worldinfo", cookie);
assertEquals(worldInfo["version"], "1.20.1");
assertEquals(worldInfo["worldName"], worldName);

console.log("Stop server");
await api("POST /api/stop", cookie);

console.log("Wait server stopped");
for (let i = 0; i < 18; i++) {
  console.log("...");
  await new Promise((resolve) => setTimeout(resolve, 10 * 1000));

  const { pageCode } = await streamEvent("/api/streaming/events", cookie);
  if (pageCode === codes.PAGE.LAUNCH) {
    break;
  } else if (pageCode === codes.PAGE.MANUAL_SETUP) {
    throw new Error("Unexpected page");
  }
}

console.log("Success");
